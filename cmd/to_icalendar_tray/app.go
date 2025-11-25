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
	ctx            context.Context
	appIcon        []byte // 应用程序图标
	isWindowVisible bool   // 窗口可见状态跟踪
	isQuitting     bool   // 退出状态跟踪
}

// NewApp creates a new App application struct
func NewApp(icon []byte) *App {
	return &App{
		appIcon:         icon,
		isWindowVisible: false,
		isQuitting:      false,
	}
}

// startup is called when the app starts up.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.isWindowVisible = true
	// Start system tray in a goroutine after a short delay to ensure Wails is ready
	go func() {
		// 等待一小段时间确保Wails完全初始化
		// time.Sleep(100 * time.Millisecond)
		a.setupSystemTray()
	}()
}

// onDomReady is called after front-end resources have been loaded
func (a *App) onDomReady(ctx context.Context) {
	// Here you could make your initial API calls or set up your frontend
}

// onBeforeClose is called when the application is about to quit,
// either by clicking the window close button or calling runtime.Quit.
// Returning true will cause the application to continue, false will continue shutdown as normal.
func (a *App) onBeforeClose(ctx context.Context) (prevent bool) {
	// 如果是用户点击窗口关闭按钮且不是正在退出，隐藏到托盘
	if !a.isQuitting {
		a.HideWindow()
		return true // 阻止窗口关闭，隐藏到托盘
	}

	// 如果是调用Quit()方法触发的关闭，允许正常退出
	return false // 允许退出
}

// onShutdown is called when the application is shutting down
func (a *App) onShutdown(ctx context.Context) {
	// Clean shutdown - stop systray
	systray.Quit()
}

// setupSystemTray configures the system tray icon and menu
func (a *App) setupSystemTray() {
	systray.Run(a.onSystrayReady, a.onSystrayExit)
}

// onSystrayReady is called when the system tray is ready
func (a *App) onSystrayReady() {
	// Set icon and title
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
	a.isWindowVisible = false
}

// ShowWindow shows the main window (alias for Show)
func (a *App) ShowWindow() {
	runtime.WindowShow(a.ctx)
	a.isWindowVisible = true
}

// IsWindowVisible returns whether the main window is visible
func (a *App) IsWindowVisible() bool {
	return a.isWindowVisible && a.ctx != nil
}

// Quit exits the application
func (a *App) Quit() {
	// 设置退出状态标志
	a.isQuitting = true

	// 先停止systray
	systray.Quit()

	// 然后退出Wails应用
	runtime.Quit(a.ctx)

	// 强制退出（最后手段，确保进程完全退出）
	go func() {
		time.Sleep(500 * time.Millisecond)
		os.Exit(0)
	}()
}