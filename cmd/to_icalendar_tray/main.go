package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Create an instance of the app structure
	app := NewApp()

	// Create tray menu
	trayMenu := menu.NewMenu()
	exitItem := menu.Text("退出", nil, func(_ *menu.CallbackData) {
		app.Quit()
	})
	trayMenu.Append(exitItem)

	// Create application with options
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
		WindowStartState: options.Minimised, // Start minimized to tray
		Menu:             trayMenu,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
