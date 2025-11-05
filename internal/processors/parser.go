package processors

import (
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/allanpk716/to_icalendar/internal/models"
)

// TaskParser handles parsing of task information from various sources
type TaskParser struct {
	// 预编译的正则表达式
	emailPattern     *regexp.Regexp
	phonePattern     *regexp.Regexp
	urlPattern       *regexp.Regexp
	durationPattern  *regexp.Regexp
	priorityPattern  *regexp.Regexp
	dateTimePattern  *regexp.Regexp
}

// NewTaskParser creates a new task parser
func NewTaskParser() *TaskParser {
	return &TaskParser{
		emailPattern:     regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`),
		phonePattern:     regexp.MustCompile(`(\d{3}[-.\s]?\d{3}[-.\s]?\d{4}|\d{11})`),
		urlPattern:       regexp.MustCompile(`https?://[^\s]+`),
		durationPattern:  regexp.MustCompile(`(\d+)(m|h|d|w|分钟|小时|天|周)`),
		priorityPattern:  regexp.MustCompile(`(?i)(紧急|急|urgent|asap|high|低|low|中|medium|一般|normal)`),
		dateTimePattern:  regexp.MustCompile(`(\d{4}[-/]\d{1,2}[-/]\d{1,2}|\d{1,2}[-/]\d{1,2}[-/]\d{4}|今天|明天|后天|昨天|前天|\d+天后|\d+天前|\d+月\d+日|\d+月\d+号)\s*(\d{1,2}:\d{2}|\d{1,2}点|\d{1,2}时)?`),
	}
}

// ParseFromDifyResponse parses task information from Dify API response
func (tp *TaskParser) ParseFromDifyResponse(response string) (*models.ParsedTaskInfo, error) {
	log.Printf("开始解析Dify响应: %s", response)

	// 尝试直接解析JSON
	var taskInfo models.ParsedTaskInfo
	if err := json.Unmarshal([]byte(response), &taskInfo); err == nil {
		return tp.enhanceParsedInfo(&taskInfo), nil
	}

	// 如果直接解析失败，尝试从文本中提取JSON
	jsonStr := tp.extractJSONFromText(response)
	if jsonStr != "" {
		if err := json.Unmarshal([]byte(jsonStr), &taskInfo); err == nil {
			return tp.enhanceParsedInfo(&taskInfo), nil
		}
	}

	// 如果仍然无法解析，尝试智能解析
	return tp.intelligentParse(response), nil
}

// ParseFromText parses task information from plain text
func (tp *TaskParser) ParseFromText(text string) (*models.ParsedTaskInfo, error) {
	log.Printf("开始解析文字内容: %s", text)

	// 使用智能解析
	taskInfo := tp.intelligentParse(text)
	return taskInfo, nil
}

// extractJSONFromText extracts JSON from mixed content
func (tp *TaskParser) extractJSONFromText(text string) string {
	// 查找JSON对象开始和结束
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")

	if start >= 0 && end > start {
		jsonStr := text[start : end+1]

		// 验证是否为有效JSON
		var test interface{}
		if json.Unmarshal([]byte(jsonStr), &test) == nil {
			return jsonStr
		}
	}

	return ""
}

// enhanceParsedInfo enhances parsed information with additional processing
func (tp *TaskParser) enhanceParsedInfo(info *models.ParsedTaskInfo) *models.ParsedTaskInfo {
	if info == nil {
		return nil
	}

	// 标准化日期格式
	if info.Date != "" {
		info.Date = tp.normalizeDate(info.Date)
	}

	// 标准化时间格式
	if info.Time != "" {
		info.Time = tp.normalizeTime(info.Time)
	}

	// 标准化优先级
	if info.Priority != "" {
		info.Priority = tp.normalizePriority(info.Priority)
	}

	// 设置默认提醒时间
	if info.RemindBefore == "" {
		info.RemindBefore = tp.inferRemindBefore(info)
	}

	// 设置默认列表
	if info.List == "" {
		info.List = tp.inferList(info)
	}

	// 增强描述信息
	info.Description = tp.enhanceDescription(info)

	return info
}

