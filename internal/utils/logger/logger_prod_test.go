//go:build production

package logger

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitLogger(t *testing.T) {
	oldHomeFn := userHomeDirFn
	defer func() { userHomeDirFn = oldHomeFn }()

	tempDir := t.TempDir()
	userHomeDirFn = func() (string, error) {
		return tempDir, nil
	}

	cleanup := InitLogger()
	if cleanup == nil {
		t.Fatal("InitLogger returned nil cleanup")
	}
	cleanup()
}

func TestInitLogger_Rotation(t *testing.T) {
	oldHomeFn := userHomeDirFn
	defer func() { userHomeDirFn = oldHomeFn }()

	tempDir := t.TempDir()
	userHomeDirFn = func() (string, error) {
		return tempDir, nil
	}

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

	// 3. ロガー初期化実行
	cleanup := InitLogger()
	cleanup()

	// 4. ローテーションされたか（本番タグ）を判定
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
}

func TestInitLogger_GetLogDirError(t *testing.T) {
	oldHomeFn := userHomeDirFn
	defer func() { userHomeDirFn = oldHomeFn }()

	userHomeDirFn = func() (string, error) {
		return "", os.ErrNotExist
	}

	cleanup := InitLogger()
	cleanup()
}

func TestInitLogger_MkdirError(t *testing.T) {
	oldHomeFn := userHomeDirFn
	defer func() { userHomeDirFn = oldHomeFn }()

	tempDir := t.TempDir()
	userHomeDirFn = func() (string, error) {
		return tempDir, nil
	}

	// ディレクトリを作成したい場所にファイルを置いて、MkdirAllを失敗させる
	logDir := filepath.Join(tempDir, ".codex-display", "logs")
	if err := os.MkdirAll(filepath.Dir(logDir), 0o755); err != nil {
		t.Fatalf("failed to prepare parent dir: %v", err)
	}
	if err := os.WriteFile(logDir, []byte("file instead of dir"), 0o644); err != nil {
		t.Fatalf("failed to write blocker file: %v", err)
	}

	cleanup := InitLogger()
	cleanup()
}

func TestInitLogger_OpenFileError(t *testing.T) {
	oldHomeFn := userHomeDirFn
	defer func() { userHomeDirFn = oldHomeFn }()

	tempDir := t.TempDir()
	userHomeDirFn = func() (string, error) {
		return tempDir, nil
	}

	logDir := filepath.Join(tempDir, ".codex-display", "logs")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		t.Fatalf("failed to create log dir: %v", err)
	}
	logPath := filepath.Join(logDir, "app.log")
	// app.logをディレクトリとして作成して、OpenFileを失敗させる
	if err := os.MkdirAll(logPath, 0o755); err != nil {
		t.Fatalf("failed to create app.log as directory: %v", err)
	}

	cleanup := InitLogger()
	cleanup()
}

