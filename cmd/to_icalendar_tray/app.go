package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/allanpk716/to_icalendar/pkg/testing"
	"github.com/getlantern/systray"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"gopkg.in/yaml.v3"
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

// TestResult 完整测试结果结构
type TestResult struct {
	ConfigTest     testing.TestItemResult  `json:"configTest"`
	TodoTest       testing.TestItemResult  `json:"todoTest"`
	DifyTest       *testing.TestItemResult `json:"difyTest,omitempty"`
	OverallSuccess bool                   `json:"overallSuccess"`
	Duration       time.Duration          `json:"duration"`
	Timestamp      string                 `json:"timestamp"`
}

// ServerConfig 配置文件结构
type ServerConfig struct {
	MicrosoftTodo testing.MicrosoftTodoConfig `yaml:"microsoft_todo"`
	Dify          testing.DifyConfig          `yaml:"dify"`
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
	// 设置系统托盘 - 增加延迟确保Wails完全初始化
	go func() {
		// 等待更长时间确保Wails完全初始化，避免竞态条件
		time.Sleep(500 * time.Millisecond)
		a.setupSystemTray()
	}()
}

// onDomReady is called after front-end resources have been loaded
func (a *App) onDomReady(ctx context.Context) {
	// 这里可以进行前端初始化后的操作
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

	// 添加调试输出，确认菜单项创建成功
	println("系统托盘菜单初始化完成")
}

// onSystrayExit is called when the system tray is exiting
func (a *App) onSystrayExit() {
	println("系统托盘清理完成")
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

			// 第一步：停止systray
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

// InitConfig 标准配置初始化方法（不发送实时日志）
// 返回JSON格式的结果字符串，供前端调用
func (a *App) InitConfig() string {
	// 获取用户目录和配置路径
	homeDir, err := os.UserHomeDir()
	if err != nil {
		result := InitResult{
			Success:      false,
			Message:      fmt.Sprintf("获取用户目录失败: %v", err),
			ConfigDir:    "",
			ServerConfig: "",
		}
		return fmt.Sprintf(`{"success":false,"message":"%s","configDir":"","serverConfig":""}`, result.Message)
	}

	configDir := filepath.Join(homeDir, ".to_icalendar")
	serverConfigPath := filepath.Join(configDir, "server.yaml")

	// 创建配置目录
	if err := os.MkdirAll(configDir, 0755); err != nil {
		result := InitResult{
			Success:      false,
			Message:      fmt.Sprintf("创建配置目录失败: %v", err),
			ConfigDir:    configDir,
			ServerConfig: serverConfigPath,
		}
		return fmt.Sprintf(`{"success":false,"message":"%s","configDir":"%s","serverConfig":"%s"}`, result.Message, result.ConfigDir, result.ServerConfig)
	}

	// 检查文件是否已存在
	if _, err := os.Stat(serverConfigPath); err == nil {
		result := InitResult{
			Success:      true,
			Message:      "配置文件已存在，无需重复初始化",
			ConfigDir:    configDir,
			ServerConfig: serverConfigPath,
		}
		return fmt.Sprintf(`{"success":true,"message":"%s","configDir":"%s","serverConfig":"%s"}`, result.Message, result.ConfigDir, result.ServerConfig)
	}

	// 创建配置文件内容（复用现有的完整模板）
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
	if err := os.WriteFile(serverConfigPath, []byte(serverConfigContent), 0600); err != nil {
		result := InitResult{
			Success:      false,
			Message:      fmt.Sprintf("创建配置文件失败: %v", err),
			ConfigDir:    configDir,
			ServerConfig: serverConfigPath,
		}
		return fmt.Sprintf(`{"success":false,"message":"%s","configDir":"%s","serverConfig":"%s"}`, result.Message, result.ConfigDir, result.ServerConfig)
	}

	// 返回成功结果
	result := InitResult{
		Success:      true,
		Message:      "初始化成功",
		ConfigDir:    configDir,
		ServerConfig: serverConfigPath,
	}
	return fmt.Sprintf(`{"success":true,"message":"%s","configDir":"%s","serverConfig":"%s"}`, result.Message, result.ConfigDir, result.ServerConfig)
}

// CheckConfigExists 检查配置文件是否存在
// 返回布尔值，表示配置是否已经初始化
func (a *App) CheckConfigExists() bool {
	// 获取用户目录和配置路径
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false
	}

	serverConfigPath := filepath.Join(homeDir, ".to_icalendar", "server.yaml")

	// 检查配置文件是否存在
	_, err = os.Stat(serverConfigPath)
	return err == nil
}

