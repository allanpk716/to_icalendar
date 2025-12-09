package commands

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/allanpk716/to_icalendar/pkg/config"
	"github.com/allanpk716/to_icalendar/pkg/logger"
)

const (
	appName = "to_icalendar"
)

// TestCommand 测试命令
type TestCommand struct {
	*BaseCommand
	container ServiceContainer
}

// TestItemResult 测试项结果
type TestItemResult struct {
	Name      string        `json:"name"`
	Success   bool          `json:"success"`
	Message   string        `json:"message"`
	Error     string        `json:"error,omitempty"`
	Details   interface{}   `json:"details,omitempty"`
	Duration  time.Duration `json:"duration"`
}

// TestResult 测试结果
type TestResult struct {
	ConfigTest     *TestItemResult `json:"config_test"`
	TodoTest       *TestItemResult `json:"todo_test"`
	DifyTest       *TestItemResult `json:"dify_test,omitempty"`
	OverallSuccess bool            `json:"overall_success"`
	Duration       time.Duration   `json:"duration"`
}

// NewTestCommand 创建测试命令
func NewTestCommand(container ServiceContainer) *TestCommand {
	return &TestCommand{
		BaseCommand: NewBaseCommand("test", "测试系统连接和配置"),
		container:   container,
	}
}

// Execute 执行测试命令
func (c *TestCommand) Execute(ctx context.Context, req *CommandRequest) (*CommandResponse, error) {
	logger.Info("🔍 开始系统诊断测试...")
	startTime := time.Now()

	result := &TestResult{}

	// 1. 配置文件验证
	logger.Debug("开始配置文件验证...")
	configTest := c.testConfigurationFile(ctx)
	result.ConfigTest = configTest
	if !configTest.Success {
		result.OverallSuccess = false
		result.Duration = time.Since(startTime)
		logger.Error("❌ 配置文件验证失败，停止后续测试")
		return ErrorResponse(&configTestError{Message: configTest.Error}), nil
	}

	// 2. Microsoft Todo 服务测试
	logger.Debug("开始 Microsoft Todo 服务测试...")
	todoTest := c.testMicrosoftTodoService(ctx)
	result.TodoTest = todoTest
	if !todoTest.Success {
		result.OverallSuccess = false
		result.Duration = time.Since(startTime)
		logger.Error("❌ Microsoft Todo 服务测试失败，停止后续测试")
		return ErrorResponse(&todoTestError{Message: todoTest.Error}), nil
	}

	// 3. Dify 服务测试
	logger.Debug("开始 Dify 服务测试...")
	difyTest := c.testDifyService(ctx)
	result.DifyTest = difyTest

	// 计算总体结果
	result.OverallSuccess = configTest.Success && todoTest.Success && (difyTest == nil || difyTest.Success)
	result.Duration = time.Since(startTime)

	if result.OverallSuccess {
		logger.Info("✅ 所有测试通过，系统运行正常")
		return SuccessResponse(result, map[string]interface{}{
			"test_completed": true,
			"duration":       result.Duration,
		}), nil
	}

	return ErrorResponse(&overallTestError{Message: "部分测试失败"}), nil
}

// Validate 验证命令参数
func (c *TestCommand) Validate(args []string) error {
	// test 命令不需要参数
	return nil
}

// ShowTestResult 显示测试结果（用于CLI调用）
func (c *TestCommand) ShowTestResult(data interface{}, metadata map[string]interface{}) {
	result, ok := data.(*TestResult)
	if !ok {
		logger.Error("❌ 无效的测试结果数据")
		return
	}

	// 显示配置文件测试结果
	c.showTestItemResult("📋 配置文件验证", result.ConfigTest)

	// 显示 Microsoft Todo 测试结果
	c.showTestItemResult("🔗 Microsoft Todo 服务测试", result.TodoTest)

	// 显示 Dify 测试结果（如果存在）
	if result.DifyTest != nil {
		c.showTestItemResult("🤖 Dify 服务测试", result.DifyTest)
	}

	// 显示总结
	c.showTestSummary(result)
}

