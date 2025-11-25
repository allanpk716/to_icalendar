package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/windows/icon.ico
var appIcon []byte

func main() {
	// Create an instance of the app structure
	app := NewApp(appIcon)

	// Create application with options - no main menu since we use systray
	err := wails.Run(&options.App{
		Title:  "to_icalendar",
		Width:  800,
		Height: 600,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		OnDomReady:       app.onDomReady,
		OnBeforeClose:    app.onBeforeClose,
		OnShutdown:       app.onShutdown,
		WindowStartState: options.Normal, // 改为Normal启动，确保窗口可见
		// 移除Menu配置，使用systray处理托盘功能
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
