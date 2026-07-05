package main

import (
	"bytes"
	"codex-session-display/internal/domain/dto"
	"codex-session-display/internal/repository"
	"codex-session-display/internal/usecase"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
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

func TestResolveClaudeProjectsDir(t *testing.T) {
	originalClaudeConfigDir := os.Getenv("CLAUDE_CONFIG_DIR")
	defer func() {
		_ = os.Setenv("CLAUDE_CONFIG_DIR", originalClaudeConfigDir)
	}()

	tests := []struct {
		name            string
		claudeConfigDir string
		home            string
		want            string
	}{
		{
			name:            "uses CLAUDE_CONFIG_DIR projects when set",
			claudeConfigDir: "/tmp/custom-claude",
			home:            "/tmp/home",
			want:            filepath.Join("/tmp/custom-claude", "projects"),
		},
		{
			name: "falls back to home claude projects",
			home: "/tmp/home",
			want: filepath.Join("/tmp/home", ".claude", "projects"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.claudeConfigDir == "" {
				_ = os.Unsetenv("CLAUDE_CONFIG_DIR")
			} else {
				_ = os.Setenv("CLAUDE_CONFIG_DIR", tt.claudeConfigDir)
			}
			got := resolveClaudeProjectsDir(tt.home)
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
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

func (s *stubSessionRepository) ListSessions(ctx context.Context, provider dto.SessionProvider, year, month int, query string) ([]dto.SessionSummary, error) {
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

func TestApp_SaveChartImage(t *testing.T) {
	originalSaveFileDialog := saveFileDialog
	defer func() {
		saveFileDialog = originalSaveFileDialog
	}()

	tempDir := t.TempDir()

	tests := []struct {
		name         string
		base64Data   string
		defaultName  string
		dialogPath   string
		dialogErr    error
		wantErr      bool
		wantErrCode  string
		expectedData []byte
	}{
		{
			name:         "success - normal base64",
			base64Data:   "SGVsbG8gV29ybGQ=", // "Hello World"
			defaultName:  "test.png",
			dialogPath:   filepath.Join(tempDir, "test1.png"),
			wantErr:      false,
			expectedData: []byte("Hello World"),
		},
		{
			name:         "success - with data uri prefix",
			base64Data:   "data:image/png;base64,SGVsbG8gV29ybGQ=",
			defaultName:  "test.png",
			dialogPath:   filepath.Join(tempDir, "test2.png"),
			wantErr:      false,
			expectedData: []byte("Hello World"),
		},
		{
			name:        "success - user cancelled",
			base64Data:  "SGVsbG8gV29ybGQ=",
			defaultName: "test.png",
			dialogPath:  "", // cancel returns empty path
			wantErr:     false,
		},
		{
			name:        "error - empty base64",
			base64Data:  "",
			defaultName: "test.png",
			wantErr:     true,
			wantErrCode: "INVALID_ARGUMENT",
		},
		{
			name:        "error - invalid base64",
			base64Data:  "invalid-base-64!!!",
			defaultName: "test.png",
			dialogPath:  filepath.Join(tempDir, "test3.png"),
			wantErr:     true,
			wantErrCode: "INVALID_ARGUMENT",
		},
		{
			name:        "error - dialog failure",
			base64Data:  "SGVsbG8gV29ybGQ=",
			defaultName: "test.png",
			dialogErr:   errors.New("dialog failed"),
			wantErr:     true,
			wantErrCode: "INTERNAL_ERROR",
		},
		{
			name:        "error - file write failure (invalid path)",
			base64Data:  "SGVsbG8gV29ybGQ=",
			defaultName: "test.png",
			dialogPath:  filepath.Join(tempDir, "nonexistent-dir", "test.png"),
			wantErr:     true,
			wantErrCode: "FILE_WRITE_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock Dialog
			saveFileDialog = func(ctx context.Context, options wailsruntime.SaveDialogOptions) (string, error) {
				if tt.dialogErr != nil {
					return "", tt.dialogErr
				}
				return tt.dialogPath, nil
			}

			app := &App{ctx: context.Background()}
			err := app.SaveChartImage(tt.base64Data, tt.defaultName)

			if (err != nil) != tt.wantErr {
				t.Fatalf("expected error=%v, got %v", tt.wantErr, err)
			}

			if tt.wantErr {
				var appErr *dto.AppError
				if !errors.As(err, &appErr) {
					t.Fatalf("expected *dto.AppError, got %T", err)
				}
				if appErr.Code != tt.wantErrCode {
					t.Fatalf("expected error code %q, got %q", tt.wantErrCode, appErr.Code)
				}
				return
			}

			// Verify file content for success cases with non-empty dialogPath
			if tt.dialogPath != "" {
				data, err := os.ReadFile(tt.dialogPath)
				if err != nil {
					t.Fatalf("failed to read written file: %v", err)
				}
				if !bytes.Equal(data, tt.expectedData) {
					t.Fatalf("expected file content %q, got %q", string(tt.expectedData), string(data))
				}
			}
		})
	}
}

func TestApp_AnalyzeClaudeCorpus(t *testing.T) {
	tempDir := t.TempDir()

	// Create dummy projects and session logs
	projectDir := filepath.Join(tempDir, "projects", "test-proj")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "session.jsonl"), []byte(`{"type":"user","sessionId":"session-1"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	uc := usecase.NewAnalyzeClaudeCorpusUseCase()
	uc.SetCustomProjectsDir(filepath.Join(tempDir, "projects"))

	app := &App{
		analyzeClaudeCorpusUC: uc,
		ctx:                   context.Background(),
	}

	t.Run("success", func(t *testing.T) {
		res, err := app.AnalyzeClaudeCorpus(dto.AnalyzeOptions{ProjectSource: "home"})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if res.TotalFiles != 1 {
			t.Errorf("expected 1 file, got %d", res.TotalFiles)
		}
	})

	t.Run("error when directory not found or config dir not set", func(t *testing.T) {
		originalConfigDir := os.Getenv("CLAUDE_CONFIG_DIR")
		os.Unsetenv("CLAUDE_CONFIG_DIR")
		defer func() {
			if originalConfigDir != "" {
				_ = os.Setenv("CLAUDE_CONFIG_DIR", originalConfigDir)
			}
		}()

		// Create a separate usecase instance without SetCustomProjectsDir to force it to use environment variables
		ucErr := usecase.NewAnalyzeClaudeCorpusUseCase()
		appErr := &App{
			analyzeClaudeCorpusUC: ucErr,
			ctx:                   context.Background(),
		}

		_, err := appErr.AnalyzeClaudeCorpus(dto.AnalyzeOptions{ProjectSource: "config"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		var appErrType *dto.AppError
		if !errors.As(err, &appErrType) {
			t.Fatalf("expected *dto.AppError, got %T", err)
		}
		if appErrType.Code != "ANALYZE_ERROR" {
			t.Errorf("expected ANALYZE_ERROR, got %s", appErrType.Code)
		}
	})
}
