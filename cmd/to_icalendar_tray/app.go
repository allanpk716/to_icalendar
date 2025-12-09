package main

import (
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
	"github.com/getlantern/systray"
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
}

// NewApp 创建应用
func NewApp() *App {
	return &App{}
}

// OnStartup 启动应用
func (a *App) OnStartup(ctx context.Context) {
	a.ctx = ctx
	a.taskManager = NewTaskManager(ctx)

	// 初始化CLI版本的应用
	a.application = app.NewApplication()
}

// OnDomReady DOM准备就绪
func (a *App) OnDomReady(ctx context.Context) {
	// 初始化系统托盘
	a.initSystray()
}

// OnBeforeClose 关闭前
func (a *App) OnBeforeClose(ctx context.Context) (prevent bool) {
	// 关闭CLI版本的应用
	if a.application != nil {
		a.application.Shutdown(ctx)
	}
	return false
}

// OnShutdown 关闭
func (a *App) OnShutdown(ctx context.Context) {
	systray.Quit()
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
	// 暂时禁用系统托盘以避免崩溃
	// TODO: 修复系统托盘图标问题后重新启用
	/*
	systray.SetIcon(iconData)
	systray.SetTitle("To iCalendar")
	systray.SetTooltip("Microsoft Todo 任务管理工具")

	// 设置菜单
	mQuit := systray.AddMenuItem("退出", "退出程序")

	go func() {
		<-mQuit.ClickedCh
		wailsRuntime.Quit(a.ctx)
	}()
	*/
}
