package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/allanpk716/to_icalendar/internal/logger"
	"github.com/allanpk716/to_icalendar/internal/models"
	"gopkg.in/yaml.v3"
)

// ConfigManager 管理配置文件和提醒事项文件
type ConfigManager struct{}

// NewConfigManager 创建配置管理器
func NewConfigManager() *ConfigManager {
	return &ConfigManager{}
}

// LoadServerConfig 加载服务器配置文件
func (cm *ConfigManager) LoadServerConfig(configPath string) (*models.ServerConfig, error) {
	// 检查文件是否存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("server config file not found: %s", configPath)
	}

	// 读取文件
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read server config file: %w", err)
	}

	// 解析YAML
	var config models.ServerConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse server config file: %w", err)
	}

	// 验证配置完整性 - 需要 Microsoft Todo 配置
	hasMicrosoftTodo := config.MicrosoftTodo.TenantID != "" && config.MicrosoftTodo.ClientID != "" && config.MicrosoftTodo.ClientSecret != ""

	if !hasMicrosoftTodo {
		return nil, fmt.Errorf("microsoft_todo configuration is required")
	}

	// 验证 Microsoft Todo 配置
	if config.MicrosoftTodo.Timezone == "" {
		config.MicrosoftTodo.Timezone = "UTC" // 默认UTC时区
	}

	// 验证提醒配置
	if err := config.Reminder.Validate(); err != nil {
		return nil, fmt.Errorf("reminder configuration validation failed: %w", err)
	}

	// 验证去重配置
	if err := config.Deduplication.Validate(); err != nil {
		return nil, fmt.Errorf("deduplication configuration validation failed: %w", err)
	}

	// 验证缓存配置
	if err := config.Cache.Validate(); err != nil {
		return nil, fmt.Errorf("cache configuration validation failed: %w", err)
	}

	// 设置默认日志配置（如果没有配置的话）
	if config.Logging.Level == "" {
		config.Logging.Level = "info"
	}
	if !config.Logging.ConsoleOutput && !config.Logging.FileOutput {
		config.Logging.ConsoleOutput = true
		config.Logging.FileOutput = true
	}
	if config.Logging.LogDir == "" {
		config.Logging.LogDir = "./Logs"
	}

	// 添加配置状态日志
	logger.Infof("提醒配置加载完成:")
	logger.Infof("  默认提醒时间: %s", config.Reminder.DefaultRemindBefore)
	logger.Infof("  智能提醒功能: %t", config.Reminder.EnableSmartReminder)

	return &config, nil
}

// LoadReminder 加载提醒事项JSON文件
func (cm *ConfigManager) LoadReminder(reminderPath string) (*models.Reminder, error) {
	// 检查文件是否存在
	if _, err := os.Stat(reminderPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("reminder file not found: %s", reminderPath)
	}

	// 读取文件
	data, err := os.ReadFile(reminderPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read reminder file: %w", err)
	}

	// 解析JSON
	var reminder models.Reminder
	err = json.Unmarshal(data, &reminder)
	if err != nil {
		return nil, fmt.Errorf("failed to parse reminder file: %w", err)
	}

	// 验证必要字段
	if reminder.Title == "" {
		return nil, fmt.Errorf("title is required in reminder")
	}
	if reminder.Date == "" {
		return nil, fmt.Errorf("date is required in reminder")
	}
	if reminder.Time == "" {
		return nil, fmt.Errorf("time is required in reminder")
	}

	// 验证日期格式
	_, err = time.Parse("2006-01-02", reminder.Date)
	if err != nil {
		return nil, fmt.Errorf("invalid date format, expected YYYY-MM-DD: %w", err)
	}

	// 验证时间格式（支持时间范围）
	processedTime := parseTimeFromRange(reminder.Time)
	_, err = time.Parse("15:04", processedTime)
	if err != nil {
		return nil, fmt.Errorf("invalid time format, expected HH:MM or time range like HH:MM - HH:MM: %w", err)
	}

	// 验证优先级
	if reminder.Priority != "" {
		switch reminder.Priority {
		case models.PriorityLow, models.PriorityMedium, models.PriorityHigh:
			// 有效优先级
		default:
			return nil, fmt.Errorf("invalid priority, must be one of: low, medium, high")
		}
	}

	return &reminder, nil
}

// LoadRemindersFromPattern 根据glob模式加载多个提醒事项文件
func (cm *ConfigManager) LoadRemindersFromPattern(pattern string) ([]*models.Reminder, error) {
	// 匹配文件
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to glob pattern: %w", err)
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("no files found matching pattern: %s", pattern)
	}

	var reminders []*models.Reminder
	for _, filePath := range matches {
		reminder, err := cm.LoadReminder(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to load reminder from %s: %w", filePath, err)
		}
		reminders = append(reminders, reminder)
	}

	return reminders, nil
}

