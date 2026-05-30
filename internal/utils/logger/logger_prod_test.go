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
	if err := os.MkdirAll(logDir, 0755); err != nil {
		t.Fatalf("failed to create log dir: %v", err)
	}

	logPath := filepath.Join(logDir, "app.log")
	logPath1 := logPath + ".1"

	// 1. 10MB を超えるダミーログファイルを作成
	dummyData := make([]byte, maxLogSize+100)
	if err := os.WriteFile(logPath, dummyData, 0644); err != nil {
		t.Fatalf("failed to write dummy log: %v", err)
	}

	// 2. 退避先（古い世代）ファイルもあらかじめ作成
	if err := os.WriteFile(logPath1, []byte("old log"), 0644); err != nil {
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
