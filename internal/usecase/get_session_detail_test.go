package usecase_test

import (
	"bytes"
	"codex-session-display/internal/domain/dto"
	"codex-session-display/internal/domain/model"
	"codex-session-display/internal/repository"
	"codex-session-display/internal/usecase"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// mockCacheRepositoryForDetail はテスト用のモックキャッシュリポジトリです。
type mockCacheRepositoryForDetail struct {
	cache      map[string]*dto.SessionDetailResponse
	modTimes   map[string]time.Time
	saveErr    error
	saveCount  int
	lastSaved  *dto.SessionDetailResponse
	lastSaveID string
}

func (m *mockCacheRepositoryForDetail) GetSessionSummary(ctx context.Context, provider dto.SessionProvider, sessionID string) (*dto.SessionSummary, error) {
	return nil, errors.New("not implemented")
}

func (m *mockCacheRepositoryForDetail) GetSessionDetail(ctx context.Context, provider dto.SessionProvider, sessionID string) (*dto.SessionDetailResponse, error) {
	if m.cache == nil {
		return nil, errors.New("not found")
	}
	if detail, ok := m.cache[sessionID]; ok {
		return detail, nil
	}
	return nil, errors.New("not found")
}

func (m *mockCacheRepositoryForDetail) GetSessionDetailModTime(ctx context.Context, provider dto.SessionProvider, sessionID string) (time.Time, error) {
	if m.modTimes != nil {
		if modTime, ok := m.modTimes[sessionID]; ok {
			return modTime, nil
		}
	}
	return time.Time{}, errors.New("not found")
}

func (m *mockCacheRepositoryForDetail) SaveSessionDetail(ctx context.Context, provider dto.SessionProvider, sessionID string, detail *dto.SessionDetailResponse) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	if m.cache == nil {
		m.cache = make(map[string]*dto.SessionDetailResponse)
	}
	m.saveCount++
	m.lastSaveID = sessionID
	m.lastSaved = detail
	m.cache[sessionID] = detail
	return nil
}

// mockSessionRepositoryForDetail はテスト用のモックセッションリポジトリです。
type mockSessionRepositoryForDetail struct {
	paths       map[string]string
	filePathErr error
	modTimeErr  error
	modTimes    map[string]time.Time
	sessions    []dto.SessionSummary
}

func (m *mockSessionRepositoryForDetail) ListSessions(ctx context.Context, provider dto.SessionProvider, year, month int, query string) ([]dto.SessionSummary, error) {
	if m.sessions != nil {
		return m.sessions, nil
	}
	return nil, errors.New("not implemented")
}

func (m *mockSessionRepositoryForDetail) GetSessionFilePath(ctx context.Context, sessionID string) (string, error) {
	if m.filePathErr != nil {
		return "", m.filePathErr
	}
	if path, ok := m.paths[sessionID]; ok {
		return path, nil
	}
	return "", errors.New("session not found")
}

func (m *mockSessionRepositoryForDetail) GetSessionIDByFilePath(ctx context.Context, filePath string) (string, error) {
	return "", errors.New("not implemented")
}

