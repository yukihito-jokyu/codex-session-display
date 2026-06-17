package usecase

import (
	"codex-session-display/internal/domain/dto"
	"codex-session-display/internal/domain/model"
	"codex-session-display/internal/utils/logger"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"time"
)

type GetSessionDetailUseCase struct {
	sessionRepo SessionRepository
	cacheRepo   CacheRepository
	parser      SessionParser
}

type batchOutputResolution struct {
	OutputRecords []*model.TypedRecord
	Matched       int
	NullAssigned  int
	Discarded     int
	FallbackUsed  bool
}

type displayUnitKind string

const (
	displayUnitReasoning    displayUnitKind = "reasoning"
	displayUnitBatch        displayUnitKind = "batch"
	displayUnitWebSearch    displayUnitKind = "webSearch"
	displayUnitItemComplete displayUnitKind = "itemComplete"
	displayUnitAgentMessage displayUnitKind = "agentMessage"
	displayUnitGeneric      displayUnitKind = "generic"
)

type displayUnit struct {
	Kind           displayUnitKind
	StartLine      int
	Records        []*model.TypedRecord
	ReasoningPairs []*model.ReasoningPair
	Batch          *model.Batch
}

type conversationDisplayUnit struct {
	Kind      string
	Label     string
	Role      string
	Body      string
	Timestamp string
	StartLine int
	Details   []dto.TimelineItemDetail
	Records   []*model.TypedRecord
}

func NewGetSessionDetailUseCase(sessionRepo SessionRepository, cacheRepo CacheRepository, parser SessionParser) *GetSessionDetailUseCase {
	return &GetSessionDetailUseCase{
		sessionRepo: sessionRepo,
		cacheRepo:   cacheRepo,
		parser:      parser,
	}
}

func buildConversationTimeline(
	turns []*model.Turn,
	turnStats []dto.TurnStatistics,
	recordToNodeID map[int]string,
	tokenCountIndexByRecordLine map[int]int,
) []dto.ConversationTimelineTurn {
	var timeline []dto.ConversationTimelineTurn
	consumedTokensByTurn := make(map[int]dto.TokenBreakdown, len(turnStats))
	for _, turnStat := range turnStats {
		consumedTokensByTurn[turnStat.Index] = turnStat.ConsumedTokens
	}

	for _, turn := range turns {
		units := make([]conversationDisplayUnit, 0)
		unitIndexes := make(map[string]int)
		for _, record := range turn.Records {
			role, body, ok := conversationMessage(record)
			if !ok {
				continue
			}
			timestamp := fmt.Sprint(record.Envelope.Timestamp)
			key := role + "\x00" + body + "\x00" + timestamp
			if index, exists := unitIndexes[key]; exists {
				units[index].Records = append(units[index].Records, record)
				continue
			}
			unitIndexes[key] = len(units)
			units = append(units, conversationDisplayUnit{
				Kind:      "conversation",
				Role:      role,
				Body:      body,
				Timestamp: timestamp,
				StartLine: record.LineNumber,
				Records:   []*model.TypedRecord{record},
			})
		}
		for _, batch := range turn.Batches {
			if len(batch.CallRecords) > 0 && batch.CallRecords[0].ResponseItem != nil {
				name := batch.CallRecords[0].ResponseItem.Name
				if name == "spawn_agent" || name == "run_command" || name == "exec_command" {
					unit, ok := batchTimelineUnit(batch)
					if ok {
						units = append(units, unit)
					}
				}
			}
		}
		for _, record := range turn.ExternalEventRecords {
			if record.SubType == "exec_command_end" {
				unit, ok := externalTimelineUnit(record)
				if ok {
					units = append(units, unit)
				}
			}
		}
		metaUnits := metadataTimelineUnits(turn)
		for i := range metaUnits {
			if metaUnits[i].Kind == "collab" {
				units = append(units, metaUnits[i])
			}
		}
		if len(units) == 0 {
			continue
		}
		sort.SliceStable(units, func(i, j int) bool {
			return units[i].StartLine < units[j].StartLine
		})

		items := make([]dto.ConversationTimelineItem, 0, len(units))
		for index := range units {
			unit := &units[index]
			lastTokenUsage, tokenCountCount, totalTokenUsage := conversationTokenUsage(turn, unit)
			nodeIDs, tokenCountIndices := conversationSelection(
				turn,
				unit,
				recordToNodeID,
				tokenCountIndexByRecordLine,
			)
			var nodeID string
			if len(nodeIDs) > 0 {
				nodeID = nodeIDs[0]
			}
			items = append(items, dto.ConversationTimelineItem{
				SelectionID:       fmt.Sprintf("timeline-%d-%d", turn.Index, unit.StartLine),
				NodeID:            nodeID,
				NodeIDs:           nodeIDs,
				TokenCountIndices: tokenCountIndices,
				Kind:              unit.Kind,
				Label:             unit.Label,
				Role:              unit.Role,
				Body:              unit.Body,
				Timestamp:         unit.Timestamp,
				RecordCount:       len(unit.Records),
				Collapsible:       unit.Kind != "conversation",
				Details:           unit.Details,
				LastTokenUsage:    lastTokenUsage,
				TokenCountCount:   tokenCountCount,
				TotalTokenUsage:   totalTokenUsage,
			})
		}

		var durationMs int64
		if turn.TaskComplete != nil {
			durationMs = turn.TaskComplete.DurationMs
			if durationMs == 0 && turn.TaskStarted != nil && turn.TaskComplete.CompletedAt >= turn.TaskStarted.StartedAt {
				durationMs = (turn.TaskComplete.CompletedAt - turn.TaskStarted.StartedAt) * 1000
			}
		}
		timeline = append(timeline, dto.ConversationTimelineTurn{
			Index:          turn.Index,
			TurnID:         turn.TurnID,
			Pseudo:         turn.Index < 0,
			DurationMs:     durationMs,
			ConsumedTokens: consumedTokensByTurn[turn.Index],
			Items:          items,
		})
	}

	return timeline
}

