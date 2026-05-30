//go:build !production

package logger

import (
	"log/slog"
	"os"
)

// InitLogger は開発環境向けのロガーを初期化します。
// 標準エラー出力 (stderr) に対し、テキスト形式 (slog.TextHandler) でログを出力します。
// 開発環境では DEBUG レベル以上のログを出力します。
func InitLogger() func() {
	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}
	handler := slog.NewTextHandler(os.Stderr, opts)
	logger := slog.New(handler)
	slog.SetDefault(logger)

	// 開発環境ではクローズ処理が必要なファイルはないため、空の関数を返します。
	return func() {}
}
