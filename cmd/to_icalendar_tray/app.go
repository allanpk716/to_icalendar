package main

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/getlantern/systray"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// 使用 main.go 中嵌入的图标
// 注意：这里不再重复嵌入，避免资源重复

// InitResult 初始化结果结构
type InitResult struct {
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	ConfigDir    string `json:"configDir"`
	ServerConfig string `json:"serverConfig"`
}

// LogMessage 日志消息结构
type LogMessage struct {
	Type    string `json:"type"`    // info, debug, error, success, warn
	Message string `json:"message"`
	Time    string `json:"time"`
}

// App struct
type App struct {
	ctx            context.Context
	appIcon        []byte // 应用程序图标
	isWindowVisible bool   // 窗口可见状态跟踪
	isQuitting     bool   // 退出状态跟踪
	quitOnce       sync.Once        // 确保Quit只执行一次
	quitWG         sync.WaitGroup   // 等待清理完成
	quitDone       chan struct{}    // 退出完成信号
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
	// Wails正在关闭，systray清理应该已经在Quit()中完成
	println("Wails shutdown completed")
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
	// 记录systray退出日志
	println("系统托盘清理完成")

	// 确保所有systray资源被正确清理
	// systray库会自动处理大部分清理工作
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
	a.quitOnce.Do(func() {
		// 设置退出状态标志
		a.isQuitting = true
		println("开始关闭应用程序...")

		// 创建退出完成通道
		a.quitDone = make(chan struct{})

		// 启动清理goroutine
		a.quitWG.Add(1)
		go func() {
			defer a.quitWG.Done()

			// 第一步：停止systray (这会触发onSystrayExit)
			println("正在停止系统托盘...")
			systray.Quit()

			// 给systray一些时间完成清理
			time.Sleep(200 * time.Millisecond)

			// 第二步：退出Wails应用
			println("正在退出Wails应用...")
			runtime.Quit(a.ctx)

			// 关闭退出完成通道
			close(a.quitDone)
		}()

		// 启动超时保护goroutine
		go func() {
			select {
			case <-a.quitDone:
				println("应用程序关闭完成")
			case <-time.After(3 * time.Second):
				println("关闭超时，强制退出...")
				os.Exit(1)
			}
		}()
	})
}

// InitConfigWithStreaming 带实时日志的初始化
func (a *App) InitConfigWithStreaming() {
	// 发送开始日志
	a.sendLog("info", "🚀 开始初始化配置...")

	// 获取用户目录和配置路径
	homeDir, err := os.UserHomeDir()
	if err != nil {
		a.sendLog("error", fmt.Sprintf("❌ 获取用户目录失败: %v", err))
		return
	}
	a.sendLog("debug", fmt.Sprintf("用户目录: %s", homeDir))

	configDir := filepath.Join(homeDir, ".to_icalendar")
	serverConfigPath := filepath.Join(configDir, "server.yaml")

	// 创建配置目录
	a.sendLog("debug", "正在创建配置目录...")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		a.sendLog("error", fmt.Sprintf("❌ 创建配置目录失败: %v", err))
		return
	}
	a.sendLog("success", fmt.Sprintf("✅ 配置目录创建成功: %s", configDir))

	// 检查文件是否已存在 - 根据用户需求，直接显示成功并跳过初始化
	a.sendLog("debug", "检查配置文件是否已存在...")
	if _, err := os.Stat(serverConfigPath); err == nil {
		a.sendLog("success", fmt.Sprintf("✅ 配置文件已存在: %s", serverConfigPath))
		a.sendLog("info", "配置已初始化，可以开始使用")
		a.sendResult(true, "配置文件已存在，无需重复初始化", configDir, serverConfigPath)
		return
	}

	// 创建配置文件内容（复用 CLI 版本的完整模板）
	a.sendLog("debug", "创建默认配置文件内容...")
	serverConfigContent := `# Microsoft Todo 配置
microsoft_todo:
  tenant_id: "YOUR_TENANT_ID"
  client_id: "YOUR_CLIENT_ID"
  client_secret: "YOUR_CLIENT_SECRET"
  user_email: ""
  timezone: "Asia/Shanghai"

# 提醒配置
reminder:
  default_remind_before: "15m"
  enable_smart_reminder: true

# 去重配置
deduplication:
  enabled: true
  time_window_minutes: 5
  similarity_threshold: 80
  check_incomplete_only: true
  enable_local_cache: true
  enable_remote_query: true

# Dify AI 配置（可选）
dify:
  api_endpoint: ""
  api_key: ""
  timeout: 60

# 缓存配置
cache:
  auto_cleanup_days: 30
  cleanup_on_startup: true
  preserve_successful_hashes: true

# 日志配置
logging:
  level: "info"
  console_output: true
  file_output: true
  log_dir: "./Logs"`

	// 写入文件
	a.sendLog("debug", "写入配置文件...")
	if err := os.WriteFile(serverConfigPath, []byte(serverConfigContent), 0600); err != nil {
		a.sendLog("error", fmt.Sprintf("❌ 创建配置文件失败: %v", err))
		return
	}
	a.sendLog("success", fmt.Sprintf("✅ 配置文件创建成功: %s", serverConfigPath))

	// 发送完成信息
	a.sendLog("info", "🎉 初始化完成！")
	a.sendLog("info", "📝 请编辑 server.yaml 文件配置 Microsoft Todo 信息")
	a.sendResult(true, "初始化成功", configDir, serverConfigPath)
}

// sendLog 发送日志到前端
func (a *App) sendLog(logType, message string) {
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "initLog", LogMessage{
			Type:    logType,
			Message: message,
			Time:    time.Now().Format("15:04:05"),
		})
	}
}

// sendResult 发送最终结果
func (a *App) sendResult(success bool, message, configDir, serverConfig string) {
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "initResult", InitResult{
			Success:      success,
			Message:      message,
			ConfigDir:    configDir,
			ServerConfig: serverConfig,
		})
	}
}