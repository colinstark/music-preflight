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
		Title:         "Music Preflight",
		Width:         680,
		Height:        640,
		MinWidth:      420,
		MinHeight:     240,
		DisableResize: false,
		AssetServer:   &assetserver.Options{Assets: assets},
		// Transparent so macOS never paints a solid (white) window background
		// behind the native titlebar — otherwise scrolling flips the titlebar
		// to that colour. The vibrancy material shows through instead.
		BackgroundColour: &options.RGBA{R: 0, G: 0, B: 0, A: 0},
		OnStartup: app.startup,
		Bind: []interface{}{
			app,
		},
		Mac: &mac.Options{
			// Transparent titlebar (traffic lights kept, native title hidden):
			// a custom header band renders the title so its colour is fully
			// controlled (white-on-dark in dark mode, theme-aware).
			TitleBar:             mac.TitleBarHidden(),
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
			Appearance:           mac.DefaultAppearance,
			About: &mac.AboutInfo{
				Title:   "Music Preflight",
				Message: "Batch-fix cover art in a music library.",
			},
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
