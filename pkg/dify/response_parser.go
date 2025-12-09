package dify

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/allanpk716/to_icalendar/pkg/models"
)

// ResponseParser defines the interface for parsing Dify responses
type ResponseParser interface {
	// ParseReminderResponse parses Dify response and extracts task information
	ParseReminderResponse(response string) (*models.ParsedTaskInfo, error)
}

// ResponseParserImpl implements ResponseParser for Dify workflow responses
type ResponseParserImpl struct{}

// NewResponseParser creates a new ResponseParser instance
func NewResponseParser() ResponseParser {
	return &ResponseParserImpl{}
}

// ParseReminderResponse parses Dify workflow response and extracts task information
func (p *ResponseParserImpl) ParseReminderResponse(response string) (*models.ParsedTaskInfo, error) {
	log.Printf("开始解析Dify响应，长度: %d", len(response))
	log.Printf("原始响应内容: %s", response)

	// 清理响应内容
	cleanedResponse := strings.TrimSpace(response)
	if cleanedResponse == "" {
		return nil, fmt.Errorf("empty response from Dify")
	}

	// 首先尝试直接解析为JSON
	var taskInfo models.ParsedTaskInfo
	if err := json.Unmarshal([]byte(cleanedResponse), &taskInfo); err == nil {
		// 验证关键字段是否存在
		if p.validateTaskInfo(&taskInfo) {
			log.Printf("直接JSON解析成功")
			return &taskInfo, nil
		}
	}

	// 尝试解析为Dify工作流响应结构
	var difyResp models.DifyResponse
	if err := json.Unmarshal([]byte(cleanedResponse), &difyResp); err == nil {
		log.Printf("识别为Dify工作流响应")

		// 尝试从Answer字段获取任务信息
		if difyResp.Answer != "" {
			log.Printf("从Answer字段解析，内容: %s", difyResp.Answer)
			if taskInfo, err := p.parseTaskJSON(difyResp.Answer); err == nil {
				return taskInfo, nil
			}
		}

		// 尝试从Data.Outputs.Text字段获取任务信息
		if difyResp.Data != nil && difyResp.Data.Outputs != nil && difyResp.Data.Outputs.Text != "" {
			log.Printf("从Data.Outputs.Text字段解析，内容: %s", difyResp.Data.Outputs.Text)
			if taskInfo, err := p.parseTaskJSON(difyResp.Data.Outputs.Text); err == nil {
				return taskInfo, nil
			}
		}
	}

	// 如果直接解析失败，尝试从响应中提取JSON
	taskInfoPtr, err := p.extractJSONFromResponse(cleanedResponse)
	if err != nil {
		log.Printf("JSON提取失败，尝试文本解析: %v", err)
		// 如果无法提取有效JSON，尝试从文本中解析任务信息
		return p.parseTaskFromText(cleanedResponse)
	}

	// 验证解析结果
	if !p.validateTaskInfo(taskInfoPtr) {
		return nil, fmt.Errorf("parsed task info is incomplete or invalid")
	}

	log.Printf("成功解析任务信息: 标题='%s', 日期='%s', 时间='%s'",
		taskInfoPtr.Title, taskInfoPtr.Date, taskInfoPtr.Time)

	return taskInfoPtr, nil
}

