package app

import (
	"context"
	"fmt"
	"time"

	"github.com/allanpk716/to_icalendar/internal/logger"
	"github.com/allanpk716/to_icalendar/internal/microsofttodo"
	"github.com/allanpk716/to_icalendar/internal/models"
	"github.com/allanpk716/to_icalendar/internal/services"
	timezonepkg "github.com/allanpk716/to_icalendar/internal/timezone"
)

// NewTodoService 创建 Todo 服务
func NewTodoService(config *models.ServerConfig, logger interface{}) services.TodoService {
	return &TodoServiceImpl{
		config: config,
		logger: logger,
	}
}

// TodoServiceImpl Todo 服务实现
type TodoServiceImpl struct {
	config *models.ServerConfig
	logger interface{}
}

// CreateTask 创建任务
func (ts *TodoServiceImpl) CreateTask(ctx context.Context, reminder *models.Reminder) error {
	if ts.config == nil {
		return fmt.Errorf("配置未初始化")
	}

	if reminder == nil {
		return fmt.Errorf("提醒对象为空")
	}

	// 验证 Microsoft Todo 配置
	if ts.config.MicrosoftTodo.TenantID == "" ||
		ts.config.MicrosoftTodo.ClientID == "" ||
		ts.config.MicrosoftTodo.ClientSecret == "" ||
		ts.config.MicrosoftTodo.UserEmail == "" {
		return fmt.Errorf("Microsoft Todo 配置不完整")
	}

	// 创建 Todo 客户端
	todoClient, err := microsofttodo.NewSimpleTodoClient(
		ts.config.MicrosoftTodo.TenantID,
		ts.config.MicrosoftTodo.ClientID,
		ts.config.MicrosoftTodo.ClientSecret,
		ts.config.MicrosoftTodo.UserEmail,
	)
	if err != nil {
		return fmt.Errorf("创建 Microsoft Todo 客户端失败: %w", err)
	}

	// 改进的时区处理逻辑（使用timezone工具函数）
	var timezone *time.Location
	if ts.config.MicrosoftTodo.Timezone != "" {
		// 使用timezone工具函数安全加载时区
		timezone = timezonepkg.GetTimezoneLocation(ts.config.MicrosoftTodo.Timezone)
		logger.Infof("使用时区: %s", ts.config.MicrosoftTodo.Timezone)

		// 验证时区是否有效
		if !timezonepkg.IsValidTimezone(ts.config.MicrosoftTodo.Timezone) {
			logger.Warnf("配置的时区 '%s' 可能无效，Windows用户建议使用 'UTC'", ts.config.MicrosoftTodo.Timezone)
			logger.Infof("支持的时区列表: %v", timezonepkg.GetSupportedTimezones())
		}
	} else {
		logger.Warnf("配置中未指定时区，使用系统本地时区")
		timezone = time.Local
	}

	// 使用工作版本的完整时间解析函数
	parsedReminder, parseErr := models.ParseReminderTimeWithConfig(*reminder, timezone, &ts.config.Reminder)
	if parseErr != nil {
		return fmt.Errorf("完整时间解析失败: %w", parseErr)
	}

	// 从解析结果中提取时间信息（使用UTC标准化字段）
	dueDateTime := parsedReminder.DueTimeUTC       // 使用UTC时间
	reminderTime := parsedReminder.AlarmTimeUTC    // 使用UTC时间
	userTimezone := parsedReminder.UserTimezone    // 用户配置的时区

	// 添加详细的时间处理调试日志
	logger.Infof("完整时间处理流程（UTC标准化）:")
	logger.Infof("  输入提醒: %+v", reminder)
	logger.Infof("  用户时区: %s", userTimezone)
	logger.Infof("  本地截止时间: %s", parsedReminder.DueTime.Format("2006-01-02 15:04:05"))
	logger.Infof("  UTC截止时间: %s", parsedReminder.DueTimeUTC.Format("2006-01-02 15:04:05"))
	logger.Infof("  本地提醒时间: %s", parsedReminder.AlarmTime.Format("2006-01-02 15:04:05"))
	logger.Infof("  UTC提醒时间: %s", parsedReminder.AlarmTimeUTC.Format("2006-01-02 15:04:05"))

	importance := 1 // 默认重要性

	// 根据优先级设置重要性
	switch reminder.Priority {
	case models.PriorityHigh:
		importance = 3
	case models.PriorityMedium:
		importance = 2
	case models.PriorityLow:
		importance = 1
	}

	// 获取或创建任务列表
	var listID string
	if reminder.List != "" {
		// 如果指定了列表名称，尝试获取列表ID
		listID, err = todoClient.GetOrCreateTaskList(reminder.List)
		if err != nil {
			return fmt.Errorf("无法创建任务列表 '%s': %w", reminder.List, err)
		}
	}

	// 如果没有指定列表，创建默认列表
	if listID == "" {
		listID, err = todoClient.GetOrCreateTaskList("Tasks")
		if err != nil {
			return fmt.Errorf("无法创建默认任务列表: %w", err)
		}
	}

	// 创建任务（使用UTC时间传递，时区信息用于API转换）
	err = todoClient.CreateTaskWithDetails(
		reminder.Title,
		reminder.Description,
		listID,
		dueDateTime,      // UTC时间
		reminderTime,     // UTC时间
		importance,
		userTimezone,     // 用户配置的时区名称
	)
	if err != nil {
		return fmt.Errorf("创建 Microsoft Todo 任务失败: %w", err)
	}

	return nil
}