func (m *mockSessionRepositoryForDetail) GetSessionModTime(ctx context.Context, sessionID string) (time.Time, error) {
	if m.modTimeErr != nil {
		return time.Time{}, m.modTimeErr
	}
	if m.modTimes != nil {
		if t, ok := m.modTimes[sessionID]; ok {
			return t, nil
		}
	}
	path, err := m.GetSessionFilePath(ctx, sessionID)
	if err != nil {
		return time.Time{}, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}

// mockSessionParser はテスト用のモックパーサーです。
type mockSessionParser struct {
	records []*model.TypedRecord
	err     error
	calls   int
}

func (m *mockSessionParser) ParseSessionFile(ctx context.Context, filePath string) ([]*model.TypedRecord, error) {
	m.calls++
	if m.err != nil {
		return nil, m.err
	}
	return m.records, nil
}

func setupUsecaseTestLogger(t *testing.T) (*bytes.Buffer, func()) {
	t.Helper()
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	oldDefault := slog.Default()
	slog.SetDefault(slog.New(handler))
	return &buf, func() {
		slog.SetDefault(oldDefault)
	}
}

func TestGetSessionDetailUseCase_Execute(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T, tmpDir string) (sessionID string, sessionRepo usecase.SessionRepository, cacheRepo usecase.CacheRepository, parser usecase.SessionParser, cleanup func())
		wantErr bool
		verify  func(t *testing.T, res *dto.SessionDetailResponse, err error)
	}{
		{
			name: "unsupported version",
			setup: func(t *testing.T, tmpDir string) (string, usecase.SessionRepository, usecase.CacheRepository, usecase.SessionParser, func()) {
				filePath := filepath.Join(tmpDir, "rollout-unsupported.jsonl")
				logs := []string{
					`{"type":"session_meta","timestamp":1717084800,"payload":{"id":"session-1","cli_version":"0.120.0"}}`,
				}
				if err := os.WriteFile(filePath, []byte(strings.Join(logs, "\n")), 0o644); err != nil {
					t.Fatal(err)
				}
				sessionRepo := &mockSessionRepositoryForDetail{
					paths: map[string]string{"session-1": filePath},
				}
				cacheRepo := &mockCacheRepositoryForDetail{}
				return "session-1", sessionRepo, cacheRepo, nil, nil
			},
			wantErr: true,
			verify: func(t *testing.T, res *dto.SessionDetailResponse, err error) {
				if !errors.Is(err, model.ErrUnsupportedFormat) {
					t.Errorf("expected unsupported format error, got %v", err)
				}
			},
		},
		{
			name: "normal flow and layout",
			setup: func(t *testing.T, tmpDir string) (string, usecase.SessionRepository, usecase.CacheRepository, usecase.SessionParser, func()) {
				filePath := filepath.Join(tmpDir, "rollout-success.jsonl")
				logs := []string{
					// Session Meta (v0.131.0)
					`{"type":"session_meta","timestamp":1717084800,"payload":{"id":"session-1234","cwd":"/path/to/project","cli_version":"v0.131.0","base_instructions":{"text":"Base System Instructions"}}}`,
					// Turn 1 Context
					`{"type":"turn_context","timestamp":1717084805,"payload":{"turn_id":"turn-1","model":"gpt-4o","user_instructions":"Fix it","collaboration_mode":{"mode":"normal","developer_instructions":"Dev Instructions"}}}`,
					// Turn 1 Start
					`{"type":"event_msg","timestamp":1717084810,"payload":{"type":"task_started","turn_id":"turn-1","started_at":1717084810,"model_context_window":128000}}`,
					// User message
					`{"type":"event_msg","timestamp":1717084812,"payload":{"type":"user_message","message":"Let's go"}}`,
					// Reasoning
					`{"type":"event_msg","timestamp":1717084815,"payload":{"type":"agent_reasoning","text":"Thinking hard"}}`,
					`{"type":"response_item","timestamp":1717084816,"payload":{"type":"reasoning","summary":[{"type":"summary_text","text":"Thinking summary"}]}}`,
					// Batch function calls
					`{"type":"response_item","timestamp":1717084820,"payload":{"type":"function_call","name":"read_file","call_id":"call-1","arguments":"{}"}}`,
					`{"type":"response_item","timestamp":1717084821,"payload":{"type":"function_call","name":"write_file","call_id":"call-2","arguments":"{}"}}`,
					// Batch Middle Message (Pattern A)
					`{"type":"event_msg","timestamp":1717084822,"payload":{"type":"agent_message","message":"Running tool"}}`,
					`{"type":"response_item","timestamp":1717084823,"payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"Tool is running"}]}}`,
					// Batch function outputs
					`{"type":"response_item","timestamp":1717084825,"payload":{"type":"function_call_output","call_id":"call-1","output":"file content"}}`,
					`{"type":"response_item","timestamp":1717084826,"payload":{"type":"function_call_output","call_id":"call-2","output":"success"}}`,
					// External event branch (exec_command_end)
					`{"type":"event_msg","timestamp":1717084828,"payload":{"type":"exec_command_end","call_id":"call-1","command":["cat","file"],"exit_code":0}}`,
					// Token count
					`{"type":"event_msg","timestamp":1717084830,"payload":{"type":"token_count","info":{"total_token_usage":{"total_tokens":1000,"input_tokens":800,"output_tokens":200},"last_token_usage":{"total_tokens":150,"input_tokens":100,"output_tokens":50},"model_context_window":64000}}}`,
					// Turn 1 End
					`{"type":"event_msg","timestamp":1717084840,"payload":{"type":"task_complete","turn_id":"turn-1","completed_at":1717084840}}`,
				}
				if err := os.WriteFile(filePath, []byte(strings.Join(logs, "\n")), 0o644); err != nil {
					t.Fatal(err)
				}
				sessionRepo := &mockSessionRepositoryForDetail{
					paths: map[string]string{"session-1234": filePath},
				}
				cacheRepo := &mockCacheRepositoryForDetail{}
				return "session-1234", sessionRepo, cacheRepo, nil, nil
			},
			wantErr: false,
			verify: func(t *testing.T, res *dto.SessionDetailResponse, err error) {
				if res.ID != "session-1234" {
					t.Errorf("expected ID 'session-1234', got '%s'", res.ID)
				}

				if len(res.Nodes) == 0 {
					t.Fatal("expected nodes to be generated, got 0")
				}

				var metaNode, baseInstNode, contextNode, startNode, reasoningNode, middleNode, endNode *dto.FlowNode
				for i := range res.Nodes {
					n := &res.Nodes[i]
					switch n.Type {
					case "sessionMeta":
						metaNode = n
					case "contextDoc":
						if strings.Contains(n.Data.Summary, "Base System Instructions") {
							baseInstNode = n
						}
					case "turnContext":
						contextNode = n
					case "taskEvent":
						switch n.Data.Label {
						case "Task Started":
							startNode = n
						case "Task Complete":
							endNode = n
						}
					case "reasoning":
						reasoningNode = n
					case "agentMessage":
						if strings.Contains(n.Data.Summary, "Tool is running") {
							middleNode = n
						}
					}
				}

				if metaNode == nil || metaNode.Position.X != 0 || metaNode.Position.Y != 0 {
					t.Errorf("invalid sessionMeta node: %+v", metaNode)
				}

				if baseInstNode == nil || baseInstNode.Position.X != 400 || baseInstNode.Position.Y != 0 {
					t.Errorf("invalid base_instructions contextDoc node: %+v", baseInstNode)
				}

				if startNode == nil || startNode.Position.X != 0 || startNode.Position.Y != 120 { // 0 + 80 + 40 = 120
					t.Errorf("invalid task_started node: %+v", startNode)
				}

				if contextNode == nil || contextNode.Position.X != 0 || contextNode.Position.Y != 240 { // 120 + 80 + 40 = 240
					t.Errorf("invalid turnContext node: %+v", contextNode)
				}

				if reasoningNode == nil || reasoningNode.Position.X != 0 {
					t.Errorf("invalid reasoningNode: %+v", reasoningNode)
				}

				if middleNode == nil || middleNode.Position.X != 0 {
					t.Errorf("invalid middleNode: %+v", middleNode)
				}

				if endNode == nil || endNode.Position.X != 0 {
					t.Errorf("invalid endNode: %+v", endNode)
				}

				if len(res.TokenCounts) != 1 {
					t.Errorf("expected 1 token count entry, got %d", len(res.TokenCounts))
				} else {
					entry := res.TokenCounts[0]
					if entry.TotalTokenUsage == nil || entry.TotalTokenUsage.TotalTokens != 1000 {
						t.Errorf("incorrect token usage in entry: %+v", entry.TotalTokenUsage)
					}
					if entry.ModelContextWindow != 64000 {
						t.Errorf("ModelContextWindow = %d, want 64000", entry.ModelContextWindow)
					}
				}
			},
		},
		{
			name: "token count context window falls back to task started",
			setup: func(t *testing.T, tmpDir string) (string, usecase.SessionRepository, usecase.CacheRepository, usecase.SessionParser, func()) {
				filePath := filepath.Join(tmpDir, "rollout-context-window-fallback.jsonl")
				sessionRepo := &mockSessionRepositoryForDetail{
					paths: map[string]string{"session-context-window-fallback": filePath},
				}
				cacheRepo := &mockCacheRepositoryForDetail{}
				parser := &mockSessionParser{
					records: []*model.TypedRecord{
						{
							LineNumber:  1,
							Type:        "session_meta",
							SessionMeta: &model.SessionMetaPayload{ID: "session-context-window-fallback", CliVersion: "v0.131.0"},
						},
						{
							LineNumber: 2,
							Type:       "event_msg",
							SubType:    "task_started",
							EventMsg: &model.EventMsgPayload{
								TurnID:             "turn-1",
								ModelContextWindow: 128000,
							},
						},
						{
							LineNumber: 3,
							Type:       "event_msg",
							SubType:    "agent_message",
							EventMsg:   &model.EventMsgPayload{Message: "Working"},
						},
						{
							LineNumber: 4,
							Type:       "event_msg",
							SubType:    "token_count",
							EventMsg: &model.EventMsgPayload{
								Info: &model.TokenInfo{
									LastTokenUsage: &model.TokenDetail{InputTokens: 64000},
								},
							},
						},
						{
							LineNumber: 5,
							Type:       "event_msg",
							SubType:    "task_complete",
							EventMsg:   &model.EventMsgPayload{TurnID: "turn-1"},
						},
					},
				}
				return "session-context-window-fallback", sessionRepo, cacheRepo, parser, nil
			},
			wantErr: false,
			verify: func(t *testing.T, res *dto.SessionDetailResponse, err error) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if len(res.TokenCounts) != 1 {
					t.Fatalf("TokenCounts length = %d, want 1", len(res.TokenCounts))
				}
				if res.TokenCounts[0].ModelContextWindow != 128000 {
					t.Errorf(
						"ModelContextWindow = %d, want 128000",
						res.TokenCounts[0].ModelContextWindow,
					)
				}
			},
		},
		{
			name: "non-positive token count context window is omitted without a valid fallback",
			setup: func(t *testing.T, tmpDir string) (string, usecase.SessionRepository, usecase.CacheRepository, usecase.SessionParser, func()) {
				filePath := filepath.Join(tmpDir, "rollout-invalid-context-window.jsonl")
				sessionRepo := &mockSessionRepositoryForDetail{
					paths: map[string]string{"session-invalid-context-window": filePath},
				}
				cacheRepo := &mockCacheRepositoryForDetail{}
				parser := &mockSessionParser{
					records: []*model.TypedRecord{
						{
							LineNumber:  1,
							Type:        "session_meta",
							SessionMeta: &model.SessionMetaPayload{ID: "session-invalid-context-window", CliVersion: "v0.131.0"},
						},
						{
							LineNumber: 2,
							Type:       "event_msg",
							SubType:    "task_started",
							EventMsg: &model.EventMsgPayload{
								TurnID:             "turn-1",
								ModelContextWindow: 0,
							},
						},
						{
							LineNumber: 3,
							Type:       "event_msg",
							SubType:    "token_count",
							EventMsg: &model.EventMsgPayload{
								Info: &model.TokenInfo{
									ModelContextWindow: -1,
									LastTokenUsage:     &model.TokenDetail{InputTokens: 100},
								},
							},
						},
						{
							LineNumber: 4,
							Type:       "event_msg",
							SubType:    "task_complete",
							EventMsg:   &model.EventMsgPayload{TurnID: "turn-1"},
						},
					},
				}
				return "session-invalid-context-window", sessionRepo, cacheRepo, parser, nil
			},
			wantErr: false,
			verify: func(t *testing.T, res *dto.SessionDetailResponse, err error) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if len(res.TokenCounts) != 1 {
					t.Fatalf("TokenCounts length = %d, want 1", len(res.TokenCounts))
				}
				if res.TokenCounts[0].ModelContextWindow != 0 {
					t.Errorf(
						"ModelContextWindow = %d, want 0",
						res.TokenCounts[0].ModelContextWindow,
					)
				}
			},
		},
		{
			name: "consecutive unreadable reasoning records are aggregated",
			setup: func(t *testing.T, tmpDir string) (string, usecase.SessionRepository, usecase.CacheRepository, usecase.SessionParser, func()) {
				filePath := filepath.Join(tmpDir, "rollout-unreadable-reasoning.jsonl")
				sessionRepo := &mockSessionRepositoryForDetail{
					paths: map[string]string{"session-unreadable-reasoning": filePath},
				}
				cacheRepo := &mockCacheRepositoryForDetail{}
				parser := &mockSessionParser{
					records: []*model.TypedRecord{
						{
							LineNumber:  1,
							Type:        "session_meta",
							SessionMeta: &model.SessionMetaPayload{ID: "session-unreadable-reasoning", CliVersion: "v0.131.0"},
						},
						{
							LineNumber: 2,
							Type:       "event_msg",
							SubType:    "task_started",
							EventMsg:   &model.EventMsgPayload{TurnID: "turn-1"},
						},
						{
							LineNumber:   3,
							Type:         "response_item",
							SubType:      "reasoning",
							ResponseItem: &model.ResponseItemPayload{EncryptedContent: "encrypted-1"},
						},
						{
							LineNumber:   4,
							Type:         "response_item",
							SubType:      "reasoning",
							ResponseItem: &model.ResponseItemPayload{EncryptedContent: "encrypted-2"},
						},
						{
							LineNumber:   5,
							Type:         "response_item",
							SubType:      "reasoning",
							ResponseItem: &model.ResponseItemPayload{EncryptedContent: "encrypted-3"},
						},
						{
							LineNumber: 6,
							Type:       "event_msg",
							SubType:    "token_count",
							EventMsg: &model.EventMsgPayload{
								Info: &model.TokenInfo{
									TotalTokenUsage: &model.TokenDetail{TotalTokens: 100},
									LastTokenUsage:  &model.TokenDetail{TotalTokens: 100},
								},
							},
						},
						{
							LineNumber: 7,
							Type:       "event_msg",
							SubType:    "task_complete",
							EventMsg:   &model.EventMsgPayload{TurnID: "turn-1"},
						},
					},
				}
				return "session-unreadable-reasoning", sessionRepo, cacheRepo, parser, nil
			},
			wantErr: false,
			verify: func(t *testing.T, res *dto.SessionDetailResponse, err error) {
				var reasoningNodes []dto.FlowNode
				for _, node := range res.Nodes {
					if node.Type == "reasoning" {
						reasoningNodes = append(reasoningNodes, node)
					}
				}

				if len(reasoningNodes) != 1 {
					t.Fatalf("expected 1 reasoning node, got %d", len(reasoningNodes))
				}
				if reasoningNodes[0].Data.Summary != "（暗号化済み・表示不可）×3" {
					t.Errorf("unexpected reasoning summary: %q", reasoningNodes[0].Data.Summary)
				}
				if reasoningNodes[0].Data.FullText != "（暗号化済み・表示不可）×3" {
					t.Errorf("unexpected reasoning full text: %q", reasoningNodes[0].Data.FullText)
				}
				if len(res.TokenCounts) != 1 {
					t.Fatalf("expected 1 token count entry, got %d", len(res.TokenCounts))
				}
				if res.TokenCounts[0].BoundToNodeID != reasoningNodes[0].ID {
					t.Errorf("token count bound to %q, want %q", res.TokenCounts[0].BoundToNodeID, reasoningNodes[0].ID)
				}
			},
		},
		{
			name: "unreadable reasoning records separated by another record form separate groups",
			setup: func(t *testing.T, tmpDir string) (string, usecase.SessionRepository, usecase.CacheRepository, usecase.SessionParser, func()) {
				filePath := filepath.Join(tmpDir, "rollout-separated-unreadable-reasoning.jsonl")
				logs := []string{
					`{"type":"session_meta","timestamp":1717084800,"payload":{"id":"session-separated-unreadable-reasoning","cli_version":"v0.131.0"}}`,
					`{"type":"event_msg","timestamp":1717084810,"payload":{"type":"task_started","turn_id":"turn-1"}}`,
					`{"type":"response_item","timestamp":1717084811,"payload":{"type":"reasoning","summary":[],"encrypted_content":"encrypted-1"}}`,
					`{"type":"response_item","timestamp":1717084812,"payload":{"type":"reasoning","summary":[],"encrypted_content":"encrypted-2"}}`,
					`{"type":"event_msg","timestamp":1717084813,"payload":{"type":"agent_message","message":"Between reasoning groups"}}`,
					`{"type":"response_item","timestamp":1717084814,"payload":{"type":"reasoning","summary":[],"encrypted_content":"encrypted-3"}}`,
					`{"type":"event_msg","timestamp":1717084815,"payload":{"type":"task_complete","turn_id":"turn-1"}}`,
				}
				if err := os.WriteFile(filePath, []byte(strings.Join(logs, "\n")), 0o644); err != nil {
					t.Fatal(err)
				}
				sessionRepo := &mockSessionRepositoryForDetail{
					paths: map[string]string{"session-separated-unreadable-reasoning": filePath},
				}
				cacheRepo := &mockCacheRepositoryForDetail{}
				return "session-separated-unreadable-reasoning", sessionRepo, cacheRepo, nil, nil
			},
			wantErr: false,
			verify: func(t *testing.T, res *dto.SessionDetailResponse, err error) {
				var summaries []string
				var dynamicNodeTypes []string
				for _, node := range res.Nodes {
					if node.Type == "reasoning" {
						summaries = append(summaries, node.Data.Summary)
					}
					if node.Type == "reasoning" || node.Type == "agentMessage" {
						dynamicNodeTypes = append(dynamicNodeTypes, node.Type)
					}
				}

				want := []string{"（暗号化済み・表示不可）×2", "（暗号化済み・表示不可）×1"}
				if len(summaries) != len(want) {
					t.Fatalf("expected %d reasoning nodes, got %d: %v", len(want), len(summaries), summaries)
				}
				for i := range want {
					if summaries[i] != want[i] {
						t.Errorf("reasoning node %d summary = %q, want %q", i, summaries[i], want[i])
					}
				}
				wantNodeTypes := []string{"reasoning", "agentMessage", "reasoning"}
				for i := range wantNodeTypes {
					if dynamicNodeTypes[i] != wantNodeTypes[i] {
						t.Errorf("dynamic node %d type = %q, want %q", i, dynamicNodeTypes[i], wantNodeTypes[i])
					}
				}
			},
		},
		{
			name: "unreadable reasoning records separated only by token counts are aggregated",
			setup: func(t *testing.T, tmpDir string) (string, usecase.SessionRepository, usecase.CacheRepository, usecase.SessionParser, func()) {
				filePath := filepath.Join(tmpDir, "rollout-token-count-between-unreadable-reasoning.jsonl")
				logs := []string{
					`{"type":"session_meta","timestamp":1717084800,"payload":{"id":"session-token-count-between-unreadable-reasoning","cli_version":"v0.131.0"}}`,
					`{"type":"event_msg","timestamp":1717084810,"payload":{"type":"task_started","turn_id":"turn-1"}}`,
					`{"type":"response_item","timestamp":1717084811,"payload":{"type":"reasoning","summary":[],"encrypted_content":"encrypted-1"}}`,
					`{"type":"event_msg","timestamp":1717084812,"payload":{"type":"token_count","info":{"total_token_usage":{"total_tokens":100},"last_token_usage":{"total_tokens":100}}}}`,
					`{"type":"response_item","timestamp":1717084813,"payload":{"type":"reasoning","summary":[],"encrypted_content":"encrypted-2"}}`,
					`{"type":"event_msg","timestamp":1717084814,"payload":{"type":"task_complete","turn_id":"turn-1"}}`,
				}
				if err := os.WriteFile(filePath, []byte(strings.Join(logs, "\n")), 0o644); err != nil {
					t.Fatal(err)
				}
				sessionRepo := &mockSessionRepositoryForDetail{
					paths: map[string]string{"session-token-count-between-unreadable-reasoning": filePath},
				}
				cacheRepo := &mockCacheRepositoryForDetail{}
				return "session-token-count-between-unreadable-reasoning", sessionRepo, cacheRepo, nil, nil
			},
			wantErr: false,
			verify: func(t *testing.T, res *dto.SessionDetailResponse, err error) {
				var reasoningNodes []dto.FlowNode
				for _, node := range res.Nodes {
					if node.Type == "reasoning" {
						reasoningNodes = append(reasoningNodes, node)
					}
				}

				if len(reasoningNodes) != 1 {
					t.Fatalf("expected 1 reasoning node, got %d", len(reasoningNodes))
				}
				if reasoningNodes[0].Data.Summary != "（暗号化済み・表示不可）×2" {
					t.Errorf("unexpected reasoning summary: %q", reasoningNodes[0].Data.Summary)
				}
			},
		},
		{
			name: "tool batch remains between unreadable reasoning nodes",
			setup: func(t *testing.T, tmpDir string) (string, usecase.SessionRepository, usecase.CacheRepository, usecase.SessionParser, func()) {
				filePath := filepath.Join(tmpDir, "rollout-tool-batch-between-unreadable-reasoning.jsonl")
				logs := []string{
					`{"type":"session_meta","timestamp":1717084800,"payload":{"id":"session-tool-batch-between-unreadable-reasoning","cli_version":"v0.131.0"}}`,
					`{"type":"event_msg","timestamp":1717084810,"payload":{"type":"task_started","turn_id":"turn-1"}}`,
					`{"type":"response_item","timestamp":1717084811,"payload":{"type":"reasoning","summary":[],"encrypted_content":"encrypted-1"}}`,
					`{"type":"response_item","timestamp":1717084812,"payload":{"type":"function_call","name":"exec_command","arguments":"{}","call_id":"call-1"}}`,
					`{"type":"response_item","timestamp":1717084813,"payload":{"type":"function_call_output","output":"done","call_id":"call-1"}}`,
					`{"type":"event_msg","timestamp":1717084814,"payload":{"type":"token_count"}}`,
					`{"type":"response_item","timestamp":1717084815,"payload":{"type":"reasoning","summary":[],"encrypted_content":"encrypted-2"}}`,
					`{"type":"event_msg","timestamp":1717084816,"payload":{"type":"task_complete","turn_id":"turn-1"}}`,
				}
				if err := os.WriteFile(filePath, []byte(strings.Join(logs, "\n")), 0o644); err != nil {
					t.Fatal(err)
				}
				sessionRepo := &mockSessionRepositoryForDetail{
					paths: map[string]string{"session-tool-batch-between-unreadable-reasoning": filePath},
				}
				cacheRepo := &mockCacheRepositoryForDetail{}
				return "session-tool-batch-between-unreadable-reasoning", sessionRepo, cacheRepo, nil, nil
			},
			wantErr: false,
			verify: func(t *testing.T, res *dto.SessionDetailResponse, err error) {
				wantEdges := map[string]string{
					"node-3": "node-4",
					"node-5": "node-7",
				}
				for source, target := range wantEdges {
					found := false
					for _, edge := range res.Edges {
						if edge.Source == source && edge.Target == target {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("expected edge %s -> %s", source, target)
					}
				}
			},
		},
		{
			name: "readable reasoning summaries remain separate nodes",
			setup: func(t *testing.T, tmpDir string) (string, usecase.SessionRepository, usecase.CacheRepository, usecase.SessionParser, func()) {
				filePath := filepath.Join(tmpDir, "rollout-readable-reasoning.jsonl")
				logs := []string{
					`{"type":"session_meta","timestamp":1717084800,"payload":{"id":"session-readable-reasoning","cli_version":"v0.131.0"}}`,
					`{"type":"event_msg","timestamp":1717084810,"payload":{"type":"task_started","turn_id":"turn-1"}}`,
					`{"type":"response_item","timestamp":1717084811,"payload":{"type":"reasoning","summary":[{"type":"summary_text","text":"Readable summary 1"}]}}`,
					`{"type":"response_item","timestamp":1717084812,"payload":{"type":"reasoning","summary":[{"type":"summary_text","text":"Readable summary 2"}]}}`,
					`{"type":"event_msg","timestamp":1717084813,"payload":{"type":"task_complete","turn_id":"turn-1"}}`,
				}
				if err := os.WriteFile(filePath, []byte(strings.Join(logs, "\n")), 0o644); err != nil {
					t.Fatal(err)
				}
				sessionRepo := &mockSessionRepositoryForDetail{
					paths: map[string]string{"session-readable-reasoning": filePath},
				}
				cacheRepo := &mockCacheRepositoryForDetail{}
				return "session-readable-reasoning", sessionRepo, cacheRepo, nil, nil
			},
			wantErr: false,
			verify: func(t *testing.T, res *dto.SessionDetailResponse, err error) {
				var summaries []string
				for _, node := range res.Nodes {
					if node.Type == "reasoning" {
						summaries = append(summaries, node.Data.Summary)
					}
				}

				want := []string{"Readable summary 1", "Readable summary 2"}
				if len(summaries) != len(want) {
					t.Fatalf("expected %d reasoning nodes, got %d: %v", len(want), len(summaries), summaries)
				}
				for i := range want {
					if summaries[i] != want[i] {
						t.Errorf("reasoning node %d summary = %q, want %q", i, summaries[i], want[i])
					}
				}
			},
		},
		{
			name: "real data",
			setup: func(t *testing.T, tmpDir string) (string, usecase.SessionRepository, usecase.CacheRepository, usecase.SessionParser, func()) {
				filePath := filepath.Join("testdata", "rollout-2026-05-27T20-40-12-019e693c-2e17-7353-949f-d1d33ccbf6ff.jsonl")
				if _, err := os.Stat(filePath); os.IsNotExist(err) {
					t.Skip("real data file not found, skipping real data test")
				}
				sessionRepo := &mockSessionRepositoryForDetail{
					paths: map[string]string{"real-session-id": filePath},
				}
				cacheRepo := &mockCacheRepositoryForDetail{}
				return "real-session-id", sessionRepo, cacheRepo, nil, nil
			},
			wantErr: false,
			verify: func(t *testing.T, res *dto.SessionDetailResponse, err error) {
				if len(res.Nodes) == 0 {
					t.Error("expected nodes from real data, got 0")
				}
				if len(res.Edges) == 0 {
					t.Error("expected edges from real data, got 0")
				}
				for _, n := range res.Nodes {
					if n.ID == "" {
						t.Error("found node with empty ID")
					}
				}
			},
		},
		{
			name: "extract child_session_ids from collab_agent_spawn_end",
			setup: func(t *testing.T, tmpDir string) (string, usecase.SessionRepository, usecase.CacheRepository, usecase.SessionParser, func()) {
				sessionID := "parent-session-1"
				filePath := filepath.Join(tmpDir, "rollout-parent.jsonl")
				sessionRepo := &mockSessionRepositoryForDetail{
					paths: map[string]string{sessionID: filePath},
				}
				cacheRepo := &mockCacheRepositoryForDetail{}
				parser := &mockSessionParser{
					records: []*model.TypedRecord{
						{
							LineNumber:  1,
							Type:        "session_meta",
							SessionMeta: &model.SessionMetaPayload{ID: sessionID, CliVersion: "v0.131.0"},
						},
						{
							LineNumber: 2,
							Type:       "event_msg",
							SubType:    "task_started",
							EventMsg: &model.EventMsgPayload{
								TurnID: "turn-1",
							},
						},
						{
							LineNumber: 3,
							Type:       "event_msg",
							SubType:    "collab_agent_spawn_end",
							EventMsg: &model.EventMsgPayload{
								NewThreadID:      "sub-session-uuid-1234",
								NewAgentNickname: "SubBot",
								NewAgentRole:     "Coder",
							},
						},
						{
							LineNumber: 4,
							Type:       "event_msg",
							SubType:    "task_complete",
							EventMsg:   &model.EventMsgPayload{TurnID: "turn-1"},
						},
					},
				}
				return sessionID, sessionRepo, cacheRepo, parser, nil
			},
			wantErr: false,
			verify: func(t *testing.T, res *dto.SessionDetailResponse, err error) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if len(res.ChildSessionIDs) != 1 || res.ChildSessionIDs[0] != "sub-session-uuid-1234" {
					t.Errorf("ChildSessionIDs = %v, want ['sub-session-uuid-1234']", res.ChildSessionIDs)
				}

				// ノードが collabAgent 型として生成されているか確認
				var collabNode *dto.FlowNode
				for i := range res.Nodes {
					if res.Nodes[i].Type == "collabAgent" {
						collabNode = &res.Nodes[i]
						break
					}
				}
				if collabNode == nil {
					t.Fatal("expected a node of type 'collabAgent'")
				}
				if collabNode.Data.Meta["new_thread_id"] != "sub-session-uuid-1234" {
					t.Errorf("expected meta new_thread_id 'sub-session-uuid-1234', got %v", collabNode.Data.Meta["new_thread_id"])
				}
				if collabNode.Data.Meta["new_agent_nickname"] != "SubBot" {
					t.Errorf("expected nickname 'SubBot', got %v", collabNode.Data.Meta["new_agent_nickname"])
				}

				// タイムライン項目が collab 型として生成されているか確認
				var collabTimelineItem *dto.ConversationTimelineItem
				for _, turn := range res.Timeline {
					for i := range turn.Items {
						if turn.Items[i].Kind == "collab" {
							collabTimelineItem = &turn.Items[i]
							break
						}
					}
				}
				if collabTimelineItem == nil {
					t.Fatal("expected a timeline item of kind 'collab'")
				}
				if collabTimelineItem.Label != "サブエージェント起動" {
					t.Errorf("expected label 'サブエージェント起動', got %s", collabTimelineItem.Label)
				}
			},
		},
		{
			name: "extract child_session_ids and generate collabNode from spawn_agent tool call",
			setup: func(t *testing.T, tmpDir string) (string, usecase.SessionRepository, usecase.CacheRepository, usecase.SessionParser, func()) {
				sessionID := "parent-session-2"
				filePath := filepath.Join(tmpDir, "rollout-parent2.jsonl")
				sessionRepo := &mockSessionRepositoryForDetail{
					paths: map[string]string{sessionID: filePath},
				}
				cacheRepo := &mockCacheRepositoryForDetail{}
				parser := &mockSessionParser{
					records: []*model.TypedRecord{
						{
							LineNumber:  1,
							Type:        "session_meta",
							SessionMeta: &model.SessionMetaPayload{ID: sessionID, CliVersion: "v0.131.0"},
						},
						{
							LineNumber: 2,
							Type:       "event_msg",
							SubType:    "task_started",
							EventMsg: &model.EventMsgPayload{
								TurnID: "turn-1",
							},
						},
						{
							LineNumber: 3,
							Type:       "response_item",
							SubType:    "function_call",
							ResponseItem: &model.ResponseItemPayload{
								CallID:    "call-1",
								Name:      "spawn_agent",
								Arguments: `{"agent_type": "Coder"}`,
							},
						},
						{
							LineNumber: 4,
							Type:       "response_item",
							SubType:    "function_call_output",
							ResponseItem: &model.ResponseItemPayload{
								CallID: "call-1",
								Name:   "spawn_agent",
								Output: `{"agent_id": "sub-session-uuid-5678", "nickname": "SubBot2"}`,
							},
						},
						{
							LineNumber: 5,
							Type:       "event_msg",
							SubType:    "task_complete",
							EventMsg:   &model.EventMsgPayload{TurnID: "turn-1"},
						},
					},
				}
				return sessionID, sessionRepo, cacheRepo, parser, nil
			},
			wantErr: false,
			verify: func(t *testing.T, res *dto.SessionDetailResponse, err error) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if len(res.ChildSessionIDs) != 1 || res.ChildSessionIDs[0] != "sub-session-uuid-5678" {
					t.Errorf("ChildSessionIDs = %v, want ['sub-session-uuid-5678']", res.ChildSessionIDs)
				}

				// ノードが collabAgent 型として生成されているか確認
				var collabNode *dto.FlowNode
				for i := range res.Nodes {
					if res.Nodes[i].Type == "collabAgent" {
						collabNode = &res.Nodes[i]
						break
					}
				}
				if collabNode == nil {
					t.Fatal("expected a node of type 'collabAgent'")
				}
				if collabNode.Data.Meta["new_thread_id"] != "sub-session-uuid-5678" {
					t.Errorf("expected meta new_thread_id 'sub-session-uuid-5678', got %v", collabNode.Data.Meta["new_thread_id"])
				}
				if collabNode.Data.Meta["new_agent_nickname"] != "SubBot2" {
					t.Errorf("expected nickname 'SubBot2', got %v", collabNode.Data.Meta["new_agent_nickname"])
				}
				if collabNode.Data.Meta["new_agent_role"] != "Coder" {
					t.Errorf("expected role 'Coder', got %v", collabNode.Data.Meta["new_agent_role"])
				}

				// タイムライン項目が collab 型として生成されているか確認
				var collabTimelineItem *dto.ConversationTimelineItem
				for _, turn := range res.Timeline {
					for i := range turn.Items {
						if turn.Items[i].Kind == "collab" {
							collabTimelineItem = &turn.Items[i]
							break
						}
					}
				}
				if collabTimelineItem == nil {
					t.Fatal("expected a timeline item of kind 'collab'")
				}
				if collabTimelineItem.Label != "サブエージェント起動" {
					t.Errorf("expected label 'サブエージェント起動', got %s", collabTimelineItem.Label)
				}
				if !strings.Contains(collabTimelineItem.Body, "sub-session-uuid-5678") {
					t.Errorf("expected body to contain thread ID, got %s", collabTimelineItem.Body)
				}
			},
		},
		{
			name: "link parent session ID when current session is child",
			setup: func(t *testing.T, tmpDir string) (string, usecase.SessionRepository, usecase.CacheRepository, usecase.SessionParser, func()) {
				sessionID := "child-session-1"
				filePath := filepath.Join(tmpDir, "rollout-child.jsonl")
				sessionRepo := &mockSessionRepositoryForDetail{
					paths: map[string]string{sessionID: filePath},
					sessions: []dto.SessionSummary{
						{
							ID:              "parent-session-123",
							ChildSessionIDs: []string{sessionID},
						},
					},
				}
				cacheRepo := &mockCacheRepositoryForDetail{}
				parser := &mockSessionParser{
					records: []*model.TypedRecord{
						{
							LineNumber:  1,
							Type:        "session_meta",
							SessionMeta: &model.SessionMetaPayload{ID: sessionID, CliVersion: "v0.131.0"},
						},
						{
							LineNumber: 2,
							Type:       "event_msg",
							SubType:    "task_started",
							EventMsg: &model.EventMsgPayload{
								TurnID: "turn-1",
							},
						},
						{
							LineNumber: 3,
							Type:       "event_msg",
							SubType:    "task_complete",
							EventMsg:   &model.EventMsgPayload{TurnID: "turn-1"},
						},
					},
				}
				return sessionID, sessionRepo, cacheRepo, parser, nil
			},
			wantErr: false,
			verify: func(t *testing.T, res *dto.SessionDetailResponse, err error) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if res.ParentSessionID == nil || *res.ParentSessionID != "parent-session-123" {
					t.Errorf("ParentSessionID = %v, want 'parent-session-123'", res.ParentSessionID)
				}
			},
		},
		{
			name: "cache hit success",
			setup: func(t *testing.T, tmpDir string) (string, usecase.SessionRepository, usecase.CacheRepository, usecase.SessionParser, func()) {
				sessionID := "cached-session"
				cachedResponse := &dto.SessionDetailResponse{
					ID:                 sessionID,
					CacheSchemaVersion: dto.CurrentSessionDetailCacheSchemaVersion,
					ParsedAt:           "2026-05-31T01:00:00Z",
				}
				cacheRepo := &mockCacheRepositoryForDetail{
					cache: map[string]*dto.SessionDetailResponse{
						sessionID: cachedResponse,
					},
					modTimes: map[string]time.Time{
						sessionID: time.Date(2026, 5, 31, 1, 0, 1, 0, time.UTC),
					},
				}
				sessionRepo := &mockSessionRepositoryForDetail{
					modTimes: map[string]time.Time{
						sessionID: time.Date(2026, 5, 30, 0, 0, 0, 0, time.UTC),
					},
				}
				return sessionID, sessionRepo, cacheRepo, nil, nil
			},
			wantErr: false,
			verify: func(t *testing.T, res *dto.SessionDetailResponse, err error) {
				if res == nil || res.ID != "cached-session" {
					t.Errorf("expected cached response, got %+v", res)
				}
			},
		},
		{
			name: "fresh cache file is used even when parsed_at is old",
			setup: func(t *testing.T, tmpDir string) (string, usecase.SessionRepository, usecase.CacheRepository, usecase.SessionParser, func()) {
				sessionID := "cached-by-file-modtime"
				cachedResponse := &dto.SessionDetailResponse{
					ID:                 sessionID,
					CacheSchemaVersion: dto.CurrentSessionDetailCacheSchemaVersion,
					ParsedAt:           "2026-05-01T00:00:00Z",
				}
				cacheRepo := &mockCacheRepositoryForDetail{
					cache: map[string]*dto.SessionDetailResponse{
						sessionID: cachedResponse,
					},
					modTimes: map[string]time.Time{
						sessionID: time.Date(2026, 5, 31, 3, 0, 0, 0, time.UTC),
					},
				}
				sessionRepo := &mockSessionRepositoryForDetail{
					modTimes: map[string]time.Time{
						sessionID: time.Date(2026, 5, 31, 2, 0, 0, 0, time.UTC),
					},
				}
				parser := &mockSessionParser{
					records: []*model.TypedRecord{
						{
							Type: "session_meta",
							SessionMeta: &model.SessionMetaPayload{
								ID:         sessionID,
								CliVersion: "0.131.0",
							},
						},
					},
				}
				return sessionID, sessionRepo, cacheRepo, parser, func() {
					if parser.calls != 0 {
						t.Fatalf("expected parser not to be called, got %d", parser.calls)
					}
					if cacheRepo.saveCount != 0 {
						t.Fatalf("expected cache save not to run, got %d", cacheRepo.saveCount)
					}
				}
			},
			wantErr: false,
			verify: func(t *testing.T, res *dto.SessionDetailResponse, err error) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if res == nil || res.ID != "cached-by-file-modtime" {
					t.Fatalf("expected cached response, got %+v", res)
				}
			},
		},
		{
			name: "schema version 5 cache triggers reparse even when cache file is fresh",
			setup: func(t *testing.T, tmpDir string) (string, usecase.SessionRepository, usecase.CacheRepository, usecase.SessionParser, func()) {
				sessionID := "legacy-cache-session"
				cacheRepo := &mockCacheRepositoryForDetail{
					cache: map[string]*dto.SessionDetailResponse{
						sessionID: {
							ID:                 sessionID,
							CacheSchemaVersion: 5,
							ParsedAt:           "2026-05-31T01:00:00Z",
						},
					},
					modTimes: map[string]time.Time{
						sessionID: time.Date(2026, 5, 31, 3, 0, 0, 0, time.UTC),
					},
				}
				sessionRepo := &mockSessionRepositoryForDetail{
					paths: map[string]string{
						sessionID: filepath.Join(tmpDir, "rollout-legacy-cache.jsonl"),
					},
					modTimes: map[string]time.Time{
						sessionID: time.Date(2026, 5, 31, 2, 0, 0, 0, time.UTC),
					},
				}
				parser := &mockSessionParser{
					records: []*model.TypedRecord{
						{
							Type: "session_meta",
							SessionMeta: &model.SessionMetaPayload{
								ID:         sessionID,
								CliVersion: "0.131.0",
							},
						},
					},
				}
				return sessionID, sessionRepo, cacheRepo, parser, func() {
					if parser.calls != 1 {
						t.Fatalf("expected parser to be called once, got %d", parser.calls)
					}
					if cacheRepo.saveCount != 1 {
						t.Fatalf("expected cache save once, got %d", cacheRepo.saveCount)
					}
				}
			},
			wantErr: false,
			verify: func(t *testing.T, res *dto.SessionDetailResponse, err error) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if res == nil || res.ID != "legacy-cache-session" {
					t.Fatalf("expected reparsed response, got %+v", res)
				}
				if res.CacheSchemaVersion != dto.CurrentSessionDetailCacheSchemaVersion {
					t.Fatalf(
						"CacheSchemaVersion = %d, want %d",
						res.CacheSchemaVersion,
						dto.CurrentSessionDetailCacheSchemaVersion,
					)
				}
			},
		},
		{
			name: "stale cache triggers reparse and save",
			setup: func(t *testing.T, tmpDir string) (string, usecase.SessionRepository, usecase.CacheRepository, usecase.SessionParser, func()) {
				sessionID := "stale-session"
				cacheRepo := &mockCacheRepositoryForDetail{
					cache: map[string]*dto.SessionDetailResponse{
						sessionID: {
							ID:                 sessionID,
							CacheSchemaVersion: dto.CurrentSessionDetailCacheSchemaVersion,
							ParsedAt:           "2026-05-31T01:00:00Z",
						},
					},
					modTimes: map[string]time.Time{
						sessionID: time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC),
					},
				}
				sessionRepo := &mockSessionRepositoryForDetail{
					paths: map[string]string{
						sessionID: filepath.Join(tmpDir, "rollout-stale.jsonl"),
					},
					modTimes: map[string]time.Time{
						sessionID: time.Date(2026, 5, 31, 2, 0, 0, 0, time.UTC),
					},
				}
				parser := &mockSessionParser{
					records: []*model.TypedRecord{
						{
							Type: "session_meta",
							SessionMeta: &model.SessionMetaPayload{
								ID:         sessionID,
								CliVersion: "0.131.0",
							},
						},
					},
				}
				return sessionID, sessionRepo, cacheRepo, parser, func() {
					if parser.calls != 1 {
						t.Fatalf("expected parser to be called once, got %d", parser.calls)
					}
					if cacheRepo.saveCount != 1 {
						t.Fatalf("expected cache save once, got %d", cacheRepo.saveCount)
					}
					if cacheRepo.lastSaveID != sessionID {
						t.Fatalf("expected saved session id %s, got %s", sessionID, cacheRepo.lastSaveID)
					}
				}
			},
			wantErr: false,
			verify: func(t *testing.T, res *dto.SessionDetailResponse, err error) {
				if res == nil {
					t.Fatal("expected response, got nil")
				}
				if res.ID != "stale-session" {
					t.Fatalf("expected reparsed response, got %+v", res)
				}
			},
		},
		{
			name: "save cache failure does not fail response",
			setup: func(t *testing.T, tmpDir string) (string, usecase.SessionRepository, usecase.CacheRepository, usecase.SessionParser, func()) {
				sessionID := "save-error-session"
				cacheRepo := &mockCacheRepositoryForDetail{
					saveErr: errors.New("disk full"),
				}
				sessionRepo := &mockSessionRepositoryForDetail{
					paths: map[string]string{
						sessionID: filepath.Join(tmpDir, "rollout-save-error.jsonl"),
					},
					modTimes: map[string]time.Time{
						sessionID: time.Date(2026, 5, 31, 2, 0, 0, 0, time.UTC),
					},
				}
				parser := &mockSessionParser{
					records: []*model.TypedRecord{
						{
							Type: "session_meta",
							SessionMeta: &model.SessionMetaPayload{
								ID:         sessionID,
								CliVersion: "0.131.0",
							},
						},
					},
				}
				return sessionID, sessionRepo, cacheRepo, parser, func() {
					if parser.calls != 1 {
						t.Fatalf("expected parser to be called once, got %d", parser.calls)
					}
				}
			},
			wantErr: false,
			verify: func(t *testing.T, res *dto.SessionDetailResponse, err error) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if res == nil || res.ID != "save-error-session" {
					t.Fatalf("expected parsed response, got %+v", res)
				}
			},
		},
		{
			name: "GetSessionFilePath error",
			setup: func(t *testing.T, tmpDir string) (string, usecase.SessionRepository, usecase.CacheRepository, usecase.SessionParser, func()) {
				sessionRepo := &mockSessionRepositoryForDetail{
					filePathErr: errors.New("file path error"),
				}
				cacheRepo := &mockCacheRepositoryForDetail{}
				return "session-id", sessionRepo, cacheRepo, nil, nil
			},
			wantErr: true,
			verify: func(t *testing.T, res *dto.SessionDetailResponse, err error) {
				if !errors.Is(err, model.ErrSessionNotFound) {
					t.Errorf("expected ErrSessionNotFound, got %v", err)
				}
			},
		},
		{
			name: "parser ErrFileTooLarge",
			setup: func(t *testing.T, tmpDir string) (string, usecase.SessionRepository, usecase.CacheRepository, usecase.SessionParser, func()) {
				sessionRepo := &mockSessionRepositoryForDetail{
					paths: map[string]string{"session-id": "dummy-path"},
				}
				parser := &mockSessionParser{
					err: model.ErrFileTooLarge,
				}
				cacheRepo := &mockCacheRepositoryForDetail{}
				return "session-id", sessionRepo, cacheRepo, parser, nil
			},
			wantErr: true,
			verify: func(t *testing.T, res *dto.SessionDetailResponse, err error) {
				if !errors.Is(err, model.ErrFileTooLarge) {
					t.Errorf("expected ErrFileTooLarge, got %v", err)
				}
			},
		},
		{
			name: "parser ErrParseFailed",
			setup: func(t *testing.T, tmpDir string) (string, usecase.SessionRepository, usecase.CacheRepository, usecase.SessionParser, func()) {
				sessionRepo := &mockSessionRepositoryForDetail{
					paths: map[string]string{"session-id": "dummy-path"},
				}
				parser := &mockSessionParser{
					err: model.ErrParseFailed,
				}
				cacheRepo := &mockCacheRepositoryForDetail{}
				return "session-id", sessionRepo, cacheRepo, parser, nil
			},
			wantErr: true,
			verify: func(t *testing.T, res *dto.SessionDetailResponse, err error) {
				if !errors.Is(err, model.ErrParseFailed) {
					t.Errorf("expected ErrParseFailed, got %v", err)
				}
			},
		},
		{
			name: "parser fs.ErrPermission",
			setup: func(t *testing.T, tmpDir string) (string, usecase.SessionRepository, usecase.CacheRepository, usecase.SessionParser, func()) {
				sessionRepo := &mockSessionRepositoryForDetail{
					paths: map[string]string{"session-id": "dummy-path"},
				}
				parser := &mockSessionParser{
					err: os.ErrPermission,
				}
				cacheRepo := &mockCacheRepositoryForDetail{}
				return "session-id", sessionRepo, cacheRepo, parser, nil
			},
			wantErr: true,
			verify: func(t *testing.T, res *dto.SessionDetailResponse, err error) {
				if !errors.Is(err, model.ErrFileReadError) {
					t.Errorf("expected ErrFileReadError, got %v", err)
				}
			},
		},
		{
			name: "parser generic error",
			setup: func(t *testing.T, tmpDir string) (string, usecase.SessionRepository, usecase.CacheRepository, usecase.SessionParser, func()) {
				sessionRepo := &mockSessionRepositoryForDetail{
					paths: map[string]string{"session-id": "dummy-path"},
				}
				parser := &mockSessionParser{
					err: errors.New("generic parser error"),
				}
				cacheRepo := &mockCacheRepositoryForDetail{}
				return "session-id", sessionRepo, cacheRepo, parser, nil
			},
			wantErr: true,
			verify: func(t *testing.T, res *dto.SessionDetailResponse, err error) {
				if !errors.Is(err, model.ErrParseFailed) {
					t.Errorf("expected ErrParseFailed, got %v", err)
				}
			},
		},
		{
			name: "cache save error ignored",
			setup: func(t *testing.T, tmpDir string) (string, usecase.SessionRepository, usecase.CacheRepository, usecase.SessionParser, func()) {
				sessionRepo := &mockSessionRepositoryForDetail{
					paths: map[string]string{"session-id": "dummy-path"},
				}
				parser := &mockSessionParser{
					records: []*model.TypedRecord{
						{
							LineNumber: 1,
							Type:       "session_meta",
							SessionMeta: &model.SessionMetaPayload{
								ID:         "session-id",
								CliVersion: "v0.125.0",
							},
						},
					},
				}
				cacheRepo := &mockCacheRepositoryForDetail{
					saveErr: errors.New("failed to save cache"),
				}
				return "session-id", sessionRepo, cacheRepo, parser, nil
			},
			wantErr: false,
			verify: func(t *testing.T, res *dto.SessionDetailResponse, err error) {
				if res == nil || res.ID != "session-id" {
					t.Errorf("expected successful response, got %+v", res)
				}
			},
		},
		{
			name: "isUnsupportedVersion invalid version Sscanf fail",
			setup: func(t *testing.T, tmpDir string) (string, usecase.SessionRepository, usecase.CacheRepository, usecase.SessionParser, func()) {
				sessionRepo := &mockSessionRepositoryForDetail{
					paths: map[string]string{"session-id": "dummy-path"},
				}
				parser := &mockSessionParser{
					records: []*model.TypedRecord{
						{
							LineNumber: 1,
							Type:       "session_meta",
							SessionMeta: &model.SessionMetaPayload{
								ID:         "session-id",
								CliVersion: "invalid",
							},
						},
					},
				}
				cacheRepo := &mockCacheRepositoryForDetail{}
				return "session-id", sessionRepo, cacheRepo, parser, nil
			},
			wantErr: true,
			verify: func(t *testing.T, res *dto.SessionDetailResponse, err error) {
				if !errors.Is(err, model.ErrUnsupportedFormat) {
					t.Errorf("expected ErrUnsupportedFormat, got %v", err)
				}
			},
		},
		{
			name: "isUnsupportedVersion empty version",
			setup: func(t *testing.T, tmpDir string) (string, usecase.SessionRepository, usecase.CacheRepository, usecase.SessionParser, func()) {
				sessionRepo := &mockSessionRepositoryForDetail{
					paths: map[string]string{"session-id": "dummy-path"},
				}
				parser := &mockSessionParser{
					records: []*model.TypedRecord{
						{
							LineNumber: 1,
							Type:       "session_meta",
							SessionMeta: &model.SessionMetaPayload{
								ID:         "session-id",
								CliVersion: "",
							},
						},
					},
				}
				cacheRepo := &mockCacheRepositoryForDetail{}
				return "session-id", sessionRepo, cacheRepo, parser, nil
			},
			wantErr: false,
			verify: func(t *testing.T, res *dto.SessionDetailResponse, err error) {
				if res == nil || res.ID != "session-id" {
					t.Errorf("expected successful response, got %+v", res)
				}
			},
		},
		{
			name: "isUnsupportedVersion major version > 0",
			setup: func(t *testing.T, tmpDir string) (string, usecase.SessionRepository, usecase.CacheRepository, usecase.SessionParser, func()) {
				sessionRepo := &mockSessionRepositoryForDetail{
					paths: map[string]string{"session-id": "dummy-path"},
				}
				parser := &mockSessionParser{
					records: []*model.TypedRecord{
						{
							LineNumber: 1,
							Type:       "session_meta",
							SessionMeta: &model.SessionMetaPayload{
								ID:         "session-id",
								CliVersion: "v1.0.0",
							},
						},
					},
				}
				cacheRepo := &mockCacheRepositoryForDetail{}
				return "session-id", sessionRepo, cacheRepo, parser, nil
			},
			wantErr: false,
			verify: func(t *testing.T, res *dto.SessionDetailResponse, err error) {
				if res == nil || res.ID != "session-id" {
					t.Errorf("expected successful response, got %+v", res)
				}
			},
		},
		{
			name: "comprehensive edge cases and branch coverage",
			setup: func(t *testing.T, tmpDir string) (string, usecase.SessionRepository, usecase.CacheRepository, usecase.SessionParser, func()) {
				sessionRepo := &mockSessionRepositoryForDetail{
					paths: map[string]string{"session-id": "dummy-path"},
				}

				records := []*model.TypedRecord{
					// session_meta with git info and base instructions
					{
						LineNumber: 1,
						Type:       "session_meta",
						SessionMeta: &model.SessionMetaPayload{
							ID:         "session-id",
							CliVersion: "v0.125.0",
							Git: &model.GitInfo{
								Branch: "feat-branch",
							},
							BaseInstructions: &model.Instructions{
								Text: "Base System Instructions",
							},
						},
					},
					// thread_name_updated (skipped)
					{
						LineNumber: 2,
						Type:       "event_msg",
						SubType:    "thread_name_updated",
					},
					// unmatched out of turn event msg (appends to outOfTurnBuffer)
					{
						LineNumber: 3,
						Type:       "event_msg",
						SubType:    "some_early_event",
						EventMsg: &model.EventMsgPayload{
							TurnID: "turn-1",
						},
					},
					// task_started triggers turn parsing
					{
						LineNumber: 4,
						Type:       "event_msg",
						SubType:    "task_started",
						EventMsg: &model.EventMsgPayload{
							TurnID:             "turn-1",
							ModelContextWindow: 64000,
							Message:            "Started turn 1",
						},
					},
					// Non-batch agent messages for merge logic (Case A: event_msg followed by response_item)
					{
						LineNumber: 37,
						Type:       "event_msg",
						SubType:    "agent_message",
						EventMsg: &model.EventMsgPayload{
							Message: "Non-batch message 1",
						},
					},
					{
						LineNumber: 38,
						Type:       "response_item",
						SubType:    "message",
						ResponseItem: &model.ResponseItemPayload{
							Role: "assistant",
							Content: []model.MessageContent{
								{Type: "text", Text: "Non-batch message 2"},
							},
						},
					},
					// Non-batch agent messages for merge logic (Case B: response_item followed by event_msg)
					{
						LineNumber: 47,
						Type:       "response_item",
						SubType:    "message",
						ResponseItem: &model.ResponseItemPayload{
							Role: "assistant",
							Content: []model.MessageContent{
								{Type: "text", Text: "Non-batch message 3"},
							},
						},
					},
					{
						LineNumber: 48,
						Type:       "event_msg",
						SubType:    "agent_message",
						EventMsg: &model.EventMsgPayload{
							Message: "Non-batch message 4",
						},
					},
					// duplicate turn_context inside the turn
					{
						LineNumber: 5,
						Type:       "turn_context",
						TurnContext: &model.TurnContextPayload{
							TurnID: "turn-1",
							Model:  "gpt-4",
							CollaborationMode: &model.CollaborationMode{
								Mode:                  "collaboration",
								DeveloperInstructions: "Do something dev",
							},
							UserInstructions: "Do something user",
						},
					},
					{
						LineNumber: 6,
						Type:       "turn_context",
						TurnContext: &model.TurnContextPayload{
							TurnID: "turn-1",
							Model:  "gpt-4-overwrite",
						},
					},
					// developer messages, one skipped because of nil/empty content, one valid
					{
						LineNumber: 7,
						Type:       "response_item",
						SubType:    "message",
						ResponseItem: &model.ResponseItemPayload{
							Role:    "developer",
							Content: nil,
						},
					},
					{
						LineNumber: 8,
						Type:       "response_item",
						SubType:    "message",
						ResponseItem: &model.ResponseItemPayload{
							Role: "developer",
							Content: []model.MessageContent{
								{Type: "text", Text: "Dev message content"},
							},
						},
					},
					// user message, one skipped because of nil/empty content, one valid
					{
						LineNumber: 9,
						Type:       "response_item",
						SubType:    "message",
						ResponseItem: &model.ResponseItemPayload{
							Role:    "user",
							Content: nil,
						},
					},
					{
						LineNumber: 10,
						Type:       "response_item",
						SubType:    "message",
						ResponseItem: &model.ResponseItemPayload{
							Role: "user",
							Content: []model.MessageContent{
								{Type: "text", Text: "User api message content"},
							},
						},
					},
					// web search call using Query
					{
						LineNumber: 11,
						Type:       "response_item",
						SubType:    "web_search_call",
						ResponseItem: &model.ResponseItemPayload{
							Action: &model.SearchAction{
								Query: "search-1",
							},
						},
					},
					// web search call using Queries[0]
					{
						LineNumber: 12,
						Type:       "response_item",
						SubType:    "web_search_call",
						ResponseItem: &model.ResponseItemPayload{
							Action: &model.SearchAction{
								Queries: []string{"search-2"},
							},
						},
					},
					// web search end
					{
						LineNumber: 13,
						Type:       "event_msg",
						SubType:    "web_search_end",
					},
					// item_completed: one valid, one nil EventMsg, one nil Item (skipped)
					{
						LineNumber: 14,
						Type:       "event_msg",
						SubType:    "item_completed",
						EventMsg: &model.EventMsgPayload{
							Item: &model.CompletedItem{
								Text: "Item text",
							},
						},
					},
					{
						LineNumber: 15,
						Type:       "event_msg",
						SubType:    "item_completed",
						EventMsg:   nil,
					},
					{
						LineNumber: 16,
						Type:       "event_msg",
						SubType:    "item_completed",
						EventMsg: &model.EventMsgPayload{
							Item: nil,
						},
					},
					// generic record of event_msg type
					{
						LineNumber: 17,
						Type:       "event_msg",
						SubType:    "generic_subtype_event",
					},
					// external event record of event_msg type
					{
						LineNumber: 18,
						Type:       "event_msg",
						SubType:    "generic_subtype_ext",
						EventMsg: &model.EventMsgPayload{
							CallID: "call-ext-99",
						},
					},
					// generic record of response_item type
					{
						LineNumber: 19,
						Type:       "response_item",
						SubType:    "generic_subtype_resp",
					},
					// external event record of response_item type
					{
						LineNumber: 20,
						Type:       "response_item",
						SubType:    "generic_subtype_resp_ext",
						ResponseItem: &model.ResponseItemPayload{
							CallID: "call-ext-98",
						},
					},
					// reasoning pairing: more reasoning than agent_reasoning
					{
						LineNumber: 21,
						Type:       "event_msg",
						SubType:    "agent_reasoning",
						EventMsg: &model.EventMsgPayload{
							Text: "Agent thoughts",
						},
					},
					{
						LineNumber: 22,
						Type:       "response_item",
						SubType:    "reasoning",
						ResponseItem: &model.ResponseItemPayload{
							Summary: []model.MessageContent{
								{Type: "text", Text: "Reasoning summary 1"},
							},
						},
					},
					{
						LineNumber: 23,
						Type:       "response_item",
						SubType:    "reasoning",
						ResponseItem: &model.ResponseItemPayload{
							Summary: []model.MessageContent{
								{Type: "text", Text: "Reasoning summary 2"},
							},
						},
					},
					// Batch call with middleMessage only (no output)
					{
						LineNumber: 24,
						Type:       "response_item",
						SubType:    "function_call",
						ResponseItem: &model.ResponseItemPayload{
							CallID:    "call-middle-only",
							Name:      "middle_only_tool",
							Arguments: "{}",
						},
					},
					{
						LineNumber: 25,
						Type:       "response_item",
						SubType:    "message",
						ResponseItem: &model.ResponseItemPayload{
							Role: "assistant",
							Content: []model.MessageContent{
								{Type: "text", Text: "Middle assistant message"},
							},
						},
					},
					// Batch call with no middleMessage
					{
						LineNumber: 26,
						Type:       "response_item",
						SubType:    "function_call",
						ResponseItem: &model.ResponseItemPayload{
							CallID:    "call-no-middle",
							Name:      "no_middle_tool",
							Arguments: "{}",
						},
					},
					{
						LineNumber: 27,
						Type:       "response_item",
						SubType:    "function_call_output",
						ResponseItem: &model.ResponseItemPayload{
							CallID: "call-no-middle",
							Output: "output-no-middle",
						},
					},
					// Custom tool call batch
					{
						LineNumber: 28,
						Type:       "response_item",
						SubType:    "custom_tool_call",
						ResponseItem: &model.ResponseItemPayload{
							CallID: "custom-1",
							Name:   "custom_tool",
						},
					},
					{
						LineNumber: 29,
						Type:       "response_item",
						SubType:    "custom_tool_call_output",
						ResponseItem: &model.ResponseItemPayload{
							CallID: "custom-1",
							Output: "custom-output",
						},
					},
					// Consecutive token counts (bound to the preceding record, skipping token_count record itself)
					{
						LineNumber: 30,
						Type:       "event_msg",
						SubType:    "token_count",
						EventMsg: &model.EventMsgPayload{
							Info: &model.TokenInfo{
								TotalTokenUsage: &model.TokenDetail{TotalTokens: 5000},
								LastTokenUsage:  &model.TokenDetail{TotalTokens: 200, InputTokens: 150, OutputTokens: 50},
							},
						},
					},
					{
						LineNumber: 31,
						Type:       "event_msg",
						SubType:    "token_count",
						EventMsg: &model.EventMsgPayload{
							Info: &model.TokenInfo{
								TotalTokenUsage: &model.TokenDetail{TotalTokens: 5200},
							},
						},
					},
					{
						LineNumber: 32,
						Type:       "event_msg",
						SubType:    "token_count",
						EventMsg: &model.EventMsgPayload{
							Info: &model.TokenInfo{
								TotalTokenUsage: &model.TokenDetail{TotalTokens: 5500},
								LastTokenUsage:  &model.TokenDetail{TotalTokens: 300, InputTokens: 200, OutputTokens: 100},
							},
						},
					},
					// function call with nil ResponseItem
					{
						LineNumber:   33,
						Type:         "response_item",
						SubType:      "function_call",
						ResponseItem: nil,
					},
					// task complete terminates turn-1
					{
						LineNumber: 34,
						Type:       "event_msg",
						SubType:    "task_complete",
						EventMsg: &model.EventMsgPayload{
							TurnID: "turn-1",
							Reason: "Done with turn 1",
						},
					},
					// orphan complete/abort messages (processed where turn index is -1)
					{
						LineNumber: 35,
						Type:       "event_msg",
						SubType:    "task_complete",
						EventMsg: &model.EventMsgPayload{
							Reason: "Orphan complete reason",
						},
					},
					{
						LineNumber: 36,
						Type:       "event_msg",
						SubType:    "turn_aborted",
						EventMsg: &model.EventMsgPayload{
							Reason: "Orphan abort reason",
						},
					},
					// leftover record (triggers pseudo-turn index -1)
					{
						LineNumber: 37,
						Type:       "event_msg",
						SubType:    "generic_leftover",
					},
				}

				parser := &mockSessionParser{
					records: records,
				}
				cacheRepo := &mockCacheRepositoryForDetail{}
				return "session-id", sessionRepo, cacheRepo, parser, nil
			},
			wantErr: false,
			verify: func(t *testing.T, res *dto.SessionDetailResponse, err error) {
				if res == nil || res.ID != "session-id" {
					t.Fatalf("expected successful response, got %+v", res)
				}

				// Verify nodes were created
				var orphanCompleteNode, orphanAbortNode, leftoverNode, customToolNode, customToolOutputNode, middleOnlyNode, mergedAgentNode *dto.FlowNode
				for i := range res.Nodes {
					n := &res.Nodes[i]
					switch n.Type {
					case "taskEvent":
						switch n.Data.Label {
						case "Orphan Complete":
							orphanCompleteNode = n
						case "Orphan Aborted":
							orphanAbortNode = n
						}
					case "generic":
						if strings.Contains(n.Data.Summary, "generic_leftover") {
							leftoverNode = n
						}
					case "action":
						switch n.Data.Label {
						case "custom_tool":
							customToolNode = n
						case "Output: custom_tool":
							customToolOutputNode = n
						}
					case "agentMessage":
						if strings.Contains(n.Data.Summary, "Middle assistant message") {
							middleOnlyNode = n
						} else if strings.Contains(n.Data.Summary, "Non-batch message 1") {
							mergedAgentNode = n
						}
					}
				}

				if orphanCompleteNode == nil {
					t.Error("expected orphan complete node, got nil")
				}
				if orphanAbortNode == nil {
					t.Error("expected orphan abort node, got nil")
				}
				if leftoverNode == nil {
					t.Error("expected leftover node, got nil")
				}
				if customToolNode == nil {
					t.Error("expected custom tool node, got nil")
				}
				if customToolOutputNode == nil {
					t.Fatal("expected custom tool output node, got nil")
				}
				if customToolOutputNode.Data.TokenBadge == nil {
					t.Fatal("expected custom tool output node to receive token badge")
				}
				if customToolOutputNode.Data.TokenBadge.ConsumedTokens != 500 {
					t.Fatalf("expected missing last token usage to be excluded from the sum, got %+v", customToolOutputNode.Data.TokenBadge)
				}
				if customToolOutputNode.Data.TokenBadge.BoundCount != 3 {
					t.Fatalf("expected bound count to include all 3 records, got %+v", customToolOutputNode.Data.TokenBadge)
				}
				if middleOnlyNode == nil {
					t.Error("expected middle only node, got nil")
				}
				if mergedAgentNode == nil {
					t.Error("expected merged non-batch agent message node, got nil")
				}
			},
		},
		{
			name: "binds token count when output is separated by other events",
			setup: func(t *testing.T, tmpDir string) (string, usecase.SessionRepository, usecase.CacheRepository, usecase.SessionParser, func()) {
				sessionRepo := &mockSessionRepositoryForDetail{
					paths: map[string]string{"session-separated-output": "dummy-path"},
				}

				records := []*model.TypedRecord{
					{
						LineNumber: 1,
						Type:       "session_meta",
						SessionMeta: &model.SessionMetaPayload{
							ID:         "session-separated-output",
							CliVersion: "v0.135.0",
						},
					},
					{
						LineNumber: 2,
						Type:       "event_msg",
						SubType:    "task_started",
						EventMsg: &model.EventMsgPayload{
							TurnID:             "turn-1",
							ModelContextWindow: 258400,
						},
					},
					{
						LineNumber: 3,
						Type:       "response_item",
						SubType:    "function_call",
						ResponseItem: &model.ResponseItemPayload{
							CallID:    "call-1",
							Name:      "exec_command",
							Arguments: "{\"cmd\":\"echo hi\"}",
						},
					},
					{
						LineNumber: 4,
						Type:       "event_msg",
						SubType:    "thread_goal_updated",
						EventMsg:   &model.EventMsgPayload{},
					},
					{
						LineNumber: 5,
						Type:       "response_item",
						SubType:    "function_call_output",
						ResponseItem: &model.ResponseItemPayload{
							CallID: "call-1",
							Output: "done",
						},
					},
					{
						LineNumber: 6,
						Type:       "event_msg",
						SubType:    "token_count",
						EventMsg: &model.EventMsgPayload{
							Info: &model.TokenInfo{
								TotalTokenUsage: &model.TokenDetail{TotalTokens: 1200},
								LastTokenUsage:  &model.TokenDetail{TotalTokens: 200},
							},
						},
					},
					{
						LineNumber: 7,
						Type:       "response_item",
						SubType:    "custom_tool_call",
						ResponseItem: &model.ResponseItemPayload{
							CallID: "custom-1",
							Name:   "apply_patch",
							Input:  "*** Begin Patch",
						},
					},
					{
						LineNumber: 8,
						Type:       "event_msg",
						SubType:    "thread_goal_updated",
						EventMsg:   &model.EventMsgPayload{},
					},
					{
						LineNumber: 9,
						Type:       "response_item",
						SubType:    "custom_tool_call_output",
						ResponseItem: &model.ResponseItemPayload{
							CallID: "custom-1",
							Output: "patched",
						},
					},
					{
						LineNumber: 10,
						Type:       "event_msg",
						SubType:    "token_count",
						EventMsg: &model.EventMsgPayload{
							Info: &model.TokenInfo{
								TotalTokenUsage: &model.TokenDetail{TotalTokens: 1400},
								LastTokenUsage:  &model.TokenDetail{TotalTokens: 200},
							},
						},
					},
					{
						LineNumber: 11,
						Type:       "event_msg",
						SubType:    "task_complete",
						EventMsg: &model.EventMsgPayload{
							TurnID: "turn-1",
						},
					},
				}

				parser := &mockSessionParser{records: records}
				cacheRepo := &mockCacheRepositoryForDetail{}
				return "session-separated-output", sessionRepo, cacheRepo, parser, nil
			},
			wantErr: false,
			verify: func(t *testing.T, res *dto.SessionDetailResponse, err error) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if len(res.TokenCounts) != 2 {
					t.Fatalf("expected 2 token count entries, got %d", len(res.TokenCounts))
				}

				if res.TokenCounts[0].BoundToNodeID != "node-5" {
					t.Fatalf("expected first token count to bind to node-5, got %q", res.TokenCounts[0].BoundToNodeID)
				}
				if res.TokenCounts[1].BoundToNodeID != "node-9" {
					t.Fatalf("expected second token count to bind to node-9, got %q", res.TokenCounts[1].BoundToNodeID)
				}

				var outputNode, customOutputNode *dto.FlowNode
				for i := range res.Nodes {
					node := &res.Nodes[i]
					switch node.ID {
					case "node-5":
						outputNode = node
					case "node-9":
						customOutputNode = node
					}
				}

				if outputNode == nil {
					t.Fatal("expected function_call_output node to exist")
				}
				if customOutputNode == nil {
					t.Fatal("expected custom_tool_call_output node to exist")
				}
				if outputNode.Data.TokenBadge == nil || outputNode.Data.TokenBadge.ConsumedTokens != 200 {
					t.Fatalf("expected output node badge to be set, got %+v", outputNode.Data.TokenBadge)
				}
				if customOutputNode.Data.TokenBadge == nil || customOutputNode.Data.TokenBadge.ConsumedTokens != 200 {
					t.Fatalf("expected custom output node badge to be set, got %+v", customOutputNode.Data.TokenBadge)
				}
			},
		},
		{
			name: "falls back to previous mapped node when bound record has no node",
			setup: func(t *testing.T, tmpDir string) (string, usecase.SessionRepository, usecase.CacheRepository, usecase.SessionParser, func()) {
				sessionRepo := &mockSessionRepositoryForDetail{
					paths: map[string]string{"session-token-fallback": "dummy-path"},
				}

				records := []*model.TypedRecord{
					{
						LineNumber: 1,
						Type:       "session_meta",
						SessionMeta: &model.SessionMetaPayload{
							ID:         "session-token-fallback",
							CliVersion: "v0.135.0",
						},
					},
					{
						LineNumber: 2,
						Type:       "event_msg",
						SubType:    "task_started",
						EventMsg: &model.EventMsgPayload{
							TurnID: "turn-1",
						},
					},
					{
						LineNumber: 3,
						Type:       "event_msg",
						SubType:    "agent_message",
						EventMsg: &model.EventMsgPayload{
							Message: "Mapped agent message",
						},
					},
					{
						LineNumber:   4,
						Type:         "response_item",
						SubType:      "function_call",
						ResponseItem: nil,
					},
					{
						LineNumber: 5,
						Type:       "event_msg",
						SubType:    "token_count",
						EventMsg: &model.EventMsgPayload{
							Info: &model.TokenInfo{
								TotalTokenUsage: &model.TokenDetail{TotalTokens: 900},
								LastTokenUsage:  &model.TokenDetail{TotalTokens: 100},
							},
						},
					},
					{
						LineNumber: 6,
						Type:       "event_msg",
						SubType:    "task_complete",
						EventMsg: &model.EventMsgPayload{
							TurnID: "turn-1",
						},
					},
				}

				parser := &mockSessionParser{records: records}
				cacheRepo := &mockCacheRepositoryForDetail{}
				return "session-token-fallback", sessionRepo, cacheRepo, parser, nil
			},
			wantErr: false,
			verify: func(t *testing.T, res *dto.SessionDetailResponse, err error) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if len(res.TokenCounts) != 1 {
					t.Fatalf("expected 1 token count entry, got %d", len(res.TokenCounts))
				}
				if res.TokenCounts[0].BoundToNodeID != "node-3" {
					t.Fatalf("expected token count to fall back to node-3, got %q", res.TokenCounts[0].BoundToNodeID)
				}

				var fallbackNode *dto.FlowNode
				for i := range res.Nodes {
					node := &res.Nodes[i]
					if node.ID == "node-3" {
						fallbackNode = node
						break
					}
				}

				if fallbackNode == nil {
					t.Fatal("expected fallback agent message node to exist")
				}
				if fallbackNode.Data.TokenBadge == nil {
					t.Fatal("expected fallback node to receive token badge")
				}
				if fallbackNode.Data.TokenBadge.ConsumedTokens != 100 {
					t.Fatalf("expected fallback badge consumed tokens to be 100, got %+v", fallbackNode.Data.TokenBadge)
				}
			},
		},
		{
			name: "falls back to index based mapping when call ids are entirely missing",
			setup: func(t *testing.T, tmpDir string) (string, usecase.SessionRepository, usecase.CacheRepository, usecase.SessionParser, func()) {
				sessionID := "session-index-fallback"
				sessionRepo := &mockSessionRepositoryForDetail{
					paths: map[string]string{sessionID: filepath.Join(tmpDir, "rollout-index-fallback.jsonl")},
				}

				records := []*model.TypedRecord{
					{
						LineNumber: 1,
						Type:       "session_meta",
						SessionMeta: &model.SessionMetaPayload{
							ID:         sessionID,
							CliVersion: "v0.135.0",
						},
					},
					{
						LineNumber: 2,
						Type:       "event_msg",
						SubType:    "task_started",
						EventMsg: &model.EventMsgPayload{
							TurnID: "turn-index-fallback",
						},
					},
					{
						LineNumber: 3,
						Type:       "response_item",
						SubType:    "function_call",
						ResponseItem: &model.ResponseItemPayload{
							Name:      "read_file",
							Arguments: "{\"path\":\"a.txt\"}",
						},
					},
					{
						LineNumber: 4,
						Type:       "response_item",
						SubType:    "function_call",
						ResponseItem: &model.ResponseItemPayload{
							Name:      "write_file",
							Arguments: "{\"path\":\"b.txt\"}",
						},
					},
					{
						LineNumber: 5,
						Type:       "response_item",
						SubType:    "function_call_output",
						ResponseItem: &model.ResponseItemPayload{
							Output: "alpha output",
						},
					},
					{
						LineNumber: 6,
						Type:       "response_item",
						SubType:    "function_call_output",
						ResponseItem: &model.ResponseItemPayload{
							Output: "beta output",
						},
					},
					{
						LineNumber: 7,
						Type:       "event_msg",
						SubType:    "task_complete",
						EventMsg: &model.EventMsgPayload{
							TurnID: "turn-index-fallback",
						},
					},
				}

				parser := &mockSessionParser{records: records}
				cacheRepo := &mockCacheRepositoryForDetail{}

				var cleanup func()
				return sessionID, sessionRepo, cacheRepo, parser, func() {
					if cleanup != nil {
						cleanup()
					}
				}
			},
			wantErr: false,
			verify: func(t *testing.T, res *dto.SessionDetailResponse, err error) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}

				var outputLabels []string
				for _, node := range res.Nodes {
					if node.Type == "action" && strings.HasPrefix(node.Data.Label, "Output:") {
						outputLabels = append(outputLabels, node.Data.Label+"="+node.Data.Summary)
					}
				}

				expected := []string{
					"Output: read_file=alpha output",
					"Output: write_file=beta output",
				}
				if strings.Join(outputLabels, "|") != strings.Join(expected, "|") {
					t.Fatalf("expected output mapping %v, got %v", expected, outputLabels)
				}
			},
		},
		{
			name: "prefers matching call ids and falls back remaining outputs by index with warn log",
			setup: func(t *testing.T, tmpDir string) (string, usecase.SessionRepository, usecase.CacheRepository, usecase.SessionParser, func()) {
				sessionID := "session-partial-fallback"
				filePath := filepath.Join(tmpDir, "rollout-partial-fallback.jsonl")
				sessionRepo := &mockSessionRepositoryForDetail{
					paths: map[string]string{sessionID: filePath},
				}

				records := []*model.TypedRecord{
					{
						LineNumber: 1,
						Type:       "session_meta",
						SessionMeta: &model.SessionMetaPayload{
							ID:         sessionID,
							CliVersion: "v0.135.0",
						},
					},
					{
						LineNumber: 2,
						Type:       "event_msg",
						SubType:    "task_started",
						EventMsg: &model.EventMsgPayload{
							TurnID: "turn-partial-fallback",
						},
					},
					{
						LineNumber: 3,
						Type:       "response_item",
						SubType:    "function_call",
						ResponseItem: &model.ResponseItemPayload{
							CallID:    "call-1",
							Name:      "read_file",
							Arguments: "{\"path\":\"a.txt\"}",
						},
					},
					{
						LineNumber: 4,
						Type:       "response_item",
						SubType:    "function_call",
						ResponseItem: &model.ResponseItemPayload{
							Name:      "write_file",
							Arguments: "{\"path\":\"b.txt\"}",
						},
					},
					{
						LineNumber: 5,
						Type:       "response_item",
						SubType:    "function_call",
						ResponseItem: &model.ResponseItemPayload{
							CallID:    "call-3",
							Name:      "delete_file",
							Arguments: "{\"path\":\"c.txt\"}",
						},
					},
					{
						LineNumber: 6,
						Type:       "response_item",
						SubType:    "function_call_output",
						ResponseItem: &model.ResponseItemPayload{
							CallID: "call-1",
							Output: "alpha output",
						},
					},
					{
						LineNumber: 7,
						Type:       "response_item",
						SubType:    "function_call_output",
						ResponseItem: &model.ResponseItemPayload{
							Output: "beta output",
						},
					},
					{
						LineNumber: 8,
						Type:       "event_msg",
						SubType:    "task_complete",
						EventMsg: &model.EventMsgPayload{
							TurnID: "turn-partial-fallback",
						},
					},
				}

				parser := &mockSessionParser{records: records}
				cacheRepo := &mockCacheRepositoryForDetail{}
				logBuf, restoreLogger := setupUsecaseTestLogger(t)

				return sessionID, sessionRepo, cacheRepo, parser, func() {
					restoreLogger()
					logOutput := logBuf.String()
					if !strings.Contains(logOutput, "call_id mismatch or missing, falling back to index-based mapping") {
						t.Fatalf("expected warn log for fallback, got %s", logOutput)
					}
					for _, expected := range []string{
						"file_path=" + filePath,
						"turn_id=turn-partial-fallback",
						"call_count=3",
						"output_count=2",
						"details=\"matched: 2, null_assigned: 1, discarded: 0\"",
					} {
						if !strings.Contains(logOutput, expected) {
							t.Fatalf("expected fallback log to contain %q, got %s", expected, logOutput)
						}
					}
				}
			},
			wantErr: false,
			verify: func(t *testing.T, res *dto.SessionDetailResponse, err error) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}

				outputs := map[string]string{}
				for _, node := range res.Nodes {
					if node.Type == "action" && strings.HasPrefix(node.Data.Label, "Output: ") {
						outputs[node.Data.Label] = node.Data.Summary
					}
				}

				if outputs["Output: read_file"] != "alpha output" {
					t.Fatalf("expected read_file output to stay id-matched, got %q", outputs["Output: read_file"])
				}
				if outputs["Output: write_file"] != "beta output" {
					t.Fatalf("expected write_file output to use index fallback, got %q", outputs["Output: write_file"])
				}
				if _, exists := outputs["Output: delete_file"]; exists {
					t.Fatalf("expected delete_file to have null output and no output node, got %v", outputs["Output: delete_file"])
				}
			},
		},
		{
			name: "consecutive batches without middle messages join back to later harness flow",
			setup: func(t *testing.T, tmpDir string) (string, usecase.SessionRepository, usecase.CacheRepository, usecase.SessionParser, func()) {
				sessionID := "session-no-middle-batches"
				filePath := filepath.Join(tmpDir, "rollout-no-middle-batches.jsonl")
				sessionRepo := &mockSessionRepositoryForDetail{
					paths: map[string]string{sessionID: filePath},
				}

				records := []*model.TypedRecord{
					{
						LineNumber: 1,
						Type:       "session_meta",
						SessionMeta: &model.SessionMetaPayload{
							ID:         sessionID,
							CliVersion: "v0.135.0",
						},
					},
					{
						LineNumber: 2,
						Type:       "event_msg",
						SubType:    "task_started",
						EventMsg: &model.EventMsgPayload{
							TurnID: "turn-no-middle-batches",
						},
					},
					{
						LineNumber: 3,
						Type:       "response_item",
						SubType:    "function_call",
						ResponseItem: &model.ResponseItemPayload{
							CallID:    "call-1",
							Name:      "read_file",
							Arguments: "{\"path\":\"a.txt\"}",
						},
					},
					{
						LineNumber: 4,
						Type:       "response_item",
						SubType:    "function_call",
						ResponseItem: &model.ResponseItemPayload{
							CallID:    "call-2",
							Name:      "write_file",
							Arguments: "{\"path\":\"b.txt\"}",
						},
					},
					{
						LineNumber: 5,
						Type:       "response_item",
						SubType:    "function_call_output",
						ResponseItem: &model.ResponseItemPayload{
							CallID: "call-1",
							Output: "read done",
						},
					},
					{
						LineNumber: 6,
						Type:       "response_item",
						SubType:    "function_call_output",
						ResponseItem: &model.ResponseItemPayload{
							CallID: "call-2",
							Output: "write done",
						},
					},
					{
						LineNumber: 7,
						Type:       "response_item",
						SubType:    "function_call",
						ResponseItem: &model.ResponseItemPayload{
							CallID:    "call-3",
							Name:      "list_files",
							Arguments: "{\"path\":\".\"}",
						},
					},
					{
						LineNumber: 8,
						Type:       "response_item",
						SubType:    "function_call_output",
						ResponseItem: &model.ResponseItemPayload{
							CallID: "call-3",
							Output: "listed",
						},
					},
					{
						LineNumber: 9,
						Type:       "event_msg",
						SubType:    "agent_message",
						EventMsg: &model.EventMsgPayload{
							Message: "Done after tools",
						},
					},
					{
						LineNumber: 10,
						Type:       "event_msg",
						SubType:    "task_complete",
						EventMsg: &model.EventMsgPayload{
							TurnID: "turn-no-middle-batches",
						},
					},
				}

				parser := &mockSessionParser{records: records}
				cacheRepo := &mockCacheRepositoryForDetail{}
				return sessionID, sessionRepo, cacheRepo, parser, nil
			},
			wantErr: false,
			verify: func(t *testing.T, res *dto.SessionDetailResponse, err error) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}

				nodesByID := map[string]dto.FlowNode{}
				for _, node := range res.Nodes {
					nodesByID[node.ID] = node
				}

				firstOutput, ok := nodesByID["node-5"]
				if !ok {
					t.Fatal("expected first batch output node-5 to exist")
				}
				secondOutput, ok := nodesByID["node-8"]
				if !ok {
					t.Fatal("expected second batch output node-8 to exist")
				}
				followingMessage, ok := nodesByID["node-9"]
				if !ok {
					t.Fatal("expected following agent message node-9 to exist")
				}

				if firstOutput.Position.Y == secondOutput.Position.Y {
					t.Fatalf("expected consecutive batch outputs to have distinct Y positions, got %v", firstOutput.Position.Y)
				}
				if secondOutput.Position.Y <= firstOutput.Position.Y {
					t.Fatalf("expected second batch output to be below first batch output, got first=%v second=%v", firstOutput.Position.Y, secondOutput.Position.Y)
				}

				hasEdge := func(source, target string) bool {
					t.Helper()
					for _, edge := range res.Edges {
						if edge.Source == source && edge.Target == target {
							return true
						}
					}
					return false
				}

				if !hasEdge("node-5", "node-7") && !hasEdge("node-6", "node-7") {
					t.Fatalf("expected first middle-less batch output to join to the next batch call, edges=%+v", res.Edges)
				}
				if !hasEdge("node-8", "node-9") {
					t.Fatalf("expected second middle-less batch output to join to following harness node, edges=%+v", res.Edges)
				}
				if followingMessage.Position.Y <= secondOutput.Position.Y {
					t.Fatalf("expected following harness node to be below second batch output, got output=%v message=%v", secondOutput.Position.Y, followingMessage.Position.Y)
				}
			},
		},
		{
			name: "consumed tokens difference calculation",
			setup: func(t *testing.T, tmpDir string) (string, usecase.SessionRepository, usecase.CacheRepository, usecase.SessionParser, func()) {
				sessionRepo := &mockSessionRepositoryForDetail{
					paths: map[string]string{"session-diff": "dummy-path"},
				}

				records := []*model.TypedRecord{
					// session_meta
					{
						LineNumber: 1,
						Type:       "session_meta",
						SessionMeta: &model.SessionMetaPayload{
							ID:         "session-diff",
							CliVersion: "v0.125.0",
						},
					},
					// Turn 1
					{
						LineNumber: 2,
						Type:       "event_msg",
						SubType:    "task_started",
						EventMsg: &model.EventMsgPayload{
							TurnID:             "turn-1",
							ModelContextWindow: 64000,
						},
					},
					{
						LineNumber: 3,
						Type:       "event_msg",
						SubType:    "token_count",
						EventMsg: &model.EventMsgPayload{
							Info: &model.TokenInfo{
								TotalTokenUsage: &model.TokenDetail{TotalTokens: 1000, InputTokens: 800, OutputTokens: 200, ReasoningOutputTokens: 0},
								LastTokenUsage:  &model.TokenDetail{TotalTokens: 1000, InputTokens: 800, OutputTokens: 200, ReasoningOutputTokens: 0},
							},
						},
					},
					{
						LineNumber: 4,
						Type:       "event_msg",
						SubType:    "task_complete",
						EventMsg: &model.EventMsgPayload{
							TurnID: "turn-1",
						},
					},
					// Turn 2
					{
						LineNumber: 5,
						Type:       "event_msg",
						SubType:    "task_started",
						EventMsg: &model.EventMsgPayload{
							TurnID:             "turn-2",
							ModelContextWindow: 64000,
						},
					},
					{
						LineNumber: 6,
						Type:       "event_msg",
						SubType:    "token_count",
						EventMsg: &model.EventMsgPayload{
							Info: &model.TokenInfo{
								TotalTokenUsage: &model.TokenDetail{TotalTokens: 1200, InputTokens: 900, OutputTokens: 300, ReasoningOutputTokens: 50},
								LastTokenUsage:  &model.TokenDetail{TotalTokens: 200, InputTokens: 100, OutputTokens: 100, ReasoningOutputTokens: 50},
							},
						},
					},
					{
						LineNumber: 7,
						Type:       "event_msg",
						SubType:    "token_count",
						EventMsg: &model.EventMsgPayload{
							Info: &model.TokenInfo{
								// ターン最後の token_count
								TotalTokenUsage: &model.TokenDetail{TotalTokens: 1500, InputTokens: 1100, OutputTokens: 400, ReasoningOutputTokens: 100},
								LastTokenUsage:  &model.TokenDetail{TotalTokens: 9999, InputTokens: 9999, OutputTokens: 9999, ReasoningOutputTokens: 9999},
							},
						},
					},
					{
						LineNumber: 8,
						Type:       "event_msg",
						SubType:    "task_complete",
						EventMsg: &model.EventMsgPayload{
							TurnID: "turn-2",
						},
					},
					// Turn 3 (No token counts)
					{
						LineNumber: 9,
						Type:       "event_msg",
						SubType:    "task_started",
						EventMsg: &model.EventMsgPayload{
							TurnID:             "turn-3",
							ModelContextWindow: 64000,
						},
					},
					{
						LineNumber: 10,
						Type:       "event_msg",
						SubType:    "task_complete",
						EventMsg: &model.EventMsgPayload{
							TurnID: "turn-3",
						},
					},
				}

				parser := &mockSessionParser{
					records: records,
				}
				cacheRepo := &mockCacheRepositoryForDetail{}
				return "session-diff", sessionRepo, cacheRepo, parser, nil
			},
			wantErr: false,
			verify: func(t *testing.T, res *dto.SessionDetailResponse, err error) {
				if res == nil {
					t.Fatalf("expected response, got nil")
				}
				turns := res.Statistics.Turns
				if len(turns) != 3 {
					t.Fatalf("expected 3 turns statistics, got %d", len(turns))
				}

				// Turn 1 (index 0) - should be 1000, 800, 200, 0
				t1 := turns[0]
				if t1.ConsumedTokens.TotalTokens != 1000 || t1.ConsumedTokens.InputTokens != 800 || t1.ConsumedTokens.OutputTokens != 200 || t1.ConsumedTokens.ReasoningOutputTokens != 0 {
					t.Errorf("turn 1 consumed tokens mismatch: %+v", t1.ConsumedTokens)
				}

				// Turn 2 (index 1) - should be:
				// Total: 1500 - 1000 = 500
				// Input: 1100 - 800 = 300
				// Output: 400 - 200 = 200
				// Reasoning: 100 - 0 = 100
				t2 := turns[1]
				if t2.ConsumedTokens.TotalTokens != 500 || t2.ConsumedTokens.InputTokens != 300 || t2.ConsumedTokens.OutputTokens != 200 || t2.ConsumedTokens.ReasoningOutputTokens != 100 {
					t.Errorf("turn 2 consumed tokens mismatch: %+v (expected: Total=500, Input=300, Output=200, Reasoning=100)", t2.ConsumedTokens)
				}

				// Turn 3 (index 2) - should be 0, 0, 0, 0 (no token counts in this turn)
				t3 := turns[2]
				if t3.ConsumedTokens.TotalTokens != 0 || t3.ConsumedTokens.InputTokens != 0 || t3.ConsumedTokens.OutputTokens != 0 || t3.ConsumedTokens.ReasoningOutputTokens != 0 {
					t.Errorf("turn 3 consumed tokens mismatch: %+v", t3.ConsumedTokens)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			sessionID, sessionRepo, cacheRepo, parser, cleanup := tt.setup(t, tmpDir)
			if cleanup != nil {
				defer cleanup()
			}

			p := parser
			if p == nil {
				p = repository.NewJSONLParser()
			}

			uc := usecase.NewGetSessionDetailUseCase(sessionRepo, cacheRepo, p)
			res, err := uc.Execute(context.Background(), sessionID)

			if (err != nil) != tt.wantErr {
				t.Fatalf("expected error: %v, got %v", tt.wantErr, err)
			}

			if tt.verify != nil {
				tt.verify(t, res, err)
			}
		})
	}
}

