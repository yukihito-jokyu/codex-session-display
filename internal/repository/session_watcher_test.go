package repository

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestSessionWatcher(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tmpDir, err := os.MkdirTemp("", "session_watcher_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 初回検知されないようにするための既存ファイルを作成
	initialFile := filepath.Join(tmpDir, "rollout-2026-06-23T12-00-00-12345678-1234-1234-1234-1234567890ab.jsonl")
	if err := os.WriteFile(initialFile, []byte("initial contents"), 0o644); err != nil {
		t.Fatalf("failed to write initial file: %v", err)
	}

	var mu sync.Mutex
	changedFiles := make([]string, 0)
	onChanged := func(filePath string) {
		mu.Lock()
		defer mu.Unlock()
		changedFiles = append(changedFiles, filePath)
	}

	// 10ms周期でスキャン
	watcher := NewSessionWatcher(tmpDir, 10*time.Millisecond, onChanged)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	watcher.Start(ctx)

	// 起動直後に少し待ち、初期ファイルによるイベントが発生しないことを確認
	time.Sleep(30 * time.Millisecond)
	mu.Lock()
	if len(changedFiles) != 0 {
		t.Errorf("expected 0 events initially, got %d", len(changedFiles))
	}
	mu.Unlock()

	// 1. 新規ファイル追加を検証（アトミックに配置してポーリングの競合を防ぐ）
	newFileTemp := filepath.Join(tmpDir, "temp-rollout.jsonl")
	if err := os.WriteFile(newFileTemp, []byte("new file contents"), 0o644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	newFile := filepath.Join(tmpDir, "rollout-2026-06-23T12-00-05-87654321-4321-4321-4321-8765432109ba.jsonl")
	if err := os.Rename(newFileTemp, newFile); err != nil {
		t.Fatalf("failed to rename file: %v", err)
	}

	// 変更検知を待つ (ポーリング間隔10msなので 30ms 程度で十分)
	time.Sleep(30 * time.Millisecond)

	mu.Lock()
	if len(changedFiles) != 1 {
		t.Errorf("expected 1 event after creation, got %d", len(changedFiles))
	} else if changedFiles[0] != newFile {
		t.Errorf("expected event for %s, got %s", newFile, changedFiles[0])
	}
	changedFiles = changedFiles[:0] // クリア
	mu.Unlock()

	// 2. 既存ファイルの変更（サイズ変更・時間変更）を検証
	// macOSなどでは同じミリ秒内の更新だと ModTime が変わらないことがあるため、少しスリープしてサイズを変更
	time.Sleep(20 * time.Millisecond)
	if err := os.WriteFile(newFile, []byte("new file contents updated"), 0o644); err != nil {
		t.Fatalf("failed to update file: %v", err)
	}

	time.Sleep(30 * time.Millisecond)

	mu.Lock()
	if len(changedFiles) != 1 {
		t.Errorf("expected 1 event after modification, got %d", len(changedFiles))
	} else if changedFiles[0] != newFile {
		t.Errorf("expected event for %s, got %s", newFile, changedFiles[0])
	}
	changedFiles = changedFiles[:0] // クリア
	mu.Unlock()

	// 3. 非対象ファイル（rollout-以外）が無視されることを検証
	ignoredFile := filepath.Join(tmpDir, "other-file.txt")
	if err := os.WriteFile(ignoredFile, []byte("ignored"), 0o644); err != nil {
		t.Fatalf("failed to write ignored file: %v", err)
	}

	time.Sleep(30 * time.Millisecond)

	mu.Lock()
	if len(changedFiles) != 0 {
		t.Errorf("expected 0 events for non-rollout file, got %d", len(changedFiles))
	}
	mu.Unlock()

	// 4. Stop による停止を検証
	watcher.Stop()

	// 停止後の新規追加が検知されないことを検証
	afterStopFile := filepath.Join(tmpDir, "rollout-2026-06-23T12-00-10-11111111-2222-3333-4444-555555555555.jsonl")
	if err := os.WriteFile(afterStopFile, []byte("after stop contents"), 0o644); err != nil {
		t.Fatalf("failed to write file after stop: %v", err)
	}

	time.Sleep(30 * time.Millisecond)

	mu.Lock()
	if len(changedFiles) != 0 {
		t.Errorf("expected 0 events after Stop(), got %d", len(changedFiles))
	}
	mu.Unlock()
}
