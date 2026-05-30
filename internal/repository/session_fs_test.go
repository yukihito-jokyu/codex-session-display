package repository

import (
	"codex-session-display/internal/domain/dto"
	"codex-session-display/internal/usecase"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type mockCacheRepository struct {
	cache map[string]*dto.SessionSummary
	err   error
}

func (m *mockCacheRepository) GetSessionSummary(ctx context.Context, sessionID string) (*dto.SessionSummary, error) {
	if m.err != nil {
		return nil, m.err
	}
	if s, ok := m.cache[sessionID]; ok {
		return s, nil
	}
	return nil, errors.New("not found")
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

					_, _ = repo.ListSessions(context.Background(), 0, 0, "")
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
				sessionsAuto, err := repo.ListSessions(context.Background(), 0, 0, "")
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
				sessions2, err := repo.ListSessions(context.Background(), 0, 0, "FEATURE-TEST")
				if err != nil {
					t.Fatal(err)
				}
				if len(sessions2) != 1 || sessions2[0].ID != "00000000-0000-0000-0000-000000000002" {
					t.Errorf("expected only session 2 to match branch, got %d results", len(sessions2))
				}

				// 3. プロバイダー名部分一致の検索 (大文字小文字無視)
				sessions3, err := repo.ListSessions(context.Background(), 0, 0, "OPENAI")
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootDir, cacheRepo, cleanup := tt.setup(t, tmpDir)
			if cleanup != nil {
				defer cleanup()
			}

			repo := NewSessionFSRepository(rootDir, cacheRepo)
			sessions, err := repo.ListSessions(context.Background(), tt.year, tt.month, tt.query)
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
