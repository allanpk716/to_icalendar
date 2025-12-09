package deduplication

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/allanpk716/to_icalendar/pkg/models"
	"github.com/allanpk716/to_icalendar/pkg/task"
)

// TaskCache 记录已提交任务的缓存结构
type TaskCache struct {
	TaskHash    string    `json:"task_hash"`
	Title       string    `json:"title"`
	Date        string    `json:"date"`
	Time        string    `json:"time"`
	List        string    `json:"list"`
	CreatedAt   time.Time `json:"created_at"`
	MicrosoftID string    `json:"microsoft_id,omitempty"`
}

// ImageHashCache 记录已处理图片哈希的缓存结构
type ImageHashCache struct {
	ImageHash    string    `json:"image_hash"`
	TaskHash     string    `json:"task_hash,omitempty"`     // 关联的任务哈希（如果有）
	Title        string    `json:"title,omitempty"`        // 关联的任务标题（如果有）
	CreatedAt    time.Time `json:"created_at"`
	ProcessedAt  time.Time `json:"processed_at"`
	Success      bool      `json:"success"`                // 处理是否成功
	FilePath     string    `json:"file_path,omitempty"`     // 缓存文件路径（如果有）
	Size         int64     `json:"size"`                   // 图片大小
	ProcessTime  string    `json:"process_time,omitempty"` // dify处理时间（如果有）
}

// CacheManager 去重缓存管理器
type CacheManager struct {
	cacheDir       string
	cacheFile      string
	imageCacheFile string
	cachedTasks    map[string]*TaskCache
	cachedImages   map[string]*ImageHashCache
	mutex          sync.RWMutex
	logger         *log.Logger
	cleanupTTL     time.Duration
	taskManager    *task.TaskManager  // 新增：集成TaskManager
}

// NewCacheManager 创建新的缓存管理器
func NewCacheManager(cacheDir string, logger *log.Logger) *CacheManager {
	if logger == nil {
		logger = log.Default()
	}

	cm := &CacheManager{
		cacheDir:       cacheDir,
		cacheFile:      filepath.Join(cacheDir, "submitted_tasks.json"),
		imageCacheFile: filepath.Join(cacheDir, "image_hashes.json"),
		cachedTasks:    make(map[string]*TaskCache),
		cachedImages:   make(map[string]*ImageHashCache),
		logger:         logger,
		cleanupTTL:     30 * 24 * time.Hour, // 30天过期
	}

	// 确保缓存目录存在
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		cm.logger.Printf("创建缓存目录失败: %v", err)
	}

	// 加载现有缓存
	if err := cm.loadCache(); err != nil {
		cm.logger.Printf("加载任务缓存失败: %v", err)
	}

	if err := cm.loadImageCache(); err != nil {
		cm.logger.Printf("加载图片缓存失败: %v", err)
	}

	return cm
}

// NewCacheManagerWithTaskManager 创建与TaskManager集成的缓存管理器
func NewCacheManagerWithTaskManager(cacheDir string, taskManager *task.TaskManager, logger *log.Logger) *CacheManager {
	if logger == nil {
		logger = log.Default()
	}

	cm := &CacheManager{
		cacheDir:       cacheDir,
		cacheFile:      filepath.Join(cacheDir, "submitted_tasks.json"),
		imageCacheFile: filepath.Join(cacheDir, "image_hashes.json"),
		cachedTasks:    make(map[string]*TaskCache),
		cachedImages:   make(map[string]*ImageHashCache),
		logger:         logger,
		cleanupTTL:     30 * 24 * time.Hour, // 30天过期
		taskManager:    taskManager,
	}

	// 确保缓存目录存在
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		cm.logger.Printf("创建缓存目录失败: %v", err)
	}

	// 加载现有缓存
	if err := cm.loadCache(); err != nil {
		cm.logger.Printf("加载任务缓存失败: %v", err)
	}

	if err := cm.loadImageCache(); err != nil {
		cm.logger.Printf("加载图片缓存失败: %v", err)
	}

	return cm
}

