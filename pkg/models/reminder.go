package models

import (
	"fmt"
	"log"
	"regexp"
	"time"
)

// Priority defines the priority level for reminders.
type Priority string

const (
	PriorityLow    Priority = "low"    // Low priority
	PriorityMedium Priority = "medium" // Medium priority (default)
	PriorityHigh   Priority = "high"   // High priority
)

// Reminder represents a reminder task with title, description, timing, and priority.
// It is used to serialize/deserialize reminder data from JSON configuration files.
type Reminder struct {
	Title        string   `json:"title"`                   // 提醒标题（必填）
	Description  string   `json:"description,omitempty"`   // 备注信息（可选）
	Date         string   `json:"date"`                    // 日期 YYYY-MM-DD（必填）
	Time         string   `json:"time"`                    // 时间 HH:MM（必填）
	RemindBefore string   `json:"remind_before,omitempty"` // 提前提醒时间（如 15m, 1h, 1d）
	Priority     Priority `json:"priority,omitempty"`      // 优先级 low/medium/high
	List         string   `json:"list,omitempty"`          // 提醒事项列表名称
}

// MicrosoftTodoConfig represents the configuration for Microsoft Todo API integration.
type MicrosoftTodoConfig struct {
	TenantID     string `yaml:"tenant_id"`     // Microsoft Azure 租户ID
	ClientID     string `yaml:"client_id"`     // 应用程序客户端ID
	ClientSecret string `yaml:"client_secret"` // 客户端密钥
	UserEmail    string `yaml:"user_email"`    // 目标用户邮箱（用于应用程序权限）
	Timezone     string `yaml:"timezone"`      // 时区设置
}

// ReminderConfig represents the configuration for reminder settings.
type ReminderConfig struct {
	DefaultRemindBefore string `yaml:"default_remind_before"` // 默认提前提醒时间（如 15m, 1h, 1d）
	EnableSmartReminder bool   `yaml:"enable_smart_reminder"` // 是否启用智能提醒（根据优先级自动调整）
}

// DeduplicationConfig represents the configuration for task deduplication settings.
type DeduplicationConfig struct {
	Enabled              bool `yaml:"enabled"`                // 是否启用去重功能
	TimeWindowMinutes    int  `yaml:"time_window_minutes"`    // 时间匹配窗口（分钟）
	SimilarityThreshold  int  `yaml:"similarity_threshold"`   // 相似度阈值（百分比 0-100）
	CheckIncompleteOnly  bool `yaml:"check_incomplete_only"`  // 只检查未完成的任务
	EnableLocalCache     bool `yaml:"enable_local_cache"`     // 启用本地缓存
	EnableRemoteQuery    bool `yaml:"enable_remote_query"`    // 启用远程查询
}

// LoggingConfig represents the configuration for logging settings.
type LoggingConfig struct {
	Level         string `yaml:"level"`           // 日志级别: debug, info, warn, error
	ConsoleOutput bool   `yaml:"console_output"`  // 是否输出到控制台
	FileOutput    bool   `yaml:"file_output"`     // 是否输出到文件
	LogDir        string `yaml:"log_dir"`         // 日志目录（可选，默认 ./Logs/）
}

// Validate validates the reminder configuration
func (c *ReminderConfig) Validate() error {
	// 如果没有设置默认提醒时间，使用默认值
	if c.DefaultRemindBefore == "" {
		c.DefaultRemindBefore = "15m" // 默认提前15分钟
	}

	// 验证默认提醒时间格式
	_, err := parseDuration(time.Now(), c.DefaultRemindBefore)
	if err != nil {
		return fmt.Errorf("invalid default_remind_before format: %w", err)
	}

	return nil
}

// Validate validates the deduplication configuration
func (c *DeduplicationConfig) Validate() error {
	// 设置默认值
	if c.TimeWindowMinutes == 0 {
		c.TimeWindowMinutes = 5 // 默认5分钟时间窗口
	}
	if c.SimilarityThreshold == 0 {
		c.SimilarityThreshold = 80 // 默认80%相似度阈值
	}

	// 验证范围
	if c.TimeWindowMinutes < 0 || c.TimeWindowMinutes > 1440 {
		return fmt.Errorf("time_window_minutes must be between 0 and 1440 (24 hours)")
	}
	if c.SimilarityThreshold < 0 || c.SimilarityThreshold > 100 {
		return fmt.Errorf("similarity_threshold must be between 0 and 100")
	}

	return nil
}

