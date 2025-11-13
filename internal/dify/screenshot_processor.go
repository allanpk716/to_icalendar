package dify

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/allanpk716/to_icalendar/internal/models"
)

// ScreenshotProcessor 定义截图处理接口
type ScreenshotProcessor interface {
	// ProcessScreenshot 处理截图并返回提醒事项
	ProcessScreenshot(ctx context.Context, screenshot *ScreenshotInput) (*models.Reminder, error)

	// ValidateInput 验证输入数据
	ValidateInput(screenshot *ScreenshotInput) error

	// GetProcessorInfo 获取处理器信息
	GetProcessorInfo() *ProcessorInfo
}

// ScreenshotInput 截图输入数据
type ScreenshotInput struct {
	Data      []byte `json:"data"`       // 图片二进制数据
	FileName  string `json:"file_name"`  // 文件名
	Format    string `json:"format"`     // 图片格式 (png, jpg, etc.)
}

// ProcessorInfo 处理器信息
type ProcessorInfo struct {
	Name             string   `json:"name"`
	Version          string   `json:"version"`
	SupportedFormats []string `json:"supported_formats"`
	MaxFileSize      int64    `json:"max_file_size"`
}

// ScreenshotProcessorImpl 截图处理器实现
type ScreenshotProcessorImpl struct {
	client  DifyClient
	parser  ResponseParser
	config  *models.DifyConfig
}

// NewScreenshotProcessor 创建截图处理器
func NewScreenshotProcessor(config *models.DifyConfig) (*ScreenshotProcessorImpl, error) {
	// 验证配置
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid dify config: %w", err)
	}

	return &ScreenshotProcessorImpl{
		client: NewDifyClient(config),
		parser: NewResponseParser(),
		config: config,
	}, nil
}

// ProcessScreenshot 实现接口方法
func (p *ScreenshotProcessorImpl) ProcessScreenshot(ctx context.Context, screenshot *ScreenshotInput) (*models.Reminder, error) {
	startTime := time.Now()
	requestID := generateRequestID()

	log.Printf("[RequestID: %s] 开始处理截图，文件: %s, 大小: %d bytes",
		requestID, screenshot.FileName, len(screenshot.Data))

	// 1. 验证输入
	if err := p.ValidateInput(screenshot); err != nil {
		return nil, fmt.Errorf("input validation failed: %w", err)
	}

	// 2. 调用Dify API处理图片
	difyResponse, err := p.client.ProcessImage(ctx, screenshot.Data, screenshot.FileName, requestID)
	if err != nil {
		return nil, fmt.Errorf("dify processing failed: %w", err)
	}

	// 3. 解析响应
	parsedInfo, err := p.parser.ParseReminderResponse(difyResponse.Answer)
	if err != nil {
		return nil, fmt.Errorf("response parsing failed: %w", err)
	}

	// 4. 转换为Reminder对象
	reminder := p.convertToReminder(parsedInfo, screenshot)

	processingTime := time.Since(startTime)
	log.Printf("[RequestID: %s] 处理完成，耗时: %v", requestID, processingTime)

	return reminder, nil
}

// ValidateInput 验证输入数据
func (p *ScreenshotProcessorImpl) ValidateInput(screenshot *ScreenshotInput) error {
	if screenshot == nil {
		return fmt.Errorf("screenshot input is nil")
	}

	if len(screenshot.Data) == 0 {
		return fmt.Errorf("screenshot data is empty")
	}

	// 检查文件大小
	maxSize := int64(10 * 1024 * 1024) // 10MB
	if int64(len(screenshot.Data)) > maxSize {
		return fmt.Errorf("file size %d exceeds maximum allowed size %d",
			len(screenshot.Data), maxSize)
	}

	// 检查文件格式
	if !p.isSupportedFormat(screenshot.Format) {
		return fmt.Errorf("unsupported image format: %s", screenshot.Format)
	}

	// 基本的图片内容验证
	if err := p.validateImageContent(screenshot.Data); err != nil {
		return fmt.Errorf("image content validation failed: %w", err)
	}

	return nil
}

// GetProcessorInfo 获取处理器信息
func (p *ScreenshotProcessorImpl) GetProcessorInfo() *ProcessorInfo {
	return &ProcessorInfo{
		Name:             "DifyScreenshotProcessor",
		Version:          "1.0.0",
		SupportedFormats: []string{"png", "jpg", "jpeg", "bmp", "gif"},
		MaxFileSize:      10 * 1024 * 1024, // 10MB
	}
}

// 私有辅助方法
func (p *ScreenshotProcessorImpl) convertToReminder(info *models.ParsedTaskInfo, screenshot *ScreenshotInput) *models.Reminder {
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
		reminder.List = "Default"
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

func (p *ScreenshotProcessorImpl) isSupportedFormat(format string) bool {
	supportedFormats := map[string]bool{
		"png":  true,
		"jpg":  true,
		"jpeg": true,
		"bmp":  true,
		"gif":  true,
	}
	return supportedFormats[strings.ToLower(format)]
}

func (p *ScreenshotProcessorImpl) validateImageContent(data []byte) error {
	// 简单的图片头验证
	if len(data) < 8 {
		return fmt.Errorf("image data too short to be valid")
	}

	// 检查常见图片格式的文件头
	imageHeaders := [][]byte{
		{0x89, 0x50, 0x4E, 0x47}, // PNG
		{0xFF, 0xD8, 0xFF},       // JPEG
		{0x42, 0x4D},             // BMP
		{0x47, 0x49, 0x46, 0x38}, // GIF
	}

	for _, header := range imageHeaders {
		if len(data) >= len(header) {
			match := true
			for i, b := range header {
				if data[i] != b {
					match = false
					break
				}
			}
			if match {
				return nil
			}
		}
	}

	return fmt.Errorf("invalid image format")
}

func generateRequestID() string {
	return fmt.Sprintf("scr_%d", time.Now().UnixNano())
}

// 辅助函数：从文件名提取格式
func ExtractImageFormat(fileName string) string {
	ext := strings.ToLower(filepath.Ext(fileName))
	if ext == "" {
		return "unknown"
	}
	return ext[1:] // 去掉点号
}