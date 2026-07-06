package repository

import (
	"codex-session-display/internal/domain/dto"
	"codex-session-display/internal/usecase"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type mockCacheRepository struct {
	cache map[string]*dto.SessionSummary
	err   error
}

func (m *mockCacheRepository) GetSessionSummary(ctx context.Context, provider dto.SessionProvider, sessionID string) (*dto.SessionSummary, error) {
	if m.err != nil {
		return nil, m.err
	}
	if s, ok := m.cache[sessionID]; ok {
		return s, nil
	}
	return nil, errors.New("not found")
}

func (m *mockCacheRepository) GetSessionDetail(ctx context.Context, provider dto.SessionProvider, sessionID string) (*dto.SessionDetailResponse, error) {
	return nil, errors.New("not implemented")
}

func (m *mockCacheRepository) GetSessionDetailModTime(ctx context.Context, provider dto.SessionProvider, sessionID string) (time.Time, error) {
	return time.Time{}, errors.New("not implemented")
}

func (m *mockCacheRepository) SaveSessionDetail(ctx context.Context, provider dto.SessionProvider, sessionID string, detail *dto.SessionDetailResponse) error {
	return nil
}

func TestNewSessionFSRepository(t *testing.T) {
	cacheRepo := &mockCacheRepository{}
	tests := []struct {
		name      string
		rootDir   string
		cacheRepo usecase.CacheRepository
	}{
		{
			name:      "success constructor",
			rootDir:   "test_dir",
			cacheRepo: cacheRepo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewSessionFSRepository(tt.rootDir, tt.cacheRepo)
			if repo.rootDir != tt.rootDir {
				t.Errorf("expected rootDir to be '%s', got '%s'", tt.rootDir, repo.rootDir)
			}
			if repo.cacheRepo != tt.cacheRepo {
				t.Errorf("expected cacheRepo to be %v, got %v", tt.cacheRepo, repo.cacheRepo)
			}
		})
	}
}