func parseCommandFromArgs(argsJSON string) string {
	var args struct {
		CommandLine string   `json:"CommandLine"`
		Cmd         string   `json:"cmd"`
		Command     string   `json:"command"`
		Cmds        []string `json:"cmds"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err == nil {
		if args.CommandLine != "" {
			return args.CommandLine
		}
		if args.Cmd != "" {
			return args.Cmd
		}
		if args.Command != "" {
			return args.Command
		}
		if len(args.Cmds) > 0 {
			return strings.Join(args.Cmds, " ")
		}
	}
	return ""
}

func batchTimelineUnit(batch *model.Batch) (conversationDisplayUnit, bool) {
	if batch == nil || len(batch.CallRecords) == 0 {
		return conversationDisplayUnit{}, false
	}

	unit := conversationDisplayUnit{
		Kind:      "tool",
		StartLine: batch.CallRecords[0].LineNumber,
		Timestamp: fmt.Sprint(batch.CallRecords[0].Envelope.Timestamp),
	}

	// spawn_agent ツール呼び出しの場合、collab タイプのタイムラインユニットを生成する
	if batch.CallRecords[0].ResponseItem != nil && batch.CallRecords[0].ResponseItem.Name == "spawn_agent" {
		unit.Kind = "collab"
		unit.Label = "サブエージェント起動"
		agentID := ""
		nickname := ""
		role := ""
		if batch.CallRecords[0].ResponseItem.Arguments != "" {
			var args struct {
				AgentType string `json:"agent_type"`
			}
			if jsonErr := json.Unmarshal([]byte(batch.CallRecords[0].ResponseItem.Arguments), &args); jsonErr == nil {
				role = args.AgentType
			}
		}
		if len(batch.OutputRecords) > 0 && batch.OutputRecords[0] != nil && batch.OutputRecords[0].ResponseItem != nil {
			var out struct {
				AgentID  string `json:"agent_id"`
				Nickname string `json:"nickname"`
			}
			if jsonErr := json.Unmarshal([]byte(batch.OutputRecords[0].ResponseItem.Output), &out); jsonErr == nil {
				agentID = out.AgentID
				nickname = out.Nickname
			}
		}
		unit.Body = fmt.Sprintf("Nickname: %s, Role: %s, Thread ID: %s", nickname, role, agentID)
		unit.Details = []dto.TimelineItemDetail{
			{Label: "Nickname", Value: nickname},
			{Label: "Role", Value: role},
			{Label: "Thread ID", Value: agentID},
		}
		return unit, true
	}

	firstCall := batch.CallRecords[0]
	if firstCall.ResponseItem != nil && (firstCall.ResponseItem.Name == "run_command" || firstCall.ResponseItem.Name == "exec_command") {
		unit.Kind = "tool"
		unit.Label = "コマンド実行"

		var cmdLines []string
		for _, call := range batch.CallRecords {
			if call != nil && call.ResponseItem != nil && (call.ResponseItem.Name == "run_command" || call.ResponseItem.Name == "exec_command") {
				arguments := call.ResponseItem.Arguments
				if arguments == "" {
					arguments = call.ResponseItem.Input
				}
				cmdLine := parseCommandFromArgs(arguments)
				if cmdLine != "" {
					cmdLines = append(cmdLines, cmdLine)
				} else {
					cmdLines = append(cmdLines, call.ResponseItem.Name)
				}
			}
		}

		if len(cmdLines) > 0 {
			unit.Body = strings.Join(cmdLines, "\n")
		} else {
			unit.Body = firstCall.ResponseItem.Name
		}

		unit.Details = nil

		for _, call := range batch.CallRecords {
			if call != nil {
				unit.Records = append(unit.Records, call)
			}
		}
		for _, output := range batch.OutputRecords {
			if output != nil {
				unit.Records = append(unit.Records, output)
			}
		}
		unit.Records = append(unit.Records, batch.MiddleMessage...)

		return unit, true
	}

	names := make([]string, 0, len(batch.CallRecords))
	for index, call := range batch.CallRecords {
		if call == nil || call.ResponseItem == nil {
			continue
		}
		unit.Records = append(unit.Records, call)
		name := call.ResponseItem.Name
		if name == "" {
			name = "tool"
		}
		names = append(names, name)
		arguments := call.ResponseItem.Arguments
		if arguments == "" {
			arguments = call.ResponseItem.Input
		}
		if arguments != "" {
			unit.Details = append(unit.Details, dto.TimelineItemDetail{
				Label: name + " 引数",
				Value: arguments,
			})
		}
		if index < len(batch.OutputRecords) && batch.OutputRecords[index] != nil {
			output := batch.OutputRecords[index]
			unit.Records = append(unit.Records, output)
			if output.ResponseItem != nil && output.ResponseItem.Output != "" {
				unit.Details = append(unit.Details, dto.TimelineItemDetail{
					Label: name + " 結果",
					Value: output.ResponseItem.Output,
				})
			}
		}
	}
	unit.Records = append(unit.Records, batch.MiddleMessage...)
	if len(names) == 0 {
		return conversationDisplayUnit{}, false
	}
	unit.Body = strings.Join(names, ", ")
	unit.Label = "ツール " + names[0]
	if len(names) > 1 {
		unit.Label += fmt.Sprintf(" ×%d", len(names))
	}
	return unit, true
}

func externalTimelineUnit(record *model.TypedRecord) (conversationDisplayUnit, bool) {
	if record == nil || record.EventMsg == nil {
		return conversationDisplayUnit{}, false
	}
	if record.SubType == "exec_command_end" {
		event := record.EventMsg
		unit := conversationDisplayUnit{
			Kind:      "tool",
			Label:     "コマンド完了",
			Body:      strings.Join(event.Command, " "),
			StartLine: record.LineNumber,
			Timestamp: fmt.Sprint(record.Envelope.Timestamp),
			Records:   []*model.TypedRecord{record},
			Details:   nil,
		}
		return unit, true
	}
	if record.SubType == "view_image_tool_call" && record.EventMsg.Path != "" {
		return conversationDisplayUnit{
			Kind:      "reference",
			Label:     "画像参照",
			Body:      record.EventMsg.Path,
			StartLine: record.LineNumber,
			Timestamp: fmt.Sprint(record.Envelope.Timestamp),
			Records:   []*model.TypedRecord{record},
			Details:   []dto.TimelineItemDetail{{Label: "パス", Value: record.EventMsg.Path}},
		}, true
	}
	if record.SubType != "mcp_tool_call_end" || record.EventMsg.Invocation == nil {
		return conversationDisplayUnit{}, false
	}
	invocation := record.EventMsg.Invocation
	unit := conversationDisplayUnit{
		Kind:      "mcp",
		Label:     "MCP " + invocation.Tool,
		Body:      invocation.Server + "/" + invocation.Tool,
		StartLine: record.LineNumber,
		Timestamp: fmt.Sprint(record.Envelope.Timestamp),
		Records:   []*model.TypedRecord{record},
		Details: []dto.TimelineItemDetail{
			{Label: "サーバー", Value: invocation.Server},
		},
	}
	if len(invocation.Arguments) > 0 {
		unit.Details = append(unit.Details, dto.TimelineItemDetail{Label: "引数", Value: string(invocation.Arguments)})
	}
	if record.EventMsg.Result != "" {
		unit.Details = append(unit.Details, dto.TimelineItemDetail{Label: "結果", Value: record.EventMsg.Result})
	}
	return unit, true
}

func metadataTimelineUnits(turn *model.Turn) []conversationDisplayUnit {
	var units []conversationDisplayUnit
	for _, record := range turn.Records {
		if record == nil {
			continue
		}
		base := conversationDisplayUnit{
			StartLine: record.LineNumber,
			Timestamp: fmt.Sprint(record.Envelope.Timestamp),
			Records:   []*model.TypedRecord{record},
		}
		switch record.Type {
		case "session_meta":
			if record.SessionMeta != nil && record.SessionMeta.BaseInstructions != nil && record.SessionMeta.BaseInstructions.Text != "" {
				unit := base
				unit.Kind = "instructions"
				unit.Label = "Base instructions"
				unit.Body = record.SessionMeta.BaseInstructions.Text
				units = append(units, unit)
			}
		case "turn_context":
			if record.TurnContext == nil {
				continue
			}
			if mode := record.TurnContext.CollaborationMode; mode != nil && mode.DeveloperInstructions != "" {
				unit := base
				unit.Kind = "instructions"
				unit.Label = "Developer instructions"
				unit.Body = mode.DeveloperInstructions
				units = append(units, unit)
			}
			if record.TurnContext.UserInstructions != "" {
				unit := base
				unit.Kind = "instructions"
				unit.Label = "User instructions"
				unit.Body = record.TurnContext.UserInstructions
				units = append(units, unit)
			}
		case "event_msg":
			switch record.SubType {
			case "task_started":
				unit := base
				unit.Kind = "system"
				unit.Label = "タスク開始"
				units = append(units, unit)
			case "task_complete":
				unit := base
				unit.Kind = "system"
				unit.Label = "タスク完了"
				units = append(units, unit)
			case "turn_aborted":
				unit := base
				unit.Kind = "system"
				unit.Label = "ターン中断"
				if record.EventMsg != nil {
					unit.Body = record.EventMsg.Reason
				}
				units = append(units, unit)
			case "thread_name_updated":
				unit := base
				unit.Kind = "system"
				unit.Label = record.SubType
				if record.EventMsg != nil {
					unit.Body = record.EventMsg.ThreadName
				}
				units = append(units, unit)
			case "item_completed":
				unit := base
				unit.Kind = "system"
				unit.Label = record.SubType
				if record.EventMsg != nil && record.EventMsg.Item != nil {
					unit.Body = record.EventMsg.Item.Text
					if unit.Body == "" {
						unit.Body = record.EventMsg.Item.Type
					}
				}
				units = append(units, unit)
			case "collab_agent_spawn_end":
				unit := base
				unit.Kind = "collab"
				unit.Label = "サブエージェント起動"
				if record.EventMsg != nil {
					unit.Body = fmt.Sprintf("Nickname: %s, Role: %s, Thread ID: %s", record.EventMsg.NewAgentNickname, record.EventMsg.NewAgentRole, record.EventMsg.NewThreadID)
					unit.Details = []dto.TimelineItemDetail{
						{Label: "Nickname", Value: record.EventMsg.NewAgentNickname},
						{Label: "Role", Value: record.EventMsg.NewAgentRole},
						{Label: "Thread ID", Value: record.EventMsg.NewThreadID},
					}
				}
				units = append(units, unit)
			default:
				isUnhandledExternal := containsRecord(turn.ExternalEventRecords, record.LineNumber) &&
					record.SubType != "exec_command_end" &&
					record.SubType != "mcp_tool_call_end" &&
					record.SubType != "view_image_tool_call"
				if containsRecord(turn.GenericRecords, record.LineNumber) || isUnhandledExternal {
					unit := base
					unit.Kind = "system"
					unit.Label = record.SubType
					if record.EventMsg != nil {
						unit.Body = record.EventMsg.Message
						if unit.Body == "" {
							unit.Body = record.EventMsg.Text
						}
					}
					units = append(units, unit)
				}
			}
		case "response_item":
			if record.SubType == "message" && record.ResponseItem != nil && record.ResponseItem.Role == "developer" {
				unit := base
				unit.Kind = "instructions"
				unit.Label = "Developer instructions"
				for _, content := range record.ResponseItem.Content {
					unit.Body += content.Text
				}
				if unit.Body != "" {
					units = append(units, unit)
				}
			} else if containsRecord(turn.GenericRecords, record.LineNumber) {
				unit := base
				unit.Kind = "system"
				unit.Label = record.SubType
				unit.Body = record.Raw
				units = append(units, unit)
			}
		}
	}
	return units
}

func conversationSelection(
	turn *model.Turn,
	unit *conversationDisplayUnit,
	recordToNodeID map[int]string,
	tokenCountIndexByRecordLine map[int]int,
) (nodeIDs []string, tokenCountIndices []int) {
	recordLines := make(map[int]struct{}, len(unit.Records))
	seenNodeIDs := make(map[string]struct{}, len(unit.Records))
	for _, record := range unit.Records {
		recordLines[record.LineNumber] = struct{}{}
		nodeID := recordToNodeID[record.LineNumber]
		if nodeID == "" {
			continue
		}
		if _, exists := seenNodeIDs[nodeID]; exists {
			continue
		}
		seenNodeIDs[nodeID] = struct{}{}
		nodeIDs = append(nodeIDs, nodeID)
	}

	for _, tokenCount := range turn.TokenCounts {
		if tokenCount.BoundToRecord == nil || tokenCount.Record == nil {
			continue
		}
		if _, ok := recordLines[tokenCount.BoundToRecord.LineNumber]; !ok {
			continue
		}
		if tokenCountIndex, ok := tokenCountIndexByRecordLine[tokenCount.Record.LineNumber]; ok {
			tokenCountIndices = append(tokenCountIndices, tokenCountIndex)
		}
	}
	return nodeIDs, tokenCountIndices
}

func conversationTokenUsage(
	turn *model.Turn,
	unit *conversationDisplayUnit,
) (lastUsage dto.TokenBreakdown, count int, totalUsage *dto.TokenBreakdown) {
	recordLines := make(map[int]struct{}, len(unit.Records))
	for _, record := range unit.Records {
		recordLines[record.LineNumber] = struct{}{}
	}

	for _, tokenCount := range turn.TokenCounts {
		if tokenCount.BoundToRecord == nil {
			continue
		}
		if _, ok := recordLines[tokenCount.BoundToRecord.LineNumber]; !ok {
			continue
		}
		count++
		if tokenCount.Record == nil || tokenCount.Record.EventMsg == nil || tokenCount.Record.EventMsg.Info == nil {
			continue
		}
		info := tokenCount.Record.EventMsg.Info
		if info.LastTokenUsage != nil {
			lastUsage.TotalTokens += info.LastTokenUsage.TotalTokens
			lastUsage.InputTokens += info.LastTokenUsage.InputTokens
			lastUsage.OutputTokens += info.LastTokenUsage.OutputTokens
			lastUsage.ReasoningOutputTokens += info.LastTokenUsage.ReasoningOutputTokens
		}
		if info.TotalTokenUsage != nil {
			totalUsage = &dto.TokenBreakdown{
				TotalTokens:           info.TotalTokenUsage.TotalTokens,
				InputTokens:           info.TotalTokenUsage.InputTokens,
				OutputTokens:          info.TotalTokenUsage.OutputTokens,
				ReasoningOutputTokens: info.TotalTokenUsage.ReasoningOutputTokens,
			}
		}
	}
	return lastUsage, count, totalUsage
}

func conversationMessage(record *model.TypedRecord) (role, body string, ok bool) {
	if record == nil {
		return "", "", false
	}
	if record.Type == "event_msg" && record.EventMsg != nil {
		switch record.SubType {
		case "user_message":
			return "user", record.EventMsg.Message, record.EventMsg.Message != ""
		case "agent_message":
			return "assistant", record.EventMsg.Message, record.EventMsg.Message != ""
		}
	}
	if record.Type != "response_item" || record.SubType != "message" || record.ResponseItem == nil {
		return "", "", false
	}
	if record.ResponseItem.Role != "user" && record.ResponseItem.Role != "assistant" {
		return "", "", false
	}

	var bodyBuilder strings.Builder
	for _, content := range record.ResponseItem.Content {
		bodyBuilder.WriteString(content.Text)
	}
	body = bodyBuilder.String()
	return record.ResponseItem.Role, body, body != ""
}

func resolveFunctionCallOutputs(filePath, turnID string, callBatch, outputBatch []*model.TypedRecord) batchOutputResolution {
	result := batchOutputResolution{
		OutputRecords: make([]*model.TypedRecord, len(callBatch)),
	}
	if len(callBatch) == 0 {
		return result
	}

	callMatched := make([]bool, len(callBatch))
	outputMatched := make([]bool, len(outputBatch))

	for callIndex, call := range callBatch {
		if call == nil || call.ResponseItem == nil || call.ResponseItem.CallID == "" {
			continue
		}
		for outputIndex, output := range outputBatch {
			if outputMatched[outputIndex] || output == nil || output.ResponseItem == nil {
				continue
			}
			if output.ResponseItem.CallID == "" || output.ResponseItem.CallID != call.ResponseItem.CallID {
				continue
			}
			result.OutputRecords[callIndex] = output
			callMatched[callIndex] = true
			outputMatched[outputIndex] = true
			result.Matched++
			break
		}
	}

	var unmatchedCallIndexes []int
	var unmatchedOutputs []*model.TypedRecord
	for i := range callBatch {
		if !callMatched[i] {
			unmatchedCallIndexes = append(unmatchedCallIndexes, i)
		}
	}
	for i, output := range outputBatch {
		if !outputMatched[i] {
			unmatchedOutputs = append(unmatchedOutputs, output)
		}
	}

	if len(unmatchedCallIndexes) > 0 || len(unmatchedOutputs) > 0 {
		result.FallbackUsed = true
	}

	for i, callIndex := range unmatchedCallIndexes {
		if i < len(unmatchedOutputs) {
			result.OutputRecords[callIndex] = unmatchedOutputs[i]
			result.Matched++
			continue
		}
		result.NullAssigned++
	}

	if len(unmatchedOutputs) > len(unmatchedCallIndexes) {
		result.Discarded = len(unmatchedOutputs) - len(unmatchedCallIndexes)
	}

	if result.FallbackUsed {
		logger.Warn(
			"call_id mismatch or missing, falling back to index-based mapping",
			"file_path", filePath,
			"turn_id", turnID,
			"call_count", len(callBatch),
			"output_count", len(outputBatch),
			"details", fmt.Sprintf("matched: %d, null_assigned: %d, discarded: %d", result.Matched, result.NullAssigned, result.Discarded),
		)
	}

	return result
}

func (uc *GetSessionDetailUseCase) Execute(ctx context.Context, sessionID string) (*dto.SessionDetailResponse, error) {
	logger.Info("GetSessionDetailUseCase start", "session_id", sessionID)

	// 1. キャッシュの確認
	if cached, err := uc.cacheRepo.GetSessionDetail(ctx, sessionID); err == nil && cached != nil {
		if cached.CacheSchemaVersion == dto.CurrentSessionDetailCacheSchemaVersion {
			sessionModTime, sessionModErr := uc.sessionRepo.GetSessionModTime(ctx, sessionID)
			cacheModTime, cacheModErr := uc.cacheRepo.GetSessionDetailModTime(ctx, sessionID)
			if sessionModErr == nil && cacheModErr == nil && !sessionModTime.After(cacheModTime) {
				logger.Info("cache hit, returning cached session detail", "session_id", sessionID)
				return cached, nil
			}

			// フォールバックとして、既存キャッシュの parsed_at も利用する。
			if sessionModErr == nil {
				if cachedTime, parseErr := time.Parse(time.RFC3339, cached.ParsedAt); parseErr == nil && !sessionModTime.After(cachedTime) {
					logger.Info("cache hit, returning cached session detail", "session_id", sessionID)
					return cached, nil
				}
			}
		}
	}

	// 2. セッションファイルパスの特定
	filePath, err := uc.sessionRepo.GetSessionFilePath(ctx, sessionID)
	if err != nil {
		logger.Error("failed to get session file path", "session_id", sessionID, "error", err)
		return nil, model.ErrSessionNotFound
	}

	// 3. JSONLファイルのパース
	records, err := uc.parser.ParseSessionFile(ctx, filePath)
	if err != nil {
		logger.Error("failed to parse session file", "file", filePath, "error", err)
		if errors.Is(err, model.ErrFileTooLarge) {
			return nil, model.ErrFileTooLarge
		}
		if errors.Is(err, model.ErrParseFailed) {
			return nil, model.ErrParseFailed
		}
		if errors.Is(err, fs.ErrPermission) {
			return nil, model.ErrFileReadError
		}
		return nil, model.ErrParseFailed
	}

	// 4. バージョンチェック (cli_version v0.121.0 未満は非対応)
	var cliVersion string
	for _, r := range records {
		if r.Type == "session_meta" && r.SessionMeta != nil {
			cliVersion = r.SessionMeta.CliVersion
			break
		}
	}
	if isUnsupportedVersion(cliVersion) {
		logger.Warn("unsupported CLI version", "version", cliVersion)
		return nil, model.ErrUnsupportedFormat
	}

	// 5. ターン分割 + thread_name 収集 (Step 3)
	var turns []*model.Turn
	var currentTurn *model.Turn
	var outOfTurnBuffer []*model.TypedRecord
	var turnIndex int

	for _, record := range records {
		if record.Type == "session_meta" {
			outOfTurnBuffer = append(outOfTurnBuffer, record)
		} else if record.Type == "event_msg" && record.SubType == "thread_name_updated" {
			if currentTurn != nil {
				currentTurn.Records = append(currentTurn.Records, record)
			} else {
				outOfTurnBuffer = append(outOfTurnBuffer, record)
			}
		} else if record.Type == "event_msg" && record.SubType == "task_started" {
			var turnID string
			if record.EventMsg != nil {
				turnID = record.EventMsg.TurnID
			}

			var matchedRecords []*model.TypedRecord
			var remainingRecords []*model.TypedRecord
			for _, r := range outOfTurnBuffer {
				isMatch := false
				if r.Type == "turn_context" && r.TurnContext != nil && r.TurnContext.TurnID == turnID {
					isMatch = true
				} else if r.Type == "event_msg" && r.EventMsg != nil && r.EventMsg.TurnID == turnID {
					isMatch = true
				}
				if isMatch {
					matchedRecords = append(matchedRecords, r)
				} else {
					remainingRecords = append(remainingRecords, r)
				}
			}

			if len(remainingRecords) > 0 {
				turns = append(turns, &model.Turn{
					Index:   -1,
					Records: remainingRecords,
				})
			}
			outOfTurnBuffer = nil

			var initialRecords []*model.TypedRecord
			initialRecords = append(initialRecords, matchedRecords...)
			initialRecords = append(initialRecords, record)

			currentTurn = &model.Turn{
				Index:       turnIndex,
				TurnID:      turnID,
				TaskStarted: record.EventMsg,
				Records:     initialRecords,
			}
			turnIndex++
		} else if record.Type == "event_msg" && (record.SubType == "task_complete" || record.SubType == "turn_aborted") {
			if currentTurn != nil {
				currentTurn.TaskComplete = record.EventMsg
				currentTurn.Aborted = (record.SubType == "turn_aborted")
				currentTurn.Records = append(currentTurn.Records, record)
				turns = append(turns, currentTurn)
				currentTurn = nil
			} else {
				outOfTurnBuffer = append(outOfTurnBuffer, record)
			}
		} else {
			if currentTurn != nil {
				currentTurn.Records = append(currentTurn.Records, record)
			} else {
				outOfTurnBuffer = append(outOfTurnBuffer, record)
			}
		}
	}

	if len(outOfTurnBuffer) > 0 {
		turns = append(turns, &model.Turn{
			Index:   -1,
			Records: outOfTurnBuffer,
		})
	}

	// 6. ターン内レコード分類 (Step 4), Reasoning ペアリング (Step 5), バッチ検出 (Step 6), Token count 紐付け (Step 7)
	for _, turn := range turns {
		var tokenCounts []*model.TypedRecord
		var externalEvents []*model.TypedRecord
		var genericRecords []*model.TypedRecord
		customOutputsByCallID := make(map[string]*model.TypedRecord)

		for _, record := range turn.Records {
			switch record.Type {
			case "turn_context":
				if turn.TurnContext != nil {
					logger.Warn("multiple turn_contexts in a turn, overwriting", "turn_id", turn.TurnID)
				}
				if record.TurnContext != nil {
					turn.TurnContext = record.TurnContext
				}
			case "event_msg":
				switch record.SubType {
				case "user_message":
					turn.UserEventMsg = record
				case "agent_message":
					turn.AgentMessages = append(turn.AgentMessages, record)
				case "item_completed":
					turn.ItemCompleted = append(turn.ItemCompleted, record)
				case "token_count":
					tokenCounts = append(tokenCounts, record)
				case "web_search_end":
					turn.WebSearchRecords = append(turn.WebSearchRecords, record)
				case "task_started", "task_complete", "turn_aborted", "thread_name_updated", "agent_reasoning":
					// Boundary or reasoning, handled separately
				default:
					if record.EventMsg != nil && record.EventMsg.CallID != "" {
						externalEvents = append(externalEvents, record)
					} else {
						genericRecords = append(genericRecords, record)
					}
				}
			case "response_item":
				switch record.SubType {
				case "message":
					if record.ResponseItem != nil {
						switch record.ResponseItem.Role {
						case "developer":
							turn.DeveloperMessages = append(turn.DeveloperMessages, record)
						case "user":
							turn.UserMessages = append(turn.UserMessages, record)
						}
					}
				case "web_search_call":
					turn.WebSearchRecords = append(turn.WebSearchRecords, record)
				case "reasoning", "function_call", "custom_tool_call", "function_call_output", "custom_tool_call_output":
					// Handled separately
				default:
					if record.ResponseItem != nil && record.ResponseItem.CallID != "" {
						externalEvents = append(externalEvents, record)
					} else {
						genericRecords = append(genericRecords, record)
					}
				}
			default:
				genericRecords = append(genericRecords, record)
			}
			if record.Type == "response_item" && record.ResponseItem != nil && record.ResponseItem.CallID != "" {
				if record.SubType == "custom_tool_call_output" {
					customOutputsByCallID[record.ResponseItem.CallID] = record
				}
			}
		}

		turn.ExternalEventRecords = externalEvents
		turn.GenericRecords = genericRecords

		// Reasoning Pairing (Step 5)
		var arRecords []*model.TypedRecord
		var riRecords []*model.TypedRecord
		for _, r := range turn.Records {
			if r.Type == "event_msg" && r.SubType == "agent_reasoning" {
				arRecords = append(arRecords, r)
			} else if r.Type == "response_item" && r.SubType == "reasoning" {
				riRecords = append(riRecords, r)
			}
		}

		maxLen := len(arRecords)
		if len(riRecords) > maxLen {
			maxLen = len(riRecords)
		}
		for i := 0; i < maxLen; i++ {
			pair := &model.ReasoningPair{}
			if i < len(arRecords) {
				pair.AgentReasoning = arRecords[i]
			}
			if i < len(riRecords) {
				pair.Reasoning = riRecords[i]
			}
			turn.AgentReasonings = append(turn.AgentReasonings, pair)
		}

		// Batch Detection (Step 6)
		var batches []*model.Batch
		batchRecordsSet := make(map[int]bool)
		for k := 0; k < len(turn.Records); {
			record := turn.Records[k]
			if record.Type == "response_item" && record.SubType == "function_call" {
				var callBatch []*model.TypedRecord
				for k < len(turn.Records) {
					r := turn.Records[k]
					if r.Type == "response_item" && r.SubType == "function_call" {
						callBatch = append(callBatch, r)
						k++
					} else {
						break
					}
				}

				var middleMessage []*model.TypedRecord
				if k < len(turn.Records) {
					r1 := turn.Records[k]
					if r1.Type == "event_msg" && r1.SubType == "agent_message" && k+1 < len(turn.Records) {
						r2 := turn.Records[k+1]
						if r2.Type == "response_item" && r2.SubType == "message" && r2.ResponseItem != nil && r2.ResponseItem.Role == "assistant" {
							middleMessage = []*model.TypedRecord{r1, r2}
							k += 2
						}
					} else if r1.Type == "response_item" && r1.SubType == "message" && r1.ResponseItem != nil && r1.ResponseItem.Role == "assistant" {
						middleMessage = []*model.TypedRecord{r1}
						k++
					}
				}

				var outputBatch []*model.TypedRecord
				outputScanIndex := k
				for outputScanIndex < len(turn.Records) {
					r := turn.Records[outputScanIndex]
					if r.Type == "response_item" && r.SubType == "function_call_output" {
						for outputScanIndex < len(turn.Records) {
							nextRecord := turn.Records[outputScanIndex]
							if nextRecord.Type == "response_item" && nextRecord.SubType == "function_call_output" {
								outputBatch = append(outputBatch, nextRecord)
								outputScanIndex++
								continue
							}
							break
						}
						break
					}
					outputScanIndex++
				}
				k = outputScanIndex
				resolution := resolveFunctionCallOutputs(filePath, turn.TurnID, callBatch, outputBatch)

				batch := &model.Batch{
					CallRecords:   callBatch,
					OutputRecords: resolution.OutputRecords,
					MiddleMessage: middleMessage,
				}
				batches = append(batches, batch)

				for _, r := range callBatch {
					batchRecordsSet[r.LineNumber] = true
				}
				for _, r := range middleMessage {
					batchRecordsSet[r.LineNumber] = true
				}
				for _, r := range outputBatch {
					batchRecordsSet[r.LineNumber] = true
				}
				continue
			}

			if record.Type == "response_item" && record.SubType == "custom_tool_call" {
				var callBatch []*model.TypedRecord
				for k < len(turn.Records) {
					r := turn.Records[k]
					if r.Type == "response_item" && r.SubType == "custom_tool_call" {
						callBatch = append(callBatch, r)
						k++
					} else {
						break
					}
				}

				outputRecords := make([]*model.TypedRecord, len(callBatch))
				for i, call := range callBatch {
					if call.ResponseItem != nil {
						if out, ok := customOutputsByCallID[call.ResponseItem.CallID]; ok && out.LineNumber > call.LineNumber {
							outputRecords[i] = out
						}
					}
				}

				batch := &model.Batch{
					CallRecords:   callBatch,
					OutputRecords: outputRecords,
				}
				batches = append(batches, batch)

				for _, r := range callBatch {
					batchRecordsSet[r.LineNumber] = true
				}
				for _, r := range outputRecords {
					if r != nil {
						batchRecordsSet[r.LineNumber] = true
					}
				}
				continue
			}
			k++
		}
		turn.Batches = batches

		// Token count binding (Step 7)
		for _, record := range tokenCounts {
			var boundTo *model.TypedRecord
			for i := len(turn.Records) - 1; i >= 0; i-- {
				r := turn.Records[i]
				if r.LineNumber < record.LineNumber {
					if r.Type == "event_msg" && r.SubType == "token_count" {
						continue
					}
					boundTo = r
					break
				}
			}
			turn.TokenCounts = append(turn.TokenCounts, &model.TokenCountWithBinding{
				Record:        record,
				BoundToRecord: boundTo,
				TurnIndex:     turn.Index,
			})
		}
	}

	// 7. React Flow形式への変換 & 座標レイアウト計算 (Step 2.5 / 2.6 / 2.7)
	var nodes []dto.FlowNode
	var edges []dto.FlowEdge
	RecordToNodeID := make(map[int]string)
	var lastHarnessNodeID string
	var lastHarnessNodeIsBatchJoin bool
	var currentY float64

	const (
		HarnessX        = 0.0
		BranchOffsetX   = 400.0
		ContextOffsetX  = 400.0
		NodeWidth       = 320.0
		NodeHeight      = 80.0
		NodeHeightLarge = 120.0
		NodeGap         = 40.0
		BatchNodeGap    = 8.0
		BatchMiddleGap  = 24.0
		BranchNodeGapX  = 20.0
	)

	pushHarnessNode := func(id, nodeType, category, label, icon, summary, fullText string, meta map[string]interface{}, turnIndex int, height float64) {
		node := dto.FlowNode{
			ID:   id,
			Type: nodeType,
			Position: dto.Position{
				X: HarnessX,
				Y: currentY,
			},
			Data: dto.NodeData{
				Category:  category,
				Label:     label,
				Icon:      icon,
				Summary:   summary,
				FullText:  fullText,
				Meta:      meta,
				TurnIndex: turnIndex,
			},
		}
		nodes = append(nodes, node)

		if lastHarnessNodeID != "" {
			edges = append(edges, dto.FlowEdge{
				ID:       fmt.Sprintf("edge-%s-%s", lastHarnessNodeID, id),
				Source:   lastHarnessNodeID,
				Target:   id,
				Type:     "default",
				Animated: false,
			})
		}

		lastHarnessNodeID = id
		lastHarnessNodeIsBatchJoin = false
		currentY += height + NodeGap
	}

	addBranchNode := func(id, nodeType, category, label, icon, summary, fullText string, meta map[string]interface{}, turnIndex int, x, y float64) {
		node := dto.FlowNode{
			ID:   id,
			Type: nodeType,
			Position: dto.Position{
				X: x,
				Y: y,
			},
			Data: dto.NodeData{
				Category:  category,
				Label:     label,
				Icon:      icon,
				Summary:   summary,
				FullText:  fullText,
				Meta:      meta,
				TurnIndex: turnIndex,
			},
		}
		nodes = append(nodes, node)
	}

	// Step 1: sessionMeta ノード生成
	var sessionMetaRecord *model.TypedRecord
	for _, turn := range turns {
		for _, r := range turn.Records {
			if r.Type == "session_meta" {
				sessionMetaRecord = r
				break
			}
		}
		if sessionMetaRecord != nil {
			break
		}
	}

	if sessionMetaRecord != nil && sessionMetaRecord.SessionMeta != nil {
		meta := sessionMetaRecord.SessionMeta
		metaMap := map[string]interface{}{
			"cwd":            meta.Cwd,
			"cli_version":    meta.CliVersion,
			"originator":     meta.Originator,
			"model_provider": meta.ModelProvider,
			"source":         meta.Source,
			"timestamp":      meta.Timestamp,
		}
		if meta.Git != nil {
			metaMap["git_branch"] = meta.Git.Branch
		}

		metaNodeID := fmt.Sprintf("node-%d", sessionMetaRecord.LineNumber)
		pushHarnessNode(metaNodeID, "sessionMeta", "meta", "Session Meta", "⚙️", "CLI Version: "+meta.CliVersion, "", metaMap, -1, NodeHeight)
		RecordToNodeID[sessionMetaRecord.LineNumber] = metaNodeID

		// base_instructions 分岐
		if meta.BaseInstructions != nil && meta.BaseInstructions.Text != "" {
			baseInstNodeID := fmt.Sprintf("node-%d-base-inst", sessionMetaRecord.LineNumber)
			addBranchNode(baseInstNodeID, "contextDoc", "context", "Base Instructions", "📜", meta.BaseInstructions.Text, meta.BaseInstructions.Text, nil, -1, ContextOffsetX, 0)
			edges = append(edges, dto.FlowEdge{
				ID:       fmt.Sprintf("edge-%s-%s", baseInstNodeID, metaNodeID),
				Source:   baseInstNodeID,
				Target:   metaNodeID,
				Type:     "step",
				Animated: false,
			})
		}
	}

	// Step 2: 各 Turn の処理
	for _, turn := range turns {
		// Map call_id -> NodeID in this turn for external events
		callIDToNodeID := make(map[string]string)

		if turn.Index >= 0 {
			// 通常のターン
			// 2a. task_started ノード
			var taskStartedRecord *model.TypedRecord
			for _, r := range turn.Records {
				if r.Type == "event_msg" && r.SubType == "task_started" {
					taskStartedRecord = r
					break
				}
			}
			var startNodeID string
			if taskStartedRecord != nil && taskStartedRecord.EventMsg != nil {
				startNodeID = fmt.Sprintf("node-%d", taskStartedRecord.LineNumber)
				pushHarnessNode(startNodeID, "taskEvent", "event", "Task Started", "🚀", taskStartedRecord.EventMsg.Message, "", nil, turn.Index, NodeHeight)
				RecordToNodeID[taskStartedRecord.LineNumber] = startNodeID
			}

			// 2b. DeveloperMessages + UserMessages (分岐)
			var branchMsgIdx int
			for _, devMsg := range turn.DeveloperMessages {
				if devMsg.ResponseItem == nil || len(devMsg.ResponseItem.Content) == 0 {
					continue
				}
				msgText := devMsg.ResponseItem.Content[0].Text
				devNodeID := fmt.Sprintf("node-%d", devMsg.LineNumber)
				yPos := currentY - NodeHeight - NodeGap + float64(branchMsgIdx)*(NodeHeight+NodeGap)
				addBranchNode(devNodeID, "developerMessage", "message", "Developer Message", "💻", msgText, msgText, nil, turn.Index, BranchOffsetX, yPos)
				RecordToNodeID[devMsg.LineNumber] = devNodeID
				if startNodeID != "" {
					edges = append(edges, dto.FlowEdge{
						ID:       fmt.Sprintf("edge-%s-%s", devNodeID, startNodeID),
						Source:   devNodeID,
						Target:   startNodeID,
						Type:     "step",
						Animated: false,
					})
				}
				branchMsgIdx++
			}
			for _, userMsg := range turn.UserMessages {
				if userMsg.ResponseItem == nil || len(userMsg.ResponseItem.Content) == 0 {
					continue
				}
				msgText := userMsg.ResponseItem.Content[0].Text
				userNodeID := fmt.Sprintf("node-%d", userMsg.LineNumber)
				yPos := currentY - NodeHeight - NodeGap + float64(branchMsgIdx)*(NodeHeight+NodeGap)
				addBranchNode(userNodeID, "userApiMessage", "message", "User API Message", "👤", msgText, msgText, nil, turn.Index, BranchOffsetX, yPos)
				RecordToNodeID[userMsg.LineNumber] = userNodeID
				if startNodeID != "" {
					edges = append(edges, dto.FlowEdge{
						ID:       fmt.Sprintf("edge-%s-%s", userNodeID, startNodeID),
						Source:   userNodeID,
						Target:   startNodeID,
						Type:     "step",
						Animated: false,
					})
				}
				branchMsgIdx++
			}

			// 2c. turnContext ノード
			var turnContextRecord *model.TypedRecord
			for _, r := range turn.Records {
				if r.Type == "turn_context" {
					turnContextRecord = r
					break
				}
			}
			var contextNodeID string
			if turnContextRecord != nil && turnContextRecord.TurnContext != nil {
				contextNodeID = fmt.Sprintf("node-%d", turnContextRecord.LineNumber)
				pushHarnessNode(contextNodeID, "turnContext", "turn", "Turn Context", "📝", "Model: "+turnContextRecord.TurnContext.Model, "", nil, turn.Index, NodeHeightLarge)
				RecordToNodeID[turnContextRecord.LineNumber] = contextNodeID

				var branchIdx int
				turnCtxY := currentY - NodeHeightLarge - NodeGap
				if turnContextRecord.TurnContext.CollaborationMode != nil && turnContextRecord.TurnContext.CollaborationMode.DeveloperInstructions != "" {
					devInstNodeID := fmt.Sprintf("node-%d-dev-inst", turnContextRecord.LineNumber)
					devText := turnContextRecord.TurnContext.CollaborationMode.DeveloperInstructions
					yPos := turnCtxY + float64(branchIdx)*(NodeHeight+NodeGap)
					addBranchNode(devInstNodeID, "contextDoc", "context", "Developer Instructions", "📜", devText, devText, nil, turn.Index, ContextOffsetX, yPos)
					edges = append(edges, dto.FlowEdge{
						ID:       fmt.Sprintf("edge-%s-%s", devInstNodeID, contextNodeID),
						Source:   devInstNodeID,
						Target:   contextNodeID,
						Type:     "step",
						Animated: false,
					})
					branchIdx++
				}
				if turnContextRecord.TurnContext.UserInstructions != "" {
					userInstNodeID := fmt.Sprintf("node-%d-user-inst", turnContextRecord.LineNumber)
					userText := turnContextRecord.TurnContext.UserInstructions
					yPos := turnCtxY + float64(branchIdx)*(NodeHeight+NodeGap)
					addBranchNode(userInstNodeID, "contextDoc", "context", "User Instructions", "📜", userText, userText, nil, turn.Index, ContextOffsetX, yPos)
					edges = append(edges, dto.FlowEdge{
						ID:       fmt.Sprintf("edge-%s-%s", userInstNodeID, contextNodeID),
						Source:   userInstNodeID,
						Target:   contextNodeID,
						Type:     "step",
						Animated: false,
					})
				}
			}

			// 2d. user_message ノード
			if turn.UserEventMsg != nil && turn.UserEventMsg.EventMsg != nil {
				userMsgNodeID := fmt.Sprintf("node-%d", turn.UserEventMsg.LineNumber)
				pushHarnessNode(userMsgNodeID, "userMessage", "turn", "User Message", "👤", turn.UserEventMsg.EventMsg.Message, turn.UserEventMsg.EventMsg.Message, nil, turn.Index, NodeHeight)
				RecordToNodeID[turn.UserEventMsg.LineNumber] = userMsgNodeID
			}

			// 2e. 動的ノードをレコード出現順に生成
			var lastWebCallID string
			for _, unit := range buildDisplayUnits(turn) {
				switch unit.Kind {
				case displayUnitReasoning:
					reasonNodeID := fmt.Sprintf("node-%d", unit.StartLine)
					if isUnreadableReasoningUnit(&unit) {
						unreadableText := fmt.Sprintf("（暗号化済み・表示不可）×%d", len(unit.ReasoningPairs))
						pushHarnessNode(reasonNodeID, "reasoning", "turn", "Reasoning", "🧠", unreadableText, unreadableText, nil, turn.Index, NodeHeightLarge)
					} else {
						pair := unit.ReasoningPairs[0]
						var summary, fullText string
						if pair.AgentReasoning != nil && pair.AgentReasoning.EventMsg != nil {
							summary = pair.AgentReasoning.EventMsg.Text
						}
						if pair.Reasoning != nil && pair.Reasoning.ResponseItem != nil {
							var parts []string
							for _, content := range pair.Reasoning.ResponseItem.Summary {
								parts = append(parts, content.Text)
							}
							fullText = strings.Join(parts, "\n")
						}
						if pair.AgentReasoning == nil && fullText != "" {
							summary = fullText
						}
						pushHarnessNode(reasonNodeID, "reasoning", "turn", "Reasoning", "🧠", summary, fullText, nil, turn.Index, NodeHeightLarge)
					}
					for _, record := range unit.Records {
						RecordToNodeID[record.LineNumber] = reasonNodeID
					}

				case displayUnitBatch:
					batch := unit.Batch
					harnessTopY := currentY - NodeGap - NodeHeight
					if lastHarnessNodeIsBatchJoin {
						harnessTopY = currentY
					} else if lastHarnessNodeID != "" {
						for idx := range nodes {
							if nodes[idx].ID == lastHarnessNodeID {
								harnessTopY = nodes[idx].Position.Y
								break
							}
						}
					}

					var callNodeIDs []string
					var outputNodeIDs []string
					for i, call := range batch.CallRecords {
						if call.ResponseItem == nil {
							continue
						}
						callNodeID := fmt.Sprintf("node-%d", call.LineNumber)
						callNodeIDs = append(callNodeIDs, callNodeID)
						xPos := BranchOffsetX + float64(i)*(NodeWidth+BranchNodeGapX)

						nodeType := "action"
						category := "action"
						icon := "🛠️"
						var meta map[string]interface{}
						label := call.ResponseItem.Name

						if call.ResponseItem.Name == "spawn_agent" {
							nodeType = "collabAgent"
							icon = "🤖"
							label = "Subagent Spawned"
							agentID := ""
							nickname := ""
							role := ""
							if call.ResponseItem.Arguments != "" {
								var args struct {
									AgentType string `json:"agent_type"`
								}
								if jsonErr := json.Unmarshal([]byte(call.ResponseItem.Arguments), &args); jsonErr == nil {
									role = args.AgentType
								}
							}
							if i < len(batch.OutputRecords) && batch.OutputRecords[i] != nil && batch.OutputRecords[i].ResponseItem != nil {
								var outData struct {
									AgentID  string `json:"agent_id"`
									Nickname string `json:"nickname"`
								}
								if jsonErr := json.Unmarshal([]byte(batch.OutputRecords[i].ResponseItem.Output), &outData); jsonErr == nil {
									agentID = outData.AgentID
									nickname = outData.Nickname
								}
							}
							meta = map[string]interface{}{
								"new_thread_id":      agentID,
								"new_agent_nickname": nickname,
								"new_agent_role":     role,
							}
						}

						addBranchNode(callNodeID, nodeType, category, label, icon, call.ResponseItem.Arguments, call.ResponseItem.Arguments, meta, turn.Index, xPos, harnessTopY)
						RecordToNodeID[call.LineNumber] = callNodeID
						callIDToNodeID[call.ResponseItem.CallID] = callNodeID

						if lastHarnessNodeID != "" {
							edges = append(edges, dto.FlowEdge{
								ID:       fmt.Sprintf("edge-%s-%s", lastHarnessNodeID, callNodeID),
								Source:   lastHarnessNodeID,
								Target:   callNodeID,
								Type:     "step",
								Animated: false,
							})
						}

						output := batch.OutputRecords[i]
						if output != nil && output.ResponseItem != nil {
							outNodeID := fmt.Sprintf("node-%d", output.LineNumber)
							outputNodeIDs = append(outputNodeIDs, outNodeID)
							yPos := harnessTopY + NodeHeight + BatchNodeGap
							outputLabel := "Tool Output"
							if call.ResponseItem.Name != "" {
								outputLabel = "Output: " + call.ResponseItem.Name
							}
							addBranchNode(outNodeID, "action", "action", outputLabel, "📤", output.ResponseItem.Output, output.ResponseItem.Output, nil, turn.Index, xPos, yPos)
							RecordToNodeID[output.LineNumber] = outNodeID
							callIDToNodeID[output.ResponseItem.CallID] = outNodeID
							edges = append(edges, dto.FlowEdge{
								ID:       fmt.Sprintf("edge-%s-%s", callNodeID, outNodeID),
								Source:   callNodeID,
								Target:   outNodeID,
								Type:     "default",
								Animated: false,
							})
						}
					}

					if len(batch.MiddleMessage) > 0 {
						var texts []string
						for _, message := range batch.MiddleMessage {
							texts = append(texts, agentMessageTexts(message)...)
						}
						middleSummary := strings.Join(texts, "\n")
						middleNodeID := fmt.Sprintf("node-%d", batch.MiddleMessage[0].LineNumber)
						middleNodeY := harnessTopY + NodeHeight + BatchNodeGap + NodeHeight + BatchMiddleGap
						nodes = append(nodes, dto.FlowNode{
							ID:       middleNodeID,
							Position: dto.Position{X: HarnessX, Y: middleNodeY},
							Type:     "agentMessage",
							Data: dto.NodeData{
								Category:  "turn",
								Label:     "Agent Message",
								Icon:      "🤖",
								Summary:   middleSummary,
								FullText:  middleSummary,
								TurnIndex: turn.Index,
							},
						})
						for _, message := range batch.MiddleMessage {
							RecordToNodeID[message.LineNumber] = middleNodeID
						}
						if len(outputNodeIDs) > 0 {
							for _, outputNodeID := range outputNodeIDs {
								edges = append(edges, dto.FlowEdge{
									ID:       fmt.Sprintf("edge-%s-%s", outputNodeID, middleNodeID),
									Source:   outputNodeID,
									Target:   middleNodeID,
									Type:     "default",
									Animated: false,
								})
							}
						} else if lastHarnessNodeID != "" {
							edges = append(edges, dto.FlowEdge{
								ID:       fmt.Sprintf("edge-%s-%s", lastHarnessNodeID, middleNodeID),
								Source:   lastHarnessNodeID,
								Target:   middleNodeID,
								Type:     "default",
								Animated: false,
							})
						}
						lastHarnessNodeID = middleNodeID
						lastHarnessNodeIsBatchJoin = false
						currentY = middleNodeY + NodeHeight + NodeGap
					} else {
						joinNodeID := ""
						if len(outputNodeIDs) > 0 {
							joinNodeID = outputNodeIDs[len(outputNodeIDs)-1]
							currentY = harnessTopY + NodeHeight + BatchNodeGap + NodeHeight + NodeGap
						} else if len(callNodeIDs) > 0 {
							joinNodeID = callNodeIDs[len(callNodeIDs)-1]
							currentY = harnessTopY + NodeHeight + NodeGap
						}
						if joinNodeID != "" {
							lastHarnessNodeID = joinNodeID
							lastHarnessNodeIsBatchJoin = true
						}
					}

				case displayUnitWebSearch:
					record := unit.Records[0]
					if record.Type == "response_item" && record.SubType == "web_search_call" {
						var query string
						if record.ResponseItem != nil && record.ResponseItem.Action != nil {
							query = record.ResponseItem.Action.Query
							if query == "" && len(record.ResponseItem.Action.Queries) > 0 {
								query = record.ResponseItem.Action.Queries[0]
							}
						}
						nodeID := fmt.Sprintf("node-%d", record.LineNumber)
						pushHarnessNode(nodeID, "webSearchAction", "action", "Web Search", "🔍", "Query: "+query, "", nil, turn.Index, NodeHeight)
						RecordToNodeID[record.LineNumber] = nodeID
						lastWebCallID = nodeID
					} else if record.Type == "event_msg" && record.SubType == "web_search_end" {
						nodeID := fmt.Sprintf("node-%d", record.LineNumber)
						pushHarnessNode(nodeID, "webSearchAction", "action", "Web Search Completed", "🔍", "Search finished", "", nil, turn.Index, NodeHeight)
						RecordToNodeID[record.LineNumber] = nodeID
						if lastWebCallID != "" {
							edges = append(edges, dto.FlowEdge{
								ID:       fmt.Sprintf("edge-%s-%s", lastWebCallID, nodeID),
								Source:   lastWebCallID,
								Target:   nodeID,
								Type:     "default",
								Animated: false,
							})
						}
					}

				case displayUnitItemComplete:
					record := unit.Records[0]
					if record.EventMsg == nil || record.EventMsg.Item == nil {
						continue
					}
					nodeID := fmt.Sprintf("node-%d", record.LineNumber)
					pushHarnessNode(nodeID, "itemCompleted", "event", "Item Completed", "✅", record.EventMsg.Item.Text, "", nil, turn.Index, NodeHeight)
					RecordToNodeID[record.LineNumber] = nodeID

				case displayUnitAgentMessage:
					var texts []string
					for _, record := range unit.Records {
						texts = append(texts, agentMessageTexts(record)...)
					}
					nodeID := fmt.Sprintf("node-%d", unit.StartLine)
					summary := strings.Join(texts, "\n")
					pushHarnessNode(nodeID, "agentMessage", "turn", "Agent Message", "🤖", summary, summary, nil, turn.Index, NodeHeight)
					for _, record := range unit.Records {
						RecordToNodeID[record.LineNumber] = nodeID
					}

				case displayUnitGeneric:
					record := unit.Records[0]
					nodeID := fmt.Sprintf("node-%d", record.LineNumber)
					if record.Type == "event_msg" && record.SubType == "collab_agent_spawn_end" && record.EventMsg != nil {
						meta := map[string]interface{}{
							"new_thread_id":      record.EventMsg.NewThreadID,
							"new_agent_nickname": record.EventMsg.NewAgentNickname,
							"new_agent_role":     record.EventMsg.NewAgentRole,
						}
						summary := fmt.Sprintf("Nickname: %s / Role: %s", record.EventMsg.NewAgentNickname, record.EventMsg.NewAgentRole)
						pushHarnessNode(nodeID, "collabAgent", "action", "Subagent Spawned", "🤖", summary, "", meta, turn.Index, NodeHeight)
					} else {
						pushHarnessNode(nodeID, "generic", "generic", "System Event", "⚙️", "Type: "+record.Type+" / Subtype: "+record.SubType, "", nil, turn.Index, NodeHeight)
					}
					RecordToNodeID[record.LineNumber] = nodeID
				}
			}

			// 2i. task_complete
			var taskCompleteRecord *model.TypedRecord
			for _, r := range turn.Records {
				if r.Type == "event_msg" && (r.SubType == "task_complete" || r.SubType == "turn_aborted") {
					taskCompleteRecord = r
					break
				}
			}
			if taskCompleteRecord != nil && taskCompleteRecord.EventMsg != nil {
				nodeID := fmt.Sprintf("node-%d", taskCompleteRecord.LineNumber)
				var label, icon string
				if taskCompleteRecord.SubType == "turn_aborted" {
					label = "Task Aborted"
					icon = "🛑"
				} else {
					label = "Task Complete"
					icon = "🏁"
				}
				pushHarnessNode(nodeID, "taskEvent", "event", label, icon, taskCompleteRecord.EventMsg.Reason, "", nil, turn.Index, NodeHeight)
				RecordToNodeID[taskCompleteRecord.LineNumber] = nodeID
			}
		} else {
			// Process pseudo-turn records (where turn index is -1)
			for _, r := range turn.Records {
				if r.Type == "session_meta" {
					if metaNodeID, ok := RecordToNodeID[r.LineNumber]; ok {
						RecordToNodeID[r.LineNumber] = metaNodeID
					}
					continue
				}

				if r.Type == "event_msg" && (r.SubType == "task_complete" || r.SubType == "turn_aborted") {
					// Orphan complete/aborted event with warning
					nodeID := fmt.Sprintf("node-%d", r.LineNumber)
					label := "Orphan Complete"
					if r.SubType == "turn_aborted" {
						label = "Orphan Aborted"
					}
					pushHarnessNode(nodeID, "taskEvent", "event", label, "⚠️", "Orphan complete/aborted event without task_started", "", nil, -1, NodeHeight)
					RecordToNodeID[r.LineNumber] = nodeID
				} else {
					// Other system events shown as generic nodes
					nodeID := fmt.Sprintf("node-%d", r.LineNumber)
					pushHarnessNode(nodeID, "generic", "generic", "[System] Event", "⚙️", "Type: "+r.Type+" / Subtype: "+r.SubType, "", nil, -1, NodeHeight)
					RecordToNodeID[r.LineNumber] = nodeID
				}
			}
		}

		// Step 3: 外部イベントブランチノードの座標・接続
		for _, ext := range turn.ExternalEventRecords {
			if ext.EventMsg == nil || ext.EventMsg.CallID == "" {
				continue
			}
			actionNodeID := callIDToNodeID[ext.EventMsg.CallID]
			if actionNodeID == "" {
				continue
			}

			// find actionNode coordinates
			var actionNode *dto.FlowNode
			for i := range nodes {
				if nodes[i].ID == actionNodeID {
					actionNode = &nodes[i]
					break
				}
			}

			extNodeID := fmt.Sprintf("node-%d", ext.LineNumber)
			xPos := actionNode.Position.X + NodeWidth + BranchNodeGapX
			yPos := actionNode.Position.Y

			label := "External Event"
			if ext.SubType == "exec_command_end" {
				label = "Command End"
			}

			addBranchNode(extNodeID, "externalEvent", "action", label, "⚡", "Call ID: "+ext.EventMsg.CallID, "", nil, turn.Index, xPos, yPos)
			RecordToNodeID[ext.LineNumber] = extNodeID

			edges = append(edges, dto.FlowEdge{
				ID:       fmt.Sprintf("edge-%s-%s", actionNodeID, extNodeID),
				Source:   actionNodeID,
				Target:   extNodeID,
				Type:     "step",
				Animated: false,
			})
		}
	}

	// Step 4: 各 token_count を Binding
	var tokenCountsList []dto.TokenCountEntry
	tokenCountIndexByRecordLine := make(map[int]int)
	var tcIdx int
	resolveBoundNodeID := func(turn *model.Turn, tc *model.TokenCountWithBinding) string {
		if tc.BoundToRecord != nil {
			if nodeID := RecordToNodeID[tc.BoundToRecord.LineNumber]; nodeID != "" {
				return nodeID
			}
		}
		for i := len(turn.Records) - 1; i >= 0; i-- {
			record := turn.Records[i]
			if record.LineNumber >= tc.Record.LineNumber {
				continue
			}
			if record.Type == "event_msg" && record.SubType == "token_count" {
				continue
			}
			if nodeID := RecordToNodeID[record.LineNumber]; nodeID != "" {
				return nodeID
			}
		}
		return ""
	}
	for _, turn := range turns {
		for _, tc := range turn.TokenCounts {
			boundNodeID := resolveBoundNodeID(turn, tc)

			var totalUsage, lastUsage *model.TokenDetail
			var modelContextWindow int64
			if tc.Record.EventMsg != nil && tc.Record.EventMsg.Info != nil {
				totalUsage = tc.Record.EventMsg.Info.TotalTokenUsage
				lastUsage = tc.Record.EventMsg.Info.LastTokenUsage
				modelContextWindow = tc.Record.EventMsg.Info.ModelContextWindow
			}
			if modelContextWindow <= 0 &&
				turn.TaskStarted != nil &&
				turn.TaskStarted.ModelContextWindow > 0 {
				modelContextWindow = turn.TaskStarted.ModelContextWindow
			}
			if modelContextWindow <= 0 {
				modelContextWindow = 0
			}

			entry := dto.TokenCountEntry{
				Index:              tcIdx,
				TurnIndex:          tc.TurnIndex,
				BoundToNodeID:      boundNodeID,
				ModelContextWindow: modelContextWindow,
				TotalTokenUsage:    totalUsage,
				LastTokenUsage:     lastUsage,
			}
			tokenCountsList = append(tokenCountsList, entry)
			tokenCountIndexByRecordLine[tc.Record.LineNumber] = tcIdx
			tcIdx++

			// NodeID にバッジを集約
			if boundNodeID != "" {
				// Node を探す
				nodeIdx := -1
				for idx := range nodes {
					if nodes[idx].ID == boundNodeID {
						nodeIdx = idx
						break
					}
				}

				if nodeIdx >= 0 {
					var consumedTokens int64
					if lastUsage != nil {
						consumedTokens = lastUsage.TotalTokens
					}

					if nodes[nodeIdx].Data.TokenBadge == nil {
						nodes[nodeIdx].Data.TokenBadge = &dto.TokenBadgeData{
							ConsumedTokens:  consumedTokens,
							TokenCountIndex: entry.Index,
							BoundCount:      1,
						}
					} else {
						// 複数ある場合は集約
						nodes[nodeIdx].Data.TokenBadge.ConsumedTokens += consumedTokens
						nodes[nodeIdx].Data.TokenBadge.BoundCount++
					}
				}
			}
		}
	}

	// 8. 統計情報 (Statistics) の算出
	var totalDuration int64
	var toolCallCount int
	var tokenCountCount int
	var windowSize int64
	var lastTotalTokens int64
	var turnStats []dto.TurnStatistics

	var prevTotalTokens, prevInputTokens, prevOutputTokens, prevReasoningTokens int64

	for _, turn := range turns {
		if turn.Index < 0 {
			continue
		}

		var turnDuration int64
		if turn.TaskComplete != nil {
			if turn.TaskComplete.DurationMs > 0 {
				turnDuration = turn.TaskComplete.DurationMs
			} else if turn.TaskStarted != nil {
				turnDuration = (turn.TaskComplete.CompletedAt - turn.TaskStarted.StartedAt) * 1000
			}
		}
		totalDuration += turnDuration

		if turn.TaskStarted != nil && turn.TaskStarted.ModelContextWindow > 0 {
			windowSize = turn.TaskStarted.ModelContextWindow
		}

		var turnToolCalls int
		for _, r := range turn.Records {
			if r.Type == "response_item" && (r.SubType == "function_call" || r.SubType == "custom_tool_call") {
				turnToolCalls++
				toolCallCount++
			}
			if r.Type == "event_msg" && r.SubType == "token_count" {
				tokenCountCount++
			}
		}

		// Turn Statistics
		var turnTotalTokens, turnInputTokens, turnOutputTokens, turnReasoningTokens int64
		if len(turn.TokenCounts) > 0 {
			lastTc := turn.TokenCounts[len(turn.TokenCounts)-1]
			if lastTc.Record.EventMsg != nil && lastTc.Record.EventMsg.Info != nil && lastTc.Record.EventMsg.Info.TotalTokenUsage != nil {
				tUsage := lastTc.Record.EventMsg.Info.TotalTokenUsage
				turnTotalTokens = tUsage.TotalTokens - prevTotalTokens
				turnInputTokens = tUsage.InputTokens - prevInputTokens
				turnOutputTokens = tUsage.OutputTokens - prevOutputTokens
				turnReasoningTokens = tUsage.ReasoningOutputTokens - prevReasoningTokens

				prevTotalTokens = tUsage.TotalTokens
				prevInputTokens = tUsage.InputTokens
				prevOutputTokens = tUsage.OutputTokens
				prevReasoningTokens = tUsage.ReasoningOutputTokens
			}
		}

		mode := "normal"
		if turn.TurnContext != nil && turn.TurnContext.CollaborationMode != nil {
			mode = turn.TurnContext.CollaborationMode.Mode
		}

		var firstTokenMs int64
		if turn.TaskComplete != nil {
			firstTokenMs = turn.TaskComplete.TimeToFirstTokenMs
		}

		tStat := dto.TurnStatistics{
			Index:                 turn.Index,
			CollaborationModeKind: mode,
			DurationMs:            turnDuration,
			TimeToFirstTokenMs:    firstTokenMs,
			TokenCountCount:       len(turn.TokenCounts),
			ConsumedTokens: dto.TokenBreakdown{
				TotalTokens:           turnTotalTokens,
				InputTokens:           turnInputTokens,
				OutputTokens:          turnOutputTokens,
				ReasoningOutputTokens: turnReasoningTokens,
			},
		}
		turnStats = append(turnStats, tStat)
	}

	if len(tokenCountsList) > 0 {
		lastEntry := tokenCountsList[len(tokenCountsList)-1]
		if lastEntry.TotalTokenUsage != nil {
			lastTotalTokens = lastEntry.TotalTokenUsage.TotalTokens
		}
	}

	stats := dto.Statistics{
		DurationMs:        totalDuration,
		TotalTokens:       lastTotalTokens,
		ToolCallCount:     toolCallCount,
		TokenCountCount:   tokenCountCount,
		ContextWindowSize: windowSize,
		TurnCount:         turnIndex,
		Turns:             turnStats,
	}

	var parentSessionID *string
	// 1. session_meta レコードの parent_thread_id から親IDを優先取得
	for _, r := range records {
		if r.Type == "session_meta" && r.SessionMeta != nil && r.SessionMeta.ParentThreadID != "" {
			pID := r.SessionMeta.ParentThreadID
			parentSessionID = &pID
			break
		}
	}

	var childSessionIDs []string
	seenChildSessionIDs := make(map[string]bool)

	// 2. 他のセッションの一覧を取得し、親子関係を逆引きで解決
	allSessions, err := uc.sessionRepo.ListSessions(ctx, 0, 0, "")
	if err == nil {
		// リスト走査により、自身を親とする子セッションIDを検索
		for i := range allSessions {
			if allSessions[i].ParentSessionID != nil && *allSessions[i].ParentSessionID == sessionID {
				tid := allSessions[i].ID
				if !seenChildSessionIDs[tid] {
					seenChildSessionIDs[tid] = true
					childSessionIDs = append(childSessionIDs, tid)
				}
			}
		}
		// 逆引きで親IDが見つからない場合のフォールバック（従来のキャッシュ解決）
		if parentSessionID == nil {
			for i := range allSessions {
				for _, cid := range allSessions[i].ChildSessionIDs {
					if cid == sessionID {
						pID := allSessions[i].ID
						parentSessionID = &pID
						break
					}
				}
				if parentSessionID != nil {
					break
				}
			}
		}
	}

	// 3. レコード内の collab_agent_spawn_end レコードも念のためパース（後方互換性）
	for _, r := range records {
		if r.Type == "event_msg" && r.SubType == "collab_agent_spawn_end" && r.EventMsg != nil && r.EventMsg.NewThreadID != "" {
			tid := r.EventMsg.NewThreadID
			if !seenChildSessionIDs[tid] {
				seenChildSessionIDs[tid] = true
				childSessionIDs = append(childSessionIDs, tid)
			}
		}
	}

	// 4. 親のレコード内の spawn_agent ツール呼び出しとその結果から抽出
	for _, r := range records {
		if r.Type == "response_item" && r.SubType == "function_call_output" && r.ResponseItem != nil && r.ResponseItem.Output != "" {
			var outData struct {
				AgentID string `json:"agent_id"`
			}
			if jsonErr := json.Unmarshal([]byte(r.ResponseItem.Output), &outData); jsonErr == nil && outData.AgentID != "" {
				tid := outData.AgentID
				if !seenChildSessionIDs[tid] {
					seenChildSessionIDs[tid] = true
					childSessionIDs = append(childSessionIDs, tid)
				}
			}
		}
	}

	res := &dto.SessionDetailResponse{
		ID:                 sessionID,
		CacheSchemaVersion: dto.CurrentSessionDetailCacheSchemaVersion,
		ParsedAt:           time.Now().UTC().Format(time.RFC3339),
		Nodes:              nodes,
		Edges:              edges,
		Statistics:         stats,
		TokenCounts:        tokenCountsList,
		Timeline: buildConversationTimeline(
			turns,
			turnStats,
			RecordToNodeID,
			tokenCountIndexByRecordLine,
		),
		ParentSessionID: parentSessionID,
		ChildSessionIDs: childSessionIDs,
	}

	// 9. キャッシュへの保存
	if cacheErr := uc.cacheRepo.SaveSessionDetail(ctx, sessionID, res); cacheErr != nil {
		logger.Error("failed to save session detail to cache", "session_id", sessionID, "error", cacheErr)
	}

	return res, nil
}

func isUnreadableReasoningPair(pair *model.ReasoningPair) bool {
	if pair == nil || pair.AgentReasoning != nil || pair.Reasoning == nil || pair.Reasoning.ResponseItem == nil {
		return false
	}
	for _, content := range pair.Reasoning.ResponseItem.Summary {
		if strings.TrimSpace(content.Text) != "" {
			return false
		}
	}
	return true
}

func buildDisplayUnits(turn *model.Turn) []displayUnit {
	var units []displayUnit
	consumedLines := make(map[int]bool)

	for _, pair := range turn.AgentReasonings {
		startLine := reasoningPairStartLine(pair)
		if startLine == 0 {
			continue
		}
		unit := displayUnit{
			Kind:           displayUnitReasoning,
			StartLine:      startLine,
			ReasoningPairs: []*model.ReasoningPair{pair},
		}
		if pair.AgentReasoning != nil {
			unit.Records = append(unit.Records, pair.AgentReasoning)
			consumedLines[pair.AgentReasoning.LineNumber] = true
		}
		if pair.Reasoning != nil {
			unit.Records = append(unit.Records, pair.Reasoning)
			consumedLines[pair.Reasoning.LineNumber] = true
		}
		units = append(units, unit)
	}

	for _, batch := range turn.Batches {
		if len(batch.CallRecords) == 0 {
			continue
		}
		unit := displayUnit{
			Kind:      displayUnitBatch,
			StartLine: batch.CallRecords[0].LineNumber,
			Batch:     batch,
		}
		unit.Records = append(unit.Records, batch.CallRecords...)
		unit.Records = append(unit.Records, batch.OutputRecords...)
		unit.Records = append(unit.Records, batch.MiddleMessage...)
		for _, record := range unit.Records {
			if record != nil {
				consumedLines[record.LineNumber] = true
			}
		}
		units = append(units, unit)
	}

	for recordIndex := 0; recordIndex < len(turn.Records); recordIndex++ {
		record := turn.Records[recordIndex]
		if consumedLines[record.LineNumber] {
			continue
		}

		switch {
		case isAgentMessageRecord(record):
			unit := displayUnit{
				Kind:      displayUnitAgentMessage,
				StartLine: record.LineNumber,
				Records:   []*model.TypedRecord{record},
			}
			consumedLines[record.LineNumber] = true
			if recordIndex+1 < len(turn.Records) {
				nextRecord := turn.Records[recordIndex+1]
				if !consumedLines[nextRecord.LineNumber] && isAgentMessageRecord(nextRecord) {
					unit.Records = append(unit.Records, nextRecord)
					consumedLines[nextRecord.LineNumber] = true
					recordIndex++
				}
			}
			units = append(units, unit)
		case containsRecord(turn.WebSearchRecords, record.LineNumber):
			units = append(units, displayUnit{Kind: displayUnitWebSearch, StartLine: record.LineNumber, Records: []*model.TypedRecord{record}})
			consumedLines[record.LineNumber] = true
		case containsRecord(turn.ItemCompleted, record.LineNumber):
			units = append(units, displayUnit{Kind: displayUnitItemComplete, StartLine: record.LineNumber, Records: []*model.TypedRecord{record}})
			consumedLines[record.LineNumber] = true
		case containsRecord(turn.GenericRecords, record.LineNumber):
			units = append(units, displayUnit{Kind: displayUnitGeneric, StartLine: record.LineNumber, Records: []*model.TypedRecord{record}})
			consumedLines[record.LineNumber] = true
		}
	}

	sort.SliceStable(units, func(i, j int) bool {
		return units[i].StartLine < units[j].StartLine
	})

	merged := make([]displayUnit, 0, len(units))
	for _, unit := range units {
		if len(merged) > 0 &&
			isUnreadableReasoningUnit(&merged[len(merged)-1]) &&
			isUnreadableReasoningUnit(&unit) {
			merged[len(merged)-1].Records = append(merged[len(merged)-1].Records, unit.Records...)
			merged[len(merged)-1].ReasoningPairs = append(merged[len(merged)-1].ReasoningPairs, unit.ReasoningPairs...)
			continue
		}
		merged = append(merged, unit)
	}
	return merged
}

func reasoningPairStartLine(pair *model.ReasoningPair) int {
	if pair == nil {
		return 0
	}
	if pair.AgentReasoning != nil && pair.Reasoning != nil {
		return min(pair.AgentReasoning.LineNumber, pair.Reasoning.LineNumber)
	}
	if pair.AgentReasoning != nil {
		return pair.AgentReasoning.LineNumber
	}
	if pair.Reasoning != nil {
		return pair.Reasoning.LineNumber
	}
	return 0
}

func isUnreadableReasoningUnit(unit *displayUnit) bool {
	if unit == nil || unit.Kind != displayUnitReasoning || len(unit.ReasoningPairs) == 0 {
		return false
	}
	for _, pair := range unit.ReasoningPairs {
		if !isUnreadableReasoningPair(pair) {
			return false
		}
	}
	return true
}

func isAgentMessageRecord(record *model.TypedRecord) bool {
	if record == nil {
		return false
	}
	return (record.Type == "event_msg" && record.SubType == "agent_message") ||
		(record.Type == "response_item" && record.SubType == "message" && record.ResponseItem != nil && record.ResponseItem.Role == "assistant")
}

func agentMessageTexts(record *model.TypedRecord) []string {
	if record == nil {
		return nil
	}
	if record.Type == "event_msg" && record.EventMsg != nil && record.EventMsg.Message != "" {
		return []string{record.EventMsg.Message}
	}
	if record.Type == "response_item" && record.ResponseItem != nil {
		texts := make([]string, 0, len(record.ResponseItem.Content))
		for _, content := range record.ResponseItem.Content {
			texts = append(texts, content.Text)
		}
		return texts
	}
	return nil
}

func containsRecord(records []*model.TypedRecord, lineNumber int) bool {
	for _, record := range records {
		if record != nil && record.LineNumber == lineNumber {
			return true
		}
	}
	return false
}

func isUnsupportedVersion(versionStr string) bool {
	versionStr = strings.TrimPrefix(versionStr, "v")
	parts := strings.Split(versionStr, ".")
	if len(parts) == 0 || parts[0] == "" {
		return false
	}
	var major, minor, patch int
	if _, err := fmt.Sscanf(versionStr, "%d.%d.%d", &major, &minor, &patch); err != nil {
		return true // パースに失敗した場合は安全側に倒して非対応形式とみなす
	}
	if major > 0 {
		return false
	}
	if minor < 121 {
		return true
	}
	return false
}
