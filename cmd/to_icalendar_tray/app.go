package main

import (
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"crypto/rand"
	"encoding/hex"

	"github.com/allanpk716/to_icalendar/pkg/app"
	"github.com/allanpk716/to_icalendar/pkg/cache"
	"github.com/allanpk716/to_icalendar/pkg/commands"
	"github.com/allanpk716/to_icalendar/pkg/config"
	"github.com/allanpk716/to_icalendar/pkg/logger"
	"github.com/allanpk716/to_icalendar/pkg/models"
	"github.com/allanpk716/to_icalendar/pkg/testing"
	"github.com/getlantern/systray"
	"gopkg.in/yaml.v3"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed assets/icon.png
var iconData []byte

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

// TaskStatus 任务状态
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
)

// TaskInfo 任务信息
type TaskInfo struct {
	ID        string     `json:"id"`
	Status    TaskStatus `json:"status"`
	Progress  int        `json:"progress"`  // 0-100
	Step      string     `json:"step"`      // 当前步骤描述
	Message   string     `json:"message"`   // 详细信息
	Result    string     `json:"result,omitempty"`    // 最终结果
	Error     string     `json:"error,omitempty"`     // 错误信息
	StartTime time.Time  `json:"start_time"`
	EndTime   time.Time  `json:"end_time,omitempty"`
}

// TaskManager 任务管理器
type TaskManager struct {
	tasks map[string]*TaskInfo
	mutex sync.RWMutex
	ctx   context.Context
}

// NewTaskManager 创建任务管理器
func NewTaskManager(ctx context.Context) *TaskManager {
	return &TaskManager{
		tasks: make(map[string]*TaskInfo),
		ctx:   ctx,
	}
}

// AddTask 添加任务
func (tm *TaskManager) AddTask(task *TaskInfo) {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()
	tm.tasks[task.ID] = task
}

// GetTask 获取任务
func (tm *TaskManager) GetTask(taskID string) (*TaskInfo, error) {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	if task, exists := tm.tasks[taskID]; exists {
		return task, nil
	}
	return nil, fmt.Errorf("任务不存在: %s", taskID)
}

// UpdateTask 更新任务状态
func (tm *TaskManager) UpdateTask(taskID string, status TaskStatus, progress int, step, message, result, error string) {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	if task, exists := tm.tasks[taskID]; exists {
		task.Status = status
		task.Progress = progress
		task.Step = step
		task.Message = message

		if result != "" {
			task.Result = result
		}
		if error != "" {
			task.Error = error
		}

		if status == TaskStatusCompleted || status == TaskStatusFailed {
			task.EndTime = time.Now()
		}

		// 发射状态变化事件
		wailsRuntime.EventsEmit(tm.ctx, "taskStatusChange", map[string]interface{}{
			"taskId":   taskID,
			"status":   string(status),
			"progress": progress,
			"step":     step,
			"message":  message,
		})
	}
}

// generateTaskID 生成任务ID
func generateTaskID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// ClipUploadResult 剪贴板上传结果
type ClipUploadResult struct {
    Success      bool     `json:"success"`
    Title        string   `json:"title"`
    Description  string   `json:"description"`
    Message      string   `json:"message"`
    List         string   `json:"list,omitempty"`
    Priority     string   `json:"priority,omitempty"`
    Error        string   `json:"error,omitempty"`
    ErrorType    string   `json:"errorType,omitempty"`
    CanRetry     bool     `json:"canRetry,omitempty"`
    Suggestions  []string `json:"suggestions,omitempty"`
    Duration     int64    `json:"duration,omitempty"`
    ParsedAnswer string   `json:"parsedAnswer,omitempty"`
}

// App 应用结构
type App struct {
	ctx              context.Context
	taskManager      *TaskManager
	serviceContainer *app.ServiceContainer
	config           *models.ServerConfig
	application      *app.Application
	appIcon          []byte // 应用程序图标
	isWindowVisible  bool   // 窗口可见状态跟踪
	isQuitting       bool   // 退出状态跟踪
	quitOnce         sync.Once // 确保Quit只执行一次
	quitWG           sync.WaitGroup // 等待清理完成
	quitDone         chan struct{} // 退出完成信号
}

