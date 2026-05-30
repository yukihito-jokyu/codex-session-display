package logger

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

const maxLogSize = 10 * 1024 * 1024 // 10MB

var userHomeDirFn = os.UserHomeDir

// GetLogDir はログファイルが格納されるディレクトリの絶対パスを返します。
func GetLogDir() (string, error) {
	home, err := userHomeDirFn()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".codex-display", "logs"), nil
}

// GetLogFilePath はメインのログファイルの絶対パスを返します。
func GetLogFilePath() (string, error) {
	dir, err := GetLogDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "app.log"), nil
}

// logHelper は呼び出し元のコールスタック情報（ファイル名・行番号）を正しく保持したまま slog 出力を行います。
// callerSkip は runtime.Callers のスキップ数を指定します（Debug/Info/Warn/Error からの呼び出しでは 3 を指定）。
func logHelper(callerSkip int, level slog.Level, msg string, args ...any) {
	l := slog.Default()
	ctx := context.Background()
	if !l.Enabled(ctx, level) {
		return
	}
	var pcs [1]uintptr
	runtime.Callers(callerSkip, pcs[:])
	r := slog.NewRecord(time.Now(), level, msg, pcs[0])
	r.Add(args...)
	_ = l.Handler().Handle(ctx, r)
}

// Debug はデバッグレベルのログを出力します。
// 開発・トラブルシューティング用の詳細ログに使用します。
func Debug(msg string, args ...any) {
	logHelper(3, slog.LevelDebug, msg, args...)
}

// Info は情報レベルのログを出力します。
// アプリケーションの正常な動作を示すイベントに使用します。
func Info(msg string, args ...any) {
	logHelper(3, slog.LevelInfo, msg, args...)
}

// Warn は警告レベルのログを出力します。
// 回復可能な問題やデータの不整合がある場合に使用します。
func Warn(msg string, args ...any) {
	logHelper(3, slog.LevelWarn, msg, args...)
}

// Error はエラーレベルのログを出力します。
// 回復不可能な問題や主要機能の失敗時に使用します。
func Error(msg string, args ...any) {
	logHelper(3, slog.LevelError, msg, args...)
}