func TestSessionFSRepository_ListSessions(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "codex-sessions-list-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	tests := []struct {
		name    string
		setup   func(t *testing.T, tmpDir string) (rootDir string, cacheRepo usecase.CacheRepository, cleanup func())
		year    int
		month   int
		query   string
		wantErr bool
		verify  func(t *testing.T, repo *SessionFSRepository, sessions []dto.SessionSummary)
	}{
		{
			name: "directory does not exist",
			setup: func(t *testing.T, tmpDir string) (string, usecase.CacheRepository, func()) {
				return filepath.Join(tmpDir, "non-existent"), &mockCacheRepository{}, nil
			},
			year:    0,
			month:   0,
			query:   "",
			wantErr: false,
			verify: func(t *testing.T, repo *SessionFSRepository, sessions []dto.SessionSummary) {
				if len(sessions) != 0 {
					t.Errorf("expected 0 sessions, got %d", len(sessions))
				}
			},
		},
		{
			name: "empty directory",
			setup: func(t *testing.T, tmpDir string) (string, usecase.CacheRepository, func()) {
				emptyDir := filepath.Join(tmpDir, "empty")
				if err := os.Mkdir(emptyDir, 0o755); err != nil {
					t.Fatal(err)
				}
				return emptyDir, &mockCacheRepository{}, nil
			},
			year:    0,
			month:   0,
			query:   "",
			wantErr: false,
			verify: func(t *testing.T, repo *SessionFSRepository, sessions []dto.SessionSummary) {
				if len(sessions) != 0 {
					t.Errorf("expected 0 sessions, got %d", len(sessions))
				}
			},
		},
		{
			name: "successful scan and invalid files skipped",
			setup: func(t *testing.T, tmpDir string) (string, usecase.CacheRepository, func()) {
				scanDir := filepath.Join(tmpDir, "scan")
				if err := os.Mkdir(scanDir, 0o755); err != nil {
					t.Fatal(err)
				}
				// 1. 有効なファイル
				filenameVal := "rollout-2026-05-23T22-44-55-019e5514-ed44-78b2-bf88-233d6e4273bf.jsonl"
				if err := os.WriteFile(filepath.Join(scanDir, filenameVal), []byte("{}"), 0o644); err != nil {
					t.Fatal(err)
				}
				// 2. 無効なプレフィックスまたはサフィックス
				filenameInvalidSfx := "rollout-2026-05-23T22-44-55-019e5514-ed44-78b2-bf88-233d6e4273bf.txt"
				if err := os.WriteFile(filepath.Join(scanDir, filenameInvalidSfx), []byte("{}"), 0o644); err != nil {
					t.Fatal(err)
				}
				// 3. 無効な名前フォーマット（parseSessionFilename が失敗するファイル名）
				filenameInvalidFmt := "rollout-invalid-format.jsonl"
				if err := os.WriteFile(filepath.Join(scanDir, filenameInvalidFmt), []byte("{}"), 0o644); err != nil {
					t.Fatal(err)
				}
				// 4. サブディレクトリ（ファイルではないためスキップされるべき）
				subDir := filepath.Join(scanDir, "rollout-2026-05-23T22-44-55-029e5514-ed44-78b2-bf88-233d6e4273bf.jsonl")
				if err := os.Mkdir(subDir, 0o755); err != nil {
					t.Fatal(err)
				}
				return scanDir, &mockCacheRepository{}, nil
			},
			year:    0,
			month:   0,
			query:   "",
			wantErr: false,
			verify: func(t *testing.T, repo *SessionFSRepository, sessions []dto.SessionSummary) {
				if len(sessions) != 1 {
					t.Errorf("expected 1 session, got %d", len(sessions))
				}
				if len(sessions) > 0 {
					if sessions[0].ID != "019e5514-ed44-78b2-bf88-233d6e4273bf" {
						t.Errorf("expected ID '019e5514-ed44-78b2-bf88-233d6e4273bf', got '%s'", sessions[0].ID)
					}
				}
			},
		},
		{
			name: "WalkDir permission error in latest month scan",
			setup: func(t *testing.T, tmpDir string) (string, usecase.CacheRepository, func()) {
				permDir := filepath.Join(tmpDir, "perm_error_latest")
				if err := os.Mkdir(permDir, 0o755); err != nil {
					t.Fatal(err)
				}
				nopermDir := filepath.Join(permDir, "noperm")
				if err := os.Mkdir(nopermDir, 0o000); err != nil {
					t.Fatal(err)
				}
				cleanup := func() {
					_ = os.Chmod(nopermDir, 0o755)
					_ = os.RemoveAll(permDir)
				}
				return permDir, &mockCacheRepository{}, cleanup
			},
			year:    0,
			month:   0,
			query:   "",
			wantErr: true,
		},
		{
			name: "WalkDir permission error in target month scan",
			setup: func(t *testing.T, tmpDir string) (string, usecase.CacheRepository, func()) {
				permDir := filepath.Join(tmpDir, "perm_error_target")
				if err := os.Mkdir(permDir, 0o755); err != nil {
					t.Fatal(err)
				}
				nopermDir := filepath.Join(permDir, "noperm")
				if err := os.Mkdir(nopermDir, 0o000); err != nil {
					t.Fatal(err)
				}
				cleanup := func() {
					_ = os.Chmod(nopermDir, 0o755)
					_ = os.RemoveAll(permDir)
				}
				return permDir, &mockCacheRepository{}, cleanup
			},
			year:    2026,
			month:   5,
			query:   "",
			wantErr: true,
		},
		{
			name: "d.Info() error (deleted file)",
			setup: func(t *testing.T, tmpDir string) (string, usecase.CacheRepository, func()) {
				infoDir := filepath.Join(tmpDir, "info_error")
				if err := os.Mkdir(infoDir, 0o755); err != nil {
					t.Fatal(err)
				}
				cleanup := func() {
					_ = os.RemoveAll(infoDir)
				}
				return infoDir, &mockCacheRepository{}, cleanup
			},
			year:    0,
			month:   0,
			query:   "",
			wantErr: false,
			verify: func(t *testing.T, repo *SessionFSRepository, sessions []dto.SessionSummary) {
				for i := 0; i < 150; i++ {
					// 5つのファイルを生成
					for j := 0; j < 5; j++ {
						filePath := filepath.Join(repo.rootDir, fmt.Sprintf("rollout-2026-05-23T22-44-55-019e5514-ed44-78b2-bf88-233d6e4273b%d.jsonl", j))
						_ = os.WriteFile(filePath, []byte("{}"), 0o644)
					}

					stop := make(chan struct{})
					go func(d time.Duration) {
						time.Sleep(d)
						// 生成したすべてのファイルを並行して削除する
						for j := 0; j < 5; j++ {
							filePath := filepath.Join(repo.rootDir, fmt.Sprintf("rollout-2026-05-23T22-44-55-019e5514-ed44-78b2-bf88-233d6e4273b%d.jsonl", j))
							_ = os.Remove(filePath)
						}
						close(stop)
					}(time.Duration(i) * 3 * time.Microsecond)

					_, _ = repo.ListSessions(context.Background(), dto.SessionProviderCodex, 0, 0, "")
					<-stop
				}
			},
		},
		{
			name: "filepath.Rel error",
			setup: func(t *testing.T, tmpDir string) (string, usecase.CacheRepository, func()) {
				relDir := filepath.Join(tmpDir, "rel_error")
				if err := os.Mkdir(relDir, 0o755); err != nil {
					t.Fatal(err)
				}
				filename := "rollout-2026-05-23T22-44-55-019e5514-ed44-78b2-bf88-233d6e4273bf.jsonl"
				filePath := filepath.Join(relDir, filename)
				if err := os.WriteFile(filePath, []byte("{}"), 0o644); err != nil {
					t.Fatal(err)
				}

				orig := filepathRelFn
				filepathRelFn = func(basepath, targpath string) (string, error) {
					return "", errors.New("mocked rel error")
				}
				cleanup := func() {
					filepathRelFn = orig
					_ = os.RemoveAll(relDir)
				}
				return relDir, &mockCacheRepository{}, cleanup
			},
			year:    0,
			month:   0,
			query:   "",
			wantErr: false,
			verify: func(t *testing.T, repo *SessionFSRepository, sessions []dto.SessionSummary) {
				if len(sessions) != 1 {
					t.Fatalf("expected 1 session, got %d", len(sessions))
				}
				expectedPath := filepath.Join(repo.rootDir, "rollout-2026-05-23T22-44-55-019e5514-ed44-78b2-bf88-233d6e4273bf.jsonl")
				if sessions[0].FilePath != expectedPath {
					t.Errorf("expected FilePath to be '%s', got '%s'", expectedPath, sessions[0].FilePath)
				}
			},
		},
		{
			name: "filter by year/month (pagination)",
			setup: func(t *testing.T, tmpDir string) (string, usecase.CacheRepository, func()) {
				filterDir := filepath.Join(tmpDir, "filter_ym")
				if err := os.Mkdir(filterDir, 0o755); err != nil {
					t.Fatal(err)
				}
				fMay := "rollout-2026-05-10T12-00-00-00000000-0000-0000-0000-000000000001.jsonl"
				_ = os.WriteFile(filepath.Join(filterDir, fMay), []byte("{}"), 0o644)

				fJune := "rollout-2026-06-25T12-00-00-00000000-0000-0000-0000-000000000002.jsonl"
				_ = os.WriteFile(filepath.Join(filterDir, fJune), []byte("{}"), 0o644)

				cleanup := func() {
					_ = os.RemoveAll(filterDir)
				}
				return filterDir, &mockCacheRepository{}, cleanup
			},
			year:    2026,
			month:   6,
			query:   "",
			wantErr: false,
			verify: func(t *testing.T, repo *SessionFSRepository, sessions []dto.SessionSummary) {
				if len(sessions) != 1 {
					t.Fatalf("expected 1 session, got %d", len(sessions))
				}
				if sessions[0].ID != "00000000-0000-0000-0000-000000000002" {
					t.Errorf("expected session ID to be 00000000-0000-0000-0000-000000000002, got %s", sessions[0].ID)
				}

				// 年月を 0, 0 にした場合は最新月 (2026年6月) が自動検出されて1件返ってくるべき
				sessionsAuto, err := repo.ListSessions(context.Background(), dto.SessionProviderCodex, 0, 0, "")
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if len(sessionsAuto) != 1 || sessionsAuto[0].ID != "00000000-0000-0000-0000-000000000002" {
					t.Errorf("expected auto-detected latest month (6月) to return 1 session, got %d", len(sessionsAuto))
				}
			},
		},
		{
			name: "filter by query (search limit)",
			setup: func(t *testing.T, tmpDir string) (string, usecase.CacheRepository, func()) {
				queryDir := filepath.Join(tmpDir, "filter_query")
				if err := os.Mkdir(queryDir, 0o755); err != nil {
					t.Fatal(err)
				}

				f1 := "rollout-2026-05-25T12-00-00-00000000-0000-0000-0000-000000000001.jsonl"
				_ = os.WriteFile(filepath.Join(queryDir, f1), []byte("{}"), 0o644)
				f2 := "rollout-2026-05-25T12-00-00-00000000-0000-0000-0000-000000000002.jsonl"
				_ = os.WriteFile(filepath.Join(queryDir, f2), []byte("{}"), 0o644)
				f3 := "rollout-2026-05-25T12-00-00-00000000-0000-0000-0000-000000000003.jsonl"
				_ = os.WriteFile(filepath.Join(queryDir, f3), []byte("{}"), 0o644)

				cwdMatch := "/users/match"
				cwdNoMatch := "/users/other-path"
				branchMatch := "feature-test"
				providerMatch := "openai"

				cacheRepo := &mockCacheRepository{
					cache: map[string]*dto.SessionSummary{
						"00000000-0000-0000-0000-000000000001": {
							ID:  "00000000-0000-0000-0000-000000000001",
							Cwd: &cwdMatch,
						},
						"00000000-0000-0000-0000-000000000002": {
							ID:     "00000000-0000-0000-0000-000000000002",
							Cwd:    &cwdNoMatch,
							Branch: &branchMatch,
						},
						"00000000-0000-0000-0000-000000000003": {
							ID:            "00000000-0000-0000-0000-000000000003",
							Cwd:           &cwdNoMatch,
							ModelProvider: &providerMatch,
						},
					},
				}
				cleanup := func() {
					_ = os.RemoveAll(queryDir)
				}
				return queryDir, cacheRepo, cleanup
			},
			year:    0,
			month:   0,
			query:   "match",
			wantErr: false,
			verify: func(t *testing.T, repo *SessionFSRepository, sessions []dto.SessionSummary) {
				if len(sessions) != 1 || sessions[0].ID != "00000000-0000-0000-0000-000000000001" {
					t.Errorf("expected only session 1 to match, got %d results", len(sessions))
				}

				// 2. ブランチ名部分一致の検索 (大文字小文字無視)
				sessions2, err := repo.ListSessions(context.Background(), dto.SessionProviderCodex, 0, 0, "FEATURE-TEST")
				if err != nil {
					t.Fatal(err)
				}
				if len(sessions2) != 1 || sessions2[0].ID != "00000000-0000-0000-0000-000000000002" {
					t.Errorf("expected only session 2 to match branch, got %d results", len(sessions2))
				}

				// 3. プロバイダー名部分一致の検索 (大文字小文字無視)
				sessions3, err := repo.ListSessions(context.Background(), dto.SessionProviderCodex, 0, 0, "OPENAI")
				if err != nil {
					t.Fatal(err)
				}
				if len(sessions3) != 1 || sessions3[0].ID != "00000000-0000-0000-0000-000000000003" {
					t.Errorf("expected only session 3 to match provider, got %d results", len(sessions3))
				}
			},
		},
		{
			name: "invalid rfc3339 timestamp filter skipped",
			setup: func(t *testing.T, tmpDir string) (string, usecase.CacheRepository, func()) {
				invalidTSDir := filepath.Join(tmpDir, "invalid_ts")
				if err := os.Mkdir(invalidTSDir, 0o755); err != nil {
					t.Fatal(err)
				}
				filename := "rollout-2026-05-23T22-44-55-019e5514-ed44-78b2-bf88-233d6e4273bf.jsonl"
				if err := os.WriteFile(filepath.Join(invalidTSDir, filename), []byte("{}"), 0o644); err != nil {
					t.Fatal(err)
				}

				orig := parseSessionFilenameFn
				parseSessionFilenameFn = func(filename string) (string, string, error) {
					return "019e5514-ed44-78b2-bf88-233d6e4273bf", "invalid-rfc3339-timestamp", nil
				}
				cleanup := func() {
					parseSessionFilenameFn = orig
					_ = os.RemoveAll(invalidTSDir)
				}
				return invalidTSDir, &mockCacheRepository{}, cleanup
			},
			year:    2026,
			month:   5,
			query:   "",
			wantErr: false,
			verify: func(t *testing.T, repo *SessionFSRepository, sessions []dto.SessionSummary) {
				if len(sessions) != 0 {
					t.Errorf("expected 0 sessions due to timestamp parse error skip, got %d", len(sessions))
				}
			},
		},
		{
			name: "resolve parent-child relationships from raw session files",
			setup: func(t *testing.T, tmpDir string) (string, usecase.CacheRepository, func()) {
				rawDir := filepath.Join(tmpDir, "raw_parent_child")
				if err := os.Mkdir(rawDir, 0o755); err != nil {
					t.Fatal(err)
				}
				// 親セッションファイル (ID: 00000000-0000-0000-0000-000000000001)
				pFilename := "rollout-2026-05-25T12-00-00-00000000-0000-0000-0000-000000000001.jsonl"
				pContent := `{"type":"session_meta", "payload":{"id":"00000000-0000-0000-0000-000000000001", "cli_version":"v0.131.0"}}` + "\n"
				if err := os.WriteFile(filepath.Join(rawDir, pFilename), []byte(pContent), 0o644); err != nil {
					t.Fatal(err)
				}

				// 子セッションファイル (ID: 00000000-0000-0000-0000-000000000002)
				cFilename := "rollout-2026-05-25T12-00-00-00000000-0000-0000-0000-000000000002.jsonl"
				cContent := `{"type":"session_meta", "payload":{"id":"00000000-0000-0000-0000-000000000002", "parent_thread_id":"00000000-0000-0000-0000-000000000001", "cli_version":"v0.131.0"}}` + "\n"
				if err := os.WriteFile(filepath.Join(rawDir, cFilename), []byte(cContent), 0o644); err != nil {
					t.Fatal(err)
				}

				cleanup := func() {
					_ = os.RemoveAll(rawDir)
				}
				return rawDir, &mockCacheRepository{}, cleanup
			},
			year:    2026,
			month:   5,
			query:   "",
			wantErr: false,
			verify: func(t *testing.T, repo *SessionFSRepository, sessions []dto.SessionSummary) {
				if len(sessions) != 2 {
					t.Fatalf("expected 2 sessions, got %d", len(sessions))
				}

				var parent, child *dto.SessionSummary
				for i := range sessions {
					switch sessions[i].ID {
					case "00000000-0000-0000-0000-000000000001":
						parent = &sessions[i]
					case "00000000-0000-0000-0000-000000000002":
						child = &sessions[i]
					}
				}

				if parent == nil {
					t.Fatal("parent session not found")
				}
				if child == nil {
					t.Fatal("child session not found")
				}

				if child.ParentSessionID == nil || *child.ParentSessionID != "00000000-0000-0000-0000-000000000001" {
					t.Errorf("expected child's ParentSessionID to be '00000000-0000-0000-0000-000000000001', got %v", child.ParentSessionID)
				}

				if len(parent.ChildSessionIDs) != 1 || parent.ChildSessionIDs[0] != "00000000-0000-0000-0000-000000000002" {
					t.Errorf("expected parent's ChildSessionIDs to contain '00000000-0000-0000-0000-000000000002', got %v", parent.ChildSessionIDs)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootDir, cacheRepo, cleanup := tt.setup(t, tmpDir)
			if cleanup != nil {
				defer cleanup()
			}

			repo := NewSessionFSRepository(rootDir, cacheRepo)
			sessions, err := repo.ListSessions(context.Background(), dto.SessionProviderCodex, tt.year, tt.month, tt.query)
			if (err != nil) != tt.wantErr {
				t.Fatalf("expected error: %v, got: %v", tt.wantErr, err)
			}

			if tt.verify != nil {
				tt.verify(t, repo, sessions)
			}
		})
	}
}