// showTestItemResult 显示单项测试结果
func (c *TestCommand) showTestItemResult(title string, result *TestItemResult) {
	logger.Infof("\n%s", title)
	if result.Success {
		logger.Info("✅ 测试通过")
		if result.Message != "" {
			logger.Infof("   %s", result.Message)
		}
	} else {
		logger.Error("❌ 测试失败")
		if result.Error != "" {
			logger.Errorf("   错误: %s", result.Error)
		}
	}

	// Debug 模式下显示详细信息
	if result.Details != nil {
		logger.Debugf("   详细信息: %+v", result.Details)
	}
	logger.Debugf("   耗时: %v", result.Duration)
}

// showTestSummary 显示测试总结
func (c *TestCommand) showTestSummary(result *TestResult) {
	logger.Infof("\n📈 测试报告总结")
	logger.Infof("总耗时: %v", result.Duration)

	if result.OverallSuccess {
		logger.Info("✅ 所有测试通过，系统运行正常")
	} else {
		logger.Error("❌ 部分测试失败，请检查上述错误信息")
	}
}

// testConfigurationFile 测试配置文件
func (c *TestCommand) testConfigurationFile(ctx context.Context) *TestItemResult {
	startTime := time.Now()
	result := &TestItemResult{
		Name:     "配置文件验证",
		Success:  false,
		Duration: 0,
	}

	logger.Debug("获取用户配置目录...")
	homeDir, err := os.UserHomeDir()
	if err != nil {
		result.Error = err.Error()
		result.Duration = time.Since(startTime)
		logger.Errorf("获取用户目录失败: %v", err)
		return result
	}
	logger.Debugf("用户目录: %s", homeDir)

	configDir := filepath.Join(homeDir, ".to_icalendar")
	serverConfigPath := filepath.Join(configDir, "server.yaml")
	logger.Debugf("配置文件路径: %s", serverConfigPath)

	// 检查配置文件是否存在
	logger.Debug("检查配置文件是否存在...")
	if _, err := os.Stat(serverConfigPath); os.IsNotExist(err) {
		result.Error = "配置文件不存在"
		result.Message = serverConfigPath
		result.Duration = time.Since(startTime)
		logger.Errorf("配置文件不存在: %s", serverConfigPath)
		logger.Infof("💡 请先运行 '%s init' 初始化配置", appName)
		return result
	}
	logger.Info("✅ 配置文件存在")
	logger.Debugf("配置文件路径: %s", serverConfigPath)

	// 创建配置管理器并加载配置
	logger.Debug("创建配置管理器并加载配置...")
	configManager := config.NewConfigManager()
	config, err := configManager.LoadServerConfig(serverConfigPath)
	if err != nil {
		result.Error = err.Error()
		result.Duration = time.Since(startTime)
		logger.Errorf("配置文件格式错误: %v", err)
		return result
	}
	logger.Info("✅ YAML 格式正确")
	logger.Debugf("配置加载成功: %+v", config)

	// 验证必需字段
	logger.Debug("验证必需字段...")
	if config.MicrosoftTodo.TenantID == "" || config.MicrosoftTodo.ClientID == "" || config.MicrosoftTodo.ClientSecret == "" {
		result.Error = "Microsoft Todo 配置缺少必需字段"
		result.Duration = time.Since(startTime)
		logger.Error("❌ Microsoft Todo 配置缺少必需字段")
		return result
	}
	logger.Info("✅ 必需字段完整")

	// 检查占位符
	logger.Debug("检查配置占位符...")
	if config.MicrosoftTodo.TenantID == "YOUR_TENANT_ID" {
		result.Error = "TenantID 仍是占位符，请更新为实际值"
		result.Duration = time.Since(startTime)
		logger.Error("❌ TenantID 仍是占位符，请更新为实际值")
		return result
	}
	if config.MicrosoftTodo.ClientID == "YOUR_CLIENT_ID" {
		result.Error = "ClientID 仍是占位符，请更新为实际值"
		result.Duration = time.Since(startTime)
		logger.Error("❌ ClientID 仍是占位符，请更新为实际值")
		return result
	}
	if config.MicrosoftTodo.ClientSecret == "YOUR_CLIENT_SECRET" {
		result.Error = "ClientSecret 仍是占位符，请更新为实际值"
		result.Duration = time.Since(startTime)
		logger.Error("❌ ClientSecret 仍是占位符，请更新为实际值")
		return result
	}

	result.Success = true
	result.Message = "配置文件验证通过"
	result.Duration = time.Since(startTime)
	logger.Debug("配置文件验证完成")
	return result
}

