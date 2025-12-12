package clipboard

import (
	"fmt"
	"sync"
	"time"

	"github.com/WQGroup/logger"
	"golang.org/x/sys/windows"
)

// ClipboardHealthChecker 剪贴板健康检查器
type ClipboardHealthChecker struct {
	mu            sync.RWMutex
	lastCheck     time.Time
	isHealthy     bool
	checkInterval time.Duration
}

// NewClipboardHealthChecker 创建新的剪贴板健康检查器
func NewClipboardHealthChecker(checkInterval time.Duration) *ClipboardHealthChecker {
	if checkInterval == 0 {
		checkInterval = 30 * time.Second // 默认30秒检查一次
	}

	return &ClipboardHealthChecker{
		checkInterval: checkInterval,
		lastCheck:     time.Time{},
		isHealthy:     true, // 初始状态为健康
	}
}

// CheckHealth 检查剪贴板健康状态
func (h *ClipboardHealthChecker) CheckHealth() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 记录检查开始时间
	checkStart := time.Now()

	// 尝试打开和关闭剪贴板
	ret, _, err := procOpenClipboard.Call(0)
	if ret == 0 {
		h.isHealthy = false
		h.lastCheck = checkStart

		// 获取详细的错误信息
		var errMsg string
		if err != nil {
			errCode := uint32(err.(windows.Errno))
			switch errCode {
			case ERROR_ACCESS_DENIED:
				errMsg = "剪贴板访问被拒绝"
			case ERROR_CLIPBOARD_LOCKED:
				errMsg = "剪贴板被锁定"
			default:
				errMsg = fmt.Sprintf("未知错误，错误码: %d", errCode)
			}
		} else {
			errMsg = "无法打开剪贴板"
		}

		logger.Warnf("剪贴板健康检查失败: %s", errMsg)
		return fmt.Errorf("剪贴板不可用: %s", errMsg)
	}

	// 成功打开，立即关闭
	if closeRet, _, _ := procCloseClipboard.Call(); closeRet == 0 {
		logger.Warnf("剪贴板关闭失败，可能已被其他进程占用")
	}

	// 更新健康状态
	h.isHealthy = true
	h.lastCheck = checkStart
	logger.Debugf("剪贴板健康检查通过，耗时: %v", time.Since(checkStart))

	return nil
}

// IsHealthy 返回剪贴板是否健康
func (h *ClipboardHealthChecker) IsHealthy() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.isHealthy
}

// GetLastCheckTime 获取最后一次检查时间
func (h *ClipboardHealthChecker) GetLastCheckTime() time.Time {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.lastCheck
}

// GetHealthStatus 获取详细的健康状态信息
func (h *ClipboardHealthChecker) GetHealthStatus() map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return map[string]interface{}{
		"is_healthy":     h.isHealthy,
		"last_check":     h.lastCheck.Format(time.RFC3339),
		"check_interval": h.checkInterval.String(),
		"time_since_check": func() string {
			if h.lastCheck.IsZero() {
				return "从未检查"
			}
			return time.Since(h.lastCheck).String()
		}(),
	}
}

// StartPeriodicCheck 启动定期健康检查
func (h *ClipboardHealthChecker) StartPeriodicCheck(stopCh <-chan struct{}) {
	ticker := time.NewTicker(h.checkInterval)
	defer ticker.Stop()

	logger.Infof("启动剪贴板健康检查，检查间隔: %v", h.checkInterval)

	for {
		select {
		case <-ticker.C:
			if err := h.CheckHealth(); err != nil {
				logger.Errorf("定期健康检查失败: %v", err)
			}
		case <-stopCh:
			logger.Info("剪贴板健康检查已停止")
			return
		}
	}
}

// ForceCheck 强制立即检查健康状态
func (h *ClipboardHealthChecker) ForceCheck() error {
	logger.Debug("执行强制剪贴板健康检查")
	return h.CheckHealth()
}

// GlobalClipboardHealthChecker 全局剪贴板健康检查器实例
var (
	globalHealthChecker *ClipboardHealthChecker
	globalHealthOnce     sync.Once
)

// GetGlobalHealthChecker 获取全局健康检查器实例
func GetGlobalHealthChecker() *ClipboardHealthChecker {
	globalHealthOnce.Do(func() {
		globalHealthChecker = NewClipboardHealthChecker(30 * time.Second)
		logger.Info("全局剪贴板健康检查器已创建")
	})
	return globalHealthChecker
}