// NewApp 创建应用
func NewApp(icon []byte) *App {
	return &App{
		appIcon:         icon,
		isWindowVisible: false,
		isQuitting:      false,
		quitDone:        make(chan struct{}),
	}
}

// OnStartup 启动应用
func (a *App) OnStartup(ctx context.Context) {
	a.ctx = ctx
	a.taskManager = NewTaskManager(ctx)
	a.isWindowVisible = true  // 启动时窗口可见

	// 初始化服务容器
	if err := a.InitializeServiceContainer(); err != nil {
		logger.Errorf("初始化服务容器失败: %v", err)
		// 即使服务容器初始化失败，也继续启动应用，但某些功能可能不可用
	}

	// 初始化CLI版本的应用
	a.application = app.NewApplication()

	// 监听前端WebView导航事件
	wailsRuntime.EventsOn(ctx, "oauthNavigation", func(data ...interface{}) {
		if len(data) > 0 {
			if callbackURL, ok := data[0].(string); ok {
				// 检查是否是OAuth回调URL
				if strings.Contains(callbackURL, "/oauth2/callback") {
					// 解析state
					u, _ := url.Parse(callbackURL)
					state := u.Query().Get("state")
					if state != "" {
						// 处理回调
						result, err := a.HandleOAuthCallback(state, callbackURL)
						if err != nil {
							wailsRuntime.EventsEmit(ctx, "oauthError", err.Error())
						} else {
							wailsRuntime.EventsEmit(ctx, "oauthResult", result)
						}
					}
				}
			}
		}
	})

	// 启动 token 刷新服务
	go a.startTokenRefresher()
}

// OnDomReady DOM准备就绪
func (a *App) OnDomReady(ctx context.Context) {
	// 初始化系统托盘
	a.initSystray()
}

// OnBeforeClose 关闭前
func (a *App) OnBeforeClose(ctx context.Context) (prevent bool) {
	// 如果是用户点击窗口关闭按钮且不是正在退出，隐藏到托盘
	if !a.isQuitting {
		a.HideWindow()
		return true // 阻止窗口关闭，隐藏到托盘
	}

	// 如果是调用Quit()方法触发的关闭，允许正常退出
	return false // 允许退出
}

// OnShutdown 关闭
func (a *App) OnShutdown(ctx context.Context) {
	systray.Quit()
}

// startTokenRefresher 启动 token 刷新服务
func (a *App) startTokenRefresher() {
	// 等待服务容器初始化完成
	for {
		if a.serviceContainer != nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	// 获取 token 刷新服务
	tokenRefresher := a.serviceContainer.GetTokenRefresherService()
	if tokenRefresher != nil {
		// 设置事件监听
		tokenRefresher.SetReauthCallback(func(err error) {
			// 发射事件到前端，显示重新认证提示
			wailsRuntime.EventsEmit(a.ctx, "reauthRequired", map[string]interface{}{
				"message": "Token 已过期，需要重新认证",
				"error":   err.Error(),
			})
		})

		// 启动服务
		if err := tokenRefresher.Start(); err != nil {
			a.sendClipboardLog("error", fmt.Sprintf("启动 token 刷新服务失败: %v", err))
		} else {
			a.sendClipboardLog("info", "Token 刷新服务已启动")
		}
	}
}

// InitializeServiceContainer 初始化服务容器
func (a *App) InitializeServiceContainer() error {
	// 获取配置目录 ~/.to_icalendar
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("获取用户目录失败: %w", err)
	}
	configDir := filepath.Join(homeDir, ".to_icalendar")

	// 加载CLI版本的配置
	configManager := config.NewConfigManager()
	serverConfigPath := filepath.Join(configDir, "server.yaml")
	serverConfig, err := configManager.LoadServerConfig(serverConfigPath)
	if err != nil {
		return fmt.Errorf("加载配置文件失败: %w", err)
	}

	// 初始化缓存管理器
	cacheManager, err := cache.NewUnifiedCacheManager(configDir, logger.GetLogger().GetStdLogger())
	if err != nil {
		return fmt.Errorf("初始化缓存管理器失败: %w", err)
	}

	// 验证缓存管理器是否正确初始化
	if cacheManager != nil {
		// 确保图片缓存目录已正确设置
		imagesDir := cacheManager.GetCacheDir(cache.CacheTypeImages)
		if imagesDir == "" {
			logger.Error("图片缓存目录未正确设置")
		} else {
			logger.Infof("图片缓存目录: %s", imagesDir)
		}
	}

	// 创建服务容器
	a.serviceContainer = app.NewServiceContainer(
		configDir,
		serverConfig,
		cacheManager,
		logger.GetLogger(),
	)

	a.config = serverConfig
	return nil
}