// testMicrosoftTodoService 测试 Microsoft Todo 服务
func (c *TestCommand) testMicrosoftTodoService(ctx context.Context) *TestItemResult {
	startTime := time.Now()
	result := &TestItemResult{
		Name:     "Microsoft Todo 服务测试",
		Success:  false,
		Duration: 0,
	}

	logger.Debug("获取 TodoService 实例...")
	todoService := c.container.GetTodoService()

	logger.Debug("开始测试连接...")
	if err := todoService.TestConnection(); err != nil {
		result.Error = err.Error()
		result.Duration = time.Since(startTime)
		logger.Errorf("Microsoft Todo 连接失败: %v", err)
		logger.Debugf("连接错误详情: %+v", err)
		return result
	}

	logger.Info("✅ 配置验证通过")
	logger.Info("✅ 服务连接成功")

	// 尝试获取服务信息
	logger.Debug("获取服务信息...")
	if serverInfo, err := todoService.GetServerInfo(); err == nil {
		logger.Info("📊 服务信息：连接正常")
		result.Details = serverInfo
		logger.Debugf("服务信息详情: %+v", serverInfo)
	}

	result.Success = true
	result.Message = "Microsoft Todo 服务连接正常"
	result.Duration = time.Since(startTime)
	logger.Debug("Microsoft Todo 服务测试完成")
	return result
}

// testDifyService 测试 Dify 服务
func (c *TestCommand) testDifyService(ctx context.Context) *TestItemResult {
	startTime := time.Now()
	result := &TestItemResult{
		Name:     "Dify 服务测试",
		Success:  false,
		Duration: 0,
	}

	logger.Debug("获取 DifyService 实例...")
	difyService := c.container.GetDifyService()

	// 如果 Dify 未配置，跳过测试
	if difyService == nil {
		result.Success = true // 跳过不算失败
		result.Message = "Dify 服务未配置，跳过测试"
		result.Duration = time.Since(startTime)
		logger.Info("⏸️ Dify 服务未配置，跳过测试")
		return result
	}

	// 验证配置
	logger.Debug("验证 Dify 配置...")
	if err := difyService.ValidateConfig(); err != nil {
		result.Error = err.Error()
		result.Duration = time.Since(startTime)
		logger.Errorf("Dify 配置验证失败: %v", err)
		return result
	}
	logger.Info("✅ 配置验证通过")

	// 测试连接
	logger.Debug("测试 Dify 连接...")
	if err := difyService.TestConnection(); err != nil {
		result.Error = err.Error()
		result.Duration = time.Since(startTime)
		logger.Errorf("Dify 连接失败: %v", err)
		return result
	}

	result.Success = true
	result.Message = "Dify API 端点连接可达"
	result.Duration = time.Since(startTime)
	logger.Info("✅ API 端点连接可达")
	logger.Debug("Dify 服务测试完成")
	return result
}

// 自定义错误类型
type configTestError struct {
	Message string
}

func (e *configTestError) Error() string {
	return e.Message
}

type todoTestError struct {
	Message string
}

func (e *todoTestError) Error() string {
	return e.Message
}

type overallTestError struct {
	Message string
}

func (e *overallTestError) Error() string {
	return e.Message
}