// GenerateTaskHash 生成任务哈希值用于去重
func (cm *CacheManager) GenerateTaskHash(reminder *models.ParsedReminder) string {
	data := fmt.Sprintf("%s|%s|%s|%s",
		reminder.Original.Title,
		reminder.Original.Date,
		reminder.Original.Time,
		reminder.List)

	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// GenerateImageHash 生成图片SHA-256哈希值用于去重
func (cm *CacheManager) GenerateImageHash(imageData []byte) string {
	hash := sha256.Sum256(imageData)
	return hex.EncodeToString(hash[:])
}

// IsDuplicate 检查任务是否重复
func (cm *CacheManager) IsDuplicate(reminder *models.ParsedReminder) bool {
	taskHash := cm.GenerateTaskHash(reminder)

	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	_, exists := cm.cachedTasks[taskHash]
	return exists
}

// GetDuplicate 获取重复任务信息
func (cm *CacheManager) GetDuplicate(reminder *models.ParsedReminder) *TaskCache {
	taskHash := cm.GenerateTaskHash(reminder)

	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	if task, exists := cm.cachedTasks[taskHash]; exists {
		return task
	}
	return nil
}

// IsImageProcessed 检查图片是否已经处理过
func (cm *CacheManager) IsImageProcessed(imageData []byte) bool {
	imageHash := cm.GenerateImageHash(imageData)

	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	_, exists := cm.cachedImages[imageHash]
	return exists
}

// GetImageCache 获取图片缓存信息
func (cm *CacheManager) GetImageCache(imageData []byte) *ImageHashCache {
	imageHash := cm.GenerateImageHash(imageData)

	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	if cache, exists := cm.cachedImages[imageHash]; exists {
		return cache
	}
	return nil
}

// AddProcessedImage 添加已处理的图片到缓存
func (cm *CacheManager) AddProcessedImage(imageData []byte, taskHash, title string, success bool, processTime string, filePath string) error {
	imageHash := cm.GenerateImageHash(imageData)

	cache := &ImageHashCache{
		ImageHash:   imageHash,
		TaskHash:    taskHash,
		Title:       title,
		CreatedAt:   time.Now(),
		ProcessedAt: time.Now(),
		Success:     success,
		FilePath:    filePath,
		Size:        int64(len(imageData)),
		ProcessTime: processTime,
	}

	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	cm.cachedImages[imageHash] = cache

	// 异步保存到文件
	go func() {
		if err := cm.saveImageCache(); err != nil {
			cm.logger.Printf("保存图片缓存失败: %v", err)
		}
	}()

	cm.logger.Printf("图片已添加到缓存: %s (哈希: %s, 大小: %d bytes)", title, imageHash[:8], len(imageData))
	return nil
}

// AddSubmittedTask 添加已提交的任务到缓存
func (cm *CacheManager) AddSubmittedTask(reminder *models.ParsedReminder, microsoftID string) error {
	taskHash := cm.GenerateTaskHash(reminder)

	task := &TaskCache{
		TaskHash:    taskHash,
		Title:       reminder.Original.Title,
		Date:        reminder.Original.Date,
		Time:        reminder.Original.Time,
		List:        reminder.List,
		CreatedAt:   time.Now(),
		MicrosoftID: microsoftID,
	}

	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	cm.cachedTasks[taskHash] = task

	// 异步保存到文件
	go func() {
		if err := cm.saveCache(); err != nil {
			cm.logger.Printf("保存缓存失败: %v", err)
		}
	}()

	cm.logger.Printf("任务已添加到缓存: %s (哈希: %s)", reminder.Original.Title, taskHash[:8])
	return nil
}

// loadCache 从文件加载缓存
func (cm *CacheManager) loadCache() error {
	if _, err := os.Stat(cm.cacheFile); os.IsNotExist(err) {
		return nil // 文件不存在，返回空缓存
	}

	data, err := os.ReadFile(cm.cacheFile)
	if err != nil {
		return fmt.Errorf("读取缓存文件失败: %w", err)
	}

	var tasks []*TaskCache
	if err := json.Unmarshal(data, &tasks); err != nil {
		return fmt.Errorf("解析缓存文件失败: %w", err)
	}

	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// 清理过期任务并重建缓存
	now := time.Now()
	cm.cachedTasks = make(map[string]*TaskCache)

	for _, task := range tasks {
		if now.Sub(task.CreatedAt) <= cm.cleanupTTL {
			cm.cachedTasks[task.TaskHash] = task
		}
	}

	cm.logger.Printf("已加载 %d 个缓存任务", len(cm.cachedTasks))
	return nil
}

// saveCache 保存缓存到文件
func (cm *CacheManager) saveCache() error {
	cm.mutex.RLock()
	tasks := make([]*TaskCache, 0, len(cm.cachedTasks))
	for _, task := range cm.cachedTasks {
		tasks = append(tasks, task)
	}
	cm.mutex.RUnlock()

	data, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化缓存失败: %w", err)
	}

	// 原子写入
	tempFile := cm.cacheFile + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("写入临时缓存文件失败: %w", err)
	}

	if err := os.Rename(tempFile, cm.cacheFile); err != nil {
		os.Remove(tempFile) // 清理临时文件
		return fmt.Errorf("重命名缓存文件失败: %w", err)
	}

	return nil
}