func TestGetSessionDetailUseCase_ExecuteBuildsConversationTimelineInRecordOrder(t *testing.T) {
	t.Parallel()

	sessionRepo := &mockSessionRepositoryForDetail{
		paths: map[string]string{"session-timeline": "/tmp/session-timeline.jsonl"},
	}
	cacheRepo := &mockCacheRepositoryForDetail{}
	parser := &mockSessionParser{
		records: []*model.TypedRecord{
			{
				LineNumber: 1,
				Type:       "session_meta",
				SessionMeta: &model.SessionMetaPayload{
					ID:         "session-timeline",
					CliVersion: "v0.131.0",
				},
			},
			{
				LineNumber: 2,
				Type:       "event_msg",
				SubType:    "task_started",
				EventMsg: &model.EventMsgPayload{
					TurnID:    "turn-1",
					StartedAt: 100,
				},
			},
			{
				LineNumber: 3,
				Type:       "event_msg",
				SubType:    "user_message",
				Envelope: model.RecordEnvelope{
					Timestamp: "2026-06-13T01:00:01Z",
				},
				EventMsg: &model.EventMsgPayload{
					Message: "テストを追加してください",
				},
			},
			{
				LineNumber: 4,
				Type:       "response_item",
				SubType:    "message",
				Envelope: model.RecordEnvelope{
					Timestamp: "2026-06-13T01:00:02Z",
				},
				ResponseItem: &model.ResponseItemPayload{
					Role: "assistant",
					Content: []model.MessageContent{
						{Type: "output_text", Text: "実装します"},
					},
				},
			},
			{
				LineNumber: 5,
				Type:       "event_msg",
				SubType:    "task_complete",
				EventMsg: &model.EventMsgPayload{
					TurnID:      "turn-1",
					CompletedAt: 130,
				},
			},
		},
	}

	uc := usecase.NewGetSessionDetailUseCase(sessionRepo, cacheRepo, parser)
	res, err := uc.Execute(context.Background(), "session-timeline")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if len(res.Timeline) != 1 {
		t.Fatalf("len(Timeline) = %d, want 1", len(res.Timeline))
	}
	turn := res.Timeline[0]
	if turn.Index != 0 || turn.TurnID != "turn-1" || turn.Pseudo {
		t.Fatalf("Timeline[0] = %+v, want normal turn 0", turn)
	}
	if turn.DurationMs != 30000 {
		t.Errorf("DurationMs = %d, want 30000", turn.DurationMs)
	}
	if len(turn.Items) != 2 {
		t.Fatalf("len(Timeline[0].Items) = %d, want 2", len(turn.Items))
	}
	if turn.Items[0].Role != "user" || turn.Items[0].Body != "テストを追加してください" {
		t.Errorf("Items[0] = %+v, want user message", turn.Items[0])
	}
	if turn.Items[1].Role != "assistant" || turn.Items[1].Body != "実装します" {
		t.Errorf("Items[1] = %+v, want assistant message", turn.Items[1])
	}
}

