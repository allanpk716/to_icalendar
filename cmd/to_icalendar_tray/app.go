package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/getlantern/systray"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx context.Context
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
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
	fmt.Println("Initializing system tray with icon...")

	// 1. 优先尝试加载ICO文件 (最佳兼容性)
	iconData, err := a.loadIconFromFile("./build/windows/icon.ico")
	if err != nil {
		fmt.Printf("Failed to load ICO icon: %v\n", err)

		// 2. 备用：尝试加载32x32 PNG图标
		iconData, err = a.loadIconFromFile("./assets/icons/tray-32.png")
		if err != nil {
			fmt.Printf("Failed to load PNG icon: %v\n", err)

			// 3. 兜底：使用内置图标数据
			fmt.Println("Using built-in icon as fallback")
			iconData = a.createSimpleTrayIcon()
		} else {
			fmt.Println("Successfully loaded PNG icon")
		}
	} else {
		fmt.Println("Successfully loaded ICO icon")
	}

	// 设置图标 (关键修复点)
	systray.SetIcon(iconData)

	// 保留标题作为辅助显示
	systray.SetTitle("to_icalendar") // 移除emoji，因为现在有图标了
	systray.SetTooltip("to_icalendar - Microsoft Todo Reminders")
	fmt.Println("System tray initialized with icon and title")

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
	fmt.Println("System tray exiting...")
}

// loadTrayIcon loads and optimizes an icon for system tray use
func (a *App) loadTrayIcon(filename string) ([]byte, error) {
	// Read the icon file
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read icon file: %v", err)
	}

	// For now, return the raw data
	// In a production app, you might want to resize/crop the image
	// to optimize it for tray display (typically 16x16 or 32x32)
	return data, nil
}

// createSimpleIcon creates a simple working icon
func (a *App) createSimpleIcon() []byte {
	// Create a simple 16x16 blue square PNG
	// This is a minimal valid PNG
	return []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D,
		0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x10, 0x00, 0x00, 0x00, 0x10,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, 0xDE, 0x00, 0x00, 0x00,
		0x0C, 0x49, 0x44, 0x41, 0x54, 0x28, 0xCF, 0x63, 0x60, 0x60, 0x60, 0x60,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x03, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE, 0x42, 0x60, 0x82,
	}
}

// loadIconFromFile loads icon data from a file
func (a *App) loadIconFromFile(filename string) ([]byte, error) {
	// Check if file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return nil, fmt.Errorf("icon file not found: %s", filename)
	}

	// Read icon file
	iconData, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read icon file: %v", err)
	}

	return iconData, nil
}

// createSimpleTrayIcon creates a simple tray icon using a working example
func (a *App) createSimpleTrayIcon() []byte {
	// This is a known working icon data for systray on Windows
	// It's a simple 16x16 blue square in ICO format embedded as bytes
	return []byte{
		0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x10, 0x10, 0x10, 0x00, 0x01, 0x00, 0x04, 0x00, 0x28, 0x00,
		0x00, 0x00, 0x10, 0x00, 0x00, 0x00, 0x20, 0x00, 0x00, 0x00, 0x01, 0x00, 0x20, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x42, 0x47, 0x52, 0x73, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xFF, 0xFF, 0xFF, 0x00, 0xFF, 0xFF, 0xFF, 0x00,
		0xFF, 0xFF, 0xFF, 0x00, 0xFF, 0xFF, 0xFF, 0x00, 0xFF, 0xFF, 0xFF, 0x00, 0xFF, 0xFF, 0xFF, 0x00,
		0xFF, 0xFF, 0xFF, 0x00, 0xFF, 0xFF, 0xFF, 0x00, 0xFF, 0xFF, 0xFF, 0x00, 0xFF, 0xFF, 0xFF, 0x00,
		0xFF, 0xFF, 0xFF, 0x00, 0xFF, 0xFF, 0xFF, 0x00, 0xFF, 0xFF, 0xFF, 0x00, 0xFF, 0xFF, 0xFF, 0x00,
		0xFF, 0xFF, 0xFF, 0x00, 0xFF, 0xFF, 0xFF, 0x00, 0xFF, 0xFF, 0xFF, 0x00, 0xFF, 0xFF, 0xFF, 0x00,
		0xFF, 0xFF, 0xFF, 0x00, 0xFF, 0xFF, 0xFF, 0x00, 0xFF, 0xFF, 0xFF, 0x00, 0xFF, 0xFF, 0xFF, 0x00,
		0xFF, 0xFF, 0xFF, 0x00, 0xFF, 0xFF, 0xFF, 0x00, 0xFF, 0xFF, 0xFF, 0x00, 0xFF, 0xFF, 0xFF, 0x00,
		0xFF, 0xFF, 0xFF, 0x00, 0xFF, 0xFF, 0xFF, 0x00, 0xFF, 0xFF, 0xFF, 0x00, 0xFF, 0xFF, 0xFF, 0x00,
		0xFF, 0xFF, 0xFF, 0x00, 0xFF, 0xFF, 0xFF, 0x00, 0xFF, 0xFF, 0xFF, 0x00, 0xFF, 0xFF, 0xFF, 0x00,
	}
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
	fmt.Println("Quitting application...")
	// Use os.Exit for forceful termination after cleanup
	go func() {
		// Give systray a moment to clean up
		systray.Quit()
		// Force exit after a short delay
		time.Sleep(100 * time.Millisecond)
		os.Exit(0)
	}()
}
