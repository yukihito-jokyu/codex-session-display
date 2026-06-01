package main

import (
	"codex-session-display/internal/domain/dto"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
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