// GetConfigStatus 获取配置状态
func (a *App) GetConfigStatus() map[string]interface{} {
	status := map[string]interface{}{
		"configDir":          "",
		"configExists":       false,
		"configValid":        false,
		"serviceInitialized": false,
		"ready":              false,
		"error":              "",
		"suggestions":        []string{},
	}

	// 获取用户目录
	homeDir, err := os.UserHomeDir()
	if err != nil {
		status["error"] = fmt.Sprintf("无法获取用户目录: %v", err)
		status["suggestions"] = []string{"请检查系统用户配置", "确保应用有足够权限访问用户目录"}
		return status
	}

	configDir := filepath.Join(homeDir, ".to_icalendar")
	serverConfigPath := filepath.Join(configDir, "server.yaml")

	// 更新状态
	status["configDir"] = configDir

	// 检查文件是否存在
	if _, err := os.Stat(serverConfigPath); os.IsNotExist(err) {
		status["error"] = "配置文件不存在"
		status["suggestions"] = []string{
			"运行 'to_icalendar init' 初始化配置",
			"手动创建配置文件: " + serverConfigPath,
		}
		return status
	}
	status["configExists"] = true

	// 尝试加载和验证配置
	configManager := config.NewConfigManager()
	serverConfig, err := configManager.LoadServerConfig(serverConfigPath)
	if err != nil {
		status["error"] = fmt.Sprintf("配置文件格式错误: %v", err)
		status["suggestions"] = []string{
			"检查 YAML 格式是否正确",
			"确保所有必需字段都已填写",
		}
		return status
	}

	// 验证配置内容
	isConfigured := serverConfig.MicrosoftTodo.TenantID != "" &&
		serverConfig.MicrosoftTodo.ClientID != "" &&
		serverConfig.MicrosoftTodo.ClientSecret != ""

	if !isConfigured {
		status["error"] = "配置文件缺少必要字段"
		status["suggestions"] = []string{
			"请配置 Microsoft Todo 相关信息（TenantID, ClientID, ClientSecret）",
			"请配置 Dify API 信息（如需要）",
		}
		return status
	}
	status["configValid"] = true

	// 检查服务是否已初始化
	if a.serviceContainer != nil {
		status["serviceInitialized"] = true
		status["ready"] = true
	} else {
		// 尝试初始化服务容器
		if err := a.InitializeServiceContainer(); err != nil {
			status["error"] = fmt.Sprintf("服务初始化失败: %v", err)
			status["suggestions"] = []string{"请检查配置文件是否正确", "请确保有足够的权限"}
		} else {
			status["serviceInitialized"] = true
			status["ready"] = true
		}
	}

	return status
}

// GetClipboardBase64 获取剪贴板图片的base64编码
func (a *App) GetClipboardBase64() (string, error) {
	// 使用CLI版本的剪贴板服务
	if a.serviceContainer == nil {
		if err := a.InitializeServiceContainer(); err != nil {
			return "", fmt.Errorf("初始化服务容器失败: %w", err)
		}
	}

	clipboardService := a.serviceContainer.GetClipboardService()

	content, err := clipboardService.ReadContent(a.ctx)
	if err != nil {
		return "", fmt.Errorf("读取剪贴板失败: %w", err)
	}

	if content.Type != models.ContentTypeImage {
		return "", fmt.Errorf("剪贴板中没有图片内容")
	}

	return base64.StdEncoding.EncodeToString(content.Image), nil
}