// GetSmartRemindTime 根据优先级获取智能提醒时间
func (c *ReminderConfig) GetSmartRemindTime(priority Priority) string {
	// 首先检查是否启用智能提醒
	if !c.EnableSmartReminder {
		log.Printf("智能提醒功能已禁用，使用默认提醒时间: %s", c.DefaultRemindBefore)
		return c.DefaultRemindBefore
	}

	// 根据优先级调整提醒时间
	switch priority {
	case PriorityHigh:
		log.Printf("高优先级任务，使用智能提醒时间: 30m")
		return "30m" // 高优先级任务提前30分钟
	case PriorityMedium:
		log.Printf("中优先级任务，使用智能提醒时间: 15m")
		return "15m" // 中优先级任务提前15分钟
	case PriorityLow:
		log.Printf("低优先级任务，使用智能提醒时间: 5m")
		return "5m"  // 低优先级任务提前5分钟
	default:
		log.Printf("未知优先级，使用默认提醒时间: %s", c.DefaultRemindBefore)
		return c.DefaultRemindBefore
	}
}

// ServerConfig contains configuration for Microsoft Todo, Dify integration, reminder settings, cache management, and logging.
// It includes Azure AD credentials, timezone settings, Dify API configuration, reminder defaults, cache configuration, and logging configuration.
type ServerConfig struct {
	MicrosoftTodo  MicrosoftTodoConfig   `yaml:"microsoft_todo"`
	Reminder       ReminderConfig        `yaml:"reminder"`
	Deduplication  DeduplicationConfig   `yaml:"deduplication"`
	Dify           DifyConfig           `yaml:"dify"`
	Cache          CacheConfig          `yaml:"cache"`
	Logging        LoggingConfig        `yaml:"logging"`
}

// ParsedReminder represents a reminder with parsed time information.
// It includes the original reminder data, calculated due/alarm times, and formatted strings.
type ParsedReminder struct {
	Original         Reminder       // 原始数据
	DueTime          time.Time      // 截止时间（保留向后兼容）
	AlarmTime        time.Time      // 提醒时间（保留向后兼容）
	PriorityValue    int            // iCalendar优先级值（1-9）
	Priority         int            // 原始优先级值（用于 Microsoft Todo）
	Timezone         *time.Location // 时区信息
	List             string         // 任务列表名称
	Description      string         // 描述信息
	DueTimeStr       string         // 格式化的截止时间字符串
	RemindTimeStr    string         // 格式化的提醒时间字符串
	// UTC标准化字段（新增）
	DueTimeUTC       time.Time      // 截止时间（UTC）
	AlarmTimeUTC     time.Time      // 提醒时间（UTC）
	UserTimezone     string         // 用户配置的时区名称
}

// ParseReminderTime parses time information from a reminder and creates a ParsedReminder.
// It converts date/time strings, calculates alarm times, and formats priority values.
// Returns a ParsedReminder with calculated times and formatted strings, or an error if parsing fails.
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

func ParseReminderTime(reminder Reminder, timezone *time.Location) (*ParsedReminder, error) {
	return ParseReminderTimeWithConfig(reminder, timezone, nil)
}

// ParseReminderTimeWithConfig 使用配置信息解析提醒时间，采用UTC标准化处理
func ParseReminderTimeWithConfig(reminder Reminder, timezone *time.Location, config *ReminderConfig) (*ParsedReminder, error) {
	// 处理时间范围，提取开始时间
	processedTime := parseTimeFromRange(reminder.Time)

	// 获取用户时区名称字符串
	userTimezone := "UTC"
	if timezone != nil {
		userTimezone = timezone.String()
	}

	// UTC标准化处理：先用本地时区解析，然后转换为UTC
	dateTimeStr := reminder.Date + " " + processedTime
	localDueTime, err := time.ParseInLocation("2006-01-02 15:04", dateTimeStr, timezone)
	if err != nil {
		return nil, err
	}

	// 转换为UTC时间（内部统一使用UTC）
	dueTimeUTC := localDueTime.UTC()
	log.Printf("时间解析: 本地时间 %s -> UTC时间 %s (时区: %s)",
		localDueTime.Format("2006-01-02 15:04:05"),
		dueTimeUTC.Format("2006-01-02 15:04:05"),
		userTimezone)

	// 解析提前提醒时间
	remindBefore := reminder.RemindBefore
	if remindBefore == "" {
		if config != nil {
			// 使用配置中的智能提醒时间
			remindBefore = config.GetSmartRemindTime(reminder.Priority)
		} else {
			log.Printf("配置为空，使用默认提醒时间: 15m")
			remindBefore = "15m" // 默认提前15分钟
		}
	} else {
		// 用户已明确设置remind_before，记录但不覆盖
		log.Printf("用户设置的提醒时间: %s，将优先使用用户设置", remindBefore)
	}

	// 计算提醒时间（基于本地时间）
	localAlarmTime, err := parseDuration(localDueTime, remindBefore)
	if err != nil {
		return nil, err
	}

	// 转换提醒时间为UTC
	alarmTimeUTC := localAlarmTime.UTC()
	log.Printf("提醒时间: 本地时间 %s -> UTC时间 %s",
		localAlarmTime.Format("2006-01-02 15:04:05"),
		alarmTimeUTC.Format("2006-01-02 15:04:05"))

	// 转换优先级
	priorityValue := 5 // 默认中等优先级
	priority := 5      // Microsoft Todo 优先级
	switch reminder.Priority {
	case PriorityLow:
		priorityValue = 9
		priority = 1 // Microsoft Todo 低优先级
	case PriorityHigh:
		priorityValue = 1
		priority = 9 // Microsoft Todo 高优先级
	case PriorityMedium:
		priorityValue = 5
		priority = 5 // Microsoft Todo 中等优先级
	}

	// 设置默认列表名称
	list := reminder.List
	if list == "" {
		list = "Default" // Microsoft Todo 默认列表名称
	}

	// 格式化时间字符串（保持向后兼容）
	dueTimeStr := localDueTime.Format("2006-01-02T15:04:05")
	remindTimeStr := localAlarmTime.Format("2006-01-02T15:04:05")

	return &ParsedReminder{
		Original:         reminder,
		DueTime:          localDueTime,      // 保留原有字段（向后兼容）
		AlarmTime:        localAlarmTime,    // 保留原有字段（向后兼容）
		PriorityValue:    priorityValue,
		Priority:         priority,
		Timezone:         timezone,
		List:             list,
		Description:      reminder.Description,
		DueTimeStr:       dueTimeStr,
		RemindTimeStr:    remindTimeStr,
		// UTC标准化字段
		DueTimeUTC:       dueTimeUTC,        // 截止时间（UTC）
		AlarmTimeUTC:     alarmTimeUTC,      // 提醒时间（UTC）
		UserTimezone:     userTimezone,      // 用户配置的时区名称
	}, nil
}

// parseDuration parses a duration string and calculates the reminder time from a given time.
// Supports formats like "15m", "1h", "2d" for minutes, hours, and days respectively.
// Returns the calculated reminder time, or an error if the duration format is invalid.
func parseDuration(from time.Time, duration string) (time.Time, error) {
	var d time.Duration
	var err error

	// 简单解析持续时间字符串
	if len(duration) < 2 {
		return time.Time{}, &time.ParseError{
			Layout: "duration",
			Value:  duration,
		}
	}

	unit := duration[len(duration)-1:]
	valueStr := duration[:len(duration)-1]

	var value int
	_, err = fmt.Sscanf(valueStr, "%d", &value)
	if err != nil {
		return time.Time{}, err
	}

	switch unit {
	case "m":
		d = time.Duration(value) * time.Minute
	case "h":
		d = time.Duration(value) * time.Hour
	case "d":
		d = time.Duration(value) * 24 * time.Hour
	default:
		return time.Time{}, &time.ParseError{
			Layout: "duration",
			Value:  duration,
		}
	}

	return from.Add(-d), nil
}