// CreateServerConfigTemplate 创建服务器配置模板文件
func (cm *ConfigManager) CreateServerConfigTemplate(configPath string) error {
	template := models.ServerConfig{}

	// Microsoft Todo 配置
	template.MicrosoftTodo.TenantID = "YOUR_TENANT_ID"
	template.MicrosoftTodo.ClientID = "YOUR_CLIENT_ID"
	template.MicrosoftTodo.ClientSecret = "YOUR_CLIENT_SECRET"
	template.MicrosoftTodo.Timezone = "Asia/Shanghai"

	// 提醒配置
	template.Reminder.DefaultRemindBefore = "15m"
	template.Reminder.EnableSmartReminder = true

	// 去重配置
	template.Deduplication.Enabled = true
	template.Deduplication.TimeWindowMinutes = 5
	template.Deduplication.SimilarityThreshold = 80
	template.Deduplication.CheckIncompleteOnly = true
	template.Deduplication.EnableLocalCache = true
	template.Deduplication.EnableRemoteQuery = true

	// 缓存配置
	template.Cache = models.DefaultCacheConfig()

	// 日志配置
	template.Logging.Level = "info"
	template.Logging.ConsoleOutput = true
	template.Logging.FileOutput = true
	template.Logging.LogDir = "./Logs"

	// 序列化为YAML
	data, err := yaml.Marshal(template)
	if err != nil {
		return fmt.Errorf("failed to marshal server config template: %w", err)
	}

	// Ensure directory exists with secure permissions
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write file with restricted permissions (owner read/write only)
	err = os.WriteFile(configPath, data, 0600)
	if err != nil {
		return fmt.Errorf("failed to write server config template: %w", err)
	}

	return nil
}

// SaveServerConfig 保存服务器配置文件
func (cm *ConfigManager) SaveServerConfig(configPath string, config *models.ServerConfig) error {
	// 序列化为YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal server config: %w", err)
	}

	// Ensure directory exists with secure permissions
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write file with restricted permissions (owner read/write only)
	err = os.WriteFile(configPath, data, 0600)
	if err != nil {
		return fmt.Errorf("failed to write server config: %w", err)
	}

	return nil
}

// CreateReminderTemplate 创建提醒事项模板文件
func (cm *ConfigManager) CreateReminderTemplate(reminderPath string) error {
	template := models.Reminder{
		Title:        "会议提醒",
		Description:  "参加产品评审会议",
		Date:         "2024-12-25",
		Time:         "14:30",
		RemindBefore: "15m",
		Priority:     models.PriorityMedium,
		List:         "工作",
	}

	return cm.SaveReminderTemplate(reminderPath, &template)
}

// SaveReminderTemplate 保存提醒事项模板文件
func (cm *ConfigManager) SaveReminderTemplate(reminderPath string, reminder *models.Reminder) error {
	// 序列化为JSON
	data, err := json.MarshalIndent(reminder, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal reminder template: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(reminderPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create reminder directory: %w", err)
	}

	// Write file with standard permissions (reminder files are not sensitive)
	err = os.WriteFile(reminderPath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write reminder template: %w", err)
	}

	return nil
}

// parseTimeFromRange 从时间字符串中解析时间，支持时间范围格式
// 如果输入是时间范围（如"14:30 - 16:30"），返回开始时间"14:30"
// 如果输入是单个时间，直接返回
func parseTimeFromRange(timeStr string) string {
	// 定义时间范围分隔符模式
	rangePatterns := []string{
		`^(\d{1,2}:\d{2})\s*[-~~到至]\s*(\d{1,2}:\d{2})$`,     // 14:30-16:30, 14:30~16:30, 14:30到16:30, 14:30至16:30
		`^(\d{1,2}:\d{2})\s*[-~到至]\s*(\d{1,2}:\d{2})$`,       // 14:30 - 16:30 (带空格)
		`^(\d{1,2}:\d{2})\s*-\s*(\d{1,2}:\d{2})$`,             // 14:30 - 16:30
	}
	
	for _, pattern := range rangePatterns {
		if re, err := regexp.Compile(pattern); err == nil {
			if matches := re.FindStringSubmatch(timeStr); len(matches) > 1 {
				startTime := matches[1]
				// 验证开始时间格式是否有效
				if isValidTimeFormat(startTime) {
					return startTime
				}
			}
		}
	}
	
	// 尝试更宽松的匹配：只提取第一个有效的时间格式
	timeRegex := regexp.MustCompile(`\d{1,2}:\d{2}`)
	matches := timeRegex.FindAllString(timeStr, 2)
	if len(matches) > 0 {
		// 返回第一个匹配的时间（开始时间）
		if isValidTimeFormat(matches[0]) {
			return matches[0]
		}
	}
	
	return timeStr // 不是时间范围格式，返回原始字符串
}

// isValidTimeFormat 验证时间格式是否有效 (HH:MM 或 H:MM)
func isValidTimeFormat(timeStr string) bool {
	formats := []string{"15:04", "3:04"}
	for _, format := range formats {
		if _, err := time.Parse(format, timeStr); err == nil {
			return true
		}
	}
	return false
}