// intelligentParse performs intelligent parsing without structured data
func (tp *TaskParser) intelligentParse(text string) *models.ParsedTaskInfo {
	// 提取基本信息
	title := tp.extractTitle(text)
	description := tp.extractDescription(text)
	date := tp.extractDate(text)
	time_ := tp.extractTime(text)
	priority := tp.extractPriority(text)
	list := tp.extractList(text)

	// 计算置信度
	confidence := tp.calculateConfidence(text, title, date, time_)

	taskInfo := &models.ParsedTaskInfo{
		Title:        title,
		Description:  description,
		Date:         date,
		Time:         time_,
		Priority:     priority,
		List:         list,
		OriginalText: text,
		Confidence:   confidence,
	}

	// 标准化处理
	return tp.enhanceParsedInfo(taskInfo)
}

// extractTitle extracts a suitable title from text
func (tp *TaskParser) extractTitle(text string) string {
	// 移除常见的无用前缀
	prefixes := []string{
		"提醒：", "提醒:", "通知：", "通知:",
		"会议：", "会议:", "任务：", "任务:",
		"待办：", "待办:", "TODO:", "todo:",
	}

	cleanText := text
	for _, prefix := range prefixes {
		cleanText = strings.TrimPrefix(cleanText, prefix)
	}

	// 按行分割，取第一行作为标题
	lines := strings.Split(strings.TrimSpace(cleanText), "\n")
	if len(lines) > 0 {
		title := strings.TrimSpace(lines[0])
		if len(title) > 50 {
			title = title[:50] + "..."
		}
		return title
	}

	// 如果没有换行，取前50个字符
	if len(cleanText) > 50 {
		cleanText = cleanText[:50] + "..."
	}

	return cleanText
}

// extractDescription extracts description information
func (tp *TaskParser) extractDescription(text string) string {
	// 移除邮件地址、电话号码等私人信息
	description := text

	// 移除邮件地址
	description = tp.emailPattern.ReplaceAllString(description, "[邮箱]")

	// 移除电话号码
	description = tp.phonePattern.ReplaceAllString(description, "[电话]")

	// 移除URL（保留域名信息）
	description = tp.urlPattern.ReplaceAllStringFunc(description, func(url string) string {
		if strings.Contains(url, "meeting") || strings.Contains(url, "zoom") || strings.Contains(url, "teams") {
			return "[会议链接]"
		}
		return "[链接]"
	})

	return description
}

// extractDate extracts date information from text
func (tp *TaskParser) extractDate(text string) string {
	matches := tp.dateTimePattern.FindStringSubmatch(text)
	if len(matches) > 1 {
		return tp.normalizeDate(matches[1])
	}
	return ""
}

// extractTime extracts time information from text
func (tp *TaskParser) extractTime(text string) string {
	matches := tp.dateTimePattern.FindStringSubmatch(text)
	if len(matches) > 2 && matches[2] != "" {
		return tp.normalizeTime(matches[2])
	}
	return ""
}

// extractPriority extracts priority information from text
func (tp *TaskParser) extractPriority(text string) string {
	matches := tp.priorityPattern.FindStringSubmatch(text)
	if len(matches) > 1 {
		return tp.normalizePriority(matches[1])
	}
	return ""
}

// extractList extracts list/category information from text
func (tp *TaskParser) extractList(text string) string {
	text = strings.ToLower(text)

	if strings.Contains(text, "会议") || strings.Contains(text, "meeting") {
		return "会议"
	}
	if strings.Contains(text, "工作") || strings.Contains(text, "work") {
		return "工作"
	}
	if strings.Contains(text, "个人") || strings.Contains(text, "personal") {
		return "个人"
	}
	if strings.Contains(text, "学习") || strings.Contains(text, "study") {
		return "学习"
	}

	return ""
}

