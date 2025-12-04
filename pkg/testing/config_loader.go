package testing

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/allanpk716/to_icalendar/internal/models"
	"gopkg.in/yaml.v3"
)

// ConfigLoader 提供统一的配置加载和验证功能
type ConfigLoader struct {
	configPath string
}

// NewConfigLoader 创建配置加载器
func NewConfigLoader(configPath string) *ConfigLoader {
	return &ConfigLoader{
		configPath: configPath,
	}
}

// LoadConfigFromFile 从文件加载配置
func (cl *ConfigLoader) LoadConfigFromFile() (*models.ServerConfig, error) {
	return LoadConfigFromFile(cl.configPath)
}

// LoadConfigFromFile 从指定路径加载配置文件
func LoadConfigFromFile(configPath string) (*models.ServerConfig, error) {
	// 检查文件是否存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("配置文件不存在: %s", configPath)
	}

	// 读取文件
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 解析YAML
	var config models.ServerConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("解析YAML配置失败: %w", err)
	}

	return &config, nil
}

// ValidateConfigFile 验证配置文件
func (cl *ConfigLoader) ValidateConfigFile() (*models.ServerConfig, error) {
	return ValidateConfigFile(cl.configPath)
}

// ValidateConfigFile 验证指定路径的配置文件
func ValidateConfigFile(configPath string) (*models.ServerConfig, error) {
	config, err := LoadConfigFromFile(configPath)
	if err != nil {
		return nil, err
	}

	// 验证 Microsoft Todo 配置
	if err := ValidateMicrosoftTodoConfig(&config.MicrosoftTodo); err != nil {
		return nil, fmt.Errorf("Microsoft Todo 配置验证失败: %w", err)
	}

	// 设置默认值
	if config.MicrosoftTodo.Timezone == "" {
		config.MicrosoftTodo.Timezone = "UTC"
	}

	return config, nil
}

// ValidateMicrosoftTodoConfig 验证 Microsoft Todo 配置
func ValidateMicrosoftTodoConfig(config *models.MicrosoftTodoConfig) error {
	// 检查必需字段
	if config.TenantID == "" {
		return fmt.Errorf("缺少 tenant_id")
	}
	if config.ClientID == "" {
		return fmt.Errorf("缺少 client_id")
	}
	if config.ClientSecret == "" {
		return fmt.Errorf("缺少 client_secret")
	}

	// 检查占位符
	if err := CheckPlaceholders(config); err != nil {
		return err
	}

	return nil
}

// CheckPlaceholders 检查配置是否仍使用占位符
func CheckPlaceholders(config *models.MicrosoftTodoConfig) error {
	const (
		placeholderTenantID     = "YOUR_TENANT_ID"
		placeholderClientID     = "YOUR_CLIENT_ID"
		placeholderClientSecret = "YOUR_CLIENT_SECRET"
	)

	if config.TenantID == placeholderTenantID {
		return fmt.Errorf("tenant_id 仍为占位符，请设置为实际的 Azure 租户 ID")
	}
	if config.ClientID == placeholderClientID {
		return fmt.Errorf("client_id 仍为占位符，请设置为实际的应用程序客户端 ID")
	}
	if config.ClientSecret == placeholderClientSecret {
		return fmt.Errorf("client_secret 仍为占位符，请设置为实际的客户端密钥")
	}

	return nil
}

// GetDefaultConfigPath 获取默认配置文件路径
func GetDefaultConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("获取用户目录失败: %w", err)
	}

	return filepath.Join(homeDir, ".to_icalendar", "server.yaml"), nil
}

// EnsureConfigDir 确保配置目录存在
func EnsureConfigDir(configPath string) error {
	dir := filepath.Dir(configPath)
	return os.MkdirAll(dir, 0755)
}