// StartProcessImageToTodo 开始异步处理图片到Todo
func (a *App) StartProcessImageToTodo(imageBase64 string) (string, error) {
	taskID := generateTaskID()

	// 创建任务信息
	taskInfo := &TaskInfo{
		ID:        taskID,
		Status:    TaskStatusPending,
		Progress:  0,
		Step:      "准备开始处理",
		StartTime: time.Now(),
	}

	a.taskManager.AddTask(taskInfo)

	// 启动异步处理
	go a.processImageAsync(taskID, imageBase64)

	return taskID, nil
}

// processImageAsync 异步处理图片
func (a *App) processImageAsync(taskID, imageBase64 string) {
	defer func() {
		if r := recover(); r != nil {
			errMsg := fmt.Sprintf("处理出现异常: %v", r)
			a.taskManager.UpdateTask(taskID, TaskStatusFailed, 0, "处理出现异常", "", "", errMsg)
			a.sendClipboardLog("error", errMsg)
		}
	}()

	// 输入验证
	if imageBase64 == "" {
		a.taskManager.UpdateTask(taskID, TaskStatusFailed, 0, "输入为空", "", "", "base64字符串为空")
		a.sendClipboardLog("error", "输入的base64字符串为空")
		return
	}

	if len(imageBase64) < 100 {
		a.taskManager.UpdateTask(taskID, TaskStatusFailed, 0, "输入无效", "", "", "base64字符串长度异常")
		a.sendClipboardLog("error", "输入的base64字符串长度异常")
		return
	}

	// 记录开始处理
	a.sendClipboardLog("info", fmt.Sprintf("开始解码图片，输入长度: %d", len(imageBase64)))

	// 步骤1：解码图片
	a.taskManager.UpdateTask(taskID, TaskStatusRunning, 10, "正在解码图片...", "", "", "")
	imageData, err := base64.StdEncoding.DecodeString(imageBase64)
	if err != nil {
		detailedError := fmt.Sprintf("base64解码失败: %v, 输入长度: %d", err, len(imageBase64))
		a.taskManager.UpdateTask(taskID, TaskStatusFailed, 0, "解码失败", "", "", detailedError)
		a.sendClipboardLog("error", detailedError)
		return
	}

	a.sendClipboardLog("success", fmt.Sprintf("解码成功，输出长度: %d", len(imageData)))
	a.taskManager.UpdateTask(taskID, TaskStatusRunning, 20, "图片解码完成", "", "", "")

	// 步骤2：调用CLI服务处理
	a.taskManager.UpdateTask(taskID, TaskStatusRunning, 30, "调用CLI服务处理...", "", "", "")

	// 步骤3：上传AI服务
	a.taskManager.UpdateTask(taskID, TaskStatusRunning, 40, "正在上传图片到AI服务...", "", "", "")

	// 获取Dify服务并处理图片
	difyService := a.serviceContainer.GetDifyService()
	difyResponse, err := difyService.ProcessImage(context.Background(), imageData)
	if err != nil {
		a.taskManager.UpdateTask(taskID, TaskStatusFailed, 0, "AI处理失败", "", "", err.Error())
		a.sendClipboardLog("error", fmt.Sprintf("AI处理失败: %v", err))
		return
	}

    a.sendClipboardLog("success", "AI服务调用成功")
    a.taskManager.UpdateTask(taskID, TaskStatusRunning, 60, "AI服务调用成功", "", "", "")

    // 步骤4：解析AI响应
    a.taskManager.UpdateTask(taskID, TaskStatusRunning, 70, "正在解析AI响应...", "", "", "")
    rawAnswer := ""
    if difyResponse.Answer != "" {
        rawAnswer = difyResponse.Answer
    } else if difyResponse.Data != nil && difyResponse.Data.Outputs != nil {
        rawAnswer = difyResponse.Data.Outputs.Text
    }
    if rawAnswer != "" {
        a.sendClipboardLog("info", fmt.Sprintf("AI响应内容: %s", rawAnswer))
    }
    reminder, err := commands.ParseDifyResponseToReminder(difyResponse, "image", "[图片内容]")
    if err != nil {
        a.taskManager.UpdateTask(taskID, TaskStatusFailed, 0, "解析AI响应失败", "", "", err.Error())
        a.sendClipboardLog("error", fmt.Sprintf("解析AI响应失败: %v", err))
        return
	}

	a.sendClipboardLog("info", fmt.Sprintf("解析任务信息: %s", reminder.Title))
	a.taskManager.UpdateTask(taskID, TaskStatusRunning, 80, "AI分析完成", "", "", "")

	// 步骤5：创建Todo任务
	a.taskManager.UpdateTask(taskID, TaskStatusRunning, 90, "正在创建Microsoft Todo任务...", "", "", "")
	todoService := a.serviceContainer.GetTodoService()
	err = todoService.CreateTask(context.Background(), reminder)
	if err != nil {
		a.taskManager.UpdateTask(taskID, TaskStatusFailed, 0, "创建任务失败", "", "", err.Error())
		a.sendClipboardLog("error", fmt.Sprintf("创建Microsoft Todo任务失败: %v", err))
		return
	}

	// 完成处理
    result := &ClipUploadResult{
        Success:     true,
        Title:       reminder.Title,
        Description: reminder.Description,
        Message:     "任务创建成功",
        List:        reminder.List,
        Priority:    string(reminder.Priority),
        Duration:    time.Since(time.Now()).Milliseconds(),
        ParsedAnswer: rawAnswer,
    }

	resultJSON, _ := json.Marshal(result)
	a.taskManager.UpdateTask(taskID, TaskStatusCompleted, 100, "任务创建成功！", string(resultJSON), "", "")
	a.sendClipboardLog("success", "处理完成")
}