func TestGetSessionDetailUseCase_ExecuteAddsReasoningToTimelineInRecordOrder(t *testing.T) {
	t.Parallel()

	sessionRepo := &mockSessionRepositoryForDetail{
		paths: map[string]string{"session-reasoning-timeline": "/tmp/session-reasoning-timeline.jsonl"},
	}
	cacheRepo := &mockCacheRepositoryForDetail{}
	parser := &mockSessionParser{
		records: []*model.TypedRecord{
			{
				LineNumber: 1,
				Type:       "session_meta",
				SessionMeta: &model.SessionMetaPayload{
					ID:         "session-reasoning-timeline",
					CliVersion: "v0.131.0",
				},
			},
			{
				LineNumber: 2,
				Type:       "event_msg",
				SubType:    "task_started",
				EventMsg:   &model.EventMsgPayload{TurnID: "turn-1"},
			},
			{
				LineNumber: 3,
				Type:       "event_msg",
				SubType:    "user_message",
				EventMsg:   &model.EventMsgPayload{Message: "原因を調べてください"},
			},
			{
				LineNumber: 4,
				Type:       "event_msg",
				SubType:    "agent_reasoning",
				Envelope:   model.RecordEnvelope{Timestamp: "2026-06-13T01:00:02Z"},
				EventMsg:   &model.EventMsgPayload{Text: "関連する実装を確認する"},
			},
			{
				LineNumber: 5,
				Type:       "response_item",
				SubType:    "reasoning",
				Envelope:   model.RecordEnvelope{Timestamp: "2026-06-13T01:00:03Z"},
				ResponseItem: &model.ResponseItemPayload{
					Summary: []model.MessageContent{{Type: "summary_text", Text: "実装確認"}},
				},
			},
			{
				LineNumber: 6,
				Type:       "event_msg",
				SubType:    "token_count",
				EventMsg: &model.EventMsgPayload{
					Info: &model.TokenInfo{
						LastTokenUsage:  &model.TokenDetail{TotalTokens: 12, InputTokens: 8, OutputTokens: 4},
						TotalTokenUsage: &model.TokenDetail{TotalTokens: 112, InputTokens: 88, OutputTokens: 24},
					},
				},
			},
			{
				LineNumber: 7,
				Type:       "response_item",
				SubType:    "message",
				ResponseItem: &model.ResponseItemPayload{
					Role:    "assistant",
					Content: []model.MessageContent{{Type: "output_text", Text: "原因を特定しました"}},
				},
			},
			{
				LineNumber: 8,
				Type:       "event_msg",
				SubType:    "task_complete",
				EventMsg:   &model.EventMsgPayload{TurnID: "turn-1"},
			},
		},
	}

	uc := usecase.NewGetSessionDetailUseCase(sessionRepo, cacheRepo, parser)
	res, err := uc.Execute(context.Background(), "session-reasoning-timeline")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	items := res.Timeline[0].Items
	if len(items) != 2 {
		t.Fatalf("len(Timeline[0].Items) = %d, want 2", len(items))
	}
	if items[0].Kind != "conversation" || items[0].Role != "user" {
		t.Errorf("Items[0] = %+v, want user conversation", items[0])
	}
	if items[1].Kind != "conversation" || items[1].Role != "assistant" {
		t.Errorf("Items[1] = %+v, want assistant conversation", items[1])
	}
	for _, item := range items {
		if item.Kind == "reasoning" {
			t.Errorf("expected no reasoning item in timeline, but found one: %+v", item)
		}
	}
}