// parseTaskJSON 解析任务JSON字符串
func (p *ResponseParserImpl) parseTaskJSON(jsonStr string) (*models.ParsedTaskInfo, error) {
	log.Printf("尝试解析任务JSON，原始长度: %d", len(jsonStr))
	log.Printf("原始JSON字符串: %s", jsonStr)

	// 清理JSON字符串
	jsonStr = strings.TrimSpace(jsonStr)
	if jsonStr == "" {
		return nil, fmt.Errorf("empty JSON string")
	}

	// 如果JSON字符串被引号包围，去掉引号
	if len(jsonStr) >= 2 && jsonStr[0] == '"' && jsonStr[len(jsonStr)-1] == '"' {
		jsonStr = jsonStr[1 : len(jsonStr)-1]
		log.Printf("去除引号后: %s", jsonStr)

		// 处理转义字符 - 使用更完整的转义处理
		jsonStr = strings.ReplaceAll(jsonStr, "\\\"", "\"")
		jsonStr = strings.ReplaceAll(jsonStr, "\\\\", "\\")
		jsonStr = strings.ReplaceAll(jsonStr, "\\n", "\n")
		jsonStr = strings.ReplaceAll(jsonStr, "\\t", "\t")
		jsonStr = strings.ReplaceAll(jsonStr, "\\r", "\r")
		jsonStr = strings.ReplaceAll(jsonStr, "\\f", "\f")
		jsonStr = strings.ReplaceAll(jsonStr, "\\b", "\b")

		log.Printf("处理转义字符后: %s", jsonStr)
	}

	// 尝试解析JSON
	var taskInfo models.ParsedTaskInfo
	if err := json.Unmarshal([]byte(jsonStr), &taskInfo); err != nil {
		log.Printf("JSON解析失败: %v, 原始内容: %s", err, jsonStr)
		return nil, fmt.Errorf("failed to parse task JSON: %w", err)
	}

	if !p.validateTaskInfo(&taskInfo) {
		log.Printf("任务信息验证失败: 标题='%s', 日期='%s', 时间='%s'",
			taskInfo.Title, taskInfo.Date, taskInfo.Time)
		return nil, fmt.Errorf("parsed task info is incomplete or invalid")
	}

	log.Printf("任务JSON解析成功: 标题='%s', 日期='%s', 时间='%s'",
		taskInfo.Title, taskInfo.Date, taskInfo.Time)

	return &taskInfo, nil
}

// extractJSONFromResponse attempts to extract JSON from a mixed response
func (p *ResponseParserImpl) extractJSONFromResponse(response string) (*models.ParsedTaskInfo, error) {
	// 查找JSON对象的开始和结束
	jsonStart := strings.Index(response, "{")
	jsonEnd := strings.LastIndex(response, "}")

	if jsonStart < 0 || jsonEnd <= jsonStart {
		return nil, fmt.Errorf("no valid JSON found in response")
	}

	jsonStr := response[jsonStart : jsonEnd+1]
	var taskInfo models.ParsedTaskInfo

	if err := json.Unmarshal([]byte(jsonStr), &taskInfo); err != nil {
		return nil, fmt.Errorf("failed to parse extracted JSON: %w", err)
	}

	return &taskInfo, nil
}

// parseTaskFromText attempts to parse task information from plain text response
func (p *ResponseParserImpl) parseTaskFromText(text string) (*models.ParsedTaskInfo, error) {
	// 这是一个简单的文本解析实现，您可以根据需要扩展
	// 尝试识别常见的任务描述模式

	lines := strings.Split(text, "\n")
	taskInfo := &models.ParsedTaskInfo{
		OriginalText: text,
		Confidence:   0.6, // 文本解析的置信度较低
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 尝试提取标题（通常是不包含日期时间的关键词行）
		if taskInfo.Title == "" && p.looksLikeTitle(line) {
			taskInfo.Title = line
			continue
		}

		// 尝试提取日期时间
		if date, time := p.extractDateTime(line); date != "" && time != "" {
			taskInfo.Date = date
			taskInfo.Time = time
			continue
		}

		// 如果还没有描述，将第一行作为描述
		if taskInfo.Description == "" && line != taskInfo.Title {
			taskInfo.Description = line
		}
	}

	// 如果没有找到标题，使用文本的第一行
	if taskInfo.Title == "" {
		taskInfo.Title = p.getFirstMeaningfulLine(text)
	}

	// 设置默认优先级
	taskInfo.Priority = "medium"
	taskInfo.RemindBefore = "15m"

	if !p.validateTaskInfo(taskInfo) {
		return nil, fmt.Errorf("could not extract valid task information from text")
	}

	return taskInfo, nil
}

// validateTaskInfo validates that the task info contains required fields
func (p *ResponseParserImpl) validateTaskInfo(taskInfo *models.ParsedTaskInfo) bool {
	if taskInfo == nil {
		return false
	}

	// 标题是必需的
	if strings.TrimSpace(taskInfo.Title) == "" {
		return false
	}

	// 日期和时间是必需的
	if strings.TrimSpace(taskInfo.Date) == "" || strings.TrimSpace(taskInfo.Time) == "" {
		return false
	}

	// 验证日期格式 (YYYY-MM-DD)
	if !p.isValidDateFormat(taskInfo.Date) {
		return false
	}

	// 验证时间格式 (HH:MM)
	if !p.isValidTimeFormat(taskInfo.Time) {
		return false
	}

	return true
}

