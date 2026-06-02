package main

import (
	"codex-session-display/internal/domain/dto"
	"codex-session-display/internal/repository"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestApp_OpenLogDirectory(t *testing.T) {
	originalHome := os.Getenv("HOME")
	originalExecCommand := execCommand
	tempHome := t.TempDir()

	if err := os.Setenv("HOME", tempHome); err != nil {
		t.Fatalf("failed to set HOME: %v", err)
	}
	defer func() {
		_ = os.Setenv("HOME", originalHome)
		execCommand = originalExecCommand
	}()

	tests := []struct {
		name          string
		commandRunner func(name string, arg ...string) *exec.Cmd
		wantErr       bool
		verify        func(t *testing.T, err error, commandName string, commandArgs []string)
	}{
		{
			name: "success",
			commandRunner: func(name string, arg ...string) *exec.Cmd {
				return exec.Command("true")
			},
			verify: func(t *testing.T, err error, commandName string, commandArgs []string) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}

				expectedCommand := "xdg-open"
				switch runtime.GOOS {
				case "darwin":
					expectedCommand = "open"
				case "windows":
					expectedCommand = "explorer"
				}

				if commandName != expectedCommand {
					t.Fatalf("expected command %q, got %q", expectedCommand, commandName)
				}

				expectedLogDir := filepath.Join(tempHome, ".codex-display", "logs")
				if len(commandArgs) != 1 || commandArgs[0] != expectedLogDir {
					t.Fatalf("expected command args [%q], got %v", expectedLogDir, commandArgs)
				}

				if info, statErr := os.Stat(expectedLogDir); statErr != nil || !info.IsDir() {
					t.Fatalf("expected log directory to exist, statErr=%v info=%v", statErr, info)
				}
			},
		},
		{
			name: "command start failure returns app error",
			commandRunner: func(name string, arg ...string) *exec.Cmd {
				return exec.Command("command-that-does-not-exist-for-open-log-dir-test")
			},
			wantErr: true,
			verify: func(t *testing.T, err error, commandName string, commandArgs []string) {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				appErr, ok := err.(*dto.AppError)
				if !ok {
					t.Fatalf("expected *dto.AppError, got %T", err)
				}
				if appErr.Code != "INTERNAL_ERROR" {
					t.Fatalf("expected INTERNAL_ERROR, got %s", appErr.Code)
				}
				if appErr.Message != "ログディレクトリを開くのに失敗しました" {
					t.Fatalf("unexpected message: %s", appErr.Message)
				}
				if commandName == "" || len(commandArgs) != 1 {
					t.Fatalf("expected command capture, got name=%q args=%v", commandName, commandArgs)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var commandName string
			var commandArgs []string
			execCommand = func(name string, arg ...string) *exec.Cmd {
				commandName = name
				commandArgs = append([]string(nil), arg...)
				return tt.commandRunner(name, arg...)
			}

			app := &App{}
			err := app.OpenLogDirectory()
			if (err != nil) != tt.wantErr {
				t.Fatalf("expected wantErr=%v, got err=%v", tt.wantErr, err)
			}
			tt.verify(t, err, commandName, commandArgs)
		})
	}
}

func TestApp_GetLogFilePath(t *testing.T) {
	originalHome := os.Getenv("HOME")
	tempHome := t.TempDir()
	if err := os.Setenv("HOME", tempHome); err != nil {
		t.Fatalf("failed to set HOME: %v", err)
	}
	defer func() {
		_ = os.Setenv("HOME", originalHome)
	}()

	app := &App{}
	got, err := app.GetLogFilePath()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expected := filepath.Join(tempHome, ".codex-display", "logs", "app.log")
	if got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}

