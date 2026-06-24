package repository

import (
	"codex-session-display/internal/domain/dto"
	"codex-session-display/internal/utils/logger"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ErrSessionMetaNotFound はキャッシュ内に sessionMeta ノードが見つからない場合のエラーです。
var ErrSessionMetaNotFound = errors.New("sessionMeta node not found in cache")

// ErrCacheSchemaMismatch はキャッシュ形式が現行バージョンと異なる場合のエラーです。
var ErrCacheSchemaMismatch = errors.New("cache schema version mismatch")

// CacheFSRepository は usecase.CacheRepository を実装します。
type CacheFSRepository struct {
	cacheDir string
}

// NewCacheFSRepository は新しい CacheFSRepository を作成します。
func NewCacheFSRepository(cacheDir string) *CacheFSRepository {
	return &CacheFSRepository{
		cacheDir: cacheDir,
	}
}

// GetSessionSummary はキャッシュファイルを読み込み、解析されたメタデータを含む SessionSummary を返します。
// まず軽量な .summary.json の読み込みを試み、なければ .json から読み込んで summary を抽出します。
func (r *CacheFSRepository) GetSessionSummary(ctx context.Context, sessionID string) (*dto.SessionSummary, error) {
	summaryPath := filepath.Join(r.cacheDir, sessionID+".summary.json")

	if data, err := os.ReadFile(summaryPath); err == nil {
		var summary dto.SessionSummary
		if unmarshalErr := json.Unmarshal(data, &summary); unmarshalErr == nil {
			summary.Parsed = true
			logger.Info("cache read successful (summary)", "session_id", sessionID)

			// 統計情報が不足している場合、詳細キャッシュから集計してマージする
			if summary.TotalTokens == nil ||
				summary.InputTokens == nil ||
				summary.OutputTokens == nil ||
				summary.ReasoningTokens == nil ||
				summary.TurnCount == nil ||
				summary.StepCount == nil ||
				summary.DurationMs == nil {
				logger.Info("statistics missing in summary cache, falling back to detail cache for merge", "session_id", sessionID)
				if detail, err := r.GetSessionDetail(ctx, sessionID); err == nil {
					var inputTokensSum int64
					var outputTokensSum int64
					var reasoningTokensSum int64
					for i := range detail.Statistics.Turns {
						inputTokensSum += detail.Statistics.Turns[i].ConsumedTokens.InputTokens
						outputTokensSum += detail.Statistics.Turns[i].ConsumedTokens.OutputTokens
						reasoningTokensSum += detail.Statistics.Turns[i].ConsumedTokens.ReasoningOutputTokens
					}

					totalTokensVal := detail.Statistics.TotalTokens
					turnCountVal := detail.Statistics.TurnCount
					stepCountVal := detail.Statistics.ToolCallCount
					durationMsVal := detail.Statistics.DurationMs

					summary.TotalTokens = &totalTokensVal
					summary.InputTokens = &inputTokensSum
					summary.OutputTokens = &outputTokensSum
					summary.ReasoningTokens = &reasoningTokensSum
					summary.TurnCount = &turnCountVal
					summary.StepCount = &stepCountVal
					summary.DurationMs = &durationMsVal

					// 更新されたサマリーを再度キャッシュに保存して以降の読み込みを高速化する（失敗しても処理は続行）
					if summaryData, err := json.Marshal(summary); err == nil {
						if writeErr := os.WriteFile(summaryPath, summaryData, 0o644); writeErr != nil {
							logger.Warn("failed to update summary cache with merged statistics", "session_id", sessionID, "error", writeErr)
						}
					}
				} else {
					logger.Warn("failed to load detail cache for merging statistics", "session_id", sessionID, "error", err)
				}
			}

			return &summary, nil
		} else {
			logger.Warn("failed to decode summary cache JSON, falling back to detail cache", "session_id", sessionID, "error", unmarshalErr)
		}
	}

	cachePath := r.cachePath(sessionID)
	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, err
	}

	var detail dto.SessionDetailResponse
	if err := json.Unmarshal(data, &detail); err != nil {
		return nil, fmt.Errorf("failed to decode cache JSON: %w", err)
	}
	if detail.CacheSchemaVersion != dto.CurrentSessionDetailCacheSchemaVersion {
		return nil, ErrCacheSchemaMismatch
	}

	// sessionMeta ノードを探す
	var metaNode *dto.FlowNode
	for i := range detail.Nodes {
		if detail.Nodes[i].Type == "sessionMeta" {
			metaNode = &detail.Nodes[i]
			break
		}
	}

	if metaNode == nil {
		summary := &dto.SessionSummary{
			ID:     sessionID,
			Parsed: true,
		}
		logger.Info("cache read successful without sessionMeta", "session_id", sessionID)
		return summary, nil
	}

	m := metaNode.Data.Meta
	cwd := getStringPtr(m, "cwd")
	cliVer := getStringPtr(m, "cli_version")
	orig := getStringPtr(m, "originator")
	provider := getStringPtr(m, "model_provider")
	branch := getStringPtr(m, "git_branch")
	source := getStringPtr(m, "source")
	timestamp := getStringPtr(m, "timestamp")

	summary := &dto.SessionSummary{
		ID:              sessionID,
		Cwd:             cwd,
		CliVersion:      cliVer,
		Originator:      orig,
		ModelProvider:   provider,
		Branch:          branch,
		Source:          source,
		Timestamp:       timestamp,
		Parsed:          true,
		ParentSessionID: detail.ParentSessionID,
		ChildSessionIDs: detail.ChildSessionIDs,
		DurationMs:      &detail.Statistics.DurationMs,
	}

	logger.Info("cache read successful (fallback to detail)", "session_id", sessionID)

	return summary, nil
}

