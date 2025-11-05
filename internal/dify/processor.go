package dify

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/allanpk716/to_icalendar/internal/models"
)

// Processor handles content processing using Dify API
type Processor struct {
	client    *Client
	options   *ProcessingOptions
	userID    string
}

// NewProcessor creates a new content processor
func NewProcessor(client *Client, userID string, options *ProcessingOptions) *Processor {
	if options == nil {
		options = DefaultProcessingOptions()
	}

	return &Processor{
		client:  client,
		options: options,
		userID:  userID,
	}
}

// ProcessImage processes image content using Dify OCR and semantic understanding
func (p *Processor) ProcessImage(ctx context.Context, imageData []byte, fileName string) (*ProcessingResponse, error) {
	startTime := time.Now()

	log.Printf("开始处理图片内容，文件名: %s, 大小: %d bytes", fileName, len(imageData))

	// 验证输入
	if len(imageData) == 0 {
		return &ProcessingResponse{
			Success:      false,
			ErrorMessage: "图片数据为空",
			ProcessingTime: time.Since(startTime),
		}, fmt.Errorf("image data is empty")
	}

	if fileName == "" {
		fileName = fmt.Sprintf("clipboard_%s.png", time.Now().Format("20060102_150405"))
	}

	// 调用Dify API处理图片
	difyResp, err := p.client.ProcessImage(ctx, imageData, fileName, p.userID)
	if err != nil {
		log.Printf("Dify API调用失败: %v", err)
		return &ProcessingResponse{
			Success:        false,
			ErrorMessage:   fmt.Sprintf("Dify API调用失败: %v", err),
			ProcessingTime: time.Since(startTime),
		}, err
	}

	log.Printf("Dify API响应: %s", difyResp.Answer)

	// 解析Dify响应
	parsedInfo, err := p.parseDifyResponse(difyResp.Answer)
	if err != nil {
		log.Printf("解析Dify响应失败: %v", err)
		return &ProcessingResponse{
			Success:        false,
			ErrorMessage:   fmt.Sprintf("解析响应失败: %v", err),
			ProcessingTime: time.Since(startTime),
		}, err
	}

	// 验证解析结果
	validation := p.validateParsedInfo(parsedInfo)

	// 生成提醒事项
	var reminder *models.Reminder
	if validation.IsValid && parsedInfo.Confidence >= p.options.ConfidenceThreshold {
		reminder = p.createReminderFromParsedInfo(parsedInfo)
	}

	return &ProcessingResponse{
		Success:        validation.IsValid,
		Reminder:       reminder,
		ParsedInfo:     parsedInfo,
		Validation:     validation,
		ProcessingTime: time.Since(startTime),
		RequestID:      fmt.Sprintf("img_%d", time.Now().Unix()),
		Timestamp:      time.Now(),
		ErrorMessage:   getErrorMessage(validation, parsedInfo),
	}, nil
}

// ProcessText processes text content using Dify semantic understanding
func (p *Processor) ProcessText(ctx context.Context, text string) (*ProcessingResponse, error) {
	startTime := time.Now()

	log.Printf("开始处理文字内容，长度: %d", len(text))

	// 验证输入
	if strings.TrimSpace(text) == "" {
		return &ProcessingResponse{
			Success:      false,
			ErrorMessage: "文字内容为空",
			ProcessingTime: time.Since(startTime),
		}, fmt.Errorf("text content is empty")
	}

	// 调用Dify API处理文字
	difyResp, err := p.client.ProcessText(ctx, text, p.userID)
	if err != nil {
		log.Printf("Dify API调用失败: %v", err)
		return &ProcessingResponse{
			Success:        false,
			ErrorMessage:   fmt.Sprintf("Dify API调用失败: %v", err),
			ProcessingTime: time.Since(startTime),
		}, err
	}

	log.Printf("Dify API响应: %s", difyResp.Answer)

	// 解析Dify响应
	parsedInfo, err := p.parseDifyResponse(difyResp.Answer)
	if err != nil {
		log.Printf("解析Dify响应失败: %v", err)
		return &ProcessingResponse{
			Success:        false,
			ErrorMessage:   fmt.Sprintf("解析响应失败: %v", err),
			ProcessingTime: time.Since(startTime),
		}, err
	}

	// 验证解析结果
	validation := p.validateParsedInfo(parsedInfo)

	// 生成提醒事项
	var reminder *models.Reminder
	if validation.IsValid && parsedInfo.Confidence >= p.options.ConfidenceThreshold {
		reminder = p.createReminderFromParsedInfo(parsedInfo)
	}

	return &ProcessingResponse{
		Success:        validation.IsValid,
		Reminder:       reminder,
		ParsedInfo:     parsedInfo,
		Validation:     validation,
		ProcessingTime: time.Since(startTime),
		RequestID:      fmt.Sprintf("txt_%d", time.Now().Unix()),
		Timestamp:      time.Now(),
		ErrorMessage:   getErrorMessage(validation, parsedInfo),
	}, nil
}

