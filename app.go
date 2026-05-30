package main

import (
	"codex-session-display/internal/domain/dto"
	"codex-session-display/internal/repository"
	"codex-session-display/internal/usecase"
	"codex-session-display/internal/utils/logger"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// App はアプリケーションの構造体です。
type App struct {
	ctx            context.Context
	listSessionsUC *usecase.ListSessionsUseCase
}

// NewApp は新しい App アプリケーション構造体を作成します。
func NewApp() (*App, error) {
	logger.Info("Initializing application...")

	home, err := os.UserHomeDir()
	if err != nil {
		logger.Warn("Failed to get user home directory, using current directory", "error", err.Error())
		home = "."
	}
	sessionsDir := filepath.Join(home, ".codex", "sessions")
	cacheDir := filepath.Join(home, ".codex-display")

	// キャッシュディレクトリが存在することを確認します。
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		logger.Error("Failed to create cache directory", "error", err.Error(), "path", cacheDir)
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	cacheRepo := repository.NewCacheFSRepository(cacheDir)
	sessionRepo := repository.NewSessionFSRepository(sessionsDir, cacheRepo)
	listSessionsUC := usecase.NewListSessionsUseCase(sessionRepo, cacheRepo)

	return &App{
		listSessionsUC: listSessionsUC,
	}, nil
}

// startup はアプリ起動時に呼び出されます。ランタイムメソッドを呼び出せるように
// コンテキストが保存されます。
func (a *App) startup(ctx context.Context) {
	logger.Info("Application startup called")
	a.ctx = ctx
}

// Greet は指定された名前に挨拶を返します。
func (a *App) Greet(name string) string {
	logger.Debug("Greet called", "name", name)
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

// ListSessions は指定された検索クエリと年月でフィルタリングされたセッションの一覧を返します。
func (a *App) ListSessions(query string, year, month int) ([]dto.SessionSummary, error) {
	logger.Info("ListSessions start", "query", query, "year", year, "month", month)
	res, err := a.listSessionsUC.Execute(a.ctx, query, year, month)
	if err != nil {
		appErr := &dto.AppError{
			Code:    "INTERNAL_ERROR",
			Message: "セッション一覧の取得に失敗しました",
		}
		logger.Error("ListSessions failed", "error", err.Error(), "code", appErr.Code)
		return nil, appErr
	}
	logger.Info("ListSessions completed", "count", len(res))
	return res, nil
}

// OpenLogDirectory はアプリケーションログのディレクトリをOSの標準ファイルマネージャーで開きます。
func (a *App) OpenLogDirectory() error {
	logger.Info("OpenLogDirectory called")

	logDir, err := logger.GetLogDir()
	if err != nil {
		appErr := &dto.AppError{
			Code:    "INTERNAL_ERROR",
			Message: "ログディレクトリの取得に失敗しました",
		}
		logger.Error("Failed to get log directory", "error", err.Error())
		return appErr
	}

	// ログディレクトリの存在確認と作成
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		appErr := &dto.AppError{
			Code:    "INTERNAL_ERROR",
			Message: "ログディレクトリの作成に失敗しました",
		}
		logger.Error("Failed to create log directory", "error", err.Error(), "path", logDir)
		return appErr
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", logDir)
	case "windows":
		cmd = exec.Command("explorer", logDir)
	default: // linux, etc.
		cmd = exec.Command("xdg-open", logDir)
	}

	if err := cmd.Start(); err != nil {
		appErr := &dto.AppError{
			Code:    "INTERNAL_ERROR",
			Message: "ログディレクトリを開くのに失敗しました",
		}
		logger.Error("Failed to open log directory", "error", err.Error(), "path", logDir)
		return appErr
	}

	logger.Info("OpenLogDirectory succeeded")
	return nil
}
