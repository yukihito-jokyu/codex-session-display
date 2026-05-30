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
)

// ErrSessionMetaNotFound はキャッシュ内に sessionMeta ノードが見つからない場合のエラーです。
var ErrSessionMetaNotFound = errors.New("sessionMeta node not found in cache")

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
func (r *CacheFSRepository) GetSessionSummary(ctx context.Context, sessionID string) (*dto.SessionSummary, error) {
	cachePath := filepath.Join(r.cacheDir, sessionID+".json")

	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, err
	}

	var detail dto.SessionDetailResponse
	if err := json.Unmarshal(data, &detail); err != nil {
		return nil, fmt.Errorf("failed to decode cache JSON: %w", err)
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
		return nil, ErrSessionMetaNotFound
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
		ID:            sessionID,
		Cwd:           cwd,
		CliVersion:    cliVer,
		Originator:    orig,
		ModelProvider: provider,
		Branch:        branch,
		Source:        source,
		Timestamp:     timestamp,
		Parsed:        true,
	}

	logger.Info("cache read successful", "session_id", sessionID)

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

// GetSessionDetail はキャッシュファイルを読み込み、SessionDetailResponse を返します。
func (r *CacheFSRepository) GetSessionDetail(ctx context.Context, sessionID string) (*dto.SessionDetailResponse, error) {
	cachePath := filepath.Join(r.cacheDir, sessionID+".json")

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
func (r *CacheFSRepository) SaveSessionDetail(ctx context.Context, sessionID string, detail *dto.SessionDetailResponse) error {
	cachePath := filepath.Join(r.cacheDir, sessionID+".json")

	data, err := json.Marshal(detail)
	if err != nil {
		return fmt.Errorf("failed to marshal cache JSON: %w", err)
	}

	if err := os.WriteFile(cachePath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	logger.Info("cache write successful", "session_id", sessionID)
	return nil
}
