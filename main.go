package main

import (
	"codex-session-display/internal/domain/dto"
	"codex-session-display/internal/utils/logger"
	"embed"
	"errors"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// ロガーの初期化とクリーンアップ設定
	cleanup := logger.InitLogger()
	defer cleanup()

	logger.Info("Starting application...")

	// アプリ構造体のインスタンスを作成
	app, err := NewApp()
	if err != nil {
		logger.Error("Failed to initialize app", "error", err.Error())
		return
	}

	// オプションを指定してアプリケーションを作成
	err = wails.Run(&options.App{
		Title:  "codex-session-display",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 0, G: 0, B: 0, A: 0},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
		ErrorFormatter: func(err error) interface{} {
			var appErr *dto.AppError
			if errors.As(err, &appErr) {
				return appErr
			}
			return map[string]string{
				"code":    "INTERNAL_ERROR",
				"message": err.Error(),
			}
		},
		// macOS 固有設定 (Vibrancy & TitleBarHidden)
		Mac: &mac.Options{
			TitleBar:             mac.TitleBarHidden(),
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
			About: &mac.AboutInfo{
				Title:   "codex-session-display",
				Message: "Codex Session Display",
			},
		},
		// Windows 固有設定 (Mica & Translucent)
		Windows: &windows.Options{
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
			BackdropType:         windows.Mica,
			DisableWindowIcon:    false,
		},
	})
	if err != nil {
		logger.Error("Wails run error", "error", err.Error())
	}
}
