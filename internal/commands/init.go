package commands

import (
	"context"
	"fmt"

	"github.com/allanpk716/to_icalendar/internal/services"
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
	// 初始化配置服务
	if err := c.configService.Initialize(ctx); err != nil {
		return ErrorResponse(fmt.Errorf("初始化配置服务失败: %w", err)), nil
	}

	// 确保配置目录存在
	configDir, err := c.configService.EnsureConfigDir()
	if err != nil {
		return ErrorResponse(fmt.Errorf("创建配置目录失败: %w", err)), nil
	}

	// 创建配置模板
	result, err := c.configService.CreateConfigTemplates(ctx, configDir)
	if err != nil {
		return ErrorResponse(fmt.Errorf("创建配置模板失败: %w", err)), nil
	}

	if !result.Success {
		return ErrorResponse(fmt.Errorf(result.Message)), nil
	}

	// 构建成功响应
	metadata := map[string]interface{}{
		"config_dir":         result.ConfigDir,
		"server_config":      result.ServerConfig,
		"reminder_template":  result.ReminderTemplate,
	}

	return SuccessResponse(result, metadata), nil
}

// Validate 验证命令参数
func (c *InitCommand) Validate(args []string) error {
	// init 命令不需要参数
	return nil
}

// ShowSuccessMessage 显示成功消息（用于CLI调用）
func (c *InitCommand) ShowSuccessMessage(metadata map[string]interface{}) {
	fmt.Println("✓ Configuration initialized successfully")

	if configDir, ok := metadata["config_dir"].(string); ok {
		fmt.Printf("  Config directory: %s\n", configDir)
	}

	if serverConfig, ok := metadata["server_config"].(string); ok {
		fmt.Printf("  Server config: %s\n", serverConfig)
	}

	if reminderTemplate, ok := metadata["reminder_template"].(string); ok {
		fmt.Printf("  Reminder template: %s\n", reminderTemplate)
	}

	fmt.Println("\nNext steps:")
	fmt.Println("1. Edit the server configuration file to configure Microsoft Todo and Dify")
	fmt.Println("2. Run 'to_icalendar test' to test connection")
	fmt.Println("3. Run 'to_icalendar upload <reminder-file.json>' to send reminders")
	fmt.Println("4. Run 'to_icalendar clip' to process clipboard content")
	fmt.Println("5. Run 'to_icalendar clip-upload' to process clipboard and upload")
}