// looksLikeTitle checks if a line looks like a task title
func (p *ResponseParserImpl) looksLikeTitle(line string) bool {
	// 排除明显是日期时间或描述的行
	lowerLine := strings.ToLower(line)

	excludePatterns := []string{
		"日期", "时间", "提醒", "priority", "date:", "time:",
		"上午", "下午", "明天", "今天", "紧急", "重要",
	}

	for _, pattern := range excludePatterns {
		if strings.Contains(lowerLine, pattern) {
			return false
		}
	}

	// 如果长度适中且不包含数字，可能是标题
	if len(line) > 5 && len(line) < 100 && !p.containsNumbers(line) {
		return true
	}

	return false
}

// extractDateTime attempts to extract date and time from a line
func (p *ResponseParserImpl) extractDateTime(line string) (string, string) {
	// 简单的日期时间提取逻辑
	// 您可以根据需要使用正则表达式来改进

	// 查找日期格式 YYYY-MM-DD
	words := strings.Fields(line)
	for i, word := range words {
		if p.isValidDateFormat(word) && i+1 < len(words) && p.isValidTimeFormat(words[i+1]) {
			return word, words[i+1]
		}
	}

	return "", ""
}

// isValidDateFormat checks if the string is a valid date format (YYYY-MM-DD)
func (p *ResponseParserImpl) isValidDateFormat(dateStr string) bool {
	if len(dateStr) != 10 {
		return false
	}

	if dateStr[4] != '-' || dateStr[7] != '-' {
		return false
	}

	// 验证数字格式
	for i, char := range dateStr {
		if i == 4 || i == 7 {
			continue
		}
		if char < '0' || char > '9' {
			return false
		}
	}

	return true
}

// isValidTimeFormat checks if the string is a valid time format (HH:MM or HH:MM - HH:MM)
func (p *ResponseParserImpl) isValidTimeFormat(timeStr string) bool {
	timeStr = strings.TrimSpace(timeStr)

	// 检查是否是时间范围格式 (HH:MM - HH:MM)
	if strings.Contains(timeStr, " - ") {
		parts := strings.Split(timeStr, " - ")
		if len(parts) != 2 {
			return false
		}
		// 检查开始时间
		if !p.isValidSingleTimeFormat(strings.TrimSpace(parts[0])) {
			return false
		}
		// 检查结束时间
		if !p.isValidSingleTimeFormat(strings.TrimSpace(parts[1])) {
			return false
		}
		return true
	}

	// 检查单个时间格式 (HH:MM)
	return p.isValidSingleTimeFormat(timeStr)
}

// isValidSingleTimeFormat checks if the string is a valid single time format
// 支持格式: HH:MM, H:MM, 下午H:MM, H:MM PM, 等
func (p *ResponseParserImpl) isValidSingleTimeFormat(timeStr string) bool {
	timeStr = strings.TrimSpace(timeStr)

	// 支持多种时间格式
	timeFormats := []string{
		"15:04",      // 14:30
		"3:04",       // 9:30
		"下午3:04",    // 中文格式
		"上午3:04",    // 中文格式
		"3:04 PM",    // 英文格式
		"3:04 AM",    // 英文格式
	}

	for _, format := range timeFormats {
		if _, err := time.Parse(format, timeStr); err == nil {
			return true
		}
	}

	// 传统格式验证（保持向后兼容）
	if len(timeStr) == 5 && timeStr[2] == ':' {
		hours := timeStr[:2]
		minutes := timeStr[3:]

		// 验证小时 (0-23)
		if hour, err := strconv.Atoi(hours); err == nil {
			if hour < 0 || hour > 23 {
				return false
			}
		} else {
			return false
		}

		// 验证分钟 (0-59)
		if minute, err := strconv.Atoi(minutes); err == nil {
			if minute < 0 || minute > 59 {
				return false
			}
		} else {
			return false
		}

		return true
	}

	return false
}

// containsNumbers checks if the string contains numbers
func (p *ResponseParserImpl) containsNumbers(s string) bool {
	for _, char := range s {
		if char >= '0' && char <= '9' {
			return true
		}
	}
	return false
}

// getFirstMeaningfulLine gets the first non-empty line from text
func (p *ResponseParserImpl) getFirstMeaningfulLine(text string) string {
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			// 截断过长的行
			if len(line) > 100 {
				line = line[:100] + "..."
			}
			return line
		}
	}
	return "未知任务"
}