// normalizeDate normalizes date to YYYY-MM-DD format
func (tp *TaskParser) normalizeDate(dateStr string) string {
	now := time.Now()

	// 处理相对日期
	switch strings.ToLower(dateStr) {
	case "今天":
		return now.Format("2006-01-02")
	case "明天":
		return now.AddDate(0, 0, 1).Format("2006-01-02")
	case "后天":
		return now.AddDate(0, 0, 2).Format("2006-01-02")
	case "昨天":
		return now.AddDate(0, 0, -1).Format("2006-01-02")
	case "前天":
		return now.AddDate(0, 0, -2).Format("2006-01-02")
	}

	// 处理"N天后"格式
	if daysMatch := regexp.MustCompile(`(\d+)天后`).FindStringSubmatch(dateStr); len(daysMatch) > 1 {
		if days, err := strconv.Atoi(daysMatch[1]); err == nil {
			return now.AddDate(0, 0, days).Format("2006-01-02")
		}
	}

	// 处理"N天前"格式
	if daysMatch := regexp.MustCompile(`(\d+)天前`).FindStringSubmatch(dateStr); len(daysMatch) > 1 {
		if days, err := strconv.Atoi(daysMatch[1]); err == nil {
			return now.AddDate(0, 0, -days).Format("2006-01-02")
		}
	}

	// 处理"X月Y日"格式
	if monthDayMatch := regexp.MustCompile(`(\d+)月(\d+)日`).FindStringSubmatch(dateStr); len(monthDayMatch) > 2 {
		if month, err := strconv.Atoi(monthDayMatch[1]); err == nil {
			if day, err := strconv.Atoi(monthDayMatch[2]); err == nil {
				return time.Date(now.Year(), time.Month(month), day, 0, 0, 0, 0, time.Local).Format("2006-01-02")
			}
		}
	}

	// 尝试解析标准日期格式
	formats := []string{
		"2006-01-02", "2006/01/02", "2006.01.02",
		"01-02-2006", "01/02/2006", "01.02.2006",
		"2006-1-2", "2006/1/2", "2006.1.2",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t.Format("2006-01-02")
		}
	}

	return dateStr // 返回原始字符串
}

// normalizeTime normalizes time to HH:MM format
func (tp *TaskParser) normalizeTime(timeStr string) string {

	// 处理相对时间
	switch strings.ToLower(timeStr) {
	case "上午", "早上", "清晨":
		return "09:00"
	case "中午", "午间":
		return "12:00"
	case "下午", "午后":
		return "14:00"
	case "晚上", "夜间", "傍晚":
		return "18:00"
	case "深夜", "凌晨":
		return "23:00"
	}

	// 处理"X点"格式
	if hourMatch := regexp.MustCompile(`(\d+)点`).FindStringSubmatch(timeStr); len(hourMatch) > 1 {
		if hour, err := strconv.Atoi(hourMatch[1]); err == nil {
			if hour >= 0 && hour <= 23 {
				return fmt.Sprintf("%02d:00", hour)
			}
		}
	}

	// 处理"X时"格式
	if hourMatch := regexp.MustCompile(`(\d+)时`).FindStringSubmatch(timeStr); len(hourMatch) > 1 {
		if hour, err := strconv.Atoi(hourMatch[1]); err == nil {
			if hour >= 0 && hour <= 23 {
				return fmt.Sprintf("%02d:00", hour)
			}
		}
	}

	// 尝试解析标准时间格式
	formats := []string{
		"15:04", "15:04:05", "3:04 PM", "3:04:05 PM",
		"下午3:04", "上午3:04",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timeStr); err == nil {
			return t.Format("15:04")
		}
	}

	return timeStr // 返回原始字符串
}

// normalizePriority normalizes priority to standard values
func (tp *TaskParser) normalizePriority(priorityStr string) string {
	priorityStr = strings.ToLower(strings.TrimSpace(priorityStr))

	switch priorityStr {
	case "紧急", "急", "urgent", "asap", "high", "高":
		return "high"
	case "低", "low", "一般", "normal":
		return "low"
	case "中", "medium", "中等":
		return "medium"
	default:
		return "medium"
	}
}

