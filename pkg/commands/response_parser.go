package commands

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/allanpk716/to_icalendar/pkg/models"
)

// ParseDifyResponseToReminder 将 Dify AI 的响应解析为 Reminder 对象
func ParseDifyResponseToReminder(difyResponse *models.DifyResponse, contentType string, originalContent string) (*models.Reminder, error) {
	if difyResponse == nil {
		return nil, fmt.Errorf("Dify 响应为空")
	}

	reminder := &models.Reminder{}

	// 解析 Dify 响应
	if difyResponse.Answer != "" {
		// 如果 Dify 返回了结构化的答案，尝试解析
		err := parseStructuredAnswer(difyResponse.Answer, reminder)
		if err != nil {
			// 如果解析失败，使用答案作为标题，原内容作为描述
			reminder.Title = truncateTitle(difyResponse.Answer)
			reminder.Description = originalContent
		}
	} else {
		// 如果没有答案，使用原内容创建基本提醒
		reminder.Title = generateDefaultTitle(contentType)
		reminder.Description = originalContent
	}

	// 设置默认值
	if reminder.Priority == "" {
		reminder.Priority = models.PriorityMedium
	}
	if reminder.List == "" {
		reminder.List = "Tasks"
	}

	// 设置默认时间和日期（如果未设置）
	if reminder.Date == "" {
		reminder.Date = time.Now().Format("2006-01-02")
	}
	if reminder.Time == "" {
		reminder.Time = time.Now().Format("15:04")
	}

	return reminder, nil
}

// parseStructuredAnswer 解析结构化的 Dify 答案
func parseStructuredAnswer(answer string, reminder *models.Reminder) error {
	// 尝试解析 JSON 格式的响应
	var structuredResponse map[string]interface{}
	if err := json.Unmarshal([]byte(answer), &structuredResponse); err != nil {
		// 如果不是 JSON，尝试简单的文本解析
		return parseTextAnswer(answer, reminder)
	}

	// 解析结构化字段
	if title, ok := structuredResponse["title"].(string); ok && title != "" {
		reminder.Title = truncateTitle(title)
	}

	if description, ok := structuredResponse["description"].(string); ok && description != "" {
		reminder.Description = description
	}

	if date, ok := structuredResponse["date"].(string); ok && date != "" {
		reminder.Date = date
	}

	if time, ok := structuredResponse["time"].(string); ok && time != "" {
		reminder.Time = time
	}

	if priority, ok := structuredResponse["priority"].(string); ok && priority != "" {
		if isValidPriority(priority) {
			reminder.Priority = models.Priority(priority)
		}
	}

	if list, ok := structuredResponse["list"].(string); ok && list != "" {
		reminder.List = list
	}

	return nil
}

// parseTextAnswer 解析文本格式的答案
func parseTextAnswer(answer string, reminder *models.Reminder) error {
	lines := splitLines(answer)

	// 寻找标题行（通常在开头）
	for i, line := range lines {
		line = trimString(line)
		if line == "" {
			continue
		}

		// 第一行非空文本作为标题
		if reminder.Title == "" {
			reminder.Title = truncateTitle(line)
			// 剩余行作为描述
			if i < len(lines)-1 {
				reminder.Description = joinLines(lines[i+1:])
			}
			break
		}
	}

	// 尝试从文本中提取时间信息
	timeInfo := extractTimeFromText(answer)
	if !timeInfo.IsZero() {
		reminder.Date = timeInfo.Format("2006-01-02")
		reminder.Time = timeInfo.Format("15:04")
	}

	return nil
}

// parseTime 解析时间字符串
func parseTime(timeStr string) (time.Time, error) {
	// 尝试常见的时间格式
	formats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
		"15:04:05",
		"15:04",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timeStr); err == nil {
			return t, nil
		}
	}

	// 如果都失败了，返回当前时间
	return time.Time{}, fmt.Errorf("无法解析时间格式: %s", timeStr)
}

// truncateTitle 截断标题长度
func truncateTitle(title string) string {
	maxLength := 100
	if len(title) > maxLength {
		return title[:maxLength-3] + "..."
	}
	return title
}

// generateDefaultTitle 生成默认标题
func generateDefaultTitle(contentType string) string {
	switch contentType {
	case "text":
		return "剪贴板文本提醒"
	case "image":
		return "剪贴板图片提醒"
	default:
		return "剪贴板内容提醒"
	}
}

// isValidPriority 检查优先级是否有效
func isValidPriority(priority string) bool {
	switch priority {
	case "high", "medium", "low":
		return true
	default:
		return false
	}
}

// extractTimeFromText 从文本中提取时间信息
func extractTimeFromText(text string) time.Time {
	// 简单的时间关键词提取
	keywords := map[string]string{
		"明天":    time.Now().AddDate(0, 0, 1).Format("2006-01-02"),
		"今天":    time.Now().Format("2006-01-02"),
		"后天":    time.Now().AddDate(0, 0, 2).Format("2006-01-02"),
		"下周":    time.Now().AddDate(0, 0, 7).Format("2006-01-02"),
	}

	for keyword, dateStr := range keywords {
		if contains(text, keyword) {
			if t, err := time.Parse("2006-01-02", dateStr); err == nil {
				return t
			}
		}
	}

	return time.Time{} // 返回零值表示没有找到时间
}

// 简单的字符串处理函数
func splitLines(text string) []string {
	lines := make([]string, 0)
	start := 0
	for i, c := range text {
		if c == '\n' {
			lines = append(lines, text[start:i])
			start = i + 1
		}
	}
	lines = append(lines, text[start:])
	return lines
}

func joinLines(lines []string) string {
	result := ""
	for i, line := range lines {
		if i > 0 {
			result += "\n"
		}
		result += line
	}
	return result
}

func trimString(s string) string {
	start := 0
	end := len(s)

	// 去除前导空格
	for start < end && s[start] == ' ' {
		start++
	}

	// 去除尾随空格
	for end > start && s[end-1] == ' ' {
		end--
	}

	return s[start:end]
}

func contains(text, substr string) bool {
	return len(text) >= len(substr) && (text == substr ||
		(len(text) > len(substr) &&
			(text[:len(substr)] == substr ||
			 text[len(text)-len(substr):] == substr ||
			 containsMiddle(text, substr))))
}

func containsMiddle(text, substr string) bool {
	for i := 0; i <= len(text)-len(substr); i++ {
		if text[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}