func TestGetSessionDetailUseCase_ExecuteAddsToolBatchToTimeline(t *testing.T) {
	t.Parallel()

	sessionRepo := &mockSessionRepositoryForDetail{
		paths: map[string]string{"session-tool-timeline": "/tmp/session-tool-timeline.jsonl"},
	}
	cacheRepo := &mockCacheRepositoryForDetail{}
	parser := &mockSessionParser{
		records: []*model.TypedRecord{
			{LineNumber: 1, Type: "session_meta", SessionMeta: &model.SessionMetaPayload{ID: "session-tool-timeline"}},
			{LineNumber: 2, Type: "event_msg", SubType: "task_started", EventMsg: &model.EventMsgPayload{TurnID: "turn-1"}},
			{
				LineNumber: 3,
				Type:       "response_item",
				SubType:    "function_call",
				ResponseItem: &model.ResponseItemPayload{
					Name: "run_command", Arguments: `{"CommandLine":"go test ./..."}`, CallID: "call-1",
				},
			},
			{
				LineNumber: 4,
				Type:       "response_item",
				SubType:    "function_call",
				ResponseItem: &model.ResponseItemPayload{
					Name: "run_command", Arguments: `{"CommandLine":"go build"}`, CallID: "call-2",
				},
			},
			{LineNumber: 5, Type: "response_item", SubType: "function_call_output", ResponseItem: &model.ResponseItemPayload{CallID: "call-2", Output: "build ok"}},
			{LineNumber: 6, Type: "response_item", SubType: "function_call_output", ResponseItem: &model.ResponseItemPayload{CallID: "call-1", Output: "test ok"}},
			{LineNumber: 7, Type: "event_msg", SubType: "task_complete", EventMsg: &model.EventMsgPayload{TurnID: "turn-1"}},
		},
	}

	uc := usecase.NewGetSessionDetailUseCase(sessionRepo, cacheRepo, parser)
	res, err := uc.Execute(context.Background(), "session-tool-timeline")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	items := res.Timeline[0].Items
	if len(items) != 1 {
		t.Fatalf("len(Timeline[0].Items) = %d, want 1", len(items))
	}
	item := items[0]
	if item.Kind != "tool" || item.Label != "コマンド実行" || item.Body != "go test ./...\ngo build" || item.RecordCount != 4 {
		t.Fatalf("tool item = %+v, want one batch containing four records", item)
	}
	if len(item.Details) != 0 {
		t.Errorf("expected no details for command execution, but got %+v", item.Details)
	}
}

