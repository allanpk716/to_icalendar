package ical

import (
	"fmt"
	"strings"
	"time"

	"github.com/allanpk716/to_icalendar/internal/models"
)

// Calendar 表示iCalendar日历
type Calendar struct {
	Events []VTODO
}

// VTODO 表示iCalendar VTODO组件
type VTODO struct {
	UID         string
	DtStamp     time.Time
	Created     time.Time
	Summary     string
	Description string
	Due         time.Time
	Status      string
	Priority    int
	Categories  string
	Alarm       Alarm
}

// Alarm 表示VALARM组件
type Alarm struct {
	Action      string
	Description string
	Trigger     string
}

// ICalCreator 创建iCalendar VTODO组件
type ICalCreator struct{}

// NewICalCreator 创建iCalendar创建器
func NewICalCreator() *ICalCreator {
	return &ICalCreator{}
}

// CreateVTODO 为提醒事项创建VTODO组件
func (ic *ICalCreator) CreateVTODO(reminder *models.ParsedReminder) (*Calendar, error) {
	// 创建VTODO组件
	todo := VTODO{
		UID:         ic.generateUID(reminder),
		DtStamp:     time.Now().UTC(),
		Created:     time.Now().UTC(),
		Summary:     reminder.Original.Title,
		Due:         reminder.DueTime.UTC(),
		Status:      "NEEDS-ACTION",
		Priority:    reminder.PriorityValue,
		Categories:  reminder.Original.List,
		Description: reminder.Original.Description,
	}

	// 添加提醒闹钟
	todo.Alarm = Alarm{
		Action:      "DISPLAY",
		Description: reminder.Original.Title,
		Trigger:     ic.formatDuration(reminder.AlarmTime.Sub(time.Now())),
	}

	// 创建日历
	cal := &Calendar{
		Events: []VTODO{todo},
	}

	return cal, nil
}

// generateUID 生成唯一的UID
func (ic *ICalCreator) generateUID(reminder *models.ParsedReminder) string {
	// 使用时间戳和标题生成唯一ID
	timestamp := reminder.DueTime.Unix()
	titleHash := ic.simpleHash(reminder.Original.Title)
	return fmt.Sprintf("%s-%d@to_icalendar", titleHash, timestamp)
}

// simpleHash 简单的字符串哈希函数
func (ic *ICalCreator) simpleHash(s string) string {
	hash := 0
	for i, c := range s {
		hash = hash*31 + int(c) + i
		if hash < 0 {
			hash = -hash
		}
	}
	return fmt.Sprintf("%x", hash%0xFFFFFFFF)
}

// formatDuration 格式化持续时间为iCalendar格式
func (ic *ICalCreator) formatDuration(d time.Duration) string {
	// iCalendar持续时间格式: PT[nH][nM][nS]
	sign := ""
	if d < 0 {
		sign = "-"
		d = -d
	}

	totalSeconds := int(d.Seconds())
	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60

	var result strings.Builder
	result.WriteString("P")
	result.WriteString(sign)
	result.WriteString("T")

	if hours > 0 {
		result.WriteString(fmt.Sprintf("%dH", hours))
	}
	if minutes > 0 {
		result.WriteString(fmt.Sprintf("%dM", minutes))
	}
	if seconds > 0 || (hours == 0 && minutes == 0) {
		result.WriteString(fmt.Sprintf("%dS", seconds))
	}

	return result.String()
}

// ValidateReminder 验证提醒事项是否有效
func (ic *ICalCreator) ValidateReminder(reminder *models.ParsedReminder) error {
	// 检查截止时间是否在未来
	if reminder.DueTime.Before(time.Now()) {
		return fmt.Errorf("reminder due time %s is in the past", reminder.DueTime.Format("2006-01-02 15:04"))
	}

	// 检查提醒时间是否合理（不能太久远）
	maxFuture := time.Now().AddDate(1, 0, 0) // 1年后
	if reminder.DueTime.After(maxFuture) {
		return fmt.Errorf("reminder due time %s is too far in the future", reminder.DueTime.Format("2006-01-02 15:04"))
	}

	// 检查标题长度
	if len(reminder.Original.Title) > 200 {
		return fmt.Errorf("reminder title is too long (max 200 characters)")
	}

	// 检查描述长度
	if len(reminder.Original.Description) > 1000 {
		return fmt.Errorf("reminder description is too long (max 1000 characters)")
	}

	return nil
}

