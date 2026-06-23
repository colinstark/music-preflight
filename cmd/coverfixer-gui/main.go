package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:         "Coverfixer",
		Width:         680,
		Height:        640,
		MinWidth:      520,
		MinHeight:     480,
		DisableResize: false,
		AssetServer:   &assetserver.Options{Assets: assets},
		// Window background is drawn by macOS vibrancy (see Mac options): the
		// webview is transparent and the CSS body is transparent, so the
		// translucent material shows through. BackgroundColour stays nil.
		OnStartup: app.startup,
		Bind: []interface{}{
			app,
		},
		Mac: &mac.Options{
			TitleBar:             mac.TitleBarHidden(),
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
			Appearance:           mac.DefaultAppearance,
			About: &mac.AboutInfo{
				Title:   "Coverfixer",
				Message: "Batch-fix cover art in a music library.",
			},
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
