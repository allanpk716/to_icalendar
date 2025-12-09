package deduplication

import (
	"fmt"
	"log"

	"github.com/allanpk716/to_icalendar/pkg/models"
)

// DeduplicationResult 去重结果
type DeduplicationResult struct {
	IsDuplicate     bool   `json:"is_duplicate"`
	DuplicateType   string `json:"duplicate_type"` // "cache", "image", "none"
	CacheHit        bool   `json:"cache_hit"`
	SkipReason      string `json:"skip_reason,omitempty"`
	SuggestedAction string `json:"suggested_action"` // "skip", "create"
	ImageHash       string `json:"image_hash,omitempty"`     // 图片哈希（如果检查了图片）
	PreviousResult  *models.ProcessingResult `json:"previous_result,omitempty"` // 之前的处理结果（针对图片重复）
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

// CheckImageDuplicate 检查图片是否已经处理过
func (d *Deduplicator) CheckImageDuplicate(imageData []byte) (*DeduplicationResult, error) {
	if !d.config.Enabled {
		return &DeduplicationResult{
			IsDuplicate:     false,
			DuplicateType:   "none",
			SuggestedAction: "create",
		}, nil
	}

	imageHash := d.cacheManager.GenerateImageHash(imageData)
	d.logger.Printf("开始图片去重检查，哈希: %s", imageHash[:8])

	// 检查本地缓存
	if d.config.EnableLocalCache {
		d.logger.Printf("检查本地图片缓存...")
		if result := d.checkImageLocalCache(imageData, imageHash); result != nil {
			return result, nil
		}
		d.logger.Printf("本地图片缓存未发现重复")
	}

	// 未发现重复
	return &DeduplicationResult{
		IsDuplicate:     false,
		DuplicateType:   "none",
		CacheHit:        false,
		SuggestedAction: "create",
		ImageHash:       imageHash[:8],
	}, nil
}

// checkImageLocalCache 检查本地图片缓存
func (d *Deduplicator) checkImageLocalCache(imageData []byte, imageHash string) *DeduplicationResult {
	if d.cacheManager.IsImageProcessed(imageData) {
		imageCache := d.cacheManager.GetImageCache(imageData)
		if imageCache != nil {
			d.logger.Printf("本地图片缓存发现重复图片 (哈希: %s, 创建时间: %s, 处理成功: %v)",
				imageHash[:8], imageCache.CreatedAt.Format("2006-01-02 15:04"), imageCache.Success)

			result := &DeduplicationResult{
				IsDuplicate:   true,
				DuplicateType: "image",
				CacheHit:      true,
				ImageHash:     imageHash[:8],
				SuggestedAction: "skip",
			}

			// 如果之前处理成功了，提供相关信息
			if imageCache.Success && imageCache.Title != "" {
				result.SkipReason = fmt.Sprintf("图片已成功处理过，标题: %s (处理时间: %s)",
					imageCache.Title, imageCache.ProcessedAt.Format("2006-01-02 15:04"))
			} else if !imageCache.Success {
				result.SkipReason = fmt.Sprintf("图片之前处理失败 (失败时间: %s), 建议重新处理",
					imageCache.ProcessedAt.Format("2006-01-02 15:04"))
				result.SuggestedAction = "create" // 之前失败，建议重新处理
			} else {
				result.SkipReason = "本地缓存中存在相同图片"
			}

			return result
		}
	}

	d.logger.Printf("本地图片缓存中未找到哈希: %s", imageHash[:8])
	return nil
}

// RecordProcessedImage 记录已处理的图片
func (d *Deduplicator) RecordProcessedImage(imageData []byte, taskHash, title string, success bool, processTime string, filePath string) error {
	if d.config.EnableLocalCache && d.cacheManager != nil {
		return d.cacheManager.AddProcessedImage(imageData, taskHash, title, success, processTime, filePath)
	}
	return nil
}

// GetCacheManager 获取缓存管理器
func (d *Deduplicator) GetCacheManager() *CacheManager {
	return d.cacheManager
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