// GetICalString 获取iCalendar字符串表示
func (ic *ICalCreator) GetICalString(cal *Calendar) (string, error) {
	var builder strings.Builder

	// 写入日历头部
	builder.WriteString("BEGIN:VCALENDAR\r\n")
	builder.WriteString("VERSION:2.0\r\n")
	builder.WriteString("PRODID:-//to_icalendar//Reminder//EN\r\n")
	builder.WriteString("CALSCALE:GREGORIAN\r\n")

	// 写入VTODO组件
	for _, todo := range cal.Events {
		ic.writeVTODO(&builder, &todo)
	}

	// 写入日历尾部
	builder.WriteString("END:VCALENDAR\r\n")

	return builder.String(), nil
}

// writeVTODO 写入VTODO组件
func (ic *ICalCreator) writeVTODO(builder *strings.Builder, todo *VTODO) {
	builder.WriteString("BEGIN:VTODO\r\n")
	builder.WriteString(fmt.Sprintf("UID:%s\r\n", todo.UID))
	builder.WriteString(fmt.Sprintf("DTSTAMP:%s\r\n", todo.DtStamp.Format("20060102T150405Z")))
	builder.WriteString(fmt.Sprintf("CREATED:%s\r\n", todo.Created.Format("20060102T150405Z")))
	builder.WriteString(fmt.Sprintf("SUMMARY:%s\r\n", escapeText(todo.Summary)))
	builder.WriteString(fmt.Sprintf("STATUS:%s\r\n", todo.Status))
	builder.WriteString(fmt.Sprintf("PRIORITY:%d\r\n", todo.Priority))
	builder.WriteString(fmt.Sprintf("DUE:%s\r\n", todo.Due.Format("20060102T150405Z")))

	if todo.Description != "" {
		builder.WriteString(fmt.Sprintf("DESCRIPTION:%s\r\n", escapeText(todo.Description)))
	}

	if todo.Categories != "" {
		builder.WriteString(fmt.Sprintf("CATEGORIES:%s\r\n", escapeText(todo.Categories)))
	}

	// 写入闹钟
	ic.writeAlarm(builder, &todo.Alarm)

	builder.WriteString("END:VTODO\r\n")
}

// writeAlarm 写入VALARM组件
func (ic *ICalCreator) writeAlarm(builder *strings.Builder, alarm *Alarm) {
	builder.WriteString("BEGIN:VALARM\r\n")
	builder.WriteString(fmt.Sprintf("ACTION:%s\r\n", alarm.Action))
	builder.WriteString(fmt.Sprintf("DESCRIPTION:%s\r\n", escapeText(alarm.Description)))
	builder.WriteString(fmt.Sprintf("TRIGGER:%s\r\n", alarm.Trigger))
	builder.WriteString("END:VALARM\r\n")
}

// escapeText 转义iCalendar文本
func escapeText(text string) string {
	// 替换特殊字符
	text = strings.ReplaceAll(text, "\\", "\\\\")
	text = strings.ReplaceAll(text, ",", "\\,")
	text = strings.ReplaceAll(text, ";", "\\;")
	text = strings.ReplaceAll(text, "\n", "\\n")

	// 处理长行（iCalendar建议每行不超过75个字符）
	if len(text) <= 75 {
		return text
	}

	var result strings.Builder
	for i := 0; i < len(text); i += 75 {
		end := i + 75
		if end > len(text) {
			end = len(text)
		}

		if i == 0 {
			result.WriteString(text[i:end])
		} else {
			result.WriteString("\r\n " + text[i:end]) // 后续行需要空格开头
		}
	}

	return result.String()
}