// TestConfiguration 测试配置完整性和服务连通性
func (a *App) TestConfiguration() string {
	startTime := time.Now()

	// 执行三个测试
	configTest := a.testConfigurationFile()
	todoTest := a.testMicrosoftTodoService()
	difyTest := a.testDifyService()

	// 构建最终结果
	result := &TestResult{
		ConfigTest:     *configTest,
		TodoTest:       *todoTest,
		DifyTest:       difyTest,
		OverallSuccess: configTest.Success && todoTest.Success && (difyTest == nil || difyTest.Success),
		Duration:       time.Since(startTime),
		Timestamp:      time.Now().Format(time.RFC3339),
	}

	// 返回 JSON 字符串
	jsonData, err := json.Marshal(result)
	if err != nil {
		return fmt.Sprintf(`{"success":false,"error":"序列化测试结果失败: %v"}`, err)
	}
	return string(jsonData)
}

// testConfigurationFile 测试配置文件的有效性
func (a *App) testConfigurationFile() *testing.TestItemResult {
	startTime := time.Now()
	result := &testing.TestItemResult{
		Name:     "配置文件验证",
		Success:  false,
		Duration: 0,
	}

	// 获取用户目录和配置文件路径
	homeDir, err := os.UserHomeDir()
	if err != nil {
		result.Error = "无法获取用户主目录"
		result.Details = "系统错误: " + err.Error()
		result.Duration = time.Since(startTime)
		return result
	}

	serverConfigPath := filepath.Join(homeDir, ".to_icalendar", "server.yaml")

	// 检查配置文件是否存在
	if _, err := os.Stat(serverConfigPath); os.IsNotExist(err) {
		result.Error = "配置文件不存在"
		result.Details = "配置文件路径: " + serverConfigPath + "\n请先运行初始化配置"
		result.Duration = time.Since(startTime)
		return result
	}

	// 读取并解析配置文件
	configData, err := os.ReadFile(serverConfigPath)
	if err != nil {
		result.Error = "配置文件读取失败"
		result.Details = "错误详情: " + err.Error() + "\n配置文件路径: " + serverConfigPath
		result.Duration = time.Since(startTime)
		return result
	}

	var config ServerConfig
	if err := yaml.Unmarshal(configData, &config); err != nil {
		result.Error = "配置文件格式错误"
		result.Details = "YAML解析错误: " + err.Error() + "\n请检查配置文件格式是否正确"
		result.Duration = time.Since(startTime)
		return result
	}

	// 验证必需字段
	missingFields := []string{}
	if config.MicrosoftTodo.TenantID == "" {
		missingFields = append(missingFields, "tenant_id (租户ID)")
	}
	if config.MicrosoftTodo.ClientID == "" {
		missingFields = append(missingFields, "client_id (客户端ID)")
	}
	if config.MicrosoftTodo.ClientSecret == "" {
		missingFields = append(missingFields, "client_secret (客户端密钥)")
	}

	if len(missingFields) > 0 {
		result.Error = "Microsoft Todo 配置缺少必需字段: " + strings.Join(missingFields, ", ")
		result.Details = "请在配置文件中填写以下必需字段:\n" + strings.Join(missingFields, "\n") +
			"\n配置文件路径: " + serverConfigPath
		result.Duration = time.Since(startTime)
		return result
	}

	// 检查占位符
	placeholderFields := []string{}
	if config.MicrosoftTodo.TenantID == "YOUR_TENANT_ID" {
		placeholderFields = append(placeholderFields, "tenant_id")
	}
	if config.MicrosoftTodo.ClientID == "YOUR_CLIENT_ID" {
		placeholderFields = append(placeholderFields, "client_id")
	}
	if config.MicrosoftTodo.ClientSecret == "YOUR_CLIENT_SECRET" {
		placeholderFields = append(placeholderFields, "client_secret")
	}

	if len(placeholderFields) > 0 {
		result.Error = "Microsoft Todo 配置包含占位符，需要更新为实际值"
		result.Details = "以下字段仍为默认占位符:\n" + strings.Join(placeholderFields, "\n") +
			"\n请访问 Azure Portal (portal.azure.com) 创建应用注册并获取实际值"
		result.Duration = time.Since(startTime)
		return result
	}

	result.Success = true
	result.Message = "配置文件验证通过"
	result.Duration = time.Since(startTime)
	return result
}

