package repository

import (
	"codex-session-display/internal/domain/dto"
	"context"
	"encoding/json"
	"fmt"
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
			wantErr:   false,
			verify: func(t *testing.T, s *dto.SessionSummary) {
				if s.ID != sessionID+"_no_meta" {
					t.Errorf("expected ID %s, got %s", sessionID+"_no_meta", s.ID)
				}
				if !s.Parsed {
					t.Errorf("expected Parsed true, got false")
				}
				if s.Cwd != nil {
					t.Errorf("expected Cwd to be nil, got %v", *s.Cwd)
				}
			},
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
			summary, err := repo.GetSessionSummary(context.Background(), dto.SessionProviderCodex, tt.sessionID)
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
	if err := repo.SaveSessionDetail(context.Background(), dto.SessionProviderCodex, sessionID, detail); err != nil {
		t.Fatalf("failed to save session detail: %v", err)
	}

	// 1.5 Verify summary file was created
	summaryPath := filepath.Join(tmpDir, sessionID+".summary.json")
	if _, err := os.Stat(summaryPath); os.IsNotExist(err) {
		t.Errorf("expected summary cache file to exist: %s", summaryPath)
	}

	// GetSessionSummary should work using summary file
	summary, err := repo.GetSessionSummary(context.Background(), dto.SessionProviderCodex, sessionID)
	if err != nil {
		t.Fatalf("failed to get session summary: %v", err)
	}
	if summary.ID != sessionID {
		t.Errorf("expected summary ID '%s', got '%s'", sessionID, summary.ID)
	}

	// 2. Get detail
	got, err := repo.GetSessionDetail(context.Background(), dto.SessionProviderCodex, sessionID)
	if err != nil {
		t.Fatalf("failed to get session detail: %v", err)
	}

	modTime, err := repo.GetSessionDetailModTime(context.Background(), dto.SessionProviderCodex, sessionID)
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
	_, err = repo.GetSessionDetail(context.Background(), dto.SessionProviderCodex, "non-existent")
	if err == nil {
		t.Error("expected error for non-existent session detail, got nil")
	}

	_, err = repo.GetSessionDetailModTime(context.Background(), dto.SessionProviderCodex, "non-existent")
	if err == nil {
		t.Error("expected error for non-existent session detail mod time, got nil")
	}

	// 4. Decode JSON error
	invalidJSONPath := filepath.Join(tmpDir, "invalid-json-detail.json")
	if err := os.WriteFile(invalidJSONPath, []byte("{invalid}"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = repo.GetSessionDetail(context.Background(), dto.SessionProviderCodex, "invalid-json-detail")
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
	err = repo.SaveSessionDetail(context.Background(), dto.SessionProviderCodex, "invalid-marshal-session", invalidDetail)
	if err == nil {
		t.Error("expected marshal error, got nil")
	}

	// 6. WriteFile error
	badRepo := NewCacheFSRepository("/non-existent-dir-12345/sub")
	err = badRepo.SaveSessionDetail(context.Background(), dto.SessionProviderCodex, "any-session", detail)
	if err == nil {
		t.Error("expected write file error, got nil")
	}
}

func TestCacheFSRepository_SummaryPriorityAndFallback(t *testing.T) {
	tmpDir := t.TempDir()
	repo := NewCacheFSRepository(tmpDir)

	sessionID := "test-priority-session"

	// 1. summary.json のみ存在し、detail.json が存在しない場合
	summaryData := dto.SessionSummary{
		ID:     sessionID,
		Parsed: true,
	}
	d, _ := json.Marshal(summaryData)
	_ = os.WriteFile(filepath.Join(tmpDir, sessionID+".summary.json"), d, 0o644)

	summary, err := repo.GetSessionSummary(context.Background(), dto.SessionProviderCodex, sessionID)
	if err != nil {
		t.Fatalf("expected no error with summary.json only, got: %v", err)
	}
	if summary.ID != sessionID || !summary.Parsed {
		t.Errorf("unexpected summary content: %+v", summary)
	}

	// 2. summary.json が壊れていて、detail.json が正常な場合（フォールバック）
	sessionIDFallback := "test-fallback-session"
	_ = os.WriteFile(filepath.Join(tmpDir, sessionIDFallback+".summary.json"), []byte("{broken}"), 0o644)

	detail := &dto.SessionDetailResponse{
		ID:                 sessionIDFallback,
		CacheSchemaVersion: dto.CurrentSessionDetailCacheSchemaVersion,
		Nodes: []dto.FlowNode{
			{
				ID:   "meta-node-1",
				Type: "sessionMeta",
				Data: dto.NodeData{
					Category: "meta",
					Label:    "Meta",
					Meta: map[string]interface{}{
						"cwd": "/fallback/cwd",
					},
				},
			},
		},
	}
	detailData, _ := json.Marshal(detail)
	_ = os.WriteFile(filepath.Join(tmpDir, sessionIDFallback+".json"), detailData, 0o644)

	summaryFallback, err := repo.GetSessionSummary(context.Background(), dto.SessionProviderCodex, sessionIDFallback)
	if err != nil {
		t.Fatalf("expected no error with broken summary.json due to fallback, got: %v", err)
	}
	if summaryFallback.Cwd == nil || *summaryFallback.Cwd != "/fallback/cwd" {
		t.Errorf("expected fallback to detail to successfully load Cwd, got %v", summaryFallback.Cwd)
	}
}

func TestCacheFSRepository_SaveSessionDetail_WithStatistics(t *testing.T) {
	tmpDir := t.TempDir()
	repo := NewCacheFSRepository(tmpDir)

	sessionID := "test-stats-session"
	detail := &dto.SessionDetailResponse{
		ID:                 sessionID,
		CacheSchemaVersion: dto.CurrentSessionDetailCacheSchemaVersion,
		ParsedAt:           "2026-05-30T07:14:40Z",
		Nodes: []dto.FlowNode{
			{
				ID:   "meta-node-1",
				Type: "sessionMeta",
				Data: dto.NodeData{
					Category: "meta",
					Label:    "Meta",
				},
			},
		},
		Statistics: dto.Statistics{
			DurationMs:    120000,
			TotalTokens:   1500,
			ToolCallCount: 5,
			TurnCount:     2,
			Turns: []dto.TurnStatistics{
				{
					Index: 0,
					ConsumedTokens: dto.TokenBreakdown{
						InputTokens:           400,
						OutputTokens:          100,
						ReasoningOutputTokens: 20,
					},
				},
				{
					Index: 1,
					ConsumedTokens: dto.TokenBreakdown{
						InputTokens:           800,
						OutputTokens:          200,
						ReasoningOutputTokens: 50,
					},
				},
			},
		},
	}

	tests := []struct {
		name      string
		sessionID string
		detail    *dto.SessionDetailResponse
		verify    func(t *testing.T, summaryPath string)
	}{
		{
			name:      "should aggregate and save statistics in summary.json",
			sessionID: sessionID,
			detail:    detail,
			verify: func(t *testing.T, summaryPath string) {
				data, err := os.ReadFile(summaryPath)
				if err != nil {
					t.Fatalf("failed to read summary cache: %v", err)
				}

				var summary dto.SessionSummary
				if err := json.Unmarshal(data, &summary); err != nil {
					t.Fatalf("failed to unmarshal summary: %v", err)
				}

				if summary.TotalTokens == nil || *summary.TotalTokens != 1500 {
					t.Errorf("expected TotalTokens 1500, got %v", getVal(summary.TotalTokens))
				}
				if summary.InputTokens == nil || *summary.InputTokens != 1200 {
					t.Errorf("expected InputTokens 1200, got %v", getVal(summary.InputTokens))
				}
				if summary.OutputTokens == nil || *summary.OutputTokens != 300 {
					t.Errorf("expected OutputTokens 300, got %v", getVal(summary.OutputTokens))
				}
				if summary.ReasoningTokens == nil || *summary.ReasoningTokens != 70 {
					t.Errorf("expected ReasoningTokens 70, got %v", getVal(summary.ReasoningTokens))
				}
				if summary.TurnCount == nil || *summary.TurnCount != 2 {
					t.Errorf("expected TurnCount 2, got %v", getValInt(summary.TurnCount))
				}
				if summary.StepCount == nil || *summary.StepCount != 5 {
					t.Errorf("expected StepCount 5, got %v", getValInt(summary.StepCount))
				}
				if summary.DurationMs == nil || *summary.DurationMs != 120000 {
					t.Errorf("expected DurationMs 120000, got %v", getVal(summary.DurationMs))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.SaveSessionDetail(context.Background(), dto.SessionProviderCodex, tt.sessionID, tt.detail)
			if err != nil {
				t.Fatalf("SaveSessionDetail failed: %v", err)
			}
			summaryPath := filepath.Join(tmpDir, tt.sessionID+".summary.json")
			tt.verify(t, summaryPath)
		})
	}
}

func getVal(ptr *int64) string {
	if ptr == nil {
		return "nil"
	}
	return fmt.Sprintf("%d", *ptr)
}

func getValInt(ptr *int) string {
	if ptr == nil {
		return "nil"
	}
	return fmt.Sprintf("%d", *ptr)
}

func TestCacheFSRepository_GetSessionSummary_FallbackMerge(t *testing.T) {
	tmpDir := t.TempDir()
	repo := NewCacheFSRepository(tmpDir)

	sessionID := "fallback-merge-session"

	// 1. 統計情報のない古い summary.json を作成
	oldSummary := dto.SessionSummary{
		ID:         sessionID,
		Cwd:        getStringPtrHelper("test-cwd"),
		CliVersion: getStringPtrHelper("1.0.0"),
		Parsed:     true,
	}
	oldSummaryData, _ := json.Marshal(oldSummary)
	_ = os.WriteFile(filepath.Join(tmpDir, sessionID+".summary.json"), oldSummaryData, 0o644)

	// 2. 統計情報と詳細データを持つ detail.json を作成
	detail := &dto.SessionDetailResponse{
		ID:                 sessionID,
		CacheSchemaVersion: dto.CurrentSessionDetailCacheSchemaVersion,
		Statistics: dto.Statistics{
			DurationMs:    180000,
			TotalTokens:   3000,
			ToolCallCount: 10,
			TurnCount:     4,
			Turns: []dto.TurnStatistics{
				{
					ConsumedTokens: dto.TokenBreakdown{
						InputTokens:           1000,
						OutputTokens:          200,
						ReasoningOutputTokens: 50,
					},
				},
				{
					ConsumedTokens: dto.TokenBreakdown{
						InputTokens:           1500,
						OutputTokens:          300,
						ReasoningOutputTokens: 100,
					},
				},
			},
		},
	}
	detailData, _ := json.Marshal(detail)
	_ = os.WriteFile(filepath.Join(tmpDir, sessionID+".json"), detailData, 0o644)

	// 3. GetSessionSummary を実行して、マージされた結果を検証
	summary, err := repo.GetSessionSummary(context.Background(), dto.SessionProviderCodex, sessionID)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if summary.ID != sessionID {
		t.Errorf("expected ID '%s', got '%s'", sessionID, summary.ID)
	}
	if summary.Cwd == nil || *summary.Cwd != "test-cwd" {
		t.Errorf("expected Cwd 'test-cwd', got '%v'", summary.Cwd)
	}
	if summary.TotalTokens == nil || *summary.TotalTokens != 3000 {
		t.Errorf("expected merged TotalTokens 3000, got %v", getVal(summary.TotalTokens))
	}
	if summary.InputTokens == nil || *summary.InputTokens != 2500 {
		t.Errorf("expected merged InputTokens 2500, got %v", getVal(summary.InputTokens))
	}
	if summary.OutputTokens == nil || *summary.OutputTokens != 500 {
		t.Errorf("expected merged OutputTokens 500, got %v", getVal(summary.OutputTokens))
	}
	if summary.ReasoningTokens == nil || *summary.ReasoningTokens != 150 {
		t.Errorf("expected merged ReasoningTokens 150, got %v", getVal(summary.ReasoningTokens))
	}
	if summary.TurnCount == nil || *summary.TurnCount != 4 {
		t.Errorf("expected merged TurnCount 4, got %v", getValInt(summary.TurnCount))
	}
	if summary.StepCount == nil || *summary.StepCount != 10 {
		t.Errorf("expected merged StepCount 10, got %v", getValInt(summary.StepCount))
	}
	if summary.DurationMs == nil || *summary.DurationMs != 180000 {
		t.Errorf("expected merged DurationMs 180000, got %v", getVal(summary.DurationMs))
	}
}

func TestCacheFSRepository_GetSessionSummary_FallbackNoDetail(t *testing.T) {
	tmpDir := t.TempDir()
	repo := NewCacheFSRepository(tmpDir)

	sessionID := "fallback-nodetail-session"

	// 1. 統計情報のない古い summary.json を作成
	oldSummary := dto.SessionSummary{
		ID:         sessionID,
		Cwd:        getStringPtrHelper("test-cwd"),
		CliVersion: getStringPtrHelper("1.0.0"),
		Parsed:     true,
	}
	oldSummaryData, _ := json.Marshal(oldSummary)
	_ = os.WriteFile(filepath.Join(tmpDir, sessionID+".summary.json"), oldSummaryData, 0o644)

	// 2. detail.json は作成しない

	// 3. GetSessionSummary を実行して、エラーにならず元の情報が取得できることを検証
	summary, err := repo.GetSessionSummary(context.Background(), dto.SessionProviderCodex, sessionID)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if summary.ID != sessionID {
		t.Errorf("expected ID '%s', got '%s'", sessionID, summary.ID)
	}
	if summary.Cwd == nil || *summary.Cwd != "test-cwd" {
		t.Errorf("expected Cwd 'test-cwd', got '%v'", summary.Cwd)
	}
	// 統計情報は nil のままであるべき
	if summary.TotalTokens != nil {
		t.Errorf("expected TotalTokens to be nil, got %d", *summary.TotalTokens)
	}
	if summary.InputTokens != nil {
		t.Errorf("expected InputTokens to be nil, got %d", *summary.InputTokens)
	}
}

func getStringPtrHelper(s string) *string {
	return &s
}
