package services

import (
	"context"
	"fmt"

	"github.com/allanpk716/to_icalendar/internal/clipboard"
	"github.com/allanpk716/to_icalendar/internal/logger"
	"github.com/allanpk716/to_icalendar/internal/models"
)

// ClipboardServiceImpl 剪贴板服务实现
type ClipboardServiceImpl struct {
	logger interface{}
}

// NewClipboardService 创建剪贴板服务
func NewClipboardService(logger interface{}) ClipboardService {
	return &ClipboardServiceImpl{
		logger: logger,
	}
}

// ReadContent 读取剪贴板内容
func (cs *ClipboardServiceImpl) ReadContent(ctx context.Context) (*models.ClipboardContent, error) {
	// 初始化剪贴板管理器
	clipboardManager, err := clipboard.NewManager()
	if err != nil {
		return nil, fmt.Errorf("初始化剪贴板管理器失败: %w", err)
	}

	// 直接调用底层实现，它已经返回 models.ClipboardContent
	return clipboardManager.Read()
}

// HasContent 检查剪贴板是否有内容
func (cs *ClipboardServiceImpl) HasContent() (bool, error) {
	// 初始化剪贴板管理器
	clipboardManager, err := clipboard.NewManager()
	if err != nil {
		return false, fmt.Errorf("初始化剪贴板管理器失败: %w", err)
	}

	return clipboardManager.HasContent()
}

// GetContentType 获取剪贴板内容类型
func (cs *ClipboardServiceImpl) GetContentType() (string, error) {
	// 初始化剪贴板管理器
	clipboardManager, err := clipboard.NewManager()
	if err != nil {
		return "", fmt.Errorf("初始化剪贴板管理器失败: %w", err)
	}

	contentType, err := clipboardManager.GetContentType()
	if err != nil {
		return "", fmt.Errorf("获取剪贴板内容类型失败: %w", err)
	}

	return string(contentType), nil
}

// ProcessContent 处理剪贴板内容
func (cs *ClipboardServiceImpl) ProcessContent(ctx context.Context, content *models.ClipboardContent) (*models.ProcessingResult, error) {
	if content == nil {
		return nil, fmt.Errorf("剪贴板内容为空")
	}

	logger.Info("📝 开始处理剪贴板内容，类型: %s", content.Type)

	// 这里可以添加内容处理逻辑
	// 目前返回基本信息，具体的处理将在后续的服务中实现
	result := &models.ProcessingResult{
		Success:      true,
		Reminder:     nil, // 将在 clip-upload 服务中处理
		ParsedInfo:   nil, // 将在 clip-upload 服务中处理
		ErrorMessage: "",
	}

	logger.Info("✓ 剪贴板内容预处理完成")
	return result, nil
}