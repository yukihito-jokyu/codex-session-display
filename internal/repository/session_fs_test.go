package repository

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"codex-session-display/internal/domain/dto"
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
	repo := NewSessionFSRepository("test_dir", cacheRepo)
	if repo.rootDir != "test_dir" {
		t.Errorf("expected rootDir to be 'test_dir', got '%s'", repo.rootDir)
	}
	if repo.cacheRepo != cacheRepo {
		t.Errorf("expected cacheRepo to be %v, got %v", cacheRepo, repo.cacheRepo)
	}
}

func TestSessionFSRepository_ListSessions(t *testing.T) {
	// テストケース用の一時ディレクトリを作成
	tmpDir, err := os.MkdirTemp("", "codex-sessions-list-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	t.Run("directory does not exist", func(t *testing.T) {
		repo := NewSessionFSRepository(filepath.Join(tmpDir, "non-existent"), &mockCacheRepository{})
		sessions, err := repo.ListSessions(context.Background(), 0, 0, "")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(sessions) != 0 {
			t.Errorf("expected 0 sessions, got %d", len(sessions))
		}
	})

	t.Run("empty directory", func(t *testing.T) {
		emptyDir := filepath.Join(tmpDir, "empty")
		if err := os.Mkdir(emptyDir, 0755); err != nil {
			t.Fatal(err)
		}
		repo := NewSessionFSRepository(emptyDir, &mockCacheRepository{})
		sessions, err := repo.ListSessions(context.Background(), 0, 0, "")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(sessions) != 0 {
			t.Errorf("expected 0 sessions, got %d", len(sessions))
		}
	})

	t.Run("successful scan and invalid files skipped", func(t *testing.T) {
		scanDir := filepath.Join(tmpDir, "scan")
		if err := os.Mkdir(scanDir, 0755); err != nil {
			t.Fatal(err)
		}

		// 1. 有効なファイル
		filenameVal := "rollout-2026-05-23T22-44-55-019e5514-ed44-78b2-bf88-233d6e4273bf.jsonl"
		if err := os.WriteFile(filepath.Join(scanDir, filenameVal), []byte("{}"), 0644); err != nil {
			t.Fatal(err)
		}

		// 2. 無効なプレフィックスまたはサフィックス
		filenameInvalidSfx := "rollout-2026-05-23T22-44-55-019e5514-ed44-78b2-bf88-233d6e4273bf.txt"
		if err := os.WriteFile(filepath.Join(scanDir, filenameInvalidSfx), []byte("{}"), 0644); err != nil {
			t.Fatal(err)
		}

		// 3. 無効な名前フォーマット（parseSessionFilename が失敗するファイル名）
		filenameInvalidFmt := "rollout-invalid-format.jsonl"
		if err := os.WriteFile(filepath.Join(scanDir, filenameInvalidFmt), []byte("{}"), 0644); err != nil {
			t.Fatal(err)
		}

		// 4. サブディレクトリ（ファイルではないためスキップされるべき）
		subDir := filepath.Join(scanDir, "rollout-2026-05-23T22-44-55-029e5514-ed44-78b2-bf88-233d6e4273bf.jsonl")
		if err := os.Mkdir(subDir, 0755); err != nil {
			t.Fatal(err)
		}

		repo := NewSessionFSRepository(scanDir, &mockCacheRepository{})
		sessions, err := repo.ListSessions(context.Background(), 0, 0, "")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(sessions) != 1 {
			t.Errorf("expected 1 session, got %d", len(sessions))
		}
		if len(sessions) > 0 {
			if sessions[0].ID != "019e5514-ed44-78b2-bf88-233d6e4273bf" {
				t.Errorf("expected ID '019e5514-ed44-78b2-bf88-233d6e4273bf', got '%s'", sessions[0].ID)
			}
		}
	})

	t.Run("WalkDir permission error in latest month scan", func(t *testing.T) {
		permDir := filepath.Join(tmpDir, "perm_error_latest")
		if err := os.Mkdir(permDir, 0755); err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.RemoveAll(permDir) }()

		nopermDir := filepath.Join(permDir, "noperm")
		if err := os.Mkdir(nopermDir, 0000); err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Chmod(nopermDir, 0755) }()

		repo := NewSessionFSRepository(permDir, &mockCacheRepository{})
		_, err := repo.ListSessions(context.Background(), 0, 0, "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("WalkDir permission error in target month scan", func(t *testing.T) {
		permDir := filepath.Join(tmpDir, "perm_error_target")
		if err := os.Mkdir(permDir, 0755); err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.RemoveAll(permDir) }()

		nopermDir := filepath.Join(permDir, "noperm")
		if err := os.Mkdir(nopermDir, 0000); err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Chmod(nopermDir, 0755) }()

		repo := NewSessionFSRepository(permDir, &mockCacheRepository{})
		_, err := repo.ListSessions(context.Background(), 2026, 5, "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("d.Info() error (deleted file)", func(t *testing.T) {
		infoDir := filepath.Join(tmpDir, "info_error")
		if err := os.Mkdir(infoDir, 0755); err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.RemoveAll(infoDir) }()

		repo := NewSessionFSRepository(infoDir, &mockCacheRepository{})

		for i := 0; i < 150; i++ {
			// 5つのファイルを生成
			for j := 0; j < 5; j++ {
				filePath := filepath.Join(infoDir, fmt.Sprintf("rollout-2026-05-23T22-44-55-019e5514-ed44-78b2-bf88-233d6e4273b%d.jsonl", j))
				_ = os.WriteFile(filePath, []byte("{}"), 0644)
			}

			stop := make(chan struct{})
			go func(d time.Duration) {
				time.Sleep(d)
				// 生成したすべてのファイルを並行して削除する
				for j := 0; j < 5; j++ {
					filePath := filepath.Join(infoDir, fmt.Sprintf("rollout-2026-05-23T22-44-55-019e5514-ed44-78b2-bf88-233d6e4273b%d.jsonl", j))
					_ = os.Remove(filePath)
				}
				close(stop)
			}(time.Duration(i) * 3 * time.Microsecond)

			_, _ = repo.ListSessions(context.Background(), 0, 0, "")
			<-stop
		}
	})

	t.Run("filepath.Rel error", func(t *testing.T) {
		relDir := filepath.Join(tmpDir, "rel_error")
		if err := os.Mkdir(relDir, 0755); err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.RemoveAll(relDir) }()

		filename := "rollout-2026-05-23T22-44-55-019e5514-ed44-78b2-bf88-233d6e4273bf.jsonl"
		filePath := filepath.Join(relDir, filename)
		if err := os.WriteFile(filePath, []byte("{}"), 0644); err != nil {
			t.Fatal(err)
		}

		// filepathRelFn をモック化してエラーを返させる
		orig := filepathRelFn
		defer func() { filepathRelFn = orig }()
		filepathRelFn = func(basepath, targpath string) (string, error) {
			return "", errors.New("mocked rel error")
		}

		repo := NewSessionFSRepository(relDir, &mockCacheRepository{})
		sessions, err := repo.ListSessions(context.Background(), 0, 0, "")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(sessions) != 1 {
			t.Fatalf("expected 1 session, got %d", len(sessions))
		}
		expectedPath := filePath
		if sessions[0].FilePath != expectedPath {
			t.Errorf("expected FilePath to be '%s', got '%s'", expectedPath, sessions[0].FilePath)
		}
	})

	t.Run("filter by year/month (pagination)", func(t *testing.T) {
		filterDir := filepath.Join(tmpDir, "filter_ym")
		if err := os.Mkdir(filterDir, 0755); err != nil {
			t.Fatal(err)
		}

		// 1. 2026年5月のファイル (2026-05-10T12-00-00 -> UTC 2026-05-10)
		fMay := "rollout-2026-05-10T12-00-00-00000000-0000-0000-0000-000000000001.jsonl"
		_ = os.WriteFile(filepath.Join(filterDir, fMay), []byte("{}"), 0644)

		// 2. 2026年6月のファイル (2026-06-25T12-00-00 -> UTC 2026-06-25)
		fJune := "rollout-2026-06-25T12-00-00-00000000-0000-0000-0000-000000000002.jsonl"
		_ = os.WriteFile(filepath.Join(filterDir, fJune), []byte("{}"), 0644)

		repo := NewSessionFSRepository(filterDir, &mockCacheRepository{})

		// 2026年6月を指定して取得
		sessions, err := repo.ListSessions(context.Background(), 2026, 6, "")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// 6月の1件だけが返されるべき
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
	})

	t.Run("filter by query (search limit)", func(t *testing.T) {
		queryDir := filepath.Join(tmpDir, "filter_query")
		if err := os.Mkdir(queryDir, 0755); err != nil {
			t.Fatal(err)
		}

		// ファイル作成
		f1 := "rollout-2026-05-25T12-00-00-00000000-0000-0000-0000-000000000001.jsonl"
		_ = os.WriteFile(filepath.Join(queryDir, f1), []byte("{}"), 0644)
		f2 := "rollout-2026-05-25T12-00-00-00000000-0000-0000-0000-000000000002.jsonl"
		_ = os.WriteFile(filepath.Join(queryDir, f2), []byte("{}"), 0644)
		f3 := "rollout-2026-05-25T12-00-00-00000000-0000-0000-0000-000000000003.jsonl"
		_ = os.WriteFile(filepath.Join(queryDir, f3), []byte("{}"), 0644)

		// モックキャッシュの準備
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

		repo := NewSessionFSRepository(queryDir, cacheRepo)

		// 1. Cwd 部分一致の検索
		sessions, err := repo.ListSessions(context.Background(), 0, 0, "match")
		if err != nil {
			t.Fatal(err)
		}
		if len(sessions) != 1 || sessions[0].ID != "00000000-0000-0000-0000-000000000001" {
			t.Errorf("expected only session 1 to match, got %d results", len(sessions))
		}

		// 2. ブランチ名部分一致の検索 (大文字小文字無視)
		sessions, err = repo.ListSessions(context.Background(), 0, 0, "FEATURE-TEST")
		if err != nil {
			t.Fatal(err)
		}
		if len(sessions) != 1 || sessions[0].ID != "00000000-0000-0000-0000-000000000002" {
			t.Errorf("expected only session 2 to match branch, got %d results", len(sessions))
		}

		// 3. プロバイダー名部分一致の検索 (大文字小文字無視)
		sessions, err = repo.ListSessions(context.Background(), 0, 0, "OPENAI")
		if err != nil {
			t.Fatal(err)
		}
		if len(sessions) != 1 || sessions[0].ID != "00000000-0000-0000-0000-000000000003" {
			t.Errorf("expected only session 3 to match provider, got %d results", len(sessions))
		}
	})

	t.Run("invalid rfc3339 timestamp filter skipped", func(t *testing.T) {
		invalidTSDir := filepath.Join(tmpDir, "invalid_ts")
		if err := os.Mkdir(invalidTSDir, 0755); err != nil {
			t.Fatal(err)
		}

		filename := "rollout-2026-05-23T22-44-55-019e5514-ed44-78b2-bf88-233d6e4273bf.jsonl"
		if err := os.WriteFile(filepath.Join(invalidTSDir, filename), []byte("{}"), 0644); err != nil {
			t.Fatal(err)
		}

		// parseSessionFilenameFn をモック化して不正な形式の timestamp を返させる
		orig := parseSessionFilenameFn
		defer func() { parseSessionFilenameFn = orig }()
		parseSessionFilenameFn = func(filename string) (string, string, error) {
			return "019e5514-ed44-78b2-bf88-233d6e4273bf", "invalid-rfc3339-timestamp", nil
		}

		repo := NewSessionFSRepository(invalidTSDir, &mockCacheRepository{})
		sessions, err := repo.ListSessions(context.Background(), 2026, 5, "")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(sessions) != 0 {
			t.Errorf("expected 0 sessions due to timestamp parse error skip, got %d", len(sessions))
		}
	})
}

func TestSessionFSRepository_ParseSessionFilename(t *testing.T) {
	// 元のローカルタイムゾーンをバックアップ
	origLocal := time.Local
	defer func() { time.Local = origLocal }()

	// time.Local を JST (+09:00) に強制設定
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