// GetTaskStatus 获取任务状态
func (a *App) GetTaskStatus(taskID string) (*TaskInfo, error) {
	return a.taskManager.GetTask(taskID)
}

// sendClipboardLog 发送剪贴板处理日志
func (a *App) sendClipboardLog(logType, message string) {
	logMessage := LogMessage{
		Type:    logType,
		Message: message,
		Time:    time.Now().Format("2006-01-02 15:04:05"),
	}

	wailsRuntime.EventsEmit(a.ctx, "clipboardLog", logMessage)
}

// initSystray 初始化系统托盘
func (a *App) initSystray() {
	// 添加延迟确保Wails完全初始化，避免竞态条件
	go func() {
		time.Sleep(500 * time.Millisecond)
		a.setupSystemTray()
	}()
}

// setupSystemTray 配置系统托盘图标和菜单
func (a *App) setupSystemTray() {
	systray.Run(a.onSystrayReady, a.onSystrayExit)
}

// onSystrayReady 系统托盘准备就绪时调用
func (a *App) onSystrayReady() {
	// 设置图标和标题
	systray.SetIcon(a.appIcon)
	systray.SetTitle("to_icalendar")
	systray.SetTooltip("to_icalendar - Microsoft Todo Reminders")

	// 显示窗口菜单项
	mShow := systray.AddMenuItem("显示窗口", "显示主窗口")
	go func() {
		for range mShow.ClickedCh {
			a.ShowWindow()
		}
	}()

	systray.AddSeparator()

	// 退出菜单项
	mQuit := systray.AddMenuItem("退出", "退出应用程序")
	go func() {
		for range mQuit.ClickedCh {
			a.Quit()
		}
	}()

	// 添加调试输出，确认菜单项创建成功
	println("系统托盘菜单初始化完成")
}

// onSystrayExit 系统托盘退出时调用
func (a *App) onSystrayExit() {
	println("系统托盘清理完成")
}

