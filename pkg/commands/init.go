package commands

import (
	"context"
	"fmt"

	"github.com/allanpk716/to_icalendar/pkg/logger"
	"github.com/allanpk716/to_icalendar/pkg/services"
)

// InitCommand 初始化命令
type InitCommand struct {
	*BaseCommand
	configService services.ConfigService
}

// NewInitCommand 创建初始化命令
func NewInitCommand(container ServiceContainer) *InitCommand {
	return &InitCommand{
		BaseCommand:  NewBaseCommand("init", "初始化配置文件"),
		configService: container.GetConfigService(),
	}
}

// Execute 执行初始化命令
func (c *InitCommand) Execute(ctx context.Context, req *CommandRequest) (*CommandResponse, error) {
	logger.Debug("开始执行初始化命令...")

	// 初始化配置服务
	logger.Debug("初始化配置服务...")
	if err := c.configService.Initialize(ctx); err != nil {
		logger.Errorf("初始化配置服务失败: %v", err)
		return ErrorResponse(fmt.Errorf("初始化配置服务失败: %w", err)), nil
	}
	logger.Debug("配置服务初始化完成")

	// 确保配置目录存在
	logger.Debug("确保配置目录存在...")
	configDir, err := c.configService.EnsureConfigDir()
	if err != nil {
		logger.Errorf("创建配置目录失败: %v", err)
		return ErrorResponse(fmt.Errorf("创建配置目录失败: %w", err)), nil
	}
	logger.Debugf("配置目录已创建: %s", configDir)

	// 创建配置模板
	logger.Debug("创建配置模板...")
	result, err := c.configService.CreateConfigTemplates(ctx, configDir)
	if err != nil {
		logger.Errorf("创建配置模板失败: %v", err)
		return ErrorResponse(fmt.Errorf("创建配置模板失败: %w", err)), nil
	}
	logger.Debugf("配置模板创建完成，成功: %t", result.Success)

	if !result.Success {
		logger.Errorf("配置模板创建失败: %s", result.Message)
		return ErrorResponse(fmt.Errorf(result.Message)), nil
	}

	// 构建成功响应
	metadata := map[string]interface{}{
		"config_dir":         result.ConfigDir,
		"server_config":      result.ServerConfig,
		"reminder_template":  result.ReminderTemplate,
	}

	logger.Debug("初始化命令执行完成")
	return SuccessResponse(result, metadata), nil
}

// Validate 验证命令参数
func (c *InitCommand) Validate(args []string) error {
	// init 命令不需要参数
	return nil
}

// ShowSuccessMessage 显示成功消息（用于CLI调用）
func (c *InitCommand) ShowSuccessMessage(metadata map[string]interface{}) {
	logger.Info("✓ Configuration initialized successfully")

	if configDir, ok := metadata["config_dir"].(string); ok {
		logger.Infof("  Config directory: %s", configDir)
		logger.Debugf("配置目录路径详情: %s", configDir)
	}

	if serverConfig, ok := metadata["server_config"].(string); ok {
		logger.Infof("  Server config: %s", serverConfig)
		logger.Debugf("服务器配置文件路径: %s", serverConfig)
	}

	if reminderTemplate, ok := metadata["reminder_template"].(string); ok {
		logger.Infof("  Reminder template: %s", reminderTemplate)
		logger.Debugf("提醒模板文件路径: %s", reminderTemplate)
	}

	logger.Info("\nNext steps:")
	logger.Info("1. Edit the server configuration file to configure Microsoft Todo and Dify")
	logger.Info("2. Run 'to_icalendar test' to test connection")
	logger.Info("3. Run 'to_icalendar upload <reminder-file.json>' to send reminders")
	logger.Info("4. Run 'to_icalendar clip' to process clipboard content")
	logger.Info("5. Run 'to_icalendar clip-upload' to process clipboard and upload")

	logger.Debug("初始化成功信息显示完成")
}