package testing

import (
	"fmt"
	"time"

	"github.com/allanpk716/to_icalendar/pkg/models"
	svcs "github.com/allanpk716/to_icalendar/pkg/services"
)

// TodoTester Microsoft Todo 连接测试器
type TodoTester struct {
	config      *models.ServerConfig
	configPath  string
	logCallback func(level, message string) // 可选的日志回调函数
}

// NewTodoTester 创建 Todo 测试器
func NewTodoTester(configPath string) (*TodoTester, error) {
	// 加载配置
	config, err := ValidateConfigFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("加载配置失败: %w", err)
	}

	return &TodoTester{
		config:     config,
		configPath: configPath,
	}, nil
}

// NewTodoTesterWithConfig 使用已加载的配置创建测试器
func NewTodoTesterWithConfig(config *models.ServerConfig, configPath string) *TodoTester {
	return &TodoTester{
		config:     config,
		configPath: configPath,
	}
}

// SetLogCallback 设置日志回调函数
func (t *TodoTester) SetLogCallback(callback func(level, message string)) {
	t.logCallback = callback
}

// log 发送日志
func (t *TodoTester) log(level, message string) {
	if t.logCallback != nil {
		t.logCallback(level, message)
	}
}

// TestConfiguration 测试配置（不包括API连接）
func (t *TodoTester) TestConfiguration() *TestItemResult {
	result := &TestItemResult{
		Name:        "Microsoft Todo 配置",
		Success:     false,
		Message:     "",
		Duration:    0,
		Details:     make(map[string]interface{}),
	}

	startTime := time.Now()

	// 检查配置文件
	t.log("info", "检查配置文件...")
	config, err := LoadConfigFromFile(t.configPath)
	if err != nil {
		result.Message = fmt.Sprintf("配置文件加载失败: %v", err)
		result.Duration = time.Since(startTime)
		return result
	}

	// 验证配置
	t.log("info", "验证配置参数...")
	if err := ValidateMicrosoftTodoConfig(&config.MicrosoftTodo); err != nil {
		result.Message = fmt.Sprintf("配置验证失败: %v", err)
		result.Duration = time.Since(startTime)
		return result
	}

	// 记录配置信息（隐藏敏感信息）
	t.log("info", "配置验证通过")
	t.log("debug", fmt.Sprintf("租户ID: %s...", config.MicrosoftTodo.TenantID[:min(8, len(config.MicrosoftTodo.TenantID))]))
	t.log("debug", fmt.Sprintf("客户端ID: %s...", config.MicrosoftTodo.ClientID[:min(8, len(config.MicrosoftTodo.ClientID))]))
	if config.MicrosoftTodo.UserEmail != "" {
		t.log("debug", fmt.Sprintf("用户邮箱: %s", config.MicrosoftTodo.UserEmail))
	}

	result.Success = true
	result.Message = "配置文件验证通过"
	result.Duration = time.Since(startTime)
	// 确保 Details 是 map[string]interface{}
	if result.Details == nil {
		result.Details = make(map[string]interface{})
	}
	result.Details["tenant_id"] = config.MicrosoftTodo.TenantID
	result.Details["client_id"] = config.MicrosoftTodo.ClientID
	result.Details["user_email"] = config.MicrosoftTodo.UserEmail
	result.Details["timezone"] = config.MicrosoftTodo.Timezone

	return result
}

// TestConnection 测试 Microsoft Todo API 连接
func (t *TodoTester) TestConnection() *TestItemResult {
	result := &TestItemResult{
		Name:        "Microsoft Todo 连接",
		Success:     false,
		Message:     "",
		Duration:    0,
		Details:     make(map[string]interface{}),
	}

	startTime := time.Now()

	// 首先测试配置
	t.log("info", "开始测试 Microsoft Todo API 连接...")
	configResult := t.TestConfiguration()
	if !configResult.Success {
		result.Message = configResult.Message
		result.Duration = time.Since(startTime)
		return result
	}

	// 检查是否缺少 UserEmail（某些场景下可选）
	if t.config.MicrosoftTodo.UserEmail == "" {
		t.log("warn", "未配置用户邮箱，将使用应用程序权限模式")
	}

	// 创建 Todo 服务
	t.log("info", "创建 Microsoft Todo 服务...")
	logger := &MockLogger{} // 创建一个简单的日志器
	todoService := svcs.NewTodoService(t.config, logger)

	// 测试连接
	t.log("info", "连接到 Microsoft Graph API...")
	err = todoService.TestConnection()
	if err != nil {
		result.Message = fmt.Sprintf("API 连接失败: %v", err)
		result.Duration = time.Since(startTime)
		result.ErrorType = "api_connection_error"
		return result
	}

	// 获取服务信息
	t.log("info", "获取服务信息...")
	serverInfo, err := t.getServerInfo(todoService)
	if err != nil {
		result.Message = fmt.Sprintf("获取服务信息失败: %v", err)
		result.Duration = time.Since(startTime)
		result.ErrorType = "server_info_error"
		return result
	}

	t.log("info", "Microsoft Todo 连接测试成功！")
	result.Success = true
	result.Message = "Microsoft Todo API 连接成功"
	result.Duration = time.Since(startTime)
	result.Details["server_info"] = serverInfo

	return result
}

// TestAll 执行完整的测试流程
func (t *TodoTester) TestAll() *TestResult {
	return &TestResult{
		Items: []*TestItemResult{
			t.TestConfiguration(),
			t.TestConnection(),
		},
		TestedAt: time.Now(),
	}
}

// getServerInfo 获取服务器信息
func (t *TodoTester) getServerInfo(todoService svcs.TodoService) (string, error) {
	// TestConnection 已经成功，说明连接正常
	// 由于没有公开的方法获取用户信息，我们返回基本信息
	serverInfo := "Microsoft Graph API 连接成功"

	if t.config.MicrosoftTodo.UserEmail != "" {
		serverInfo += fmt.Sprintf(" (用户: %s)", t.config.MicrosoftTodo.UserEmail)
	} else {
		serverInfo += " (应用程序权限模式)"
	}

	return serverInfo, nil
}

// MockLogger 简单的日志器实现
type MockLogger struct{}

func (m *MockLogger) Debug(args ...interface{})                 { fmt.Print(args...) }
func (m *MockLogger) Debugf(format string, args ...interface{}) { fmt.Printf(format+"\n", args...) }
func (m *MockLogger) Info(args ...interface{})                  { fmt.Print(args...) }
func (m *MockLogger) Infof(format string, args ...interface{})  { fmt.Printf(format+"\n", args...) }
func (m *MockLogger) Warn(args ...interface{})                  { fmt.Print(args...) }
func (m *MockLogger) Warnf(format string, args ...interface{})  { fmt.Printf(format+"\n", args...) }
func (m *MockLogger) Error(args ...interface{})                 { fmt.Print(args...) }
func (m *MockLogger) Errorf(format string, args ...interface{}) { fmt.Printf(format+"\n", args...) }
func (m *MockLogger) Fatal(args ...interface{})                 { fmt.Print(args...) }
func (m *MockLogger) Fatalf(format string, args ...interface{}) { fmt.Printf(format+"\n", args...) }

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}