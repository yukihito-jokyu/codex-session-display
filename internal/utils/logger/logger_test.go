package logger

import (
	"bytes"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetLogDir(t *testing.T) {
	oldHomeFn := userHomeDirFn
	defer func() { userHomeDirFn = oldHomeFn }()

	tempDir := t.TempDir()
	userHomeDirFn = func() (string, error) {
		return tempDir, nil
	}

	dir, err := GetLogDir()
	if err != nil {
		t.Fatalf("GetLogDir failed: %v", err)
	}
	expected := filepath.Join(tempDir, ".codex-display", "logs")
	if dir != expected {
		t.Errorf("expected %s, got %s", expected, dir)
	}
}

func TestGetLogFilePath(t *testing.T) {
	oldHomeFn := userHomeDirFn
	defer func() { userHomeDirFn = oldHomeFn }()

	tempDir := t.TempDir()
	userHomeDirFn = func() (string, error) {
		return tempDir, nil
	}

	path, err := GetLogFilePath()
	if err != nil {
		t.Fatalf("GetLogFilePath failed: %v", err)
	}
	expected := filepath.Join(tempDir, ".codex-display", "logs", "app.log")
	if path != expected {
		t.Errorf("expected %s, got %s", expected, path)
	}
}

// setupTestLogger はテスト用にバッファに出力するロガーをセットアップし、
// テスト終了時にデフォルトロガーを復元する cleanup 関数とバッファを返します。
func setupTestLogger(t *testing.T, level slog.Level, addSource bool) (*bytes.Buffer, func()) {
	t.Helper()
	var buf bytes.Buffer
	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: addSource,
	}
	handler := slog.NewTextHandler(&buf, opts)
	oldDefault := slog.Default()
	slog.SetDefault(slog.New(handler))
	return &buf, func() { slog.SetDefault(oldDefault) }
}

func TestLoggerInfoOutput(t *testing.T) {
	buf, cleanup := setupTestLogger(t, slog.LevelInfo, false)
	defer cleanup()

	Info("test info message", "key1", "value1")

	output := buf.String()
	if !strings.Contains(output, "test info message") {
		t.Errorf("expected output to contain 'test info message', got: %s", output)
	}
	if !strings.Contains(output, "key1=value1") {
		t.Errorf("expected output to contain 'key1=value1', got: %s", output)
	}
}

func TestLoggerDebugOutput(t *testing.T) {
	buf, cleanup := setupTestLogger(t, slog.LevelDebug, false)
	defer cleanup()

	Debug("debug detail", "step", 42)

	output := buf.String()
	if !strings.Contains(output, "debug detail") {
		t.Errorf("expected output to contain 'debug detail', got: %s", output)
	}
	if !strings.Contains(output, "step=42") {
		t.Errorf("expected output to contain 'step=42', got: %s", output)
	}
}

func TestLoggerWarnOutput(t *testing.T) {
	buf, cleanup := setupTestLogger(t, slog.LevelWarn, false)
	defer cleanup()

	Warn("something suspicious", "file_path", "/tmp/test.jsonl")

	output := buf.String()
	if !strings.Contains(output, "something suspicious") {
		t.Errorf("expected output to contain 'something suspicious', got: %s", output)
	}
}

func TestLoggerErrorOutput(t *testing.T) {
	buf, cleanup := setupTestLogger(t, slog.LevelError, false)
	defer cleanup()

	Error("critical failure", "error", "disk full")

	output := buf.String()
	if !strings.Contains(output, "critical failure") {
		t.Errorf("expected output to contain 'critical failure', got: %s", output)
	}
	if !strings.Contains(output, "error=\"disk full\"") {
		t.Errorf("expected output to contain 'error=\"disk full\"', got: %s", output)
	}
}

func TestLoggerDebugSuppressedAtInfoLevel(t *testing.T) {
	buf, cleanup := setupTestLogger(t, slog.LevelInfo, false)
	defer cleanup()

	Debug("should not appear")

	output := buf.String()
	if output != "" {
		t.Errorf("expected no output for DEBUG at INFO level, got: %s", output)
	}
}

func TestLoggerSourceAttribution(t *testing.T) {
	buf, cleanup := setupTestLogger(t, slog.LevelInfo, true)
	defer cleanup()

	Info("source check")

	output := buf.String()
	// ソース情報が logger.go ではなく、このテストファイル（logger_test.go）を指すことを確認
	if !strings.Contains(output, "logger_test.go") {
		t.Errorf("expected source to point to logger_test.go, got: %s", output)
	}
	if strings.Contains(output, "source=") && strings.Contains(output, "logger.go") && !strings.Contains(output, "logger_test.go") {
		t.Errorf("source incorrectly points to logger.go wrapper instead of caller, got: %s", output)
	}
}

type testStruct struct {
	Name  string
	Count int
}

func TestLoggerStructOutput(t *testing.T) {
	buf, cleanup := setupTestLogger(t, slog.LevelInfo, false)
	defer cleanup()

	s := testStruct{Name: "hello", Count: 42}
	Info("struct test", "data", s)

	output := buf.String()
	t.Logf("struct output: %s", output)

	// slog の TextHandler は構造体を {Name:hello Count:42} のように出力する
	if !strings.Contains(output, "struct test") {
		t.Errorf("expected output to contain 'struct test', got: %s", output)
	}
	if !strings.Contains(output, "hello") {
		t.Errorf("expected output to contain struct field value 'hello', got: %s", output)
	}
	if !strings.Contains(output, "42") {
		t.Errorf("expected output to contain struct field value '42', got: %s", output)
	}
}
