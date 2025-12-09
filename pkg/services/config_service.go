package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/allanpk716/to_icalendar/pkg/config"
	"github.com/allanpk716/to_icalendar/pkg/logger"
	"github.com/allanpk716/to_icalendar/pkg/models"
)

// ConfigServiceImpl 配置服务实现
type ConfigServiceImpl struct {
	configDir string
	logger    interface{}
}

// NewConfigService 创建配置服务
func NewConfigService(configDir string, logger interface{}) ConfigService {
	return &ConfigServiceImpl{
		configDir: configDir,
		logger:    logger,
	}
}

// Initialize 初始化配置服务
func (cs *ConfigServiceImpl) Initialize(ctx context.Context) error {
	// 确保配置目录存在
	_, err := cs.EnsureConfigDir()
	if err != nil {
		return fmt.Errorf("确保配置目录失败: %w", err)
	}

	logger.Info("配置服务初始化完成，配置目录: %s", cs.configDir)
	return nil
}

// GetConfigDir 获取配置目录
func (cs *ConfigServiceImpl) GetConfigDir() (string, error) {
	return cs.configDir, nil
}

// EnsureConfigDir 确保配置目录存在
func (cs *ConfigServiceImpl) EnsureConfigDir() (string, error) {
	// 如果配置目录为空，使用默认位置
	if cs.configDir == "" {
		usr, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("获取用户主目录失败: %w", err)
		}
		cs.configDir = filepath.Join(usr, ".to_icalendar")
	}

	// 创建配置目录
	if err := os.MkdirAll(cs.configDir, 0755); err != nil {
		return "", fmt.Errorf("创建配置目录失败: %w", err)
	}

	return cs.configDir, nil
}

// CreateConfigTemplates 创建配置文件模板
func (cs *ConfigServiceImpl) CreateConfigTemplates(ctx context.Context, configDir string) (*ConfigResult, error) {
	result := &ConfigResult{
		ConfigDir: configDir,
		Success:   true,
	}

	// 创建配置管理器
	configManager := config.NewConfigManager()

	// 创建服务器配置文件
	serverConfigPath := filepath.Join(configDir, "server.yaml")
	if _, err := os.Stat(serverConfigPath); os.IsNotExist(err) {
		err = configManager.CreateServerConfigTemplate(serverConfigPath)
		if err != nil {
			return &ConfigResult{
				Success: false,
				Message: fmt.Sprintf("创建服务器配置文件失败: %v", err),
			}, err
		}
		result.ServerConfig = serverConfigPath
		logger.Info("✓ 已创建服务器配置文件: %s", serverConfigPath)
	} else {
		result.ServerConfig = serverConfigPath
		logger.Info("✓ 服务器配置文件已存在: %s", serverConfigPath)
	}

	// 创建提醒模板文件
	reminderTemplatePath := filepath.Join(configDir, "reminder.json")
	if _, err := os.Stat(reminderTemplatePath); os.IsNotExist(err) {
		err = configManager.CreateReminderTemplate(reminderTemplatePath)
		if err != nil {
			return &ConfigResult{
				Success: false,
				Message: fmt.Sprintf("创建提醒模板失败: %v", err),
			}, err
		}
		result.ReminderTemplate = reminderTemplatePath
		logger.Info("✓ 已创建提醒模板: %s", reminderTemplatePath)
	} else {
		result.ReminderTemplate = reminderTemplatePath
		logger.Info("✓ 提醒模板已存在: %s", reminderTemplatePath)
	}

	result.Message = "配置文件创建成功"
	return result, nil
}

// LoadServerConfig 加载服务器配置
func (cs *ConfigServiceImpl) LoadServerConfig(ctx context.Context) (*models.ServerConfig, error) {
	configManager := config.NewConfigManager()
	serverConfigPath := filepath.Join(cs.configDir, "server.yaml")

	serverConfig, err := configManager.LoadServerConfig(serverConfigPath)
	if err != nil {
		return nil, fmt.Errorf("加载服务器配置失败: %w", err)
	}

	logger.Info("✓ 服务器配置加载成功")
	return serverConfig, nil
}