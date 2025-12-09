package wails

import (
	"context"

	"github.com/allanpk716/to_icalendar/pkg/app"
	"github.com/allanpk716/to_icalendar/pkg/logger"
	"github.com/allanpk716/to_icalendar/pkg/commands"
)

// DefaultToICalendarBindings 默认ToICalendar绑定实现
type DefaultToICalendarBindings struct {
	executor commands.CommandExecutor
	logger   interface{}
	options  *WailsOptions
}

// NewToICalendarBindings 创建ToICalendar绑定
func NewToICalendarBindings() ToICalendarBindings {
	return &DefaultToICalendarBindings{
		executor: commands.NewCommandExecutor(),
		logger:   logger.GetLogger(),
		options: &WailsOptions{
			EnableNotifications: true,
			ShowProgress:        true,
			LogLevel:            "info",
		},
	}
}

// NewToICalendarBindingsWithOptions 创建带选项的ToICalendar绑定
func NewToICalendarBindingsWithOptions(options *WailsOptions) ToICalendarBindings {
	return &DefaultToICalendarBindings{
		executor: commands.NewCommandExecutor(),
		logger:   logger.GetLogger(),
		options:  options,
	}
}

// InitConfig 初始化配置
func (b *DefaultToICalendarBindings) InitConfig() *commands.ConfigResult {
	result, err := b.executor.InitConfig()
	if err != nil {
		logger.Error("配置初始化失败: %v", err)
		return &commands.ConfigResult{
			Success: false,
			Message: err.Error(),
		}
	}

	if result.Success && b.options.EnableNotifications {
		// 这里可以添加桌面通知
		logger.Info("配置初始化成功")
	}

	return result
}

// CleanCache 清理缓存
func (b *DefaultToICalendarBindings) CleanCache(options *commands.CleanupOptions) *commands.CleanupResult {
	// 使用默认选项如果没有提供
	if options == nil {
		options = &commands.CleanupOptions{
			All: true,
		}
	}

	result, err := b.executor.CleanCache(options)
	if err != nil {
		logger.Error("缓存清理失败: %v", err)
		return &commands.CleanupResult{
			Success: false,
			Message: err.Error(),
		}
	}

	if result.Success && b.options.EnableNotifications {
		logger.Info("缓存清理完成，删除了 %d 个文件", result.TotalFiles)
	}

	return result
}

// ProcessClipboard 处理剪贴板并上传
func (b *DefaultToICalendarBindings) ProcessClipboard() *commands.ProcessClipboardResult {
	result, err := b.executor.ProcessClipboard()
	if err != nil {
		logger.Error("剪贴板处理失败: %v", err)
		return &commands.ProcessClipboardResult{
			Success: false,
			Message: err.Error(),
		}
	}

	if result.Success && b.options.EnableNotifications {
		logger.Info("剪贴板处理成功: %s", result.Title)
	}

	return result
}

// TestConnection 测试连接
func (b *DefaultToICalendarBindings) TestConnection() *TestConnectionResult {
	// 初始化应用以获取配置
	application := app.NewApplication()
	ctx := context.Background()

	if err := application.Initialize(ctx); err != nil {
		return &TestConnectionResult{
			Success: false,
			Message: "应用初始化失败: " + err.Error(),
		}
	}

	// 获取Todo服务并测试连接
	container := application.GetServiceContainer()
	todoService := container.GetTodoService()

	var err = todoService.TestConnection()
	if err != nil {
		return &TestConnectionResult{
			Success: false,
			Message: "Microsoft Todo连接测试失败: " + err.Error(),
		}
	}

	// 获取服务器信息
	serverInfo, err := todoService.GetServerInfo()
	if err != nil {
		logger.Warn("获取服务器信息失败: %v", err)
	}

	return &TestConnectionResult{
		Success: true,
		Message: "Microsoft Todo连接成功",
		Details: serverInfo,
	}
}

// SetOptions 设置Wails选项
func (b *DefaultToICalendarBindings) SetOptions(options *WailsOptions) {
	b.options = options
}

// GetOptions 获取Wails选项
func (b *DefaultToICalendarBindings) GetOptions() *WailsOptions {
	return b.options
}

// SetLogLevel 设置日志级别
func (b *DefaultToICalendarBindings) SetLogLevel(level string) {
	if b.options != nil {
		b.options.LogLevel = level
	}
}
