package repository

import (
	"codex-session-display/internal/domain/dto"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestNewCacheFSRepository(t *testing.T) {
	tests := []struct {
		name     string
		cacheDir string
	}{
		{
			name:     "success constructor",
			cacheDir: "test_dir",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewCacheFSRepository(tt.cacheDir)
			if repo.cacheDir != tt.cacheDir {
				t.Errorf("expected cacheDir to be '%s', got '%s'", tt.cacheDir, repo.cacheDir)
			}
		})
	}
}

func TestCacheFSRepository_GetSessionSummary(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "codex-cache-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	sessionID := "019e5514-ed44-78b2-bf88-233d6e4273bf"

	// ファイル書き込み用のヘルパー関数
	writeFile := func(filename string, content []byte) {
		err := os.WriteFile(filepath.Join(tmpDir, filename), content, 0o644)
		if err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}
	}

	// 1. 成功ケースの詳細情報
	detailSuccess := dto.SessionDetailResponse{
		ID:                 sessionID,
		CacheSchemaVersion: dto.CurrentSessionDetailCacheSchemaVersion,
		ParsedAt:           "2026-05-30T07:14:40Z",
		Nodes: []dto.FlowNode{
			{
				ID:   "meta-node-1",
				Type: "sessionMeta",
				Data: dto.NodeData{
					Category: "meta",
					Label:    "Session Meta",
					Meta: map[string]interface{}{
						"cwd":            "/Users/yukihito/workspace",
						"cli_version":    "0.131.0",
						"originator":     "codex-tui",
						"model_provider": "openai",
						"git_branch":     "main",
						"source":         "cli",
						"timestamp":      "2026-05-23T13:44:55Z",
					},
				},
			},
		},
	}
	dataSuccess, _ := json.Marshal(detailSuccess)
	writeFile(sessionID+"_success.json", dataSuccess)

	// 2. 無効な JSON
	writeFile(sessionID+"_invalid.json", []byte("{invalid_json}"))

	// 3. sessionMeta ノードが存在しないケース
	detailNoMeta := dto.SessionDetailResponse{
		ID:                 sessionID,
		CacheSchemaVersion: dto.CurrentSessionDetailCacheSchemaVersion,
		Nodes:              []dto.FlowNode{{ID: "n1", Type: "other"}},
	}
	dataNoMeta, _ := json.Marshal(detailNoMeta)
	writeFile(sessionID+"_no_meta.json", dataNoMeta)

	// 4. Meta マップが nil のケース
	detailNilMeta := dto.SessionDetailResponse{
		ID:                 sessionID,
		CacheSchemaVersion: dto.CurrentSessionDetailCacheSchemaVersion,
		Nodes: []dto.FlowNode{
			{
				Type: "sessionMeta",
				Data: dto.NodeData{
					Meta: nil,
				},
			},
		},
	}
	dataNilMeta, _ := json.Marshal(detailNilMeta)
	writeFile(sessionID+"_nil_meta.json", dataNilMeta)

	// 5. Meta 内の値が nil、欠損、または文字列ではないケース
	detailMixedMeta := dto.SessionDetailResponse{
		ID:                 sessionID,
		CacheSchemaVersion: dto.CurrentSessionDetailCacheSchemaVersion,
		Nodes: []dto.FlowNode{
			{
				Type: "sessionMeta",
				Data: dto.NodeData{
					Meta: map[string]interface{}{
						"cwd":         nil,         // nil 値
						"cli_version": 123,         // 文字列ではない型
						"originator":  "codex-tui", // 有効な文字列
						// model_provider は欠損している
					},
				},
			},
		},
	}
	dataMixedMeta, _ := json.Marshal(detailMixedMeta)
	writeFile(sessionID+"_mixed_meta.json", dataMixedMeta)

	// 6. 旧キャッシュスキーマ
	dataLegacy, _ := json.Marshal(dto.SessionDetailResponse{
		ID:                 sessionID,
		CacheSchemaVersion: 1,
	})
	writeFile(sessionID+"_legacy.json", dataLegacy)

	tests := []struct {
		name      string
		sessionID string
		wantErr   bool
		verify    func(t *testing.T, summary *dto.SessionSummary)
	}{
		{
			name:      "file not found",
			sessionID: "non-existent-id",
			wantErr:   true,
		},
		{
			name:      "invalid json",
			sessionID: sessionID + "_invalid",
			wantErr:   true,
		},
		{
			name:      "missing sessionMeta node",
			sessionID: sessionID + "_no_meta",
			wantErr:   true,
		},
		{
			name:      "legacy cache schema",
			sessionID: sessionID + "_legacy",
			wantErr:   true,
		},
		{
			name:      "nil meta map",
			sessionID: sessionID + "_nil_meta",
			wantErr:   false,
			verify: func(t *testing.T, s *dto.SessionSummary) {
				if s.Cwd != nil {
					t.Errorf("expected nil Cwd, got %v", *s.Cwd)
				}
			},
		},
		{
			name:      "mixed meta values (nil, non-string, missing, valid)",
			sessionID: sessionID + "_mixed_meta",
			wantErr:   false,
			verify: func(t *testing.T, s *dto.SessionSummary) {
				if s.Cwd != nil {
					t.Errorf("expected Cwd to be nil (was value nil), got %v", *s.Cwd)
				}
				if s.CliVersion != nil {
					t.Errorf("expected CliVersion to be nil (was non-string), got %v", *s.CliVersion)
				}
				if s.ModelProvider != nil {
					t.Errorf("expected ModelProvider to be nil (was missing), got %v", *s.ModelProvider)
				}
				if s.Originator == nil || *s.Originator != "codex-tui" {
					t.Errorf("expected Originator to be 'codex-tui', got %v", s.Originator)
				}
			},
		},
		{
			name:      "success case",
			sessionID: sessionID + "_success",
			wantErr:   false,
			verify: func(t *testing.T, s *dto.SessionSummary) {
				if s.ID != sessionID+"_success" {
					t.Errorf("expected ID %s, got %s", sessionID+"_success", s.ID)
				}
				if s.Cwd == nil || *s.Cwd != "/Users/yukihito/workspace" {
					t.Errorf("expected Cwd '/Users/yukihito/workspace', got '%v'", s.Cwd)
				}
				if s.CliVersion == nil || *s.CliVersion != "0.131.0" {
					t.Errorf("expected CliVersion '0.131.0', got '%v'", s.CliVersion)
				}
				if s.Originator == nil || *s.Originator != "codex-tui" {
					t.Errorf("expected Originator 'codex-tui', got '%v'", s.Originator)
				}
				if s.ModelProvider == nil || *s.ModelProvider != "openai" {
					t.Errorf("expected ModelProvider 'openai', got '%v'", s.ModelProvider)
				}
				if s.Branch == nil || *s.Branch != "main" {
					t.Errorf("expected Branch 'main', got '%v'", s.Branch)
				}
				if s.Source == nil || *s.Source != "cli" {
					t.Errorf("expected Source 'cli', got '%v'", s.Source)
				}
				if s.Timestamp == nil || *s.Timestamp != "2026-05-23T13:44:55Z" {
					t.Errorf("expected Timestamp '2026-05-23T13:44:55Z', got '%v'", s.Timestamp)
				}
				if !s.Parsed {
					t.Errorf("expected Parsed true, got false")
				}
			},
		},
	}

	repo := NewCacheFSRepository(tmpDir)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary, err := repo.GetSessionSummary(context.Background(), tt.sessionID)
			if (err != nil) != tt.wantErr {
				t.Fatalf("expected error: %v, got: %v", tt.wantErr, err)
			}
			if tt.verify != nil {
				tt.verify(t, summary)
			}
		})
	}
}

