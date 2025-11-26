package app

import (
	"context"
	"fmt"
	"time"

	"github.com/allanpk716/to_icalendar/internal/microsofttodo"
	"github.com/allanpk716/to_icalendar/internal/models"
	"github.com/allanpk716/to_icalendar/internal/services"
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

	// 解析时间和优先级
	dueDateTime, err := parseReminderDateTime(reminder.Date, reminder.Time)
	if err != nil {
		return fmt.Errorf("解析日期时间失败: %w", err)
	}

	reminderTime, err := parseReminderBeforeTime(reminder.RemindBefore)
	if err != nil {
		// 如果解析失败，使用默认提醒时间
		reminderTime = time.Time{}
	}

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

	// 创建任务
	err = todoClient.CreateTaskWithDetails(
		reminder.Title,
		reminder.Description,
		listID,
		dueDateTime,
		reminderTime,
		importance,
		ts.config.MicrosoftTodo.Timezone,
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

// parseReminderDateTime 解析提醒日期时间
func parseReminderDateTime(dateStr, timeStr string) (time.Time, error) {
	if dateStr == "" {
		return time.Time{}, fmt.Errorf("日期为空")
	}

	// 组合日期和时间
	var datetimeStr string
	if timeStr != "" {
		datetimeStr = fmt.Sprintf("%s %s", dateStr, timeStr)
	} else {
		datetimeStr = dateStr
	}

	// 尝试常见的时间格式
	formats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, datetimeStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("无法解析日期时间格式: %s", datetimeStr)
}

// parseReminderBeforeTime 解析提前提醒时间
func parseReminderBeforeTime(remindBefore string) (time.Time, error) {
	if remindBefore == "" {
		return time.Time{}, nil
	}

	// 简单的时间解析，支持 m/h/d 后缀
	if len(remindBefore) < 2 {
		return time.Time{}, fmt.Errorf("无效的提醒时间格式: %s", remindBefore)
	}

	durationStr := remindBefore[:len(remindBefore)-1]
	unit := remindBefore[len(remindBefore)-1]

	var duration time.Duration
	switch unit {
	case 'm', 'M':
		// 分钟
		if minutes, err := time.ParseDuration(durationStr + "m"); err == nil {
			duration = minutes
		}
	case 'h', 'H':
		// 小时
		if hours, err := time.ParseDuration(durationStr + "h"); err == nil {
			duration = hours
		}
	case 'd', 'D':
		// 天
		if days, err := time.ParseDuration(durationStr + "d"); err == nil {
			duration = days
		}
	default:
		return time.Time{}, fmt.Errorf("不支持的时间单位: %c", unit)
	}

	if duration <= 0 {
		return time.Time{}, fmt.Errorf("无效的提醒时间: %s", remindBefore)
	}

	// 返回当前时间减去提前时间（这表示要在指定时间之前提醒）
	return time.Now().Add(-duration), nil
}