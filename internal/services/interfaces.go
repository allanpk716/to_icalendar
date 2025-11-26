package services

import (
	"context"

	"github.com/allanpk716/to_icalendar/internal/models"
	"github.com/allanpk716/to_icalendar/internal/cache"
)

// ConfigService 配置服务接口
type ConfigService interface {
	Initialize(ctx context.Context) error
	GetConfigDir() (string, error)
	EnsureConfigDir() (string, error)
	CreateConfigTemplates(ctx context.Context, configDir string) (*ConfigResult, error)
	LoadServerConfig(ctx context.Context) (*models.ServerConfig, error)
}

// CacheService 缓存服务接口
type CacheService interface {
	Initialize() error
	GetManager() *cache.UnifiedCacheManager
	GetCacheDir() string
	Cleanup() error
}

// CleanupService 清理服务接口
type CleanupService interface {
	Cleanup(ctx context.Context, options *CleanupOptions) (*CleanupResult, error)
	GetCleanupStats(ctx context.Context) (*CleanupStats, error)
	ParseCleanOptions(args []string) (*CleanupOptions, error)
}

// ClipboardService 剪贴板服务接口
type ClipboardService interface {
	ReadContent(ctx context.Context) (*models.ClipboardContent, error)
	HasContent() (bool, error)
	GetContentType() (string, error)
	ProcessContent(ctx context.Context, content *models.ClipboardContent) (*models.ProcessingResult, error)
}

// TodoService Microsoft Todo 服务接口
type TodoService interface {
	CreateTask(ctx context.Context, reminder *models.Reminder) error
	TestConnection() error
	GetServerInfo() (map[string]interface{}, error)
}

// DifyService Dify AI 服务接口
type DifyService interface {
	ProcessText(ctx context.Context, text string) (*models.DifyResponse, error)
	ProcessImage(ctx context.Context, imageData []byte) (*models.DifyResponse, error)
	ValidateConfig() error
	TestConnection() error
}

// 结果类型定义

// ConfigResult 配置操作结果
type ConfigResult struct {
	ConfigDir        string `json:"config_dir"`
	ServerConfig     string `json:"server_config"`
	ReminderTemplate string `json:"reminder_template"`
	Success          bool   `json:"success"`
	Message          string `json:"message"`
}

// CleanupOptions 清理选项
type CleanupOptions struct {
	All          bool   `json:"all"`
	Tasks        bool   `json:"tasks"`
	Images       bool   `json:"images"`
	ImageHashes  bool   `json:"image_hashes"`
	Temp         bool   `json:"temp"`
	Generated    bool   `json:"generated"`
	DryRun       bool   `json:"dry_run"`
	Force        bool   `json:"force"`
	OlderThan    string `json:"older_than"`
	ClearAll     bool   `json:"clear_all"`
}

// CleanupResult 清理结果
type CleanupResult struct {
	TotalFiles  int64            `json:"total_files"`
	TotalSize   int64            `json:"total_size"`
	FilesByType map[string]int64 `json:"files_by_type"`
	Skipped     bool             `json:"skipped"`
	Message     string           `json:"message"`
}

// CleanupStats 清理统计信息
type CleanupStats struct {
	TaskCount         int    `json:"task_count"`
	RecentTasks7Days  int    `json:"recent_tasks_7_days"`
	RecentTasks30Days int    `json:"recent_tasks_30_days"`
	Size              int64  `json:"size"`
	CacheFiles        int    `json:"cache_files"`
	CacheSize         int64  `json:"cache_size"`
}


// ProcessClipboardResult 剪贴板处理结果
type ProcessClipboardResult struct {
	Success      bool                     `json:"success"`
	Title        string                   `json:"title,omitempty"`
	Description  string                   `json:"description,omitempty"`
	Date         string                   `json:"date,omitempty"`
	Time         string                   `json:"time,omitempty"`
	Message      string                   `json:"message,omitempty"`
	Data         *models.ProcessingResult `json:"data,omitempty"`
}