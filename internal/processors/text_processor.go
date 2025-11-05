package processors

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/allanpk716/to_icalendar/internal/dify"
	"github.com/allanpk716/to_icalendar/internal/models"
)

// TextProcessor handles text content processing workflow
type TextProcessor struct {
	difyProcessor *dify.Processor
	// 预编译的正则表达式用于常见任务模式识别
	datePattern   *regexp.Regexp
	timePattern   *regexp.Regexp
	urgentPattern *regexp.Regexp
	meetingPattern *regexp.Regexp
}

// NewTextProcessor creates a new text processor
func NewTextProcessor(difyProcessor *dify.Processor) (*TextProcessor, error) {
	// 编译正则表达式
	datePattern := regexp.MustCompile(`(\d{4}[-/]\d{1,2}[-/]\d{1,2}|今天|明天|后天|\d+天后|\d+月\d+日)`)
	timePattern := regexp.MustCompile(`(\d{1,2}:\d{2}|\d{1,2}点|\d{1,2}时|上午|下午|早上|晚上|中午)`)
	urgentPattern := regexp.MustCompile(`(?i)(紧急|急|urgent|asap|尽快|立即|马上)`)
	meetingPattern := regexp.MustCompile(`(?i)(会议|meeting|开会|讨论|讨论会|评审会|周会|例会)`)

	return &TextProcessor{
		difyProcessor: difyProcessor,
		datePattern:   datePattern,
		timePattern:   timePattern,
		urgentPattern: urgentPattern,
		meetingPattern: meetingPattern,
	}, nil
}

// ProcessClipboardText processes text from clipboard
func (tp *TextProcessor) ProcessClipboardText(ctx context.Context, text string) (*models.ProcessingResult, error) {
	startTime := time.Now()

	log.Printf("开始处理剪贴板文字，长度: %d", len(text))

	// 预处理文字内容
	cleanText := tp.preprocessText(text)
	log.Printf("预处理后文字: %s", cleanText)

	// 使用Dify处理器处理文字
	response, err := tp.difyProcessor.ProcessText(ctx, cleanText)
	if err != nil {
		return &models.ProcessingResult{
			Success:        false,
			ErrorMessage:   fmt.Sprintf("Dify处理失败: %v", err),
			ProcessingTime: time.Since(startTime),
		}, err
	}

	// 转换为处理结果格式
	result := &models.ProcessingResult{
		Success:        response.Success,
		Reminder:       response.Reminder,
		ParsedInfo:     response.ParsedInfo,
		ErrorMessage:   response.ErrorMessage,
		ProcessingTime: time.Since(startTime),
	}

	log.Printf("文字处理完成，成功: %v", result.Success)

	return result, nil
}

// ProcessTextFile processes text from a file
func (tp *TextProcessor) ProcessTextFile(ctx context.Context, filePath string) (*models.ProcessingResult, error) {
	startTime := time.Now()

	log.Printf("开始处理文字文件: %s", filePath)

	// 这里可以实现文件读取功能
	// 由于当前需求主要是剪贴板处理，暂时返回不支持
	return &models.ProcessingResult{
		Success:      false,
		ErrorMessage: "暂不支持文件处理，请使用剪贴板功能",
		ProcessingTime: time.Since(startTime),
	}, fmt.Errorf("file processing not supported")
}

// QuickAnalyze performs a quick analysis without calling Dify API
func (tp *TextProcessor) QuickAnalyze(text string) *QuickAnalysisResult {
	result := &QuickAnalysisResult{
		OriginalText: text,
		HasDate:      tp.datePattern.MatchString(text),
		HasTime:      tp.timePattern.MatchString(text),
		IsUrgent:     tp.urgentPattern.MatchString(text),
		IsMeeting:    tp.meetingPattern.MatchString(text),
		WordCount:    len(strings.Fields(text)),
	}

	// 提取日期和时间
	if result.HasDate {
		result.Dates = tp.datePattern.FindAllString(text, -1)
	}
	if result.HasTime {
		result.Times = tp.timePattern.FindAllString(text, -1)
	}

	// 估算置信度
	result.Confidence = tp.calculateConfidence(result)

	return result
}

// preprocessText cleans and normalizes text content
func (tp *TextProcessor) preprocessText(text string) string {
	// 移除多余空白字符
	text = strings.TrimSpace(text)

	// 替换多个连续空白为单个空格
	spacePattern := regexp.MustCompile(`\s+`)
	text = spacePattern.ReplaceAllString(text, " ")

	// 移除常见的无用字符
	text = strings.ReplaceAll(text, "\u200B", "") // 零宽空格
	text = strings.ReplaceAll(text, "\u00A0", " ") // 不间断空格

	return text
}