// parseDifyResponse parses the response from Dify API
func (p *Processor) parseDifyResponse(response string) (*models.ParsedTaskInfo, error) {
	// 尝试解析JSON响应
	var parsedInfo models.ParsedTaskInfo

	// 首先尝试直接解析JSON
	if err := json.Unmarshal([]byte(response), &parsedInfo); err == nil {
		return &parsedInfo, nil
	}

	// 如果直接解析失败，尝试从响应中提取JSON
	jsonStart := strings.Index(response, "{")
	jsonEnd := strings.LastIndex(response, "}")

	if jsonStart >= 0 && jsonEnd > jsonStart {
		jsonStr := response[jsonStart : jsonEnd+1]
		if err := json.Unmarshal([]byte(jsonStr), &parsedInfo); err == nil {
			return &parsedInfo, nil
		}
	}

	// 如果仍然无法解析，创建一个错误响应
	return &models.ParsedTaskInfo{
		OriginalText: response,
		Confidence:   0.0,
		Description:  fmt.Sprintf("无法解析Dify响应: %s", response),
	}, fmt.Errorf("failed to parse Dify response as JSON")
}

// validateParsedInfo validates the parsed task information
func (p *Processor) validateParsedInfo(info *models.ParsedTaskInfo) *ValidationResult {
	if info == nil {
		return &ValidationResult{
			IsValid:   false,
			ErrorType: "null_response",
			Message:   "解析结果为空",
			Score:     0.0,
		}
	}

	// 检查是否有错误
	if info.Description != "" && strings.Contains(strings.ToLower(info.Description), "未识别到任务信息") {
		return &ValidationResult{
			IsValid:   false,
			ErrorType: "no_task_detected",
			Message:   "未识别到任务信息",
			Score:     info.Confidence,
		}
	}

	// 检查必需字段
	if info.Title == "" {
		return &ValidationResult{
			IsValid:   false,
			ErrorType: "missing_title",
			Message:   "缺少任务标题",
			Score:     info.Confidence * 0.5, // 降低分数
		}
	}

	// 检查日期时间格式
	if info.Date == "" || info.Time == "" {
		return &ValidationResult{
			IsValid:   false,
			ErrorType: "missing_datetime",
			Message:   "缺少日期或时间信息",
			Score:     info.Confidence * 0.7,
		}
	}

	// 验证日期格式
	if !isValidDate(info.Date) {
		return &ValidationResult{
			IsValid:   false,
			ErrorType: "invalid_date_format",
			Message:   fmt.Sprintf("无效的日期格式: %s", info.Date),
			Score:     info.Confidence * 0.8,
		}
	}

	// 验证时间格式
	if !isValidTime(info.Time) {
		return &ValidationResult{
			IsValid:   false,
			ErrorType: "invalid_time_format",
			Message:   fmt.Sprintf("无效的时间格式: %s", info.Time),
			Score:     info.Confidence * 0.8,
		}
	}

	// 所有检查通过
	return &ValidationResult{
		IsValid:   true,
		ErrorType: "",
		Message:   "验证通过",
		Score:     info.Confidence,
	}
}

// createReminderFromParsedInfo creates a Reminder from parsed task info
func (p *Processor) createReminderFromParsedInfo(info *models.ParsedTaskInfo) *models.Reminder {
	reminder := &models.Reminder{
		Title:        info.Title,
		Description:  info.Description,
		Date:         info.Date,
		Time:         info.Time,
		RemindBefore: info.RemindBefore,
		List:         info.List,
	}

	// 设置默认值
	if reminder.RemindBefore == "" {
		reminder.RemindBefore = "15m"
	}

	if reminder.List == "" {
		reminder.List = p.options.DefaultList
	}

	// 转换优先级
	switch info.Priority {
	case "high", "高", "紧急":
		reminder.Priority = models.PriorityHigh
	case "low", "低", "一般":
		reminder.Priority = models.PriorityLow
	default:
		reminder.Priority = models.PriorityMedium
	}

	return reminder
}

// isValidDate checks if the date string is in valid format (YYYY-MM-DD)
func isValidDate(dateStr string) bool {
	if len(dateStr) != 10 {
		return false
	}

	if dateStr[4] != '-' || dateStr[7] != '-' {
		return false
	}

	// 简单验证数字格式
	for _, char := range dateStr {
		if char != '-' && (char < '0' || char > '9') {
			return false
		}
	}

	return true
}

// isValidTime checks if the time string is in valid format (HH:MM)
func isValidTime(timeStr string) bool {
	if len(timeStr) != 5 {
		return false
	}

	if timeStr[2] != ':' {
		return false
	}

	// 简单验证数字格式
	hours := timeStr[:2]
	minutes := timeStr[3:]

	for _, char := range hours {
		if char < '0' || char > '9' {
			return false
		}
	}

	for _, char := range minutes {
		if char < '0' || char > '9' {
			return false
		}
	}

	return true
}

// getErrorMessage extracts error message from validation result
func getErrorMessage(validation *ValidationResult, info *models.ParsedTaskInfo) string {
	if validation.IsValid {
		return ""
	}

	if validation.ErrorType == "no_task_detected" {
		return "内容中未识别到任务信息"
	}

	return validation.Message
}

// SetOptions updates processing options
func (p *Processor) SetOptions(options *ProcessingOptions) {
	if options != nil {
		p.options = options
	}
}

// GetOptions returns current processing options
func (p *Processor) GetOptions() *ProcessingOptions {
	return p.options
}