func TestCacheFSRepository_SaveAndGetSessionDetail(t *testing.T) {
	tmpDir := t.TempDir()
	repo := NewCacheFSRepository(tmpDir)

	sessionID := "test-detail-session"
	detail := &dto.SessionDetailResponse{
		ID:       sessionID,
		ParsedAt: "2026-05-30T07:14:40Z",
		Nodes: []dto.FlowNode{
			{
				ID:   "node-1",
				Type: "sessionMeta",
				Data: dto.NodeData{
					Category: "meta",
					Label:    "Meta",
				},
			},
		},
		Edges: []dto.FlowEdge{
			{
				ID:     "edge-1",
				Source: "node-1",
				Target: "node-2",
			},
		},
	}

	// 1. Save detail
	if err := repo.SaveSessionDetail(context.Background(), sessionID, detail); err != nil {
		t.Fatalf("failed to save session detail: %v", err)
	}

	// 2. Get detail
	got, err := repo.GetSessionDetail(context.Background(), sessionID)
	if err != nil {
		t.Fatalf("failed to get session detail: %v", err)
	}

	modTime, err := repo.GetSessionDetailModTime(context.Background(), sessionID)
	if err != nil {
		t.Fatalf("failed to get session detail mod time: %v", err)
	}
	if modTime.IsZero() {
		t.Fatal("expected non-zero mod time")
	}

	if got.ID != detail.ID || len(got.Nodes) != len(detail.Nodes) || got.Nodes[0].ID != detail.Nodes[0].ID {
		t.Errorf("got detail mismatch. Expected %+v, got %+v", detail, got)
	}

	// 3. Get non-existent detail
	_, err = repo.GetSessionDetail(context.Background(), "non-existent")
	if err == nil {
		t.Error("expected error for non-existent session detail, got nil")
	}

	_, err = repo.GetSessionDetailModTime(context.Background(), "non-existent")
	if err == nil {
		t.Error("expected error for non-existent session detail mod time, got nil")
	}

	// 4. Decode JSON error
	invalidJSONPath := filepath.Join(tmpDir, "invalid-json-detail.json")
	if err := os.WriteFile(invalidJSONPath, []byte("{invalid}"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = repo.GetSessionDetail(context.Background(), "invalid-json-detail")
	if err == nil {
		t.Error("expected error for invalid json session detail, got nil")
	}

	// 5. Marshal error (channel cannot be marshaled to JSON)
	invalidDetail := &dto.SessionDetailResponse{
		ID: "invalid-marshal-session",
		Nodes: []dto.FlowNode{
			{
				ID: "node-1",
				Data: dto.NodeData{
					Meta: map[string]interface{}{
						"unmarshalable": make(chan int),
					},
				},
			},
		},
	}
	err = repo.SaveSessionDetail(context.Background(), "invalid-marshal-session", invalidDetail)
	if err == nil {
		t.Error("expected marshal error, got nil")
	}

	// 6. WriteFile error
	badRepo := NewCacheFSRepository("/non-existent-dir-12345/sub")
	err = badRepo.SaveSessionDetail(context.Background(), "any-session", detail)
	if err == nil {
		t.Error("expected write file error, got nil")
	}
}
