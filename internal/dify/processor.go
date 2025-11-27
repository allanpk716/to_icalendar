package dify

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strconv"
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

	// 解析Dify响应 - 使用工作流响应解析器
	parsedInfo, err := p.parseDifyWorkflowResponse(difyResp)
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

	// 解析Dify响应 - 使用工作流响应解析器
	parsedInfo, err := p.parseDifyWorkflowResponse(difyResp)
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

// parseDifyWorkflowResponse parses the response from Dify workflow API
func (p *Processor) parseDifyWorkflowResponse(difyResp *models.DifyResponse) (*models.ParsedTaskInfo, error) {
	// 首先检查Answer字段（适用于chat-messages）
	if difyResp.Answer != "" {
		log.Printf("从Answer字段解析响应")
		return p.parseDifyResponse(difyResp.Answer)
	}

	// 检查工作流响应的Data.Outputs.Text字段
	if difyResp.Data != nil && difyResp.Data.Outputs != nil && difyResp.Data.Outputs.Text != "" {
		log.Printf("从工作流Data.Outputs.Text字段解析响应")
		return p.parseTaskJSON(difyResp.Data.Outputs.Text)
	}

	// 如果两者都为空，返回错误
	return &models.ParsedTaskInfo{
		OriginalText: "",
		Confidence:   0.0,
		Description:  "Dify响应中没有找到有效内容",
	}, fmt.Errorf("no valid content found in Dify response")
}

// parseTaskJSON 解析任务JSON字符串（处理转义字符）
func (p *Processor) parseTaskJSON(jsonStr string) (*models.ParsedTaskInfo, error) {
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

	log.Printf("任务JSON解析成功: 标题='%s', 日期='%s', 时间='%s'",
		taskInfo.Title, taskInfo.Date, taskInfo.Time)

	// 对于成功解析的JSON，设置高置信度
	taskInfo.Confidence = 0.9

	return &taskInfo, nil
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

	// 保留图片识别的提醒时间，仅在缺失时使用默认值

	// 设置默认值（作为最后的保险）
	if reminder.RemindBefore == "" {
		if p.options != nil && p.options.DefaultRemindBefore != "" {
			reminder.RemindBefore = p.options.DefaultRemindBefore
		} else {
			reminder.RemindBefore = "15m"
		}
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
	// 首先尝试解析时间范围，如果是时间范围格式，提取开始时间
	if processedTime := parseTimeFromRange(timeStr); processedTime != timeStr {
		// 是时间范围格式，验证提取出的开始时间
		timeStr = processedTime
	}
	
	// 验证单个时间格式（支持HH:MM和H:MM）
	if len(timeStr) == 4 || len(timeStr) == 5 {
		// 查找冒号位置
		var colonIndex int
		if len(timeStr) == 5 && timeStr[2] == ':' {
			colonIndex = 2
		} else if len(timeStr) == 4 && timeStr[1] == ':' {
			colonIndex = 1
		} else {
			return false
		}
		
		// 确保冒号数量正确
		if strings.Count(timeStr, ":") != 1 {
			return false
		}
		
		// 验证小时部分
		hours := timeStr[:colonIndex]
		minutes := timeStr[colonIndex+1:]
		
		// 验证小时部分都是数字
		for _, char := range hours {
			if char < '0' || char > '9' {
				return false
			}
		}
		
		// 验证分钟部分都是数字且长度为2
		if len(minutes) != 2 {
			return false
		}
		for _, char := range minutes {
			if char < '0' || char > '9' {
				return false
			}
		}
		
		// 验证时间值的有效性
		if hour, err := strconv.Atoi(hours); err == nil {
			if minute, err := strconv.Atoi(minutes); err == nil {
				return hour >= 0 && hour <= 23 && minute >= 0 && minute <= 59
			}
		}
	}
	
	return false
}

// getErrorMessage extracts error message from validation result
// parseTimeFromRange 从时间字符串中解析时间，支持时间范围格式
// 如果输入是时间范围（如"14:30 - 16:30"），返回开始时间"14:30"
// 如果输入是单个时间，直接返回
// parseTimeFromRange 从时间字符串中解析时间，支持时间范围格式
// 如果输入是时间范围（如"14:30 - 16:30"），返回开始时间"14:30"
// 如果输入是单个时间，直接返回
func parseTimeFromRange(timeStr string) string {
	// 定义时间范围分隔符模式，确保是真正的时间范围
	rangePatterns := []string{
		`^(\d{1,2}:\d{2})\s*[-~~到至]\s*(\d{1,2}:\d{2})\s*$`,   // 14:30-16:30, 14:30~16:30, 14:30到16:30, 14:30至16:30
	}
	
	for _, pattern := range rangePatterns {
		if re, err := regexp.Compile(pattern); err == nil {
			if matches := re.FindStringSubmatch(timeStr); len(matches) > 2 {
				startTime := matches[1]
				endTime := matches[2]
				// 验证开始时间格式是否有效
				if isValidTimeFormat(startTime) && isValidTimeFormat(endTime) {
					return startTime
				}
			}
		}
	}
	
	// 如果不是时间范围，返回原始字符串
	return timeStr
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