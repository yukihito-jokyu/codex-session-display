package usecase_test

import (
	"archive/zip"
	"bytes"
	"codex-session-display/internal/usecase"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
)

// mockEmitter は AppEventsEmitter のモックです。
type mockEmitter struct {
	mu     sync.Mutex
	events []capturedEvent
}

type capturedEvent struct {
	name string
	data []interface{}
}

func (m *mockEmitter) Emit(eventName string, optionalData ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, capturedEvent{name: eventName, data: optionalData})
}

func (m *mockEmitter) getEvents() []capturedEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]capturedEvent(nil), m.events...)
}

func createTestZip() ([]byte, error) {
	var zipBuf bytes.Buffer
	zw := zip.NewWriter(&zipBuf)

	files := map[string]string{
		"codex-session-display.app/Contents/Info.plist": "dummy-plist-content",
		"codex-session-display.app/Contents/MacOS/bin":  "dummy-binary",
	}

	for name, content := range files {
		f, err := zw.Create(name)
		if err != nil {
			return nil, err
		}
		_, err = f.Write([]byte(content))
		if err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return zipBuf.Bytes(), nil
}

func TestApplyUpdateUseCase_Execute_DownloadProgress(t *testing.T) {
	zipData, err := createTestZip()
	if err != nil {
		t.Fatalf("Failed to create test zip: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(zipData)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(zipData)
	}))
	defer server.Close()

	tempDir := t.TempDir()
	emitter := &mockEmitter{}
	uc := usecase.NewApplyUpdateUseCase(emitter, tempDir)

	// テストランナーのos.Exit終了を防止するモック
	uc.CurrentAppPath = filepath.Join(tempDir, "dummy-current.app")
	if err := os.MkdirAll(uc.CurrentAppPath, 0o755); err != nil {
		t.Fatalf("Failed to create mock current app dir: %v", err)
	}
	uc.CmdStartFunc = func(cmd *exec.Cmd) error {
		return nil
	}
	uc.ExitFunc = func(code int) {}

	err = uc.Execute(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	events := emitter.getEvents()
	if len(events) == 0 {
		t.Fatal("Expected progress events, but got none")
	}

	// 最後のイベントの前のイベントが progress 100 であるか、全体のいずれかに進捗が含まれていることを確認
	foundDownloadComplete := false
	for _, event := range events {
		if event.name == "update-progress" && len(event.data) > 0 {
			progressMap, ok := event.data[0].(map[string]interface{})
			if ok && progressMap["status"] == "download_complete" && progressMap["progress"] == 100.0 {
				foundDownloadComplete = true
				break
			}
		}
	}

	if !foundDownloadComplete {
		t.Error("Expected to find update-progress event with status=download_complete and progress=100.0")
	}
}

func TestApplyUpdateUseCase_Execute_ExtractAndApply(t *testing.T) {
	zipData, err := createTestZip()
	if err != nil {
		t.Fatalf("Failed to create test zip: %v", err)
	}

	// 2. モックサーバーで zip データを配信する
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(zipData)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(zipData)
	}))
	defer server.Close()

	// 3. テスト用のダミーの「現在のアプリ」の配置
	tempDir := t.TempDir()
	currentAppPath := filepath.Join(tempDir, "current-app.app")
	if err := os.MkdirAll(currentAppPath, 0o755); err != nil {
		t.Fatalf("Failed to create mock current app dir: %v", err)
	}

	emitter := &mockEmitter{}
	uc := usecase.NewApplyUpdateUseCase(emitter, tempDir)

	// テスト用のフィールド設定
	uc.CurrentAppPath = currentAppPath

	var capturedCmd *exec.Cmd
	uc.CmdStartFunc = func(cmd *exec.Cmd) error {
		capturedCmd = cmd
		return nil // 実際には起動しない
	}

	exitCalled := false
	var exitCode int
	uc.ExitFunc = func(code int) {
		exitCalled = true
		exitCode = code
	}

	// 4. 実行
	err = uc.Execute(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	// 5. アサーション
	if capturedCmd == nil {
		t.Fatal("Expected cmdStartFunc to be called, but it was not")
	}

	if !exitCalled {
		t.Fatal("Expected exitFunc to be called, but it was not")
	}

	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", exitCode)
	}

	// スクリプトファイルが作成されたか確認
	scriptPath := filepath.Join(tempDir, "update.sh")
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		t.Errorf("Expected update.sh script to be created, but it does not exist")
	}

	// 解凍された一時ファイルの中に .app があるか確認
	extractedAppPath := filepath.Join(tempDir, "extracted", "codex-session-display.app")
	if _, err := os.Stat(extractedAppPath); os.IsNotExist(err) {
		t.Errorf("Expected extracted app to exist, but it does not")
	}
}
