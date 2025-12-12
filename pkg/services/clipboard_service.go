package services

import (
	"context"
	"fmt"
	"sync"

	"github.com/allanpk716/to_icalendar/pkg/clipboard"
	"github.com/allanpk716/to_icalendar/pkg/logger"
	"github.com/allanpk716/to_icalendar/pkg/models"
)

// ClipboardServiceImpl 剪贴板服务实现
type ClipboardServiceImpl struct {
	mu           sync.RWMutex
	manager      *clipboard.Manager
	initialized  bool
}

var (
	globalClipboardService *ClipboardServiceImpl
	globalClipboardOnce    sync.Once
)

// NewClipboardService 创建剪贴板服务
func NewClipboardService() ClipboardService {
	return &ClipboardServiceImpl{}
}

// GetGlobalClipboardService 获取全局剪贴板服务实例（单例模式）
func GetGlobalClipboardService() (ClipboardService, error) {
	var err error
	globalClipboardOnce.Do(func() {
		globalClipboardService = &ClipboardServiceImpl{}
		// 初始化剪贴板管理器
		globalClipboardService.manager, err = clipboard.NewManager()
		if err == nil {
			globalClipboardService.initialized = true
			logger.Infof("全局剪贴板服务初始化成功")
		}
	})

	if err != nil {
		return nil, fmt.Errorf("初始化全局剪贴板服务失败: %w", err)
	}

	return globalClipboardService, nil
}

// ReadContent 读取剪贴板内容
func (cs *ClipboardServiceImpl) ReadContent(ctx context.Context) (*models.ClipboardContent, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	// 检查是否已初始化
	if !cs.initialized {
		// 初始化剪贴板管理器
		manager, err := clipboard.NewManager()
		if err != nil {
			return nil, fmt.Errorf("初始化剪贴板管理器失败: %w", err)
		}
		cs.manager = manager
		cs.initialized = true
		logger.Infof("剪贴板管理器初始化完成")
	}

	// 直接调用底层实现，它已经返回 models.ClipboardContent
	return cs.manager.Read()
}

// HasContent 检查剪贴板是否有内容
func (cs *ClipboardServiceImpl) HasContent() (bool, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	// 检查是否已初始化
	if !cs.initialized {
		return false, fmt.Errorf("剪贴板管理器尚未初始化，请先调用 ReadContent")
	}

	return cs.manager.HasContent()
}

// GetContentType 获取剪贴板内容类型
func (cs *ClipboardServiceImpl) GetContentType() (string, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	// 检查是否已初始化
	if !cs.initialized {
		return "", fmt.Errorf("剪贴板管理器尚未初始化，请先调用 ReadContent")
	}

	contentType, err := cs.manager.GetContentType()
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

	logger.Infof("📝 开始处理剪贴板内容，类型: %s", content.Type)

	// 这里可以添加内容处理逻辑
	// 目前返回基本信息，具体的处理将在后续的服务中实现
	result := &models.ProcessingResult{
		Success:      true,
		Reminder:     nil, // 将在 clip-upload 服务中处理
		ParsedInfo:   nil, // 将在 clip-upload 服务中处理
		ErrorMessage: "",
	}

	logger.Infof("✓ 剪贴板内容预处理完成")
	return result, nil
}