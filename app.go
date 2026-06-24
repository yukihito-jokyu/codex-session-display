package main

import (
	"codex-session-display/internal/domain/dto"
	"codex-session-display/internal/domain/model"
	"codex-session-display/internal/repository"
	"codex-session-display/internal/usecase"
	"codex-session-display/internal/utils/logger"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

var (
	execCommand    = exec.Command
	windowShow     = wailsruntime.WindowShow
	eventsEmit     = wailsruntime.EventsEmit
	saveFileDialog = wailsruntime.SaveFileDialog
)

// App はアプリケーションの構造体です。
type App struct {
	mu                 sync.Mutex
	ctx                context.Context
	listSessionsUC     *usecase.ListSessionsUseCase
	getSessionDetailUC *usecase.GetSessionDetailUseCase
	sessionRepo        usecase.SessionRepository
	frontendReady      bool
	pendingSessionFile []string
	sessionWatcher     *repository.SessionWatcher
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

	var app *App
	watcher := repository.NewSessionWatcher(sessionsDir, 1*time.Second, func(filePath string) {
		if app != nil {
			app.emitSessionDirChanged(filePath)
		}
	})

	app = &App{
		listSessionsUC:     listSessionsUC,
		getSessionDetailUC: getSessionDetailUC,
		sessionRepo:        sessionRepo,
		sessionWatcher:     watcher,
	}

	return app, nil
}

// startup はアプリ起動時に呼び出されます。ランタイムメソッドを呼び出せるように
// コンテキストが保存されます。
func (a *App) startup(ctx context.Context) {
	logger.Info("Application startup called")
	a.mu.Lock()
	a.ctx = ctx
	a.mu.Unlock()
	if a.sessionWatcher != nil {
		a.sessionWatcher.Start(ctx)
	}
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

// ResolveSessionIDFromPath はセッションファイルパスからセッションIDを解決します。
func (a *App) ResolveSessionIDFromPath(filePath string) (string, error) {
	logger.Info("ResolveSessionIDFromPath called", "file_path", filePath)

	sessionID, err := a.sessionRepo.GetSessionIDByFilePath(a.ctx, filePath)
	if err != nil {
		appErr := &dto.AppError{
			Code:    "INTERNAL_ERROR",
			Message: "セッションファイルの解決に失敗しました",
		}
		if errors.Is(err, repository.ErrSessionNotFound) {
			appErr.Code = "SESSION_NOT_FOUND"
			appErr.Message = "セッションファイルが見つかりません"
		}
		logger.Error("ResolveSessionIDFromPath failed", "file_path", filePath, "error", err, "code", appErr.Code)
		return "", appErr
	}

	logger.Info("ResolveSessionIDFromPath succeeded", "file_path", filePath, "session_id", sessionID)
	return sessionID, nil
}

// FrontendReady はフロントエンドのイベント購読準備完了を通知します。
func (a *App) FrontendReady() {
	a.mu.Lock()
	a.frontendReady = true
	a.mu.Unlock()
	a.flushPendingSessionFiles()
}

// HandleOpenSessionFile はシングルインスタンス転送で受信したセッションファイルを処理します。
func (a *App) HandleOpenSessionFile(filePath string) {
	if filePath == "" {
		return
	}

	a.mu.Lock()
	if a.ctx == nil || !a.frontendReady {
		a.pendingSessionFile = append(a.pendingSessionFile, filePath)
		a.mu.Unlock()
		return
	}
	ctx := a.ctx
	a.mu.Unlock()

	a.emitOpenSessionFile(ctx, filePath)
}

func (a *App) flushPendingSessionFiles() {
	a.mu.Lock()
	pending := append([]string(nil), a.pendingSessionFile...)
	a.pendingSessionFile = nil
	ctx := a.ctx
	ready := a.frontendReady
	a.mu.Unlock()

	if ctx == nil || !ready {
		return
	}

	for _, filePath := range pending {
		a.emitOpenSessionFile(ctx, filePath)
	}
}

func (a *App) emitOpenSessionFile(ctx context.Context, filePath string) {
	windowShow(ctx)
	eventsEmit(ctx, "open-session-file", filePath)
}

func (a *App) emitSessionDirChanged(filePath string) {
	a.mu.Lock()
	ctx := a.ctx
	ready := a.frontendReady
	a.mu.Unlock()

	if ctx != nil && ready {
		eventsEmit(ctx, "session-dir-changed", filePath)
	}
}

// Emit は Wails イベントをフロントエンドに送信します（usecase.AppEventsEmitter インターフェースの実装）。
func (a *App) Emit(eventName string, optionalData ...interface{}) {
	if a.ctx != nil {
		eventsEmit(a.ctx, eventName, optionalData...)
	}
}

// CheckUpdate は最新のアップデートがあるか確認します。
func (a *App) CheckUpdate() (*dto.UpdateResult, error) {
	logger.Info("CheckUpdate called")

	owner := "yukihito-jokyu"
	repo := "codex-session-display"
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)

	uc := usecase.NewCheckUpdateUseCase(Version, apiURL)
	res, err := uc.Execute(a.ctx)
	if err != nil {
		appErr := &dto.AppError{
			Code:    "UPDATE_CHECK_ERROR",
			Message: "アップデートの確認に失敗しました",
		}
		logger.Error("CheckUpdate failed", "error", err)
		return nil, appErr
	}

	logger.Info("CheckUpdate completed", "hasUpdate", res.HasUpdate, "latest", res.Latest)
	return res, nil
}

// ApplyUpdate は指定されたURLからアップデートパッケージをダウンロードして適用します。
func (a *App) ApplyUpdate(downloadURL string) error {
	logger.Info("ApplyUpdate called", "url", downloadURL)
	if downloadURL == "" {
		return &dto.AppError{
			Code:    "INVALID_URL",
			Message: "ダウンロードURLが空です",
		}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	tempDir := filepath.Join(home, ".codex-display", "update-temp")

	uc := usecase.NewApplyUpdateUseCase(a, tempDir)

	err = uc.Execute(a.ctx, downloadURL)
	if err != nil {
		appErr := &dto.AppError{
			Code:    "UPDATE_APPLY_ERROR",
			Message: fmt.Sprintf("アップデートの適用に失敗しました: %v", err),
		}
		logger.Error("ApplyUpdate failed", "error", err)
		return appErr
	}

	return nil
}

// SaveChartImage は base64 形式の画像データをOSネイティブのダイアログで指定されたパスに PNG ファイルとして保存します。
func (a *App) SaveChartImage(base64Data, defaultName string) error {
	logger.Info("SaveChartImage called", "defaultName", defaultName)

	if base64Data == "" {
		return &dto.AppError{
			Code:    "INVALID_ARGUMENT",
			Message: "画像データが空です",
		}
	}

	filePath, err := saveFileDialog(a.ctx, wailsruntime.SaveDialogOptions{
		DefaultFilename: defaultName,
		Title:           "画像を保存",
		Filters: []wailsruntime.FileFilter{
			{
				DisplayName: "PNG Image (*.png)",
				Pattern:     "*.png",
			},
		},
	})
	if err != nil {
		logger.Error("Failed to show SaveFileDialog", "error", err)
		return &dto.AppError{
			Code:    "INTERNAL_ERROR",
			Message: "保存ダイアログの表示に失敗しました",
		}
	}

	if filePath == "" {
		logger.Info("SaveChartImage cancelled by user")
		return nil
	}

	rawBase64 := base64Data
	if idx := strings.Index(base64Data, ","); idx != -1 {
		rawBase64 = base64Data[idx+1:]
	}

	data, err := base64.StdEncoding.DecodeString(rawBase64)
	if err != nil {
		logger.Error("Failed to decode base64 data", "error", err)
		return &dto.AppError{
			Code:    "INVALID_ARGUMENT",
			Message: "画像データのデコードに失敗しました",
		}
	}

	if err := os.WriteFile(filePath, data, 0o644); err != nil {
		logger.Error("Failed to write image file", "error", err, "path", filePath)
		return &dto.AppError{
			Code:    "FILE_WRITE_ERROR",
			Message: "ファイルの書き込みに失敗しました",
		}
	}

	logger.Info("SaveChartImage succeeded", "path", filePath)
	return nil
}