func TestApp_ResolveSessionIDFromPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		app         *App
		path        string
		wantID      string
		wantErr     bool
		wantErrCode string
	}{
		{
			name: "returns session id from repository",
			app: &App{
				sessionRepo: &stubSessionRepository{
					sessionIDByPath: map[string]string{
						"/tmp/session.jsonl": "session-123",
					},
				},
			},
			path:   "/tmp/session.jsonl",
			wantID: "session-123",
		},
		{
			name: "returns session not found app error",
			app: &App{
				sessionRepo: &stubSessionRepository{
					errByPath: map[string]error{
						"/tmp/missing.jsonl": repository.ErrSessionNotFound,
					},
				},
			},
			path:        "/tmp/missing.jsonl",
			wantErr:     true,
			wantErrCode: "SESSION_NOT_FOUND",
		},
		{
			name: "returns internal error for repository scan failure",
			app: &App{
				sessionRepo: &stubSessionRepository{
					errByPath: map[string]error{
						"/tmp/broken.jsonl": os.ErrPermission,
					},
				},
			},
			path:        "/tmp/broken.jsonl",
			wantErr:     true,
			wantErrCode: "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.app.ResolveSessionIDFromPath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Fatalf("expected error=%v, got %v", tt.wantErr, err)
			}
			if tt.wantErr {
				appErr, ok := err.(*dto.AppError)
				if !ok {
					t.Fatalf("expected *dto.AppError, got %T", err)
				}
				if appErr.Code != tt.wantErrCode {
					t.Fatalf("expected error code %q, got %q", tt.wantErrCode, appErr.Code)
				}
				return
			}
			if !tt.wantErr && got != tt.wantID {
				t.Fatalf("expected %q, got %q", tt.wantID, got)
			}
		})
	}
}

func TestApp_HandleOpenSessionFile_FlushesAfterFrontendReady(t *testing.T) {
	t.Parallel()

	originalWindowShow := windowShow
	originalEventsEmit := eventsEmit
	defer func() {
		windowShow = originalWindowShow
		eventsEmit = originalEventsEmit
	}()

	var shownCount int
	var emittedPaths []string
	windowShow = func(ctx context.Context) {
		shownCount++
	}
	eventsEmit = func(ctx context.Context, eventName string, optionalData ...interface{}) {
		if eventName != "open-session-file" {
			t.Fatalf("unexpected event name %q", eventName)
		}
		if len(optionalData) != 1 {
			t.Fatalf("expected one payload, got %d", len(optionalData))
		}
		filePath, ok := optionalData[0].(string)
		if !ok {
			t.Fatalf("expected string payload, got %T", optionalData[0])
		}
		emittedPaths = append(emittedPaths, filePath)
	}

	app := &App{}

	app.HandleOpenSessionFile("/tmp/before-startup.jsonl")
	app.startup(context.Background())
	app.HandleOpenSessionFile("/tmp/before-ready.jsonl")

	if shownCount != 0 {
		t.Fatalf("expected no window show before frontend ready, got %d", shownCount)
	}
	if len(emittedPaths) != 0 {
		t.Fatalf("expected no emitted paths before frontend ready, got %v", emittedPaths)
	}

	app.FrontendReady()

	if shownCount != 2 {
		t.Fatalf("expected 2 window show calls after frontend ready, got %d", shownCount)
	}
	expected := []string{"/tmp/before-startup.jsonl", "/tmp/before-ready.jsonl"}
	for i, want := range expected {
		if emittedPaths[i] != want {
			t.Fatalf("expected emitted path %q at index %d, got %q", want, i, emittedPaths[i])
		}
	}
	if len(app.pendingSessionFile) != 0 {
		t.Fatalf("expected pending queue to be empty, got %v", app.pendingSessionFile)
	}

	app.HandleOpenSessionFile("/tmp/after-ready.jsonl")

	if shownCount != 3 {
		t.Fatalf("expected immediate window show after frontend ready, got %d", shownCount)
	}
	if len(emittedPaths) != 3 || emittedPaths[2] != "/tmp/after-ready.jsonl" {
		t.Fatalf("expected immediate emit after frontend ready, got %v", emittedPaths)
	}
}

type stubSessionRepository struct {
	sessionIDByPath map[string]string
	errByPath       map[string]error
}

func (s *stubSessionRepository) ListSessions(ctx context.Context, year, month int, query string) ([]dto.SessionSummary, error) {
	return nil, nil
}

func (s *stubSessionRepository) GetSessionFilePath(ctx context.Context, sessionID string) (string, error) {
	return "", nil
}

func (s *stubSessionRepository) GetSessionIDByFilePath(ctx context.Context, filePath string) (string, error) {
	if err, ok := s.errByPath[filePath]; ok {
		return "", err
	}
	return s.sessionIDByPath[filePath], nil
}

func (s *stubSessionRepository) GetSessionModTime(ctx context.Context, sessionID string) (time.Time, error) {
	return time.Time{}, nil
}