// ShowWindow 显示窗口
func (a *App) ShowWindow() {
	if !a.isWindowVisible {
		wailsRuntime.WindowShow(a.ctx)
		a.isWindowVisible = true
	}
}

// HideWindow 隐藏窗口
func (a *App) HideWindow() {
	if a.isWindowVisible {
		wailsRuntime.WindowHide(a.ctx)
		a.isWindowVisible = false
	}
}

// Show 显示窗口（别名）
func (a *App) Show() {
	a.ShowWindow()
}

// Hide 隐藏窗口（别名）
func (a *App) Hide() {
	a.HideWindow()
}

// Quit 退出应用程序
func (a *App) Quit() {
	a.quitOnce.Do(func() {
		a.isQuitting = true
		a.quitWG.Add(1)

		// 停止 token 刷新服务
		if a.serviceContainer != nil {
			if tokenRefresher := a.serviceContainer.GetTokenRefresherService(); tokenRefresher != nil {
				if err := tokenRefresher.Stop(); err != nil {
					a.sendClipboardLog("error", fmt.Sprintf("停止 token 刷新服务失败: %v", err))
				} else {
					a.sendClipboardLog("info", "Token 刷新服务已停止")
				}
			}
		}

		// 先停止托盘
		systray.Quit()

		// 关闭CLI版本的应用
		if a.application != nil {
			go func() {
				a.application.Shutdown(a.ctx)
				a.quitWG.Done()
			}()
		} else {
			a.quitWG.Done()
		}

		// 退出Wails应用
		go func() {
			a.quitWG.Wait()
			close(a.quitDone)
			wailsRuntime.Quit(a.ctx)
		}()
	})
}

// StartBrowserOAuth 启动浏览器OAuth认证
func (a *App) StartBrowserOAuth() (map[string]interface{}, error) {
	if a.serviceContainer == nil {
		return nil, fmt.Errorf("服务未初始化")
	}

	// 获取Todo服务
	todoService := a.serviceContainer.GetTodoService()
	if todoService == nil {
		return nil, fmt.Errorf("Todo服务未初始化")
	}

	// 获取SimpleTodoClient
	client := todoService.GetClient()
	if client == nil {
		return nil, fmt.Errorf("无法获取Todo客户端")
	}

	// 在后台goroutine中执行认证，避免阻塞前端
	go func() {
		logger.Info("开始后台浏览器OAuth认证流程")

		result, err := client.AuthenticateWithBrowser(context.Background())
		if err != nil {
			logger.Errorf("浏览器OAuth认证失败: %v", err)
			// 发射错误事件
			if a.ctx != nil {
				wailsRuntime.EventsEmit(a.ctx, "oauthError", map[string]interface{}{
					"error": err.Error(),
					"type":  "browser_auth_failed",
				})
			}
		} else {
			logger.Info("浏览器OAuth认证成功")
			// 发射成功事件
			if a.ctx != nil {
				wailsRuntime.EventsEmit(a.ctx, "oauthResult", map[string]interface{}{
					"success":      result.Success,
					"access_token": result.AccessToken,
					"expires_in":   result.ExpiresIn,
					"type":         "browser_auth_success",
				})
			}
		}
	}()

	return map[string]interface{}{
		"message": "正在启动浏览器进行OAuth认证，请在浏览器中完成登录",
		"type":    "browser_auth_started",
	}, nil
}

// HandleOAuthCallback 处理OAuth回调
func (a *App) HandleOAuthCallback(state, callbackURL string) (map[string]interface{}, error) {
	// 由于OAuth回调处理的复杂性，这里暂时返回一个占位符实现
	// 实际的OAuth处理应该通过 AuthenticateWithBrowser 方法完成
	return map[string]interface{}{
		"success": false,
		"error":   "OAuth callback handling not implemented. Please use AuthenticateWithBrowser instead.",
		"type":    "oauth_callback_not_implemented",
	}, nil
}