// testMicrosoftTodoService 测试 Microsoft Todo 服务
func (a *App) testMicrosoftTodoService() *testing.TestItemResult {
	startTime := time.Now()
	result := &testing.TestItemResult{
		Name:     "Microsoft Todo 服务测试",
		Success:  false,
		Duration: 0,
	}

	// 获取配置目录和文件
	homeDir, err := os.UserHomeDir()
	if err != nil {
		result.Error = "无法获取用户主目录"
		result.Details = "系统错误: " + err.Error()
		result.Duration = time.Since(startTime)
		return result
	}

	serverConfigPath := filepath.Join(homeDir, ".to_icalendar", "server.yaml")

	// 加载配置文件
	configData, err := os.ReadFile(serverConfigPath)
	if err != nil {
		result.Error = "Microsoft Todo 配置文件读取失败"
		result.Details = "错误详情: " + err.Error() + "\n配置文件路径: " + serverConfigPath
		result.Duration = time.Since(startTime)
		return result
	}

	var config ServerConfig
	if err := yaml.Unmarshal(configData, &config); err != nil {
		result.Error = "Microsoft Todo 配置文件格式错误"
		result.Details = "YAML解析错误: " + err.Error() + "\n请检查配置文件格式"
		result.Duration = time.Since(startTime)
		return result
	}

	// 验证配置完整性
	missingFields := []string{}
	if config.MicrosoftTodo.TenantID == "" {
		missingFields = append(missingFields, "TenantID")
	}
	if config.MicrosoftTodo.ClientID == "" {
		missingFields = append(missingFields, "ClientID")
	}
	if config.MicrosoftTodo.ClientSecret == "" {
		missingFields = append(missingFields, "ClientSecret")
	}

	if len(missingFields) > 0 {
		result.Error = "Microsoft Todo 配置不完整，缺少必需字段: " + strings.Join(missingFields, ", ")
		result.Details = "缺少字段: " + strings.Join(missingFields, ", ")
		result.Duration = time.Since(startTime)
		return result
	}

	// 检查占位符
	placeholderFields := []string{}
	if config.MicrosoftTodo.TenantID == "YOUR_TENANT_ID" {
		placeholderFields = append(placeholderFields, "TenantID")
	}
	if config.MicrosoftTodo.ClientID == "YOUR_CLIENT_ID" {
		placeholderFields = append(placeholderFields, "ClientID")
	}
	if config.MicrosoftTodo.ClientSecret == "YOUR_CLIENT_SECRET" {
		placeholderFields = append(placeholderFields, "ClientSecret")
	}

	if len(placeholderFields) > 0 {
		result.Error = "Microsoft Todo 配置仍使用默认占位符"
		result.Details = "以下字段需要更新为实际的Azure AD凭证:\n" + strings.Join(placeholderFields, "\n")
		result.Duration = time.Since(startTime)
		return result
	}

	// 注意：这里不做实际的API连接测试，只做配置验证
	// 实际的API测试需要更复杂的OAuth流程，这里简化处理
	result.Success = true
	result.Message = "Microsoft Todo 配置验证通过（仅配置检查，未进行API连接测试）"
	result.Duration = time.Since(startTime)
	return result
}

// testDifyService 测试 Dify 服务（使用共享测试器）
func (a *App) testDifyService() *testing.TestItemResult {
	// 获取配置目录和文件
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return &testing.TestItemResult{
			Name:     "Dify 服务测试",
			Success:  false,
			Error:    "无法获取用户主目录: " + err.Error(),
			Duration: 0,
		}
	}

	serverConfigPath := filepath.Join(homeDir, ".to_icalendar", "server.yaml")

	// 加载配置文件
	configData, err := os.ReadFile(serverConfigPath)
	if err != nil {
		return &testing.TestItemResult{
			Name:     "Dify 服务测试",
			Success:  false,
			Error:    "配置文件读取失败: " + err.Error(),
			Details:  "配置文件路径: " + serverConfigPath,
			Duration: 0,
		}
	}

	var config ServerConfig
	if err := yaml.Unmarshal(configData, &config); err != nil {
		return &testing.TestItemResult{
			Name:     "Dify 服务测试",
			Success:  false,
			Error:    "配置文件解析错误: " + err.Error(),
			Details:  "YAML解析错误，请检查配置文件格式",
			Duration: 0,
		}
	}

	// 使用共享测试器进行测试
	difyTester := testing.NewDifyTester()
	return difyTester.TestDifyService(&config.Dify)
}