func TestGetSessionDetailUseCase_ExecuteAddsWebAndMCPEventsToTimeline(t *testing.T) {
	t.Parallel()

	sessionRepo := &mockSessionRepositoryForDetail{
		paths: map[string]string{"session-reference-timeline": "/tmp/session-reference-timeline.jsonl"},
	}
	cacheRepo := &mockCacheRepositoryForDetail{}
	parser := &mockSessionParser{
		records: []*model.TypedRecord{
			{LineNumber: 1, Type: "session_meta", SessionMeta: &model.SessionMetaPayload{ID: "session-reference-timeline"}},
			{LineNumber: 2, Type: "event_msg", SubType: "task_started", EventMsg: &model.EventMsgPayload{TurnID: "turn-1"}},
			{
				LineNumber: 3,
				Type:       "response_item",
				SubType:    "web_search_call",
				ResponseItem: &model.ResponseItemPayload{
					Action: &model.SearchAction{Type: "search", Query: "Go stable release"},
				},
			},
			{
				LineNumber: 4,
				Type:       "event_msg",
				SubType:    "web_search_end",
				EventMsg: &model.EventMsgPayload{
					Action: &model.SearchAction{Type: "open", Query: "https://go.dev/doc/devel/release"},
				},
			},
			{
				LineNumber: 5,
				Type:       "event_msg",
				SubType:    "mcp_tool_call_end",
				EventMsg: &model.EventMsgPayload{
					CallID: "mcp-1",
					Invocation: &model.MCPInvocation{
						Server: "figma", Tool: "whoami", Arguments: json.RawMessage(`{"verbose":true}`),
					},
					Result: `{"user":"codex"}`,
				},
			},
			{LineNumber: 6, Type: "event_msg", SubType: "task_complete", EventMsg: &model.EventMsgPayload{TurnID: "turn-1"}},
		},
	}

	uc := usecase.NewGetSessionDetailUseCase(sessionRepo, cacheRepo, parser)
	res, err := uc.Execute(context.Background(), "session-reference-timeline")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if len(res.Timeline) != 0 {
		t.Fatalf("len(Timeline) = %d, want 0", len(res.Timeline))
	}
}