func TestSessionFSRepository_ParseSessionFilename(t *testing.T) {
	origLocal := time.Local
	defer func() { time.Local = origLocal }()

	time.Local = time.FixedZone("JST", 9*60*60)

	tests := []struct {
		name     string
		filename string
		wantID   string
		wantTime string
		wantErr  bool
	}{
		{
			name:     "valid JST filename",
			filename: "rollout-2026-05-23T22-44-55-019e5514-ed44-78b2-bf88-233d6e4273bf.jsonl",
			wantID:   "019e5514-ed44-78b2-bf88-233d6e4273bf",
			wantTime: "2026-05-23T13:44:55Z",
			wantErr:  false,
		},
		{
			name:     "invalid prefix",
			filename: "invalid-2026-05-23T22-44-55-019e5514-ed44-78b2-bf88-233d6e4273bf.jsonl",
			wantErr:  true,
		},
		{
			name:     "invalid suffix",
			filename: "rollout-2026-05-23T22-44-55-019e5514-ed44-78b2-bf88-233d6e4273bf.txt",
			wantErr:  true,
		},
		{
			name:     "too short filename",
			filename: "rollout-2026.jsonl",
			wantErr:  true,
		},
		{
			name:     "missing T in timestamp",
			filename: "rollout-2026-05-23-22-44-55-019e5514-ed44-78b2-bf88-233d6e4273bf.jsonl",
			wantErr:  true,
		},
		{
			name:     "invalid time format (non-existent date)",
			filename: "rollout-2026-05-99T22-44-55-019e5514-ed44-78b2-bf88-233d6e4273bf.jsonl",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, timestamp, err := parseSessionFilename(tt.filename)
			if (err != nil) != tt.wantErr {
				t.Fatalf("expected error: %v, got: %v", tt.wantErr, err)
			}
			if !tt.wantErr {
				if id != tt.wantID {
					t.Errorf("expected ID '%s', got '%s'", tt.wantID, id)
				}
				if timestamp != tt.wantTime {
					t.Errorf("expected timestamp '%s', got '%s'", tt.wantTime, timestamp)
				}
			}
		})
	}
}