// CleanupExpiredTasks 清理过期任务
func (cm *CacheManager) CleanupExpiredTasks() error {
	// 清理过期任务缓存
	cm.mutex.Lock()
	now := time.Now()
	expiredCount := 0

	for hash, task := range cm.cachedTasks {
		if now.Sub(task.CreatedAt) > cm.cleanupTTL {
			delete(cm.cachedTasks, hash)
			expiredCount++
		}
	}
	taskExpiredCount := expiredCount
	cm.mutex.Unlock()

	if taskExpiredCount > 0 {
		cm.logger.Printf("清理了 %d 个过期缓存任务", taskExpiredCount)
		// 异步保存
		go func() {
			if err := cm.saveCache(); err != nil {
				cm.logger.Printf("保存清理后的任务缓存失败: %v", err)
			}
		}()
	}

	// 同时清理过期图片缓存
	if err := cm.CleanupExpiredImages(); err != nil {
		return fmt.Errorf("清理过期图片缓存失败: %w", err)
	}

	return nil
}

// GetCacheStats 获取缓存统计信息
func (cm *CacheManager) GetCacheStats() map[string]interface{} {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	stats := map[string]interface{}{
		"total_tasks":    len(cm.cachedTasks),
		"cache_file":     cm.cacheFile,
		"cleanup_ttl":    cm.cleanupTTL.String(),
	}

	// 统计各列表的任务数量
	listStats := make(map[string]int)
	now := time.Now()
	recentCount := 0

	for _, task := range cm.cachedTasks {
		listStats[task.List]++
		if now.Sub(task.CreatedAt) <= 24*time.Hour {
			recentCount++
		}
	}

	stats["tasks_by_list"] = listStats
	stats["recent_tasks_24h"] = recentCount

	// 添加图片缓存统计
	imagesStats := cm.GetImageCacheStats()
	for key, value := range imagesStats {
		stats[key] = value
	}

	return stats
}

// ClearCache 清空所有缓存
func (cm *CacheManager) ClearCache() error {
	// 清空任务缓存
	cm.mutex.Lock()
	cm.cachedTasks = make(map[string]*TaskCache)
	cm.mutex.Unlock()

	if err := os.Remove(cm.cacheFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除任务缓存文件失败: %w", err)
	}

	// 清空图片缓存
	if err := cm.ClearImageCache(); err != nil {
		return fmt.Errorf("清空图片缓存失败: %w", err)
	}

	cm.logger.Printf("已清空所有缓存")
	return nil
}

