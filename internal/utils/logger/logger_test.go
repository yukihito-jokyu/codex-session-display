package logger

import (
	"bytes"
	"errors"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetLogDir(t *testing.T) {
	oldHomeFn := userHomeDirFn
	defer func() { userHomeDirFn = oldHomeFn }()

	tempDir := t.TempDir()

	tests := []struct {
		name    string
		homeFn  func() (string, error)
		want    string
		wantErr bool
	}{
		{
			name: "success",
			homeFn: func() (string, error) {
				return tempDir, nil
			},
			want:    filepath.Join(tempDir, ".codex-display", "logs"),
			wantErr: false,
		},
		{
			name: "error retrieving home directory",
			homeFn: func() (string, error) {
				return "", errors.New("home error")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userHomeDirFn = tt.homeFn
			dir, err := GetLogDir()
			if (err != nil) != tt.wantErr {
				t.Fatalf("expected error: %v, got: %v", tt.wantErr, err)
			}
			if !tt.wantErr && dir != tt.want {
				t.Errorf("expected %s, got %s", tt.want, dir)
			}
		})
	}
}

func TestGetLogFilePath(t *testing.T) {
	oldHomeFn := userHomeDirFn
	defer func() { userHomeDirFn = oldHomeFn }()

	tempDir := t.TempDir()

	tests := []struct {
		name    string
		homeFn  func() (string, error)
		want    string
		wantErr bool
	}{
		{
			name: "success",
			homeFn: func() (string, error) {
				return tempDir, nil
			},
			want:    filepath.Join(tempDir, ".codex-display", "logs", "app.log"),
			wantErr: false,
		},
		{
			name: "error retrieving home directory",
			homeFn: func() (string, error) {
				return "", errors.New("home error")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userHomeDirFn = tt.homeFn
			path, err := GetLogFilePath()
			if (err != nil) != tt.wantErr {
				t.Fatalf("expected error: %v, got: %v", tt.wantErr, err)
			}
			if !tt.wantErr && path != tt.want {
				t.Errorf("expected %s, got %s", tt.want, path)
			}
		})
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

type testStruct struct {
	Name  string
	Count int
}

func TestLoggerOutputs(t *testing.T) {
	tests := []struct {
		name      string
		logLevel  slog.Level
		addSource bool
		logFunc   func()
		verify    func(t *testing.T, output string)
	}{
		{
			name:      "info output",
			logLevel:  slog.LevelInfo,
			addSource: false,
			logFunc: func() {
				Info("test info message", "key1", "value1")
			},
			verify: func(t *testing.T, output string) {
				if !strings.Contains(output, "test info message") {
					t.Errorf("expected output to contain 'test info message', got: %s", output)
				}
				if !strings.Contains(output, "key1=value1") {
					t.Errorf("expected output to contain 'key1=value1', got: %s", output)
				}
			},
		},
		{
			name:      "debug output",
			logLevel:  slog.LevelDebug,
			addSource: false,
			logFunc: func() {
				Debug("debug detail", "step", 42)
			},
			verify: func(t *testing.T, output string) {
				if !strings.Contains(output, "debug detail") {
					t.Errorf("expected output to contain 'debug detail', got: %s", output)
				}
				if !strings.Contains(output, "step=42") {
					t.Errorf("expected output to contain 'step=42', got: %s", output)
				}
			},
		},
		{
			name:      "warn output",
			logLevel:  slog.LevelWarn,
			addSource: false,
			logFunc: func() {
				Warn("something suspicious", "file_path", "/tmp/test.jsonl")
			},
			verify: func(t *testing.T, output string) {
				if !strings.Contains(output, "something suspicious") {
					t.Errorf("expected output to contain 'something suspicious', got: %s", output)
				}
			},
		},
		{
			name:      "error output",
			logLevel:  slog.LevelError,
			addSource: false,
			logFunc: func() {
				Error("critical failure", "error", "disk full")
			},
			verify: func(t *testing.T, output string) {
				if !strings.Contains(output, "critical failure") {
					t.Errorf("expected output to contain 'critical failure', got: %s", output)
				}
				if !strings.Contains(output, "error=\"disk full\"") {
					t.Errorf("expected output to contain 'error=\"disk full\"', got: %s", output)
				}
			},
		},
		{
			name:      "debug suppressed at info level",
			logLevel:  slog.LevelInfo,
			addSource: false,
			logFunc: func() {
				Debug("should not appear")
			},
			verify: func(t *testing.T, output string) {
				if output != "" {
					t.Errorf("expected no output for DEBUG at INFO level, got: %s", output)
				}
			},
		},
		{
			name:      "source attribution",
			logLevel:  slog.LevelInfo,
			addSource: true,
			logFunc: func() {
				Info("source check")
			},
			verify: func(t *testing.T, output string) {
				if !strings.Contains(output, "logger_test.go") {
					t.Errorf("expected source to point to logger_test.go, got: %s", output)
				}
				if strings.Contains(output, "source=") && strings.Contains(output, "logger.go") && !strings.Contains(output, "logger_test.go") {
					t.Errorf("source incorrectly points to logger.go wrapper instead of caller, got: %s", output)
				}
			},
		},
		{
			name:      "struct output",
			logLevel:  slog.LevelInfo,
			addSource: false,
			logFunc: func() {
				s := testStruct{Name: "hello", Count: 42}
				Info("struct test", "data", s)
			},
			verify: func(t *testing.T, output string) {
				if !strings.Contains(output, "struct test") {
					t.Errorf("expected output to contain 'struct test', got: %s", output)
				}
				if !strings.Contains(output, "hello") {
					t.Errorf("expected output to contain struct field value 'hello', got: %s", output)
				}
				if !strings.Contains(output, "42") {
					t.Errorf("expected output to contain struct field value '42', got: %s", output)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf, cleanup := setupTestLogger(t, tt.logLevel, tt.addSource)
			defer cleanup()

			tt.logFunc()

			output := buf.String()
			tt.verify(t, output)
		})
	}
}
