package wails

import "github.com/allanpk716/to_icalendar/pkg/commands"

// ToICalendarBindings Wails绑定接口
type ToICalendarBindings interface {
	// InitConfig 初始化配置
	InitConfig() *commands.ConfigResult

	// CleanCache 清理缓存
	CleanCache(options *commands.CleanupOptions) *commands.CleanupResult

	// ProcessClipboard 处理剪贴板并上传
	ProcessClipboard() *commands.ProcessClipboardResult

	// TestConnection 测试连接
	TestConnection() *TestConnectionResult
}

// TestConnectionResult 连接测试结果
type TestConnectionResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// WailsOptions Wails选项
type WailsOptions struct {
	// 这里可以添加Wails特定的选项
	EnableNotifications bool `json:"enable_notifications"`
	ShowProgress        bool `json:"show_progress"`
	LogLevel           string `json:"log_level"`
}