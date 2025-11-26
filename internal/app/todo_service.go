package app

import (
	"context"
	"fmt"

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
	err = todoClient.TestConnection()
	if err != nil {
		return fmt.Errorf("Microsoft Todo 连接测试失败: %w", err)
	}

	// 创建任务（这里需要根据实际需求实现）
	// 目前返回成功，具体实现将在 clip-upload 命令中处理
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