// inferRemindBefore infers appropriate reminder time based on task content
func (tp *TaskParser) inferRemindBefore(info *models.ParsedTaskInfo) string {
	// 会议类型提前15分钟
	if strings.Contains(strings.ToLower(info.List), "会议") ||
	   strings.Contains(strings.ToLower(info.Title), "会议") {
		return "15m"
	}

	// 紧急任务提前5分钟
	if info.Priority == "high" {
		return "5m"
	}

	// 默认提前15分钟
	return "15m"
}

// inferList infers appropriate list based on task content
func (tp *TaskParser) inferList(info *models.ParsedTaskInfo) string {
	title := strings.ToLower(info.Title)
	description := strings.ToLower(info.Description)

	// 会议相关
	if strings.Contains(title, "会议") || strings.Contains(description, "会议") ||
	   strings.Contains(title, "meeting") || strings.Contains(description, "meeting") {
		return "会议"
	}

	// 工作相关
	if strings.Contains(title, "工作") || strings.Contains(description, "工作") ||
	   strings.Contains(title, "work") || strings.Contains(description, "work") {
		return "工作"
	}

	// 学习相关
	if strings.Contains(title, "学习") || strings.Contains(description, "学习") ||
	   strings.Contains(title, "study") || strings.Contains(description, "study") {
		return "学习"
	}

	return "Default"
}

// enhanceDescription enhances description with additional context
func (tp *TaskParser) enhanceDescription(info *models.ParsedTaskInfo) string {
	if info.Description == "" {
		return info.Title
	}

	// 如果描述和标题相同，只返回标题
	if strings.TrimSpace(info.Description) == strings.TrimSpace(info.Title) {
		return info.Title
	}

	return info.Description
}

// calculateConfidence calculates confidence score for parsed information
func (tp *TaskParser) calculateConfidence(text, title, date, time_ string) float64 {
	confidence := 0.0

	// 基础分数
	if len(strings.TrimSpace(text)) > 10 {
		confidence += 0.1
	}

	// 标题分数
	if title != "" {
		confidence += 0.2
	}

	// 日期分数
	if date != "" {
		confidence += 0.3
		if tp.isValidDate(date) {
			confidence += 0.1
		}
	}

	// 时间分数
	if time_ != "" {
		confidence += 0.2
		if tp.isValidTime(time_) {
			confidence += 0.1
		}
	}

	// 关键词分数
	if strings.Contains(strings.ToLower(text), "会议") || strings.Contains(strings.ToLower(text), "meeting") {
		confidence += 0.1
	}

	if strings.Contains(strings.ToLower(text), "提醒") || strings.Contains(strings.ToLower(text), "reminder") {
		confidence += 0.1
	}

	// 确保置信度在0-1范围内
	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

// isValidDate checks if date string is valid
func (tp *TaskParser) isValidDate(dateStr string) bool {
	// 尝试解析为日期
	formats := []string{
		"2006-01-02", "2006/01/02", "2006.01.02",
		"01-02-2006", "01/02/2006", "01.02.2006",
	}

	for _, format := range formats {
		if _, err := time.Parse(format, dateStr); err == nil {
			return true
		}
	}

	return false
}

// isValidTime checks if time string is valid
func (tp *TaskParser) isValidTime(timeStr string) bool {
	formats := []string{
		"15:04", "15:04:05", "3:04 PM", "3:04:05 PM",
	}

	for _, format := range formats {
		if _, err := time.Parse(format, timeStr); err == nil {
			return true
		}
	}

	return false
}

// GetParserInfo returns parser information and capabilities
func (tp *TaskParser) GetParserInfo() map[string]interface{} {
	return map[string]interface{}{
		"version": "1.0.0",
		"capabilities": []string{
			"dify_response_parsing",
			"plain_text_parsing",
			"date_normalization",
			"time_normalization",
			"priority_detection",
			"list_inference",
		},
		"supported_formats": []string{
			"json",
			"plain_text",
			"mixed_content",
		},
		"confidence_threshold": 0.5,
	}
}