// TestConnection 测试连接
func (ts *TodoServiceImpl) TestConnection() error {
	if ts.config == nil {
		return fmt.Errorf("配置未初始化")
	}

	// 验证 Microsoft Todo 配置
	if ts.config.MicrosoftTodo.TenantID == "" ||
		ts.config.MicrosoftTodo.ClientID == "" ||
		ts.config.MicrosoftTodo.ClientSecret == "" ||
		ts.config.MicrosoftTodo.UserEmail == "" {
		return fmt.Errorf("Microsoft Todo 配置不完整")
	}

	// 创建 Todo 客户端
	todoClient, err := microsofttodo.NewSimpleTodoClient(
		ts.config.MicrosoftTodo.TenantID,
		ts.config.MicrosoftTodo.ClientID,
		ts.config.MicrosoftTodo.ClientSecret,
		ts.config.MicrosoftTodo.UserEmail,
	)
	if err != nil {
		return fmt.Errorf("创建 Microsoft Todo 客户端失败: %w", err)
	}

	// 测试连接
	return todoClient.TestConnection()
}

// GetServerInfo 获取服务器信息
func (ts *TodoServiceImpl) GetServerInfo() (map[string]interface{}, error) {
	if ts.config == nil {
		return nil, fmt.Errorf("配置未初始化")
	}

	// 验证 Microsoft Todo 配置
	if ts.config.MicrosoftTodo.TenantID == "" ||
		ts.config.MicrosoftTodo.ClientID == "" ||
		ts.config.MicrosoftTodo.ClientSecret == "" ||
		ts.config.MicrosoftTodo.UserEmail == "" {
		return nil, fmt.Errorf("Microsoft Todo 配置不完整")
	}

	// 创建 Todo 客户端
	todoClient, err := microsofttodo.NewSimpleTodoClient(
		ts.config.MicrosoftTodo.TenantID,
		ts.config.MicrosoftTodo.ClientID,
		ts.config.MicrosoftTodo.ClientSecret,
		ts.config.MicrosoftTodo.UserEmail,
	)
	if err != nil {
		return nil, fmt.Errorf("创建 Microsoft Todo 客户端失败: %w", err)
	}

	// 获取服务器信息
	return todoClient.GetServerInfo()
}

// 注意：parseReminderDateTime 和 parseReminderBeforeTime 函数已被移除
// 现在使用 models.ParseReminderTimeWithConfig 进行完整的时间解析
// 这提供了更好的时区处理、智能提醒功能和错误处理