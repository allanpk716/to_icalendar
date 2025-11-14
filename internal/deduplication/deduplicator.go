package deduplication

import (
	"log"

	"github.com/allanpk716/to_icalendar/internal/models"
)

// DeduplicationResult 去重结果
type DeduplicationResult struct {
	IsDuplicate     bool   `json:"is_duplicate"`
	DuplicateType   string `json:"duplicate_type"` // "cache", "none"
	CacheHit        bool   `json:"cache_hit"`
	SkipReason      string `json:"skip_reason,omitempty"`
	SuggestedAction string `json:"suggested_action"` // "skip", "create"
}

// Deduplicator 去重服务（简化版 - 仅本地缓存）
type Deduplicator struct {
	config       *models.DeduplicationConfig
	cacheManager *CacheManager
	logger       *log.Logger
}

// NewDeduplicator 创建新的去重服务（简化版 - 仅本地缓存）
func NewDeduplicator(config *models.DeduplicationConfig, cacheManager *CacheManager) *Deduplicator {
	if config == nil {
		// 使用默认配置
		config = &models.DeduplicationConfig{
			Enabled:              true,
			EnableLocalCache:     true,
			EnableRemoteQuery:    false, // 禁用云端查询
		}
	}

	deduplicator := &Deduplicator{
		config:       config,
		cacheManager: cacheManager,
		logger:       log.Default(),
	}

	return deduplicator
}

// CheckDuplicate 检查任务是否重复（简化版 - 仅本地缓存）
func (d *Deduplicator) CheckDuplicate(reminder *models.ParsedReminder) (*DeduplicationResult, error) {
	if !d.config.Enabled {
		return &DeduplicationResult{
			IsDuplicate:     false,
			DuplicateType:   "none",
			SuggestedAction: "create",
		}, nil
	}

	d.logger.Printf("开始去重检查: %s (列表: %s, 日期: %s, 时间: %s)",
		reminder.Original.Title, reminder.List, reminder.Original.Date, reminder.Original.Time)

	// 仅检查本地缓存
	if d.config.EnableLocalCache {
		d.logger.Printf("检查本地缓存...")
		if result := d.checkLocalCache(reminder); result != nil {
			return result, nil
		}
		d.logger.Printf("本地缓存未发现重复")
	}

	// 未发现重复
	return &DeduplicationResult{
		IsDuplicate:     false,
		DuplicateType:   "none",
		CacheHit:        false,
		SuggestedAction: "create",
	}, nil
}

// checkLocalCache 检查本地缓存
func (d *Deduplicator) checkLocalCache(reminder *models.ParsedReminder) *DeduplicationResult {
	taskHash := d.cacheManager.GenerateTaskHash(reminder)
	d.logger.Printf("生成任务哈希: %s", taskHash[:8])

	if d.cacheManager.IsDuplicate(reminder) {
		duplicatedTask := d.cacheManager.GetDuplicate(reminder)
		d.logger.Printf("本地缓存发现重复任务: %s (哈希: %s, 创建时间: %s)",
			reminder.Original.Title, taskHash[:8], duplicatedTask.CreatedAt.Format("2006-01-02 15:04"))

		return &DeduplicationResult{
			IsDuplicate:     true,
			DuplicateType:   "cache",
			CacheHit:        true,
			SkipReason:      "本地缓存中存在相同任务",
			SuggestedAction: "skip",
		}
	}

	d.logger.Printf("本地缓存中未找到哈希: %s", taskHash[:8])
	return nil
}

// RecordSubmittedTask 记录已提交的任务
func (d *Deduplicator) RecordSubmittedTask(reminder *models.ParsedReminder, microsoftID string) error {
	if d.config.EnableLocalCache && d.cacheManager != nil {
		return d.cacheManager.AddSubmittedTask(reminder, microsoftID)
	}
	return nil
}

// GetStats 获取去重统计信息（简化版）
func (d *Deduplicator) GetStats() map[string]interface{} {
	stats := map[string]interface{}{
		"deduplication_enabled": d.config.Enabled,
		"local_cache_enabled":   d.config.EnableLocalCache,
	}

	if d.cacheManager != nil {
		cacheStats := d.cacheManager.GetCacheStats()
		stats["cached_tasks"] = cacheStats["total_tasks"]
		stats["recent_tasks_24h"] = cacheStats["recent_tasks_24h"]
	}

	return stats
}

// Cleanup 清理过期数据
func (d *Deduplicator) Cleanup() error {
	if d.cacheManager != nil {
		return d.cacheManager.CleanupExpiredTasks()
	}
	return nil
}