func TestSessionFSRepository_GetSessionFilePath(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test directories and files
	sessionID := "019e5514-ed44-78b2-bf88-233d6e4273bf"
	filename := "rollout-2026-05-23T22-44-55-" + sessionID + ".jsonl"
	subDir := filepath.Join(tmpDir, "2026", "05", "23")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}

	filePath := filepath.Join(subDir, filename)
	if err := os.WriteFile(filePath, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	repo := NewSessionFSRepository(tmpDir, &mockCacheRepository{})

	tests := []struct {
		name      string
		sessionID string
		wantPath  string
		wantErr   bool
	}{
		{
			name:      "success",
			sessionID: sessionID,
			wantPath:  filePath,
			wantErr:   false,
		},
		{
			name:      "not found",
			sessionID: "non-existent-session-id",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := repo.GetSessionFilePath(context.Background(), tt.sessionID)
			if (err != nil) != tt.wantErr {
				t.Fatalf("expected error: %v, got %v", tt.wantErr, err)
			}
			if !tt.wantErr && path != tt.wantPath {
				t.Errorf("expected path '%s', got '%s'", tt.wantPath, path)
			}
		})
	}

	t.Run("claude transcript found by embedded session id", func(t *testing.T) {
		projectsDir := filepath.Join(tmpDir, "claude-projects")
		projectDir := filepath.Join(projectsDir, "-Users-test-project")
		if err := os.MkdirAll(projectDir, 0o755); err != nil {
			t.Fatal(err)
		}

		claudeSessionID := "claude-session-1"
		claudePath := filepath.Join(projectDir, "transcript-file-name.jsonl")
		transcript := `{"type":"user","sessionId":"claude-session-1","timestamp":"2026-06-02T10:00:00.000Z","message":{"content":"hello"}}`
		if err := os.WriteFile(claudePath, []byte(transcript), 0o644); err != nil {
			t.Fatal(err)
		}

		repo.SetClaudeProjectsDir(projectsDir)
		path, err := repo.GetSessionFilePath(context.Background(), claudeSessionID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if path != claudePath {
			t.Fatalf("expected path %q, got %q", claudePath, path)
		}
	})

	t.Run("permission error", func(t *testing.T) {
		permDir := filepath.Join(tmpDir, "perm_error")
		if err := os.Mkdir(permDir, 0o755); err != nil {
			t.Fatal(err)
		}
		nopermDir := filepath.Join(permDir, "noperm")
		if err := os.Mkdir(nopermDir, 0o000); err != nil {
			t.Fatal(err)
		}
		defer func() {
			_ = os.Chmod(nopermDir, 0o755)
			_ = os.RemoveAll(permDir)
		}()

		repoErr := NewSessionFSRepository(permDir, &mockCacheRepository{})
		_, err := repoErr.GetSessionFilePath(context.Background(), "any-id")
		if err == nil {
			t.Error("expected error due to permission issues, got nil")
		}
	})

	t.Run("non-matching file skipped", func(t *testing.T) {
		skipDir := filepath.Join(tmpDir, "skip_dir")
		if err := os.Mkdir(skipDir, 0o755); err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.RemoveAll(skipDir) }()

		// Create non-matching files
		if err := os.WriteFile(filepath.Join(skipDir, "rollout-2026.txt"), []byte("{}"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(skipDir, "other.jsonl"), []byte("{}"), 0o644); err != nil {
			t.Fatal(err)
		}

		repoSkip := NewSessionFSRepository(skipDir, &mockCacheRepository{})
		_, err := repoSkip.GetSessionFilePath(context.Background(), "any-id")
		if err == nil {
			t.Error("expected session not found error, got nil")
		}
	})
}

