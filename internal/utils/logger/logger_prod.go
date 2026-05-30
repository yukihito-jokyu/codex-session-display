//go:build production

package logger

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

const maxLogSize = 10 * 1024 * 1024 // 10MB

// InitLogger は本番環境向けのロガーを初期化します。
// 起動時に簡易ログローテーションを実行し、
// 標準エラー出力 (stderr) および ~/.codex-display/logs/app.log に JSON形式 (slog.JSONHandler) で出力します。
// 本番環境では INFO レベル以上のログを出力します。
func InitLogger() func() {
	logDir, err := GetLogDir()
	if err != nil {
		setupFallbackLogger("failed to get log dir: " + err.Error())
		return func() {}
	}

	// ログディレクトリの作成
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		setupFallbackLogger("failed to create log dir: " + err.Error())
		return func() {}
	}

	logPath := filepath.Join(logDir, "app.log")
	logPath1 := logPath + ".1"

	// 起動時の簡易ログローテーション
	if info, err := os.Stat(logPath); err == nil {
		if info.Size() >= maxLogSize {
			// 古い世代が存在する場合は削除
			_ = os.Remove(logPath1)
			// 現在のログを退避
			_ = os.Rename(logPath, logPath1)
		}
	}

	// 新しいログファイルを作成、または追記モードで開く
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		setupFallbackLogger("failed to open log file: " + err.Error())
		return func() {}
	}

	// 標準エラー出力とログファイルの両方に出力する MultiWriter
	mw := io.MultiWriter(os.Stderr, file)

	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}
	handler := slog.NewJSONHandler(mw, opts)
	logger := slog.New(handler)
	slog.SetDefault(logger)

	return func() {
		_ = file.Close()
	}
}

func setupFallbackLogger(reason string) {
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}
	handler := slog.NewJSONHandler(os.Stderr, opts)
	logger := slog.New(handler)
	slog.SetDefault(logger)
	slog.Warn("falling back to stderr only logging", slog.String("reason", reason))
}
