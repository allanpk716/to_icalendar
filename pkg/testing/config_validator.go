package testing

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// ConfigValidator 配置文件验证器
type ConfigValidator struct{}

// NewConfigValidator 创建新的配置验证器
func NewConfigValidator() *ConfigValidator {
	return &ConfigValidator{}
}

// ValidateConfigFile 验证配置文件的有效性
func (cv *ConfigValidator) ValidateConfigFile() (*ConfigFileResult, error) {
	startTime := time.Now()
	result := &ConfigFileResult{
		Name:     "配置文件验证",
		Success:  false,
		Duration: 0,
	}

	// 获取用户目录和配置文件路径
	homeDir, err := os.UserHomeDir()
	if err != nil {
		result.Error = "无法获取用户主目录"
		result.Details = "系统错误: " + err.Error()
		result.Duration = time.Since(startTime)
		return result, nil
	}

	serverConfigPath := filepath.Join(homeDir, ".to_icalendar", "server.yaml")

	// 检查配置文件是否存在
	if _, err := os.Stat(serverConfigPath); os.IsNotExist(err) {
		result.Error = "配置文件不存在"
		result.Details = "配置文件路径: " + serverConfigPath + "\n请先运行初始化配置"
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// 读取并解析配置文件
	configData, err := os.ReadFile(serverConfigPath)
	if err != nil {
		result.Error = "配置文件读取失败"
		result.Details = "错误详情: " + err.Error() + "\n配置文件路径: " + serverConfigPath
		result.Duration = time.Since(startTime)
		return result, nil
	}

	var config ServerConfigFile
	if err := yaml.Unmarshal(configData, &config); err != nil {
		result.Error = "配置文件格式错误"
		result.Details = "YAML解析错误: " + err.Error() + "\n请检查配置文件格式是否正确"
		result.Duration = time.Since(startTime)
		return result, nil
	}

	result.Success = true
	result.Message = "配置文件验证通过"
	result.Details = "配置文件路径: " + serverConfigPath + "\n文件格式正确"
	result.Duration = time.Since(startTime)
	result.Config = &config
	return result, nil
}

// ValidateMicrosoftTodoConfig 验证 Microsoft Todo 配置
func (cv *ConfigValidator) ValidateMicrosoftTodoConfig(config *MicrosoftTodoConfig) (*ValidationResult, error) {
	startTime := time.Now()
	result := &ValidationResult{
		Name:     "Microsoft Todo 配置验证",
		Success:  false,
		Duration: 0,
	}

	// 验证必需字段
	missingFields := []string{}
	if config.TenantID == "" {
		missingFields = append(missingFields, "tenant_id (租户ID)")
	}
	if config.ClientID == "" {
		missingFields = append(missingFields, "client_id (客户端ID)")
	}
	if config.ClientSecret == "" {
		missingFields = append(missingFields, "client_secret (客户端密钥)")
	}

	if len(missingFields) > 0 {
		result.Error = "Microsoft Todo 配置缺少必需字段: " + strings.Join(missingFields, ", ")
		result.Details = "请在配置文件中填写以下必需字段:\n" + strings.Join(missingFields, "\n")
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// 检查占位符
	placeholderFields := []string{}
	if config.TenantID == "YOUR_TENANT_ID" {
		placeholderFields = append(placeholderFields, "tenant_id (当前值: YOUR_TENANT_ID)")
	}
	if config.ClientID == "YOUR_CLIENT_ID" {
		placeholderFields = append(placeholderFields, "client_id (当前值: YOUR_CLIENT_ID)")
	}
	if config.ClientSecret == "YOUR_CLIENT_SECRET" {
		placeholderFields = append(placeholderFields, "client_secret (当前值: YOUR_CLIENT_SECRET)")
	}

	if len(placeholderFields) > 0 {
		result.Error = "Microsoft Todo 配置包含占位符，需要更新为实际值"
		result.Details = "以下字段仍为默认占位符:\n" + strings.Join(placeholderFields, "\n") +
			"\n请访问 Azure Portal (portal.azure.com) 创建应用注册并获取实际值"
		result.Duration = time.Since(startTime)
		return result, nil
	}

	result.Success = true
	result.Message = "Microsoft Todo 配置验证通过"
	result.Duration = time.Since(startTime)
	return result, nil
}

// ConfigFileResult 配置文件验证结果
type ConfigFileResult struct {
	Name     string           `json:"name"`
	Success  bool             `json:"success"`
	Message  string           `json:"message"`
	Error    string           `json:"error,omitempty"`
	Details  string           `json:"details,omitempty"`
	Duration time.Duration    `json:"duration"`
	Config   *ServerConfigFile `json:"config,omitempty"`
}

// ValidationResult 验证结果
type ValidationResult struct {
	Name     string        `json:"name"`
	Success  bool          `json:"success"`
	Message  string        `json:"message"`
	Error    string        `json:"error,omitempty"`
	Details  string        `json:"details,omitempty"`
	Duration time.Duration `json:"duration"`
}

// ServerConfigFile 服务器配置文件结构
type ServerConfigFile struct {
	MicrosoftTodo MicrosoftTodoConfig `yaml:"microsoft_todo"`
	Dify          DifyConfig          `yaml:"dify"`
}