func TestSessionFSRepository_GetSessionIDByFilePath(t *testing.T) {
	tmpDir := t.TempDir()

	sessionID := "019e5514-ed44-78b2-bf88-233d6e4273bf"
	filename := "rollout-2026-05-23T22-44-55-" + sessionID + ".jsonl"
	subDir := filepath.Join(tmpDir, "2026", "05", "23")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}

	filePath := filepath.Join(subDir, filename)
	if err := os.WriteFile(filePath, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	repo := NewSessionFSRepository(tmpDir, &mockCacheRepository{})

	tests := []struct {
		name     string
		filePath string
		wantID   string
		wantErr  bool
	}{
		{
			name:     "success with exact path",
			filePath: filePath,
			wantID:   sessionID,
		},
		{
			name:     "success with cleaned path",
			filePath: filepath.Join(subDir, ".", filename),
			wantID:   sessionID,
		},
		{
			name:     "not found",
			filePath: filepath.Join(tmpDir, "2026", "05", "23", "missing.jsonl"),
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, err := repo.GetSessionIDByFilePath(context.Background(), tt.filePath)
			if (err != nil) != tt.wantErr {
				t.Fatalf("expected error: %v, got %v", tt.wantErr, err)
			}
			if !tt.wantErr && gotID != tt.wantID {
				t.Fatalf("expected session ID %q, got %q", tt.wantID, gotID)
			}
		})
	}
}

