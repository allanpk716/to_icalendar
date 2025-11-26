package services

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/allanpk716/to_icalendar/internal/cleanup"
	"github.com/allanpk716/to_icalendar/internal/config"
	"github.com/allanpk716/to_icalendar/internal/deduplication"
	"github.com/allanpk716/to_icalendar/internal/logger"
	"github.com/allanpk716/to_icalendar/internal/models"
	"github.com/allanpk716/to_icalendar/internal/task"
)

// CleanupServiceImpl 清理服务实现
type CleanupServiceImpl struct {
	configDir string
	logger    interface{}
}

// NewCleanupService 创建清理服务
func NewCleanupService(configDir string, logger interface{}) CleanupService {
	return &CleanupServiceImpl{
		configDir: configDir,
		logger:    logger,
	}
}

// Cleanup 执行清理操作
func (cs *CleanupServiceImpl) Cleanup(ctx context.Context, options *CleanupOptions) (*CleanupResult, error) {
	// 创建清理器
	cleaner := cleanup.NewCleaner()

	// 初始化必要的组件
	configManager := config.NewConfigManager()
	cleaner.SetConfig(configManager)

	// 尝试初始化缓存管理器
	cacheDir := filepath.Join(cs.configDir, "cache")
	cacheManager := deduplication.NewCacheManager(cacheDir, logger.GetLogger().GetStdLogger())
	cleaner.SetCacheManager(cacheManager)

	// 准备清理选项
	cleanOptions := cleanup.CleanOptions{
		All:         options.All,
		Tasks:       options.Tasks,
		Images:      options.Images,
		ImageHashes: options.ImageHashes,
		Temp:        options.Temp,
		Generated:   options.Generated,
		DryRun:      options.DryRun,
		Force:       options.Force,
		OlderThan:   options.OlderThan,
		ClearAll:    options.ClearAll,
	}

	// 执行清理
	summary, err := cleaner.Clean(cleanOptions)
	if err != nil {
		return &CleanupResult{
			Skipped: true,
			Message: fmt.Sprintf("清理失败: %v", err),
		}, err
	}

	// 转换结果格式
	filesByType := make(map[string]int64)
	for _, result := range summary.Results {
		filesByType[result.CacheType] = int64(result.FilesCount)
	}

	result := &CleanupResult{
		TotalFiles:  int64(summary.TotalFiles),
		TotalSize:   summary.TotalSize,
		FilesByType: filesByType,
		Skipped:     false,
		Message:     "清理完成",
	}

	return result, nil
}

// GetCleanupStats 获取清理统计信息
func (cs *CleanupServiceImpl) GetCleanupStats(ctx context.Context) (*CleanupStats, error) {
	// 创建清理器
	cleaner := cleanup.NewCleaner()

	// 初始化必要的组件
	configManager := config.NewConfigManager()
	cleaner.SetConfig(configManager)

	// 尝试初始化缓存管理器
	cacheDir := filepath.Join(cs.configDir, "cache")
	cacheManager := deduplication.NewCacheManager(cacheDir, logger.GetLogger().GetStdLogger())
	cleaner.SetCacheManager(cacheManager)

	// 创建任务管理器
	serverConfig := &models.ServerConfig{} // 使用默认配置
	taskManager, err := task.NewTaskManager(cs.configDir, serverConfig.Cache, logger.GetLogger().GetStdLogger())
	if err != nil {
		return nil, fmt.Errorf("创建任务管理器失败: %w", err)
	}

	// 创建清理器
	taskCleaner := task.NewTaskCleaner(taskManager, logger.GetLogger().GetStdLogger())

	// 获取统计信息
	stats, err := taskCleaner.CleanupStatistics()
	if err != nil {
		return nil, fmt.Errorf("获取清理统计信息失败: %w", err)
	}

	return &CleanupStats{
		TaskCount:         stats.TaskCount,
		RecentTasks7Days:  stats.RecentTasks7Days,
		RecentTasks30Days: stats.RecentTasks30Days,
		Size:              stats.TotalSizeBytes,
		CacheFiles:        stats.CacheFiles,
		CacheSize:         stats.CacheSizeBytes,
	}, nil
}

// ParseCleanOptions 解析清理选项
func (cs *CleanupServiceImpl) ParseCleanOptions(args []string) (*CleanupOptions, error) {
	options := &CleanupOptions{
		All:         false,
		Tasks:       false,
		Images:      false,
		ImageHashes: false,
		Temp:        false,
		Generated:   false,
		DryRun:      false,
		Force:       false,
		OlderThan:   "",
		ClearAll:    false,
	}

	for i, arg := range args {
		switch arg {
		case "--all":
			options.All = true
		case "--tasks":
			options.Tasks = true
		case "--images":
			options.Images = true
		case "--image-hashes":
			options.ImageHashes = true
		case "--temp":
			options.Temp = true
		case "--generated":
			options.Generated = true
		case "--dry-run":
			options.DryRun = true
		case "--force":
			options.Force = true
		case "--older-than":
			if i+1 < len(args) {
				options.OlderThan = args[i+1]
			}
		case "--clear-all":
			options.ClearAll = true
		}
	}

	// 如果没有指定任何具体类型，默认清理所有
	if !options.Tasks && !options.Images && !options.ImageHashes && !options.Temp && !options.Generated {
		options.All = true
	}

	return options, nil
}