package usecase

import (
	"codex-session-display/internal/domain/dto"
	"codex-session-display/internal/domain/model"
	"codex-session-display/internal/utils/logger"
	"context"
	"errors"
	"fmt"
	"io/fs"
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

func NewGetSessionDetailUseCase(sessionRepo SessionRepository, cacheRepo CacheRepository, parser SessionParser) *GetSessionDetailUseCase {
	return &GetSessionDetailUseCase{
		sessionRepo: sessionRepo,
		cacheRepo:   cacheRepo,
		parser:      parser,
	}
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
			// thread_name is not currently used in the response
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

			// 2e. reasoning ノード
			for _, pair := range turn.AgentReasonings {
				var rLine int
				var summary, fullText string
				if pair.AgentReasoning != nil && pair.AgentReasoning.EventMsg != nil {
					rLine = pair.AgentReasoning.LineNumber
					summary = pair.AgentReasoning.EventMsg.Text
				}
				if pair.Reasoning != nil && pair.Reasoning.ResponseItem != nil {
					if rLine == 0 {
						rLine = pair.Reasoning.LineNumber
					}
					var parts []string
					for _, c := range pair.Reasoning.ResponseItem.Summary {
						parts = append(parts, c.Text)
					}
					fullText = strings.Join(parts, "\n")
				}

				if pair.AgentReasoning == nil && pair.Reasoning != nil {
					summary = "（暗号化済み・表示不可）"
					fullText = "（暗号化済み・表示不可）"
				}

				reasonNodeID := fmt.Sprintf("node-%d", rLine)
				pushHarnessNode(reasonNodeID, "reasoning", "turn", "Reasoning", "🧠", summary, fullText, nil, turn.Index, NodeHeightLarge)
				if pair.AgentReasoning != nil {
					RecordToNodeID[pair.AgentReasoning.LineNumber] = reasonNodeID
				}
				if pair.Reasoning != nil {
					RecordToNodeID[pair.Reasoning.LineNumber] = reasonNodeID
				}
			}

			// 2f. Batches (fork-join)
			for _, batch := range turn.Batches {
				harnessTopY := currentY - NodeGap - NodeHeight
				if lastHarnessNodeIsBatchJoin {
					harnessTopY = currentY
				} else if lastHarnessNodeID != "" {
					// find last harness node Y to align tools
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

					addBranchNode(callNodeID, "action", "action", call.ResponseItem.Name, "🛠️", call.ResponseItem.Arguments, call.ResponseItem.Arguments, nil, turn.Index, xPos, harnessTopY)
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

					// Corresponding Output
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

				// Middle Message
				if len(batch.MiddleMessage) > 0 {
					var texts []string
					for _, m := range batch.MiddleMessage {
						if m.Type == "event_msg" && m.EventMsg != nil && m.EventMsg.Message != "" {
							texts = append(texts, m.EventMsg.Message)
						} else if m.Type == "response_item" && m.ResponseItem != nil {
							for _, c := range m.ResponseItem.Content {
								texts = append(texts, c.Text)
							}
						}
					}
					middleSummary := strings.Join(texts, "\n")
					middleNodeID := fmt.Sprintf("node-%d", batch.MiddleMessage[0].LineNumber)

					middleNodeY := harnessTopY + NodeHeight + BatchNodeGap + NodeHeight + BatchMiddleGap
					middleNode := dto.FlowNode{
						ID: middleNodeID,
						Position: dto.Position{
							X: HarnessX,
							Y: middleNodeY,
						},
						Type: "agentMessage",
						Data: dto.NodeData{
							Category:  "turn",
							Label:     "Agent Message",
							Icon:      "🤖",
							Summary:   middleSummary,
							FullText:  middleSummary,
							TurnIndex: turn.Index,
						},
					}
					nodes = append(nodes, middleNode)

					for _, m := range batch.MiddleMessage {
						RecordToNodeID[m.LineNumber] = middleNodeID
					}

					// Connect outputs to MiddleMessage
					if len(outputNodeIDs) > 0 {
						for _, outID := range outputNodeIDs {
							edges = append(edges, dto.FlowEdge{
								ID:       fmt.Sprintf("edge-%s-%s", outID, middleNodeID),
								Source:   outID,
								Target:   middleNodeID,
								Type:     "default",
								Animated: false,
							})
						}
					} else if lastHarnessNodeID != "" {
						// Fallback: connect from harnessTop to middle message directly
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
			}

			// 2f'. web_search_call / web_search_end
			var lastWebCallID string
			for _, r := range turn.WebSearchRecords {
				if r.Type == "response_item" && r.SubType == "web_search_call" {
					var q string
					if r.ResponseItem != nil && r.ResponseItem.Action != nil {
						q = r.ResponseItem.Action.Query
						if q == "" && len(r.ResponseItem.Action.Queries) > 0 {
							q = r.ResponseItem.Action.Queries[0]
						}
					}
					nodeID := fmt.Sprintf("node-%d", r.LineNumber)
					pushHarnessNode(nodeID, "webSearchAction", "action", "Web Search", "🔍", "Query: "+q, "", nil, turn.Index, NodeHeight)
					RecordToNodeID[r.LineNumber] = nodeID
					lastWebCallID = nodeID
				} else if r.Type == "event_msg" && r.SubType == "web_search_end" {
					nodeID := fmt.Sprintf("node-%d", r.LineNumber)
					pushHarnessNode(nodeID, "webSearchAction", "action", "Web Search Completed", "🔍", "Search finished", "", nil, turn.Index, NodeHeight)
					RecordToNodeID[r.LineNumber] = nodeID
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
			}

			// 2g. item_completed
			for _, r := range turn.ItemCompleted {
				if r.EventMsg == nil || r.EventMsg.Item == nil {
					continue
				}
				nodeID := fmt.Sprintf("node-%d", r.LineNumber)
				pushHarnessNode(nodeID, "itemCompleted", "event", "Item Completed", "✅", r.EventMsg.Item.Text, "", nil, turn.Index, NodeHeight)
				RecordToNodeID[r.LineNumber] = nodeID
			}

			// 2g'. Non-batch agent_messages
			batchRecordsSet := make(map[int]bool)
			for _, batch := range turn.Batches {
				for _, r := range batch.CallRecords {
					batchRecordsSet[r.LineNumber] = true
				}
				for _, r := range batch.OutputRecords {
					if r != nil {
						batchRecordsSet[r.LineNumber] = true
					}
				}
				for _, r := range batch.MiddleMessage {
					batchRecordsSet[r.LineNumber] = true
				}
			}

			for idx := 0; idx < len(turn.Records); {
				r := turn.Records[idx]
				isAgentMsg := (r.Type == "event_msg" && r.SubType == "agent_message") ||
					(r.Type == "response_item" && r.SubType == "message" && r.ResponseItem != nil && r.ResponseItem.Role == "assistant")

				if isAgentMsg && !batchRecordsSet[r.LineNumber] {
					// Check if consecutive agent messages can be merged
					var texts []string
					if r.Type == "event_msg" && r.EventMsg != nil && r.EventMsg.Message != "" {
						texts = append(texts, r.EventMsg.Message)
					} else if r.Type == "response_item" && r.ResponseItem != nil {
						for _, c := range r.ResponseItem.Content {
							texts = append(texts, c.Text)
						}
					}

					nodeID := fmt.Sprintf("node-%d", r.LineNumber)
					RecordToNodeID[r.LineNumber] = nodeID

					// Look ahead
					if idx+1 < len(turn.Records) {
						nextR := turn.Records[idx+1]
						isNextAgentMsg := (nextR.Type == "event_msg" && nextR.SubType == "agent_message") ||
							(nextR.Type == "response_item" && nextR.SubType == "message" && nextR.ResponseItem != nil && nextR.ResponseItem.Role == "assistant")
						if isNextAgentMsg && !batchRecordsSet[nextR.LineNumber] {
							if nextR.Type == "event_msg" && nextR.EventMsg != nil && nextR.EventMsg.Message != "" {
								texts = append(texts, nextR.EventMsg.Message)
							} else if nextR.Type == "response_item" && nextR.ResponseItem != nil {
								for _, c := range nextR.ResponseItem.Content {
									texts = append(texts, c.Text)
								}
							}
							RecordToNodeID[nextR.LineNumber] = nodeID
							idx++
						}
					}

					pushHarnessNode(nodeID, "agentMessage", "turn", "Agent Message", "🤖", strings.Join(texts, "\n"), strings.Join(texts, "\n"), nil, turn.Index, NodeHeight)
				}
				idx++
			}

			// 2h. Generic records (non-batch, non-event boundary, non-token counts, etc.)
			// Filter out already handled categories
			for _, r := range turn.GenericRecords {
				nodeID := fmt.Sprintf("node-%d", r.LineNumber)
				pushHarnessNode(nodeID, "generic", "generic", "System Event", "⚙️", "Type: "+r.Type+" / Subtype: "+r.SubType, "", nil, turn.Index, NodeHeight)
				RecordToNodeID[r.LineNumber] = nodeID
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
			if tc.Record.EventMsg != nil && tc.Record.EventMsg.Info != nil {
				totalUsage = tc.Record.EventMsg.Info.TotalTokenUsage
				lastUsage = tc.Record.EventMsg.Info.LastTokenUsage
			}

			entry := dto.TokenCountEntry{
				Index:           tcIdx,
				TurnIndex:       tc.TurnIndex,
				BoundToNodeID:   boundNodeID,
				TotalTokenUsage: totalUsage,
				LastTokenUsage:  lastUsage,
			}
			tokenCountsList = append(tokenCountsList, entry)
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
					var totalTokens int64
					if totalUsage != nil {
						totalTokens = totalUsage.TotalTokens
					}

					if nodes[nodeIdx].Data.TokenBadge == nil {
						nodes[nodeIdx].Data.TokenBadge = &dto.TokenBadgeData{
							TotalTokens:     totalTokens,
							TokenCountIndex: entry.Index,
							BoundCount:      1,
						}
					} else {
						// 複数ある場合は集約
						nodes[nodeIdx].Data.TokenBadge.TotalTokens = totalTokens
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

	res := &dto.SessionDetailResponse{
		ID:          sessionID,
		ParsedAt:    time.Now().UTC().Format(time.RFC3339),
		Nodes:       nodes,
		Edges:       edges,
		Statistics:  stats,
		TokenCounts: tokenCountsList,
	}

	// 9. キャッシュへの保存
	if cacheErr := uc.cacheRepo.SaveSessionDetail(ctx, sessionID, res); cacheErr != nil {
		logger.Error("failed to save session detail to cache", "session_id", sessionID, "error", cacheErr)
	}

	return res, nil
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
