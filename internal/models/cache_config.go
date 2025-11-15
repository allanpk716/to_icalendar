package models

import "fmt"

// CacheConfig 缓存配置
type CacheConfig struct {
	// 自动清理配置
	AutoCleanupDays         int  `yaml:"auto_cleanup_days"`          // 自动清理天数，默认30天
	CleanupOnStartup        bool `yaml:"cleanup_on_startup"`         // 是否在启动时清理，默认true
	PreserveSuccessfulHashes bool `yaml:"preserve_successful_hashes"` // 是否保留成功的图片哈希记录，默认true

	// 任务管理配置
	TaskRetentionDays    int `yaml:"task_retention_days"`     // 任务保留天数，0表示使用auto_cleanup_days
	MaxTaskDirectories   int `yaml:"max_task_directories"`    // 最大任务目录数量，0表示无限制
	CompressOldTasks     bool `yaml:"compress_old_tasks"`      // 是否压缩旧任务

	// 图片缓存配置
	ImageCacheMaxSize   int64 `yaml:"image_cache_max_size"`    // 图片缓存最大大小(MB)，0表示无限制
	ImageCacheMaxFiles  int   `yaml:"image_cache_max_files"`   // 图片缓存最大文件数量，0表示无限制
	EnableImageBackup   bool  `yaml:"enable_image_backup"`     // 是否启用图片备份

	// 全局缓存配置
	GlobalCacheEnabled  bool `yaml:"global_cache_enabled"`     // 是否启用全局缓存
	EnableCacheMetrics  bool `yaml:"enable_cache_metrics"`     // 是否启用缓存统计
	MetricsRetentionDays int  `yaml:"metrics_retention_days"`  // 指标保留天数
}

// DefaultCacheConfig 返回默认缓存配置
func DefaultCacheConfig() CacheConfig {
	return CacheConfig{
		AutoCleanupDays:         30,
		CleanupOnStartup:        true,
		PreserveSuccessfulHashes: true,
		TaskRetentionDays:       0, // 使用AutoCleanupDays
		MaxTaskDirectories:      0, // 无限制
		CompressOldTasks:        false,
		ImageCacheMaxSize:       0,  // 无限制
		ImageCacheMaxFiles:      0,  // 无限制
		EnableImageBackup:       true,
		GlobalCacheEnabled:      true,
		EnableCacheMetrics:      true,
		MetricsRetentionDays:    7,
	}
}

// Validate 验证缓存配置
func (cc *CacheConfig) Validate() error {
	if cc.AutoCleanupDays < 0 {
		return fmt.Errorf("auto_cleanup_days cannot be negative")
	}

	if cc.TaskRetentionDays < 0 {
		return fmt.Errorf("task_retention_days cannot be negative")
	}

	if cc.MaxTaskDirectories < 0 {
		return fmt.Errorf("max_task_directories cannot be negative")
	}

	if cc.ImageCacheMaxSize < 0 {
		return fmt.Errorf("image_cache_max_size cannot be negative")
	}

	if cc.ImageCacheMaxFiles < 0 {
		return fmt.Errorf("image_cache_max_files cannot be negative")
	}

	if cc.MetricsRetentionDays < 0 {
		return fmt.Errorf("metrics_retention_days cannot be negative")
	}

	// 如果任务保留天数为0，使用自动清理天数
	if cc.TaskRetentionDays == 0 {
		cc.TaskRetentionDays = cc.AutoCleanupDays
	}

	return nil
}

// GetEffectiveTaskRetentionDays 获取有效的任务保留天数
func (cc *CacheConfig) GetEffectiveTaskRetentionDays() int {
	if cc.TaskRetentionDays > 0 {
		return cc.TaskRetentionDays
	}
	return cc.AutoCleanupDays
}

// GetImageCacheMaxSizeBytes 获取图片缓存最大大小(字节)
func (cc *CacheConfig) GetImageCacheMaxSizeBytes() int64 {
	if cc.ImageCacheMaxSize <= 0 {
		return 0 // 无限制
	}
	return cc.ImageCacheMaxSize * 1024 * 1024 // MB转换为字节
}