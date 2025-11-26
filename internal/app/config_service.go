package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/allanpk716/to_icalendar/internal/config"
	"github.com/allanpk716/to_icalendar/internal/models"
	"github.com/allanpk716/to_icalendar/internal/services"
)

// ConfigServiceImpl 配置服务实现
type ConfigServiceImpl struct {
	configManager *config.ConfigManager
}

// NewConfigService 创建配置服务
func NewConfigService() services.ConfigService {
	return &ConfigServiceImpl{
		configManager: config.NewConfigManager(),
	}
}

// Initialize 初始化配置服务
func (s *ConfigServiceImpl) Initialize(ctx context.Context) error {
	return nil // 配置管理器会在使用时自动初始化
}

// GetConfigDir 获取配置目录
func (s *ConfigServiceImpl) GetConfigDir() (string, error) {
	configDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("获取用户目录失败: %w", err)
	}
	configDir = filepath.Join(configDir, ".to_icalendar")

	// 确保配置目录存在
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("创建配置目录失败: %w", err)
	}

	return configDir, nil
}

// EnsureConfigDir 确保配置目录存在
func (s *ConfigServiceImpl) EnsureConfigDir() (string, error) {
	return s.GetConfigDir()
}

// CreateConfigTemplates 创建配置模板
func (s *ConfigServiceImpl) CreateConfigTemplates(ctx context.Context, configDir string) (*services.ConfigResult, error) {
	// 创建配置目录
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return &services.ConfigResult{
			Success: false,
			Message: fmt.Sprintf("创建配置目录失败: %v", err),
		}, nil
	}

	// 检查是否已存在配置文件
	serverConfigPath := filepath.Join(configDir, "server.yaml")
	reminderTemplatePath := filepath.Join(configDir, "reminder.json")

	serverConfigExists := false
	reminderTemplateExists := false

	if _, err := os.Stat(serverConfigPath); err == nil {
		serverConfigExists = true
	}
	if _, err := os.Stat(reminderTemplatePath); err == nil {
		reminderTemplateExists = true
	}

	// 创建服务器配置模板
	if !serverConfigExists {
		serverConfig := &models.ServerConfig{
			MicrosoftTodo: models.MicrosoftTodoConfig{
				TenantID:     "YOUR_TENANT_ID",
				ClientID:     "YOUR_CLIENT_ID",
				ClientSecret: "YOUR_CLIENT_SECRET",
				Timezone:     "Asia/Shanghai",
			},
		}

		if err := s.configManager.SaveServerConfig(serverConfigPath, serverConfig); err != nil {
			return &services.ConfigResult{
				Success: false,
				Message: fmt.Sprintf("保存服务器配置失败: %v", err),
			}, nil
		}
	}

	// 创建提醒模板
	if !reminderTemplateExists {
		reminderTemplate := &models.Reminder{
			Title:       "Meeting Reminder",
			Description: "Attend product review meeting",
			Date:        "2024-12-25",
			Time:        "14:30",
			RemindBefore: "15m",
			Priority:    "medium",
			List:        "Work",
		}

		if err := s.configManager.SaveReminderTemplate(reminderTemplatePath, reminderTemplate); err != nil {
			return &services.ConfigResult{
				Success: false,
				Message: fmt.Sprintf("保存提醒模板失败: %v", err),
			}, nil
		}
	}

	return &services.ConfigResult{
		Success:          true,
		ConfigDir:        configDir,
		ServerConfig:     serverConfigPath,
		ReminderTemplate: reminderTemplatePath,
		Message:          "配置文件创建成功",
	}, nil
}

// LoadServerConfig 加载服务器配置
func (s *ConfigServiceImpl) LoadServerConfig(ctx context.Context) (*models.ServerConfig, error) {
	configDir, err := s.GetConfigDir()
	if err != nil {
		return nil, err
	}

	serverConfigPath := filepath.Join(configDir, "server.yaml")
	serverConfig, err := s.configManager.LoadServerConfig(serverConfigPath)
	if err != nil {
		return nil, fmt.Errorf("加载服务器配置失败: %w", err)
	}

	return serverConfig, nil
}