func getStringPtr(m map[string]interface{}, key string) *string {
	if m == nil {
		return nil
	}
	val, ok := m[key]
	if !ok || val == nil {
		return nil
	}
	str, ok := val.(string)
	if !ok {
		return nil
	}
	return &str
}

// GetSessionDetailModTime はキャッシュファイルの最終更新日時を返します。
func (r *CacheFSRepository) GetSessionDetailModTime(ctx context.Context, sessionID string) (time.Time, error) {
	info, err := os.Stat(r.cachePath(sessionID))
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}

// GetSessionDetail はキャッシュファイルを読み込み、SessionDetailResponse を返します。
func (r *CacheFSRepository) GetSessionDetail(ctx context.Context, sessionID string) (*dto.SessionDetailResponse, error) {
	cachePath := r.cachePath(sessionID)

	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, err
	}

	var detail dto.SessionDetailResponse
	if err := json.Unmarshal(data, &detail); err != nil {
		return nil, fmt.Errorf("failed to decode cache JSON: %w", err)
	}

	return &detail, nil
}

// SaveSessionDetail は SessionDetailResponse をキャッシュファイルに書き込みます。
// 同時に、軽量な .summary.json も保存します。
func (r *CacheFSRepository) SaveSessionDetail(ctx context.Context, sessionID string, detail *dto.SessionDetailResponse) error {
	cachePath := r.cachePath(sessionID)

	data, err := json.Marshal(detail)
	if err != nil {
		return fmt.Errorf("failed to marshal cache JSON: %w", err)
	}

	if err := os.WriteFile(cachePath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	// summary も保存する
	var metaNode *dto.FlowNode
	for i := range detail.Nodes {
		if detail.Nodes[i].Type == "sessionMeta" {
			metaNode = &detail.Nodes[i]
			break
		}
	}

	var summary dto.SessionSummary
	if metaNode == nil {
		summary = dto.SessionSummary{
			ID:     sessionID,
			Parsed: true,
		}
	} else {
		m := metaNode.Data.Meta
		summary = dto.SessionSummary{
			ID:              sessionID,
			Cwd:             getStringPtr(m, "cwd"),
			CliVersion:      getStringPtr(m, "cli_version"),
			Originator:      getStringPtr(m, "originator"),
			ModelProvider:   getStringPtr(m, "model_provider"),
			Branch:          getStringPtr(m, "git_branch"),
			Source:          getStringPtr(m, "source"),
			Timestamp:       getStringPtr(m, "timestamp"),
			Parsed:          true,
			ParentSessionID: detail.ParentSessionID,
			ChildSessionIDs: detail.ChildSessionIDs,
		}
	}

	// 統計情報の集計
	var inputTokensSum int64
	var outputTokensSum int64
	var reasoningTokensSum int64
	for i := range detail.Statistics.Turns {
		inputTokensSum += detail.Statistics.Turns[i].ConsumedTokens.InputTokens
		outputTokensSum += detail.Statistics.Turns[i].ConsumedTokens.OutputTokens
		reasoningTokensSum += detail.Statistics.Turns[i].ConsumedTokens.ReasoningOutputTokens
	}

	totalTokensVal := detail.Statistics.TotalTokens
	turnCountVal := detail.Statistics.TurnCount
	stepCountVal := detail.Statistics.ToolCallCount
	durationMsVal := detail.Statistics.DurationMs

	summary.TotalTokens = &totalTokensVal
	summary.InputTokens = &inputTokensSum
	summary.OutputTokens = &outputTokensSum
	summary.ReasoningTokens = &reasoningTokensSum
	summary.TurnCount = &turnCountVal
	summary.StepCount = &stepCountVal
	summary.DurationMs = &durationMsVal

	summaryData, err := json.Marshal(summary)
	if err != nil {
		return fmt.Errorf("failed to marshal summary cache JSON: %w", err)
	}

	summaryPath := filepath.Join(r.cacheDir, sessionID+".summary.json")
	if err := os.WriteFile(summaryPath, summaryData, 0o644); err != nil {
		return fmt.Errorf("failed to write summary cache file: %w", err)
	}

	logger.Info("cache write successful", "session_id", sessionID)
	return nil
}

func (r *CacheFSRepository) cachePath(sessionID string) string {
	return filepath.Join(r.cacheDir, sessionID+".json")
}
