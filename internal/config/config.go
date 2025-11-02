package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

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

	// 验证必要字段
	if config.CalDAV.ServerURL == "" {
		return nil, fmt.Errorf("server_url is required in server config")
	}
	if config.CalDAV.Username == "" {
		return nil, fmt.Errorf("username is required in server config")
	}
	if config.CalDAV.Timezone == "" {
		config.CalDAV.Timezone = "UTC" // 默认UTC时区
	}

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

	// 验证时间格式
	_, err = time.Parse("15:04", reminder.Time)
	if err != nil {
		return nil, fmt.Errorf("invalid time format, expected HH:MM: %w", err)
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
	template.CalDAV.ServerURL = "https://caldav.icloud.com"
	template.CalDAV.Username = "your_apple_id@icloud.com"
	template.CalDAV.Timezone = "Asia/Shanghai"

	// 序列化为YAML
	data, err := yaml.Marshal(template)
	if err != nil {
		return fmt.Errorf("failed to marshal server config template: %w", err)
	}

	// 确保目录存在
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// 写入文件
	err = os.WriteFile(configPath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write server config template: %w", err)
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

	// 序列化为JSON
	data, err := json.MarshalIndent(template, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal reminder template: %w", err)
	}

	// 确保目录存在
	dir := filepath.Dir(reminderPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create reminder directory: %w", err)
	}

	// 写入文件
	err = os.WriteFile(reminderPath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write reminder template: %w", err)
	}

	return nil
}