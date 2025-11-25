package tray

import (
	"context"
	"fmt"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// TrayManager 管理系统托盘功能
type TrayManager struct {
	ctx     context.Context
	app     *TrayApplication
	icon    *TrayIcon
	menu    *TrayMenu
	running bool
}

// NewTrayManager 创建新的托盘管理器
func NewTrayManager(ctx context.Context) *TrayManager {
	return &TrayManager{
		ctx:     ctx,
		running: false,
	}
}

// Initialize 初始化托盘管理器
func (tm *TrayManager) Initialize() error {
	if tm.running {
		return NewTrayError(ErrCodeRuntime, "tray manager is already running", nil)
	}

	// 创建托盘应用程序配置
	tm.app = NewTrayApplication("to_icalendar_tray", "to_icalendar", "1.0.0")

	// 创建默认托盘图标
	tm.icon = NewTrayIcon("assets/icons/tray-32.png", 32)

	// 创建默认托盘菜单
	tm.menu = NewTrayMenu()
	tm.menu.AddItem(NewMenuItem("exit", "退出", "退出应用程序", MenuTypeAction, "quit", "", false, true, 1))

	LogInfo("Tray manager initialized for app: %s", tm.app.Name)
	return nil
}

// Show 显示托盘图标
func (tm *TrayManager) Show() error {
	if !tm.running {
		return fmt.Errorf("tray manager is not initialized")
	}

	// 设置托盘图标
	if tm.icon != nil {
		if err := tm.setIcon(); err != nil {
			return NewTrayError(ErrCodeIconLoad, "failed to set tray icon", err)
		}
	}

	// 设置托盘菜单
	if tm.menu != nil {
		if err := tm.setMenu(); err != nil {
			return NewTrayError(ErrCodeMenuCreation, "failed to set tray menu", err)
		}
	}

	LogInfo("Tray icon shown for app: %s", tm.app.Name)
	return nil
}

// Hide 隐藏托盘图标
func (tm *TrayManager) Hide() {
	if tm.running {
		// 清理托盘资源
		runtime.Hide(tm.ctx)
		LogInfo("Tray icon hidden for app: %s", tm.app.Name)
	}
}

// SetTooltip 设置托盘图标提示文本
func (tm *TrayManager) SetTooltip(tooltip string) {
	if tm.app != nil {
		tm.app.Tooltip = tooltip
	}
}

// IsRunning 检查托盘管理器是否正在运行
func (tm *TrayManager) IsRunning() bool {
	return tm.running
}

// Start 启动托盘管理器
func (tm *TrayManager) Start() error {
	if err := tm.Initialize(); err != nil {
		return err
	}

	tm.running = true

	if err := tm.Show(); err != nil {
		tm.running = false
		return err
	}

	return nil
}

// Stop 停止托盘管理器
func (tm *TrayManager) Stop() {
	if tm.running {
		tm.Hide()
		tm.running = false
		LogInfo("Tray manager stopped for app: %s", tm.app.Name)
	}
}

// setIcon 设置托盘图标
func (tm *TrayManager) setIcon() error {
	// TODO: 实现实际的图标设置逻辑
	// 这里需要使用Wails的runtime.SystemTray功能
	LogDebug("Setting tray icon: %s", tm.icon.FilePath)
	return nil
}

// setMenu 设置托盘菜单
func (tm *TrayManager) setMenu() error {
	// TODO: 实现实际的菜单设置逻辑
	// 这里需要使用Wails的菜单功能
	LogDebug("Setting tray menu with %d items", len(tm.menu.Items))
	return nil
}

// Quit 退出应用程序
func (tm *TrayManager) Quit() {
	LogInfo("Quitting application: %s", tm.app.Name)
	tm.Stop()
	runtime.Quit(tm.ctx)
}