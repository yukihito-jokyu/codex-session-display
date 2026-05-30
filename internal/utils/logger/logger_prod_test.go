//go:build production

package logger

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitLogger(t *testing.T) {
	tests := []struct {
		name   string
		setup  func(t *testing.T, tempDir string) (homeFn func() (string, error), cleanup func())
		verify func(t *testing.T, tempDir string)
	}{
		{
			name: "success initialization",
			setup: func(t *testing.T, tempDir string) (func() (string, error), func()) {
				return func() (string, error) {
					return tempDir, nil
				}, nil
			},
		},
		{
			name: "rotation",
			setup: func(t *testing.T, tempDir string) (func() (string, error), func()) {
				logDir := filepath.Join(tempDir, ".codex-display", "logs")
				if err := os.MkdirAll(logDir, 0o755); err != nil {
					t.Fatalf("failed to create log dir: %v", err)
				}

				logPath := filepath.Join(logDir, "app.log")
				logPath1 := logPath + ".1"

				// 1. 10MB を超えるダミーログファイルを作成
				dummyData := make([]byte, maxLogSize+100)
				if err := os.WriteFile(logPath, dummyData, 0o644); err != nil {
					t.Fatalf("failed to write dummy log: %v", err)
				}

				// 2. 退避先（古い世代）ファイルもあらかじめ作成
				if err := os.WriteFile(logPath1, []byte("old log"), 0o644); err != nil {
					t.Fatalf("failed to write dummy log.1: %v", err)
				}

				homeFn := func() (string, error) {
					return tempDir, nil
				}
				return homeFn, nil
			},
			verify: func(t *testing.T, tempDir string) {
				logDir := filepath.Join(tempDir, ".codex-display", "logs")
				logPath := filepath.Join(logDir, "app.log")
				logPath1 := logPath + ".1"

				infoNew, err := os.Stat(logPath)
				if err != nil {
					t.Fatalf("new app.log not found: %v", err)
				}

				if infoNew.Size() >= int64(maxLogSize) {
					t.Errorf("expected rotation to reduce app.log size, but size is %d", infoNew.Size())
				}

				info1, err := os.Stat(logPath1)
				if err != nil {
					t.Fatalf("expected rotated log.1 to exist: %v", err)
				}
				if info1.Size() != int64(maxLogSize+100) {
					t.Errorf("rotated file size mismatch: expected %d, got %d", maxLogSize+100, info1.Size())
				}
			},
		},
		{
			name: "get log dir error",
			setup: func(t *testing.T, tempDir string) (func() (string, error), func()) {
				return func() (string, error) {
					return "", os.ErrNotExist
				}, nil
			},
		},
		{
			name: "mkdir error",
			setup: func(t *testing.T, tempDir string) (func() (string, error), func()) {
				// ディレクトリを作成したい場所にファイルを置いて、MkdirAllを失敗させる
				logDir := filepath.Join(tempDir, ".codex-display", "logs")
				if err := os.MkdirAll(filepath.Dir(logDir), 0o755); err != nil {
					t.Fatalf("failed to prepare parent dir: %v", err)
				}
				if err := os.WriteFile(logDir, []byte("file instead of dir"), 0o644); err != nil {
					t.Fatalf("failed to write blocker file: %v", err)
				}
				homeFn := func() (string, error) {
					return tempDir, nil
				}
				return homeFn, nil
			},
		},
		{
			name: "open file error",
			setup: func(t *testing.T, tempDir string) (func() (string, error), func()) {
				logDir := filepath.Join(tempDir, ".codex-display", "logs")
				if err := os.MkdirAll(logDir, 0o755); err != nil {
					t.Fatalf("failed to create log dir: %v", err)
				}
				logPath := filepath.Join(logDir, "app.log")
				// app.logをディレクトリとして作成して、OpenFileを失敗させる
				if err := os.MkdirAll(logPath, 0o755); err != nil {
					t.Fatalf("failed to create app.log as directory: %v", err)
				}
				homeFn := func() (string, error) {
					return tempDir, nil
				}
				return homeFn, nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldHomeFn := userHomeDirFn
			defer func() { userHomeDirFn = oldHomeFn }()

			tempDir := t.TempDir()
			homeFn, cleanup := tt.setup(t, tempDir)
			if cleanup != nil {
				defer cleanup()
			}
			userHomeDirFn = homeFn

			initCleanup := InitLogger()
			if initCleanup == nil {
				t.Fatal("InitLogger returned nil cleanup")
			}
			initCleanup()

			if tt.verify != nil {
				tt.verify(t, tempDir)
			}
		})
	}
}
