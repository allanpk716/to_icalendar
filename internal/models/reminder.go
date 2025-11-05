package models

import (
	"fmt"
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

// ServerConfig contains configuration for Microsoft Todo integration.
// It includes Azure AD credentials and timezone settings for proper time handling.
type ServerConfig struct {
	MicrosoftTodo struct {
		TenantID     string `yaml:"tenant_id"`     // Microsoft Azure 租户ID
		ClientID     string `yaml:"client_id"`     // 应用程序客户端ID
		ClientSecret string `yaml:"client_secret"` // 客户端密钥
		UserEmail    string `yaml:"user_email"`    // 目标用户邮箱（用于应用程序权限）
		Timezone     string `yaml:"timezone"`      // 时区设置
	} `yaml:"microsoft_todo"`
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
func ParseReminderTime(reminder Reminder, timezone *time.Location) (*ParsedReminder, error) {
	// 解析日期和时间
	dateTimeStr := reminder.Date + " " + reminder.Time
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
