package main

import (
	"codex-session-display/internal/domain/dto"
	"codex-session-display/internal/domain/model"
	"codex-session-display/internal/repository"
	"codex-session-display/internal/usecase"
	"codex-session-display/internal/utils/logger"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

var execCommand = exec.Command

// App はアプリケーションの構造体です。
type App struct {
	ctx                context.Context
	listSessionsUC     *usecase.ListSessionsUseCase
	getSessionDetailUC *usecase.GetSessionDetailUseCase
}

// NewApp は新しい App アプリケーション構造体を作成します。
func NewApp() (*App, error) {
	logger.Info("Initializing application...")

	home, err := os.UserHomeDir()
	if err != nil {
		logger.Warn("Failed to get user home directory, using current directory", "error", err)
		home = "."
	}
	sessionsDir := filepath.Join(home, ".codex", "sessions")
	cacheDir := filepath.Join(home, ".codex-display")

	// キャッシュディレクトリが存在することを確認します。
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		logger.Error("Failed to create cache directory", "error", err, "path", cacheDir)
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	cacheRepo := repository.NewCacheFSRepository(cacheDir)
	sessionRepo := repository.NewSessionFSRepository(sessionsDir, cacheRepo)
	listSessionsUC := usecase.NewListSessionsUseCase(sessionRepo, cacheRepo)
	getSessionDetailUC := usecase.NewGetSessionDetailUseCase(sessionRepo, cacheRepo, repository.NewJSONLParser())

	return &App{
		listSessionsUC:     listSessionsUC,
		getSessionDetailUC: getSessionDetailUC,
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
		logger.Error("ListSessions failed", "error", err, "code", appErr.Code)
		return nil, appErr
	}
	logger.Info("ListSessions completed", "count", len(res))
	return res, nil
}

// GetSessionDetail は指定されたセッションIDの詳細情報を取得します。
func (a *App) GetSessionDetail(sessionID string) (*dto.SessionDetailResponse, error) {
	logger.Info("GetSessionDetail start", "session_id", sessionID)
	res, err := a.getSessionDetailUC.Execute(a.ctx, sessionID)
	if err != nil {
		appErr := &dto.AppError{
			Code:    "INTERNAL_ERROR",
			Message: "内部エラーが発生しました",
		}
		if errors.Is(err, model.ErrSessionNotFound) {
			appErr.Code = "SESSION_NOT_FOUND"
			appErr.Message = "セッションファイルが見つかりません"
		} else if errors.Is(err, model.ErrFileTooLarge) {
			appErr.Code = "FILE_TOO_LARGE"
			appErr.Message = "ファイルサイズが制限値を超えています"
		} else if errors.Is(err, model.ErrUnsupportedFormat) {
			appErr.Code = "UNSUPPORTED_FORMAT"
			appErr.Message = "非対応のフォーマットです"
		} else if errors.Is(err, model.ErrParseFailed) {
			appErr.Code = "PARSE_ERROR"
			appErr.Message = "セッションファイルの解析に失敗しました"
		} else if errors.Is(err, model.ErrFileReadError) {
			appErr.Code = "FILE_READ_ERROR"
			appErr.Message = "セッションファイルの読み込みに失敗しました"
		}
		logger.Error("GetSessionDetail failed", "error", err, "code", appErr.Code)
		return nil, appErr
	}
	logger.Info("GetSessionDetail completed", "session_id", sessionID)
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
		logger.Error("Failed to get log directory", "error", err)
		return appErr
	}

	// ログディレクトリの存在確認と作成
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		appErr := &dto.AppError{
			Code:    "INTERNAL_ERROR",
			Message: "ログディレクトリの作成に失敗しました",
		}
		logger.Error("Failed to create log directory", "error", err, "path", logDir)
		return appErr
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = execCommand("open", logDir)
	case "windows":
		cmd = execCommand("explorer", logDir)
	default: // linux, etc.
		cmd = execCommand("xdg-open", logDir)
	}

	if err := cmd.Start(); err != nil {
		appErr := &dto.AppError{
			Code:    "INTERNAL_ERROR",
			Message: "ログディレクトリを開くのに失敗しました",
		}
		logger.Error("Failed to open log directory", "error", err, "path", logDir)
		return appErr
	}

	logger.Info("OpenLogDirectory succeeded")
	return nil
}

// GetLogFilePath はアプリケーションログファイルの絶対パスを返します。
func (a *App) GetLogFilePath() (string, error) {
	logger.Info("GetLogFilePath called")

	logPath, err := logger.GetLogFilePath()
	if err != nil {
		appErr := &dto.AppError{
			Code:    "INTERNAL_ERROR",
			Message: "ログファイルパスの取得に失敗しました",
		}
		logger.Error("Failed to get log file path", "error", err)
		return "", appErr
	}

	logger.Info("GetLogFilePath succeeded", "path", logPath)
	return logPath, nil
}
