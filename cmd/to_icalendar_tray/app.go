package main

import (
	"context"
	_ "embed"
	"os"
	"time"

	"github.com/getlantern/systray"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// 使用 main.go 中嵌入的图标
// 注意：这里不再重复嵌入，避免资源重复

// App struct
type App struct {
	ctx       context.Context
	appIcon   []byte // 应用程序图标
}

// NewApp creates a new App application struct
func NewApp(icon []byte) *App {
	return &App{
		appIcon: icon,
	}
}

// startup is called when the app starts up.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	// Start system tray in a goroutine
	go a.setupSystemTray()
}

// onDomReady is called after front-end resources have been loaded
func (a *App) onDomReady(ctx context.Context) {
	// Here you could make your initial API calls or set up your frontend
}

// onBeforeClose is called when the application is about to quit,
// either by clicking the window close button or calling runtime.Quit.
// Returning true will cause the application to continue, false will continue shutdown as normal.
func (a *App) onBeforeClose(ctx context.Context) (prevent bool) {
	// Hide window instead of closing to keep tray running
	a.HideWindow()
	return true // Prevent the window from closing
}

// onShutdown is called when the application is shutting down
func (a *App) onShutdown(ctx context.Context) {
	// Perform your teardown here
}

// setupSystemTray configures the system tray icon and menu
func (a *App) setupSystemTray() {
	systray.Run(a.onSystrayReady, a.onSystrayExit)
}

// onSystrayReady is called when the system tray is ready
func (a *App) onSystrayReady() {
	// Use the same icon as the main application
	systray.SetIcon(a.appIcon)
	systray.SetTitle("to_icalendar")
	systray.SetTooltip("to_icalendar - Microsoft Todo Reminders")

	// Show window menu item
	mShow := systray.AddMenuItem("显示窗口", "显示主窗口")
	go func() {
		for range mShow.ClickedCh {
			a.ShowWindow()
		}
	}()

	// Hide window menu item
	mHide := systray.AddMenuItem("隐藏窗口", "隐藏主窗口")
	go func() {
		for range mHide.ClickedCh {
			a.HideWindow()
		}
	}()

	systray.AddSeparator()

	// Exit menu item
	mQuit := systray.AddMenuItem("退出", "退出应用程序")
	go func() {
		for range mQuit.ClickedCh {
			a.Quit()
		}
	}()
}

// onSystrayExit is called when the system tray is exiting
func (a *App) onSystrayExit() {
	// Clean shutdown
}

// Show shows the main window
func (a *App) Show() {
	runtime.WindowShow(a.ctx)
}

// Hide hides the main window
func (a *App) Hide() {
	runtime.WindowHide(a.ctx)
}

// HideWindow hides the main window (alias for Hide)
func (a *App) HideWindow() {
	runtime.WindowHide(a.ctx)
}

// ShowWindow shows the main window (alias for Show)
func (a *App) ShowWindow() {
	runtime.WindowShow(a.ctx)
}

// IsWindowVisible returns whether the main window is visible
func (a *App) IsWindowVisible() bool {
	return a.ctx != nil
}

// Quit exits the application
func (a *App) Quit() {
	go func() {
		// Give systray a moment to clean up
		systray.Quit()
		// Force exit after a short delay
		time.Sleep(100 * time.Millisecond)
		os.Exit(0)
	}()
}