// GetCacheDir 获取缓存目录路径
func (cm *CacheManager) GetCacheDir() string {
	return cm.cacheDir
}

// loadImageCache 从文件加载图片缓存
func (cm *CacheManager) loadImageCache() error {
	if _, err := os.Stat(cm.imageCacheFile); os.IsNotExist(err) {
		return nil // 文件不存在，返回空缓存
	}

	data, err := os.ReadFile(cm.imageCacheFile)
	if err != nil {
		return fmt.Errorf("读取图片缓存文件失败: %w", err)
	}

	var images []*ImageHashCache
	if err := json.Unmarshal(data, &images); err != nil {
		return fmt.Errorf("解析图片缓存文件失败: %w", err)
	}

	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// 清理过期图片并重建缓存
	now := time.Now()
	cm.cachedImages = make(map[string]*ImageHashCache)

	for _, image := range images {
		if now.Sub(image.CreatedAt) <= cm.cleanupTTL {
			cm.cachedImages[image.ImageHash] = image
		}
	}

	cm.logger.Printf("已加载 %d 个图片缓存", len(cm.cachedImages))
	return nil
}

// saveImageCache 保存图片缓存到文件
func (cm *CacheManager) saveImageCache() error {
	cm.mutex.RLock()
	images := make([]*ImageHashCache, 0, len(cm.cachedImages))
	for _, image := range cm.cachedImages {
		images = append(images, image)
	}
	cm.mutex.RUnlock()

	data, err := json.MarshalIndent(images, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化图片缓存失败: %w", err)
	}

	// 原子写入
	tempFile := cm.imageCacheFile + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("写入临时图片缓存文件失败: %w", err)
	}

	if err := os.Rename(tempFile, cm.imageCacheFile); err != nil {
		os.Remove(tempFile) // 清理临时文件
		return fmt.Errorf("重命名图片缓存文件失败: %w", err)
	}

	return nil
}

// CleanupExpiredImages 清理过期图片缓存
func (cm *CacheManager) CleanupExpiredImages() error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	now := time.Now()
	expiredCount := 0

	for hash, image := range cm.cachedImages {
		if now.Sub(image.CreatedAt) > cm.cleanupTTL {
			delete(cm.cachedImages, hash)
			expiredCount++
		}
	}

	if expiredCount > 0 {
		cm.logger.Printf("清理了 %d 个过期图片缓存", expiredCount)
		// 异步保存
		go func() {
			if err := cm.saveImageCache(); err != nil {
				cm.logger.Printf("保存清理后的图片缓存失败: %v", err)
			}
		}()
	}

	return nil
}

// GetImageCacheStats 获取图片缓存统计信息
func (cm *CacheManager) GetImageCacheStats() map[string]interface{} {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	stats := map[string]interface{}{
		"total_images":     len(cm.cachedImages),
		"image_cache_file": cm.imageCacheFile,
		"cleanup_ttl":      cm.cleanupTTL.String(),
	}

	// 统计成功和失败的处理数量
	successCount := 0
	failureCount := 0
	now := time.Now()
	recentCount := 0
	totalSize := int64(0)

	for _, image := range cm.cachedImages {
		if image.Success {
			successCount++
		} else {
			failureCount++
		}
		if now.Sub(image.CreatedAt) <= 24*time.Hour {
			recentCount++
		}
		totalSize += image.Size
	}

	stats["successful_processed"] = successCount
	stats["failed_processed"] = failureCount
	stats["recent_images_24h"] = recentCount
	stats["total_size_mb"] = float64(totalSize) / (1024 * 1024)

	return stats
}

// ClearImageCache 清空所有图片缓存
func (cm *CacheManager) ClearImageCache() error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	cm.cachedImages = make(map[string]*ImageHashCache)

	if err := os.Remove(cm.imageCacheFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除图片缓存文件失败: %w", err)
	}

	cm.logger.Printf("已清空所有图片缓存")
	return nil
}