func TestSessionFSRepository_GetSessionModTime(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test directories and files
	sessionID := "019e5514-ed44-78b2-bf88-233d6e4273bf"
	filename := "rollout-2026-05-23T22-44-55-" + sessionID + ".jsonl"
	subDir := filepath.Join(tmpDir, "2026", "05", "23")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}

	filePath := filepath.Join(subDir, filename)
	if err := os.WriteFile(filePath, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	repo := NewSessionFSRepository(tmpDir, &mockCacheRepository{})

	// Get expected modification time
	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatal(err)
	}
	wantTime := info.ModTime()

	tests := []struct {
		name      string
		sessionID string
		wantTime  time.Time
		wantErr   bool
	}{
		{
			name:      "success",
			sessionID: sessionID,
			wantTime:  wantTime,
			wantErr:   false,
		},
		{
			name:      "not found",
			sessionID: "non-existent-session-id",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTime, err := repo.GetSessionModTime(context.Background(), tt.sessionID)
			if (err != nil) != tt.wantErr {
				t.Fatalf("expected error: %v, got %v", tt.wantErr, err)
			}
			if !tt.wantErr && !gotTime.Equal(tt.wantTime) {
				t.Errorf("expected modification time '%v', got '%v'", tt.wantTime, gotTime)
			}
		})
	}

	t.Run("stat error with symlink", func(t *testing.T) {
		symlinkDir := filepath.Join(tmpDir, "symlink_dir")
		if err := os.MkdirAll(symlinkDir, 0o755); err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.RemoveAll(symlinkDir) }()

		symlinkID := "11111111-2222-3333-4444-555555555555"
		symlinkName := "rollout-2026-05-23T22-44-55-" + symlinkID + ".jsonl"
		symlinkPath := filepath.Join(symlinkDir, symlinkName)

		// Create a symlink pointing to a non-existent file
		if err := os.Symlink(filepath.Join(symlinkDir, "non-existent-target.jsonl"), symlinkPath); err != nil {
			t.Fatal(err)
		}

		repoSym := NewSessionFSRepository(symlinkDir, &mockCacheRepository{})
		_, err := repoSym.GetSessionModTime(context.Background(), symlinkID)
		if err == nil {
			t.Error("expected error statting a broken symlink, got nil")
		}
	})
}