func TestGetSessionDetailUseCase_ExecuteAddsInstructionsAndSystemEventsToTimeline(t *testing.T) {
	t.Parallel()

	sessionRepo := &mockSessionRepositoryForDetail{
		paths: map[string]string{"session-instructions-timeline": "/tmp/session-instructions-timeline.jsonl"},
	}
	cacheRepo := &mockCacheRepositoryForDetail{}
	parser := &mockSessionParser{
		records: []*model.TypedRecord{
			{
				LineNumber: 1,
				Type:       "session_meta",
				SessionMeta: &model.SessionMetaPayload{
					ID: "session-instructions-timeline", BaseInstructions: &model.Instructions{Text: "base rules"},
				},
			},
			{
				LineNumber: 2,
				Type:       "turn_context",
				TurnContext: &model.TurnContextPayload{
					TurnID:            "turn-1",
					UserInstructions:  "user rules",
					CollaborationMode: &model.CollaborationMode{DeveloperInstructions: "developer rules"},
				},
			},
			{LineNumber: 3, Type: "event_msg", SubType: "task_started", EventMsg: &model.EventMsgPayload{TurnID: "turn-1"}},
			{LineNumber: 4, Type: "event_msg", SubType: "custom_notice", EventMsg: &model.EventMsgPayload{Message: "unknown event"}},
			{LineNumber: 5, Type: "event_msg", SubType: "task_complete", EventMsg: &model.EventMsgPayload{TurnID: "turn-1"}},
		},
	}

	uc := usecase.NewGetSessionDetailUseCase(sessionRepo, cacheRepo, parser)
	res, err := uc.Execute(context.Background(), "session-instructions-timeline")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if len(res.Timeline) != 0 {
		t.Fatalf("len(Timeline) = %d, want 0", len(res.Timeline))
	}
}

func TestGetSessionDetailUseCase_ExecuteAddsCommandAndReferenceEventsToTimeline(t *testing.T) {
	t.Parallel()

	exitCode := 0
	sessionRepo := &mockSessionRepositoryForDetail{
		paths: map[string]string{"session-command-timeline": "/tmp/session-command-timeline.jsonl"},
	}
	cacheRepo := &mockCacheRepositoryForDetail{}
	parser := &mockSessionParser{
		records: []*model.TypedRecord{
			{LineNumber: 1, Type: "session_meta", SessionMeta: &model.SessionMetaPayload{ID: "session-command-timeline"}},
			{LineNumber: 2, Type: "event_msg", SubType: "task_started", EventMsg: &model.EventMsgPayload{TurnID: "turn-1"}},
			{
				LineNumber: 3,
				Type:       "event_msg",
				SubType:    "exec_command_end",
				EventMsg: &model.EventMsgPayload{
					CallID: "command-1", Command: []string{"go", "test", "./..."}, ExitCode: &exitCode, AggregatedOutput: "ok packages",
				},
			},
			{
				LineNumber: 4,
				Type:       "event_msg",
				SubType:    "view_image_tool_call",
				EventMsg:   &model.EventMsgPayload{CallID: "image-1", Path: "/tmp/screenshot.png"},
			},
			{LineNumber: 5, Type: "event_msg", SubType: "task_complete", EventMsg: &model.EventMsgPayload{TurnID: "turn-1"}},
		},
	}

	uc := usecase.NewGetSessionDetailUseCase(sessionRepo, cacheRepo, parser)
	res, err := uc.Execute(context.Background(), "session-command-timeline")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	items := res.Timeline[0].Items
	if len(items) != 1 {
		t.Fatalf("len(Timeline[0].Items) = %d, want 1 (command): %+v", len(items), items)
	}
	command := items[0]
	if command.Kind != "tool" || command.Label != "コマンド完了" || command.Body != "go test ./..." {
		t.Errorf("command item = %+v, want Kind: tool, Label: コマンド完了, Body: go test ./...", command)
	}
	if len(command.Details) != 0 {
		t.Errorf("expected no details for command complete event, but got %+v", command.Details)
	}
}