// InitConfig 初始化配置
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
		result.Details = map[string]interface{}{
			"error": "系统错误: " + err.Error(),
		}
		result.Duration = time.Since(startTime)
		return result
	}

	serverConfigPath := filepath.Join(homeDir, ".to_icalendar", "server.yaml")

	// 检查配置文件是否存在
	if _, err := os.Stat(serverConfigPath); os.IsNotExist(err) {
		result.Error = "配置文件不存在"
		result.Details = map[string]interface{}{
			"config_path": serverConfigPath,
			"message":     "请先运行初始化配置",
		}
		result.Duration = time.Since(startTime)
		return result
	}

	// 读取并解析配置文件
	configData, err := os.ReadFile(serverConfigPath)
	if err != nil {
		result.Error = "配置文件读取失败"
		result.Details = map[string]interface{}{
			"error":        err.Error(),
			"config_path":  serverConfigPath,
		}
		result.Duration = time.Since(startTime)
		return result
	}

	var config models.ServerConfig
	if err := yaml.Unmarshal(configData, &config); err != nil {
		result.Error = "配置文件格式错误"
		result.Details = map[string]interface{}{
			"error":   "YAML解析错误: " + err.Error(),
			"message": "请检查配置文件格式是否正确",
		}
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
		result.Details = map[string]interface{}{
			"missing_fields": missingFields,
			"config_path":    serverConfigPath,
			"message":        "请在配置文件中填写以上必需字段",
		}
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
		result.Details = map[string]interface{}{
			"placeholder_fields": placeholderFields,
			"message":            "请访问 Azure Portal (portal.azure.com) 创建应用注册并获取实际值",
		}
		result.Duration = time.Since(startTime)
		return result
	}

	result.Success = true
	result.Message = "配置文件验证通过"
	result.Duration = time.Since(startTime)
	return result
}

// testMicrosoftTodoService 测试 Microsoft Todo 服务（使用新的测试器）
func (a *App) testMicrosoftTodoService() *testing.TestItemResult {
	// 获取配置目录和文件
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return &testing.TestItemResult{
			Name:     "Microsoft Todo 服务测试",
			Success:  false,
			Error:    "无法获取用户主目录: " + err.Error(),
			Duration: 0,
		}
	}

	serverConfigPath := filepath.Join(homeDir, ".to_icalendar", "server.yaml")

	// 创建 TodoTester
	tester, err := testing.NewTodoTester(serverConfigPath)
	if err != nil {
		return &testing.TestItemResult{
			Name:     "Microsoft Todo 服务测试",
			Success:  false,
			Error:    "创建测试器失败: " + err.Error(),
			Duration: 0,
		}
	}

	// 设置日志回调
	tester.SetLogCallback(func(level, message string) {
		a.sendTestLog(level, message)
	})

	// 执行连接测试
	result := tester.TestConnection()
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
			Details: map[string]interface{}{
				"config_path": serverConfigPath,
			},
			Duration: 0,
		}
	}

	var config models.ServerConfig
	if err := yaml.Unmarshal(configData, &config); err != nil {
		return &testing.TestItemResult{
			Name:     "Dify 服务测试",
			Success:  false,
			Error:    "配置文件解析错误: " + err.Error(),
			Details: map[string]interface{}{
				"message": "YAML解析错误，请检查配置文件格式",
			},
			Duration: 0,
		}
	}

	// 转换 models.DifyConfig 到 testing.DifyConfig
	testingDifyConfig := &testing.DifyConfig{
		APIEndpoint: config.Dify.APIEndpoint,
		APIKey:      config.Dify.APIKey,
		Timeout:     config.Dify.Timeout,
	}

	// 使用共享测试器进行测试
	difyTester := testing.NewDifyTester()
	return difyTester.TestDifyService(testingDifyConfig)
}

// sendTestLog 发送测试日志到前端
func (a *App) sendTestLog(level, message string) {
	logMsg := &LogMessage{
		Type:    level,
		Message: message,
		Time:    time.Now().Format("2006-01-02 15:04:05"),
	}

	// 发送事件到前端
	wailsRuntime.EventsEmit(a.ctx, "testLog", logMsg)
}