func TestSessionFSRepository_ListSessions_ClaudeProvider(t *testing.T) {
	tmpDir := t.TempDir()
	projectsDir := filepath.Join(tmpDir, "projects")
	encodedProject := "-Users-test-project"
	projectDir := filepath.Join(projectsDir, encodedProject)
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}

	sessionID := "claude-session-1"
	transcript := strings.Join([]string{
		`{"type":"user","sessionId":"claude-session-1","cwd":"/Users/test/project","timestamp":"2026-06-02T10:00:00.000Z","message":{"content":"hello"}}`,
		`{"type":"assistant","sessionId":"claude-session-1","cwd":"/Users/test/project","timestamp":"2026-06-02T10:00:03.000Z","costUSD":0.0123,"message":{"content":[{"type":"text","text":"hi"},{"type":"tool_use","name":"Read"}]}}`,
		`{"type":"assistant","sessionId":"claude-session-1","cwd":"/Users/test/project","timestamp":"2026-06-02T10:00:05.000Z","costUSD":0.0200,"message":{"content":[{"type":"tool_use","name":"Bash"}]}}`,
	}, "\n")
	if err := os.WriteFile(filepath.Join(projectDir, sessionID+".jsonl"), []byte(transcript), 0o644); err != nil {
		t.Fatal(err)
	}

	repo := NewSessionFSRepository(filepath.Join(tmpDir, "codex"), &mockCacheRepository{})
	repo.claudeProjectsDir = projectsDir

	sessions, err := repo.ListSessions(context.Background(), dto.SessionProviderClaude, 2026, 6, "project")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}

	got := sessions[0]
	if got.Provider != dto.SessionProviderClaude {
		t.Fatalf("expected claude provider, got %q", got.Provider)
	}
	if got.ID != sessionID {
		t.Fatalf("expected session ID %q, got %q", sessionID, got.ID)
	}
	if got.EncodedProject == nil || *got.EncodedProject != encodedProject {
		t.Fatalf("expected encoded project %q, got %v", encodedProject, got.EncodedProject)
	}
	if got.Cwd == nil || *got.Cwd != "/Users/test/project" {
		t.Fatalf("expected cwd, got %v", got.Cwd)
	}
	if got.Timestamp == nil || *got.Timestamp != "2026-06-02T10:00:00.000Z" {
		t.Fatalf("expected first timestamp, got %v", got.Timestamp)
	}
	if got.MessageCount == nil || *got.MessageCount != 3 {
		t.Fatalf("expected message count 3, got %v", got.MessageCount)
	}
	if got.ToolCallCount == nil || *got.ToolCallCount != 2 {
		t.Fatalf("expected tool call count 2, got %v", got.ToolCallCount)
	}
	if got.TotalCostUSD == nil || *got.TotalCostUSD != 0.0323 {
		t.Fatalf("expected total cost 0.0323, got %v", got.TotalCostUSD)
	}
}