// calculateConfidence calculates confidence score for text analysis
func (tp *TextProcessor) calculateConfidence(analysis *QuickAnalysisResult) float64 {
	confidence := 0.0

	// 基础分数
	if analysis.WordCount > 5 {
		confidence += 0.2
	}

	// 日期时间分数
	if analysis.HasDate {
		confidence += 0.3
	}
	if analysis.HasTime {
		confidence += 0.3
	}

	// 上下文分数
	if analysis.IsMeeting {
		confidence += 0.1
	}
	if analysis.IsUrgent {
		confidence += 0.1
	}

	// 确保置信度在0-1范围内
	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

// extractTaskInfo attempts to extract task information without AI
func (tp *TextProcessor) extractTaskInfo(text string) *models.ParsedTaskInfo {
	analysis := tp.QuickAnalyze(text)

	// 如果置信度太低，返回空结果
	if analysis.Confidence < 0.3 {
		return &models.ParsedTaskInfo{
			OriginalText: text,
			Confidence:   0.0,
			Description:  "文字内容置信度太低，无法提取任务信息",
		}
	}

	// 构建基础任务信息
	info := &models.ParsedTaskInfo{
		Title:        tp.extractTitle(text),
		Description:  text,
		OriginalText: text,
		Confidence:   analysis.Confidence,
	}

	// 尝试提取日期时间
	if len(analysis.Dates) > 0 {
		info.Date = tp.normalizeDate(analysis.Dates[0])
	}
	if len(analysis.Times) > 0 {
		info.Time = tp.normalizeTime(analysis.Times[0])
	}

	// 设置优先级
	if analysis.IsUrgent {
		info.Priority = "high"
	} else if analysis.IsMeeting {
		info.Priority = "medium"
	}

	// 设置列表
	if analysis.IsMeeting {
		info.List = "会议"
	}

	return info
}

// extractTitle extracts a suitable title from text
func (tp *TextProcessor) extractTitle(text string) string {
	// 取前50个字符作为标题
	title := text
	if len(title) > 50 {
		title = title[:50] + "..."
	}

	return title
}

// normalizeDate normalizes various date formats to YYYY-MM-DD
func (tp *TextProcessor) normalizeDate(dateStr string) string {
	now := time.Now()

	switch dateStr {
	case "今天":
		return now.Format("2006-01-02")
	case "明天":
		return now.AddDate(0, 0, 1).Format("2006-01-02")
	case "后天":
		return now.AddDate(0, 0, 2).Format("2006-01-02")
	default:
		// 尝试解析其他格式
		if strings.Contains(dateStr, "天后") {
			if daysMatch := regexp.MustCompile(`(\d+)天后`).FindStringSubmatch(dateStr); len(daysMatch) > 1 {
				if days, err := time.ParseDuration(daysMatch[1] + "d"); err == nil {
					return now.Add(days).Format("2006-01-02")
				}
			}
		}

		// 返回原始字符串，让Dify处理
		return dateStr
	}
}

// normalizeTime normalizes various time formats to HH:MM
func (tp *TextProcessor) normalizeTime(timeStr string) string {
	now := time.Now()

	// 处理相对时间
	switch timeStr {
	case "上午", "早上":
		return "09:00"
	case "中午":
		return "12:00"
	case "下午":
		return "14:00"
	case "晚上":
		return "18:00"
	default:
		// 尝试解析数字时间
		if hourMatch := regexp.MustCompile(`(\d+)点`).FindStringSubmatch(timeStr); len(hourMatch) > 1 {
			if hour, err := time.ParseDuration(hourMatch[1] + "h"); err == nil {
				return now.Add(hour).Format("15:04")
			}
		}

		// 返回原始字符串，让Dify处理
		return timeStr
	}
}

// QuickAnalysisResult represents the result of quick text analysis
type QuickAnalysisResult struct {
	OriginalText string    `json:"original_text"`
	HasDate      bool      `json:"has_date"`
	HasTime      bool      `json:"has_time"`
	IsUrgent     bool      `json:"is_urgent"`
	IsMeeting    bool      `json:"is_meeting"`
	Dates        []string  `json:"dates,omitempty"`
	Times        []string  `json:"times,omitempty"`
	WordCount    int       `json:"word_count"`
	Confidence   float64   `json:"confidence"`
}

// GetProcessingStats returns processing statistics
func (tp *TextProcessor) GetProcessingStats() map[string]interface{} {
	return map[string]interface{}{
		"supported_patterns": []string{
			"date_pattern",
			"time_pattern",
			"urgent_pattern",
			"meeting_pattern",
		},
		"max_text_length": 10000, // 最大文字长度
		"min_confidence":  0.3,   // 最小置信度阈值
	}
}

// ValidateText validates text content for processing
func (tp *TextProcessor) ValidateText(text string) error {
	if strings.TrimSpace(text) == "" {
		return fmt.Errorf("text content is empty")
	}

	if len(text) > 10000 {
		return fmt.Errorf("text content too long: %d characters (max: 10000)", len(text))
	}

	return nil
}

// SetDifyProcessor updates the Dify processor
func (tp *TextProcessor) SetDifyProcessor(processor *dify.Processor) {
	tp.difyProcessor = processor
}

// GetDifyProcessor returns the current Dify processor
func (tp *TextProcessor) GetDifyProcessor() *dify.Processor {
	return tp.difyProcessor
}