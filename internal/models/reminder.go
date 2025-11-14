package models

import (
	"fmt"
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

// ServerConfig contains configuration for Microsoft Todo and Dify integration.
// It includes Azure AD credentials, timezone settings, and Dify API configuration.
type ServerConfig struct {
	MicrosoftTodo MicrosoftTodoConfig `yaml:"microsoft_todo"`
	Dify          DifyConfig         `yaml:"dify"`
}

// ParsedReminder represents a reminder with parsed time information.
// It includes the original reminder data, calculated due/alarm times, and formatted strings.
type ParsedReminder struct {
	Original      Reminder       // 原始数据
	DueTime       time.Time      // 截止时间
	AlarmTime     time.Time      // 提醒时间
	PriorityValue int            // iCalendar优先级值（1-9）
	Priority      int            // 原始优先级值（用于 Microsoft Todo）
	Timezone      *time.Location // 时区信息
	List          string         // 任务列表名称
	Description   string         // 描述信息
	DueTimeStr    string         // 格式化的截止时间字符串
	RemindTimeStr string         // 格式化的提醒时间字符串
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
	// 处理时间范围，提取开始时间
	processedTime := parseTimeFromRange(reminder.Time)
	
	// 解析日期和时间
	dateTimeStr := reminder.Date + " " + processedTime
	dueTime, err := time.ParseInLocation("2006-01-02 15:04", dateTimeStr, timezone)
	if err != nil {
		return nil, err
	}

	// 解析提前提醒时间
	remindBefore := reminder.RemindBefore
	if remindBefore == "" {
		remindBefore = "15m" // 默认提前15分钟
	}

	alarmTime, err := parseDuration(dueTime, remindBefore)
	if err != nil {
		return nil, err
	}

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

	// 格式化时间字符串
	dueTimeStr := dueTime.Format("2006-01-02T15:04:05")
	remindTimeStr := alarmTime.Format("2006-01-02T15:04:05")

	return &ParsedReminder{
		Original:      reminder,
		DueTime:       dueTime,
		AlarmTime:     alarmTime,
		PriorityValue: priorityValue,
		Priority:      priority,
		Timezone:      timezone,
		List:          list,
		Description:   reminder.Description,
		DueTimeStr:    dueTimeStr,
		RemindTimeStr: remindTimeStr,
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