func TestGetSessionDetailUseCase_ExecuteKeepsOtherSystemEventsInTimeline(t *testing.T) {
	t.Parallel()

	sessionRepo := &mockSessionRepositoryForDetail{
		paths: map[string]string{"session-system-timeline": "/tmp/session-system-timeline.jsonl"},
	}
	cacheRepo := &mockCacheRepositoryForDetail{}
	parser := &mockSessionParser{
		records: []*model.TypedRecord{
			{LineNumber: 1, Type: "session_meta", SessionMeta: &model.SessionMetaPayload{ID: "session-system-timeline"}},
			{LineNumber: 2, Type: "event_msg", SubType: "thread_name_updated", EventMsg: &model.EventMsgPayload{ThreadName: "Issue 176"}},
			{LineNumber: 3, Type: "event_msg", SubType: "task_started", EventMsg: &model.EventMsgPayload{TurnID: "turn-1"}},
			{
				LineNumber: 4,
				Type:       "event_msg",
				SubType:    "item_completed",
				EventMsg:   &model.EventMsgPayload{Item: &model.CompletedItem{Type: "plan", Text: "step done"}},
			},
			{LineNumber: 5, Type: "response_item", SubType: "unknown_response", Raw: `{"payload":{"type":"unknown_response"}}`},
			{
				LineNumber: 6,
				Type:       "event_msg",
				SubType:    "collab_waiting_end",
				EventMsg:   &model.EventMsgPayload{CallID: "collab-1", Statuses: map[string]string{"agent": "completed"}},
			},
			{LineNumber: 7, Type: "event_msg", SubType: "task_complete", EventMsg: &model.EventMsgPayload{TurnID: "turn-1"}},
		},
	}

	uc := usecase.NewGetSessionDetailUseCase(sessionRepo, cacheRepo, parser)
	res, err := uc.Execute(context.Background(), "session-system-timeline")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if len(res.Timeline) != 0 {
		t.Fatalf("len(Timeline) = %d, want 0", len(res.Timeline))
	}
}

func TestGetSessionDetailUseCase_ExecuteDeduplicatesConversationMessages(t *testing.T) {
	t.Parallel()

	sessionRepo := &mockSessionRepositoryForDetail{
		paths: map[string]string{"session-deduplicate": "/tmp/session-deduplicate.jsonl"},
	}
	cacheRepo := &mockCacheRepositoryForDetail{}
	parser := &mockSessionParser{
		records: []*model.TypedRecord{
			{
				LineNumber: 1,
				Type:       "session_meta",
				SessionMeta: &model.SessionMetaPayload{
					ID:         "session-deduplicate",
					CliVersion: "v0.131.0",
				},
			},
			{
				LineNumber: 2,
				Type:       "event_msg",
				SubType:    "task_started",
				EventMsg:   &model.EventMsgPayload{TurnID: "turn-1"},
			},
			{
				LineNumber: 3,
				Type:       "event_msg",
				SubType:    "agent_message",
				Envelope: model.RecordEnvelope{
					Timestamp: "2026-06-13T01:00:01Z",
				},
				EventMsg: &model.EventMsgPayload{Message: "同じ応答"},
			},
			{
				LineNumber: 4,
				Type:       "response_item",
				SubType:    "message",
				Envelope: model.RecordEnvelope{
					Timestamp: "2026-06-13T01:00:01Z",
				},
				ResponseItem: &model.ResponseItemPayload{
					Role:    "assistant",
					Content: []model.MessageContent{{Type: "output_text", Text: "同じ応答"}},
				},
			},
			{
				LineNumber: 5,
				Type:       "event_msg",
				SubType:    "task_complete",
				EventMsg:   &model.EventMsgPayload{TurnID: "turn-1"},
			},
		},
	}

	uc := usecase.NewGetSessionDetailUseCase(sessionRepo, cacheRepo, parser)
	res, err := uc.Execute(context.Background(), "session-deduplicate")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if len(res.Timeline) != 1 {
		t.Fatalf("len(Timeline) = %d, want 1", len(res.Timeline))
	}
	if len(res.Timeline[0].Items) != 1 {
		t.Fatalf("len(Timeline[0].Items) = %d, want 1", len(res.Timeline[0].Items))
	}
}

func TestGetSessionDetailUseCase_ExecuteAggregatesConversationTokenUsage(t *testing.T) {
	t.Parallel()

	sessionRepo := &mockSessionRepositoryForDetail{
		paths: map[string]string{"session-token-usage": "/tmp/session-token-usage.jsonl"},
	}
	cacheRepo := &mockCacheRepositoryForDetail{}
	parser := &mockSessionParser{
		records: []*model.TypedRecord{
			{
				LineNumber: 1,
				Type:       "session_meta",
				SessionMeta: &model.SessionMetaPayload{
					ID:         "session-token-usage",
					CliVersion: "v0.131.0",
				},
			},
			{
				LineNumber: 2,
				Type:       "event_msg",
				SubType:    "task_started",
				EventMsg:   &model.EventMsgPayload{TurnID: "turn-1"},
			},
			{
				LineNumber: 3,
				Type:       "event_msg",
				SubType:    "agent_message",
				Envelope:   model.RecordEnvelope{Timestamp: "2026-06-13T01:00:01Z"},
				EventMsg:   &model.EventMsgPayload{Message: "統合対象"},
			},
			{
				LineNumber: 4,
				Type:       "event_msg",
				SubType:    "token_count",
				EventMsg: &model.EventMsgPayload{
					Info: &model.TokenInfo{
						LastTokenUsage:  &model.TokenDetail{TotalTokens: 10, InputTokens: 7, OutputTokens: 3},
						TotalTokenUsage: &model.TokenDetail{TotalTokens: 100, InputTokens: 70, OutputTokens: 30},
					},
				},
			},
			{
				LineNumber: 5,
				Type:       "response_item",
				SubType:    "message",
				Envelope:   model.RecordEnvelope{Timestamp: "2026-06-13T01:00:01Z"},
				ResponseItem: &model.ResponseItemPayload{
					Role:    "assistant",
					Content: []model.MessageContent{{Type: "output_text", Text: "統合対象"}},
				},
			},
			{
				LineNumber: 6,
				Type:       "event_msg",
				SubType:    "token_count",
				EventMsg: &model.EventMsgPayload{
					Info: &model.TokenInfo{
						LastTokenUsage:  &model.TokenDetail{TotalTokens: 20, InputTokens: 12, OutputTokens: 8},
						TotalTokenUsage: &model.TokenDetail{TotalTokens: 120, InputTokens: 82, OutputTokens: 38},
					},
				},
			},
			{
				LineNumber: 7,
				Type:       "event_msg",
				SubType:    "task_complete",
				EventMsg:   &model.EventMsgPayload{TurnID: "turn-1"},
			},
		},
	}

	uc := usecase.NewGetSessionDetailUseCase(sessionRepo, cacheRepo, parser)
	res, err := uc.Execute(context.Background(), "session-token-usage")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	item := res.Timeline[0].Items[0]
	if item.TokenCountCount != 2 {
		t.Errorf("TokenCountCount = %d, want 2", item.TokenCountCount)
	}
	if item.LastTokenUsage.TotalTokens != 30 || item.LastTokenUsage.InputTokens != 19 || item.LastTokenUsage.OutputTokens != 11 {
		t.Errorf("LastTokenUsage = %+v, want totals 30/19/11", item.LastTokenUsage)
	}
	if item.TotalTokenUsage == nil || item.TotalTokenUsage.TotalTokens != 120 {
		t.Errorf("TotalTokenUsage = %+v, want latest total 120", item.TotalTokenUsage)
	}
	if len(item.TokenCountIndices) != 2 || item.TokenCountIndices[0] != 0 || item.TokenCountIndices[1] != 1 {
		t.Errorf("TokenCountIndices = %v, want [0 1]", item.TokenCountIndices)
	}
	if res.Timeline[0].ConsumedTokens.TotalTokens != 120 {
		t.Errorf("ConsumedTokens.TotalTokens = %d, want 120", res.Timeline[0].ConsumedTokens.TotalTokens)
	}
}

func TestGetSessionDetailUseCase_ExecuteBuildsPseudoConversationTurn(t *testing.T) {
	t.Parallel()

	sessionRepo := &mockSessionRepositoryForDetail{
		paths: map[string]string{"session-pseudo-turn": "/tmp/session-pseudo-turn.jsonl"},
	}
	cacheRepo := &mockCacheRepositoryForDetail{}
	parser := &mockSessionParser{
		records: []*model.TypedRecord{
			{
				LineNumber: 1,
				Type:       "session_meta",
				SessionMeta: &model.SessionMetaPayload{
					ID:         "session-pseudo-turn",
					CliVersion: "v0.131.0",
				},
			},
			{
				LineNumber: 2,
				Type:       "event_msg",
				SubType:    "user_message",
				Envelope:   model.RecordEnvelope{Timestamp: "2026-06-13T00:59:59Z"},
				EventMsg:   &model.EventMsgPayload{Message: "ターン外の発言"},
			},
			{
				LineNumber: 3,
				Type:       "event_msg",
				SubType:    "task_started",
				EventMsg:   &model.EventMsgPayload{TurnID: "turn-1"},
			},
			{
				LineNumber: 4,
				Type:       "event_msg",
				SubType:    "agent_message",
				Envelope:   model.RecordEnvelope{Timestamp: "2026-06-13T01:00:00Z"},
				EventMsg:   &model.EventMsgPayload{Message: "通常ターンの発言"},
			},
			{
				LineNumber: 5,
				Type:       "event_msg",
				SubType:    "task_complete",
				EventMsg:   &model.EventMsgPayload{TurnID: "turn-1"},
			},
		},
	}

	uc := usecase.NewGetSessionDetailUseCase(sessionRepo, cacheRepo, parser)
	res, err := uc.Execute(context.Background(), "session-pseudo-turn")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if len(res.Timeline) != 2 {
		t.Fatalf("len(Timeline) = %d, want 2", len(res.Timeline))
	}
	if !res.Timeline[0].Pseudo || res.Timeline[0].Index != -1 {
		t.Errorf("Timeline[0] = %+v, want pseudo turn", res.Timeline[0])
	}
	if res.Timeline[0].Items[0].Body != "ターン外の発言" {
		t.Errorf("Timeline[0].Items[0].Body = %q", res.Timeline[0].Items[0].Body)
	}
	if res.Timeline[1].Pseudo || res.Timeline[1].Items[0].Body != "通常ターンの発言" {
		t.Errorf("Timeline[1] = %+v, want normal turn", res.Timeline[1])
	}
}

func TestGetSessionDetailUseCase_RecursiveSubagents(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "rollout-parent.jsonl")

	// 親セッションのログレコード
	logs := []string{
		`{"type":"session_meta","timestamp":1717084800,"payload":{"id":"parent-session-3","cli_version":"v0.131.0"}}`,
		`{"type":"event_msg","timestamp":1717084810,"payload":{"type":"task_started","turn_id":"turn-1"}}`,
		`{"type":"event_msg","timestamp":1717084820,"payload":{"type":"collab_agent_spawn_end","new_thread_id":"sub-session-uuid-5678","new_agent_nickname":"SubBot3"}}`,
		`{"type":"event_msg","timestamp":1717084830,"payload":{"type":"task_complete","turn_id":"turn-1"}}`,
	}
	if err := os.WriteFile(filePath, []byte(strings.Join(logs, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}

	sessionRepo := &mockSessionRepositoryForDetail{
		paths: map[string]string{"parent-session-3": filePath},
		sessions: []dto.SessionSummary{
			{
				ID:              "parent-session-3",
				ChildSessionIDs: []string{"sub-session-uuid-5678"},
			},
			{
				ID:              "sub-session-uuid-5678",
				ParentSessionID: getStringPtrRef("parent-session-3"),
				ChildSessionIDs: []string{"grandchild-session-uuid-9999"},
				TotalTokens:     getInt64PtrRef(200),
				InputTokens:     getInt64PtrRef(150),
				OutputTokens:    getInt64PtrRef(50),
				TurnCount:       getIntPtrRef(5),
				StepCount:       getIntPtrRef(10),
				DurationMs:      getInt64PtrRef(30000),
			},
			{
				ID:              "grandchild-session-uuid-9999",
				ParentSessionID: getStringPtrRef("sub-session-uuid-5678"),
				Originator:      getStringPtrRef("GrandBot"),
				TotalTokens:     getInt64PtrRef(100),
				InputTokens:     getInt64PtrRef(80),
				OutputTokens:    getInt64PtrRef(20),
				TurnCount:       getIntPtrRef(2),
				StepCount:       getIntPtrRef(4),
				DurationMs:      getInt64PtrRef(15000),
			},
		},
	}
	cacheRepo := &mockCacheRepositoryForDetail{}

	parser := repository.NewJSONLParser()
	uc := usecase.NewGetSessionDetailUseCase(sessionRepo, cacheRepo, parser)
	res, err := uc.Execute(context.Background(), "parent-session-3")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if len(res.Subagents) != 2 {
		t.Fatalf("len(Subagents) = %d, want 2", len(res.Subagents))
	}

	sub := res.Subagents[0]
	if sub.ID != "sub-session-uuid-5678" || sub.Nickname != "SubBot3" || sub.TotalTokens != 200 || sub.InputTokens != 150 || sub.OutputTokens != 50 || *sub.TurnCount != 5 || *sub.StepCount != 10 || *sub.DurationMs != 30000 {
		t.Errorf("Subagent[0] mismatch: %+v", sub)
	}

	grand := res.Subagents[1]
	if grand.ID != "grandchild-session-uuid-9999" || grand.Nickname != "GrandBot" || grand.TotalTokens != 100 || grand.InputTokens != 80 || grand.OutputTokens != 20 || *grand.TurnCount != 2 || *grand.StepCount != 4 || *grand.DurationMs != 15000 {
		t.Errorf("Subagent[1] mismatch: %+v", grand)
	}
}

func getStringPtrRef(s string) *string {
	return &s
}

func getInt64PtrRef(i int64) *int64 {
	return &i
}

func getIntPtrRef(i int) *int {
	return &i
}

func TestGetSessionDetailUseCase_ToolCallCountInTurnStats(t *testing.T) {
	t.Parallel()

	sessionRepo := &mockSessionRepositoryForDetail{
		paths: map[string]string{"session-tool-count-test": "/tmp/session-tool-count-test.jsonl"},
	}
	cacheRepo := &mockCacheRepositoryForDetail{}
	parser := &mockSessionParser{
		records: []*model.TypedRecord{
			{LineNumber: 1, Type: "session_meta", SessionMeta: &model.SessionMetaPayload{ID: "session-tool-count-test", CliVersion: "v0.131.0"}},
			{LineNumber: 2, Type: "event_msg", SubType: "task_started", EventMsg: &model.EventMsgPayload{TurnID: "turn-1"}},
			{
				LineNumber: 3,
				Type:       "response_item",
				SubType:    "function_call",
				ResponseItem: &model.ResponseItemPayload{
					Name: "run_command", Arguments: `{"CommandLine":"go test ./..."}`, CallID: "call-1",
				},
			},
			{
				LineNumber: 4,
				Type:       "response_item",
				SubType:    "function_call",
				ResponseItem: &model.ResponseItemPayload{
					Name: "run_command", Arguments: `{"CommandLine":"go build"}`, CallID: "call-2",
				},
			},
			{LineNumber: 5, Type: "response_item", SubType: "function_call_output", ResponseItem: &model.ResponseItemPayload{CallID: "call-2", Output: "build ok"}},
			{LineNumber: 6, Type: "response_item", SubType: "function_call_output", ResponseItem: &model.ResponseItemPayload{CallID: "call-1", Output: "test ok"}},
			{LineNumber: 7, Type: "event_msg", SubType: "task_complete", EventMsg: &model.EventMsgPayload{TurnID: "turn-1"}},
		},
	}

	uc := usecase.NewGetSessionDetailUseCase(sessionRepo, cacheRepo, parser)
	res, err := uc.Execute(context.Background(), "session-tool-count-test")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if len(res.Statistics.Turns) != 1 {
		t.Fatalf("len(Turns) = %d, want 1", len(res.Statistics.Turns))
	}
	turn := res.Statistics.Turns[0]
	if turn.ToolCallCount != 2 {
		t.Errorf("turn.ToolCallCount = %d, want 2", turn.ToolCallCount)
	}
}