func TestSessionFSRepository_ListSessions_ClaudeSubagent(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	projectsDir := filepath.Join(tmpDir, "claude-projects")
	projectDir := filepath.Join(projectsDir, "my-project")
	parentSessionID := "parent-session-1"
	subagentSessionID := "subagent-session-1"

	// Create directories
	parentDir := filepath.Join(projectDir, parentSessionID)
	subagentsDir := filepath.Join(parentDir, "subagents")
	if err := os.MkdirAll(subagentsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write parent transcript
	parentTranscript := strings.Join([]string{
		`{"type":"user","sessionId":"parent-session-1","cwd":"/Users/test/project","timestamp":"2026-06-02T10:00:00.000Z","message":{"content":"hello"}}`,
	}, "\n")
	if err := os.WriteFile(filepath.Join(projectDir, parentSessionID+".jsonl"), []byte(parentTranscript), 0o644); err != nil {
		t.Fatal(err)
	}

	// Write subagent transcript
	subagentTranscript := strings.Join([]string{
		`{"type":"user","sessionId":"subagent-session-1","cwd":"/Users/test/project","timestamp":"2026-06-02T10:01:00.000Z","message":{"content":"sub task"}}`,
	}, "\n")
	if err := os.WriteFile(filepath.Join(subagentsDir, subagentSessionID+".jsonl"), []byte(subagentTranscript), 0o644); err != nil {
		t.Fatal(err)
	}

	repo := NewSessionFSRepository(filepath.Join(tmpDir, "codex"), &mockCacheRepository{})
	repo.claudeProjectsDir = projectsDir

	sessions, err := repo.ListSessions(context.Background(), dto.SessionProviderClaude, 2026, 6, "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// We expect both parent and subagent sessions to be present in the returned sessions list
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}

	var parent, subagent *dto.SessionSummary
	for i := range sessions {
		switch sessions[i].ID {
		case parentSessionID:
			parent = &sessions[i]
		case subagentSessionID:
			subagent = &sessions[i]
		}
	}

	if parent == nil {
		t.Fatal("parent session not found")
	}
	if subagent == nil {
		t.Fatal("subagent session not found")
	}

	// Verify Parent-Child relations are resolved
	if parent.ParentSessionID != nil {
		t.Errorf("parent should not have a parent ID, got %v", *parent.ParentSessionID)
	}
	if len(parent.ChildSessionIDs) != 1 || parent.ChildSessionIDs[0] != subagentSessionID {
		t.Errorf("expected parent to have child session ID %q, got %v", subagentSessionID, parent.ChildSessionIDs)
	}

	if subagent.ParentSessionID == nil || *subagent.ParentSessionID != parentSessionID {
		t.Errorf("expected subagent to point to parent session ID %q, got %v", parentSessionID, subagent.ParentSessionID)
	}
}

func TestSessionFSRepository_ListSessions_ClaudeCacheHitWithMismatchID(t *testing.T) {
	tmpDir := t.TempDir()
	projectsDir := filepath.Join(tmpDir, "projects")
	projectDir := filepath.Join(projectsDir, "-Users-test-project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// ファイル名と内部の sessionId が異なるファイルを作成
	filename := "mismatch-filename.jsonl"
	internalSessionID := "claude-internal-session-1"
	transcript := `{"type":"user","sessionId":"` + internalSessionID + `","cwd":"/Users/test/project","timestamp":"2026-06-02T10:00:00.000Z","message":{"content":"hello"}}`
	if err := os.WriteFile(filepath.Join(projectDir, filename), []byte(transcript), 0o644); err != nil {
		t.Fatal(err)
	}

	// 内部IDである "claude-internal-session-1" をキーとするキャッシュを準備
	cachedSummary := &dto.SessionSummary{
		ID:        internalSessionID,
		Provider:  dto.SessionProviderClaude,
		Parsed:    true,
		Cwd:       func(s string) *string { return &s }("/Users/test/project"),
		Timestamp: func(s string) *string { return &s }("2026-06-02T10:00:00.000Z"),
	}

	mockCache := &mockCacheRepository{
		cache: map[string]*dto.SessionSummary{
			internalSessionID: cachedSummary,
		},
	}

	repo := NewSessionFSRepository(filepath.Join(tmpDir, "codex"), mockCache)
	repo.claudeProjectsDir = projectsDir

	sessions, err := repo.ListSessions(context.Background(), dto.SessionProviderClaude, 2026, 6, "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}

	got := sessions[0]
	if got.ID != internalSessionID {
		t.Errorf("expected session ID %q, got %q", internalSessionID, got.ID)
	}
	if !got.Parsed {
		t.Errorf("expected session to be parsed (cache hit), but got parsed = false")
	}
}
