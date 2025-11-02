package models

import (
	"fmt"
	"time"
)

// Priority 定义提醒事项优先级
type Priority string

const (
	PriorityLow    Priority = "low"
	PriorityMedium Priority = "medium"
	PriorityHigh   Priority = "high"
)

// Reminder 表示一个提醒事项
type Reminder struct {
	Title         string    `json:"title"`                     // 提醒标题（必填）
	Description   string    `json:"description,omitempty"`    // 备注信息（可选）
	Date          string    `json:"date"`                      // 日期 YYYY-MM-DD（必填）
	Time          string    `json:"time"`                      // 时间 HH:MM（必填）
	RemindBefore  string    `json:"remind_before,omitempty"`   // 提前提醒时间（如 15m, 1h, 1d）
	Priority      Priority  `json:"priority,omitempty"`        // 优先级 low/medium/high
	List          string    `json:"list,omitempty"`            // 提醒事项列表名称
}

// ServerConfig 表示服务器配置
type ServerConfig struct {
	CalDAV struct {
		ServerURL string `yaml:"server_url"` // CalDAV服务器地址
		Username  string `yaml:"username"`   // Apple ID
		Timezone  string `yaml:"timezone"`   // 时区设置
	} `yaml:"caldav"`
}

// ParsedReminder 表示解析后的提醒事项，包含时间处理
type ParsedReminder struct {
	Original      Reminder          // 原始数据
	DueTime       time.Time         // 截止时间
	AlarmTime     time.Time         // 提醒时间
	PriorityValue int               // iCalendar优先级值（1-9）
	Timezone      *time.Location    // 时区信息
}

// ParseReminderTime 解析提醒事项的时间信息
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
	switch reminder.Priority {
	case PriorityLow:
		priorityValue = 9
	case PriorityHigh:
		priorityValue = 1
	case PriorityMedium:
		priorityValue = 5
	}

	// 设置默认列表名称
	list := reminder.List
	if list == "" {
		list = "提醒事项"
	}

	return &ParsedReminder{
		Original:      reminder,
		DueTime:       dueTime,
		AlarmTime:     alarmTime,
		PriorityValue: priorityValue,
		Timezone:      timezone,
	}, nil
}

// parseDuration 解析持续时间字符串并计算提醒时间
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