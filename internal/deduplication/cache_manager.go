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

	"github.com/allanpk716/to_icalendar/internal/models"
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

// CacheManager 去重缓存管理器
type CacheManager struct {
	cacheDir     string
	cacheFile    string
	cachedTasks  map[string]*TaskCache
	mutex        sync.RWMutex
	logger       *log.Logger
	cleanupTTL   time.Duration
}

// NewCacheManager 创建新的缓存管理器
func NewCacheManager(cacheDir string, logger *log.Logger) *CacheManager {
	if logger == nil {
		logger = log.Default()
	}

	cm := &CacheManager{
		cacheDir:    cacheDir,
		cacheFile:   filepath.Join(cacheDir, "submitted_tasks.json"),
		cachedTasks: make(map[string]*TaskCache),
		logger:      logger,
		cleanupTTL:  30 * 24 * time.Hour, // 30天过期
	}

	// 确保缓存目录存在
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		cm.logger.Printf("创建缓存目录失败: %v", err)
	}

	// 加载现有缓存
	if err := cm.loadCache(); err != nil {
		cm.logger.Printf("加载缓存失败: %v", err)
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
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	now := time.Now()
	expiredCount := 0

	for hash, task := range cm.cachedTasks {
		if now.Sub(task.CreatedAt) > cm.cleanupTTL {
			delete(cm.cachedTasks, hash)
			expiredCount++
		}
	}

	if expiredCount > 0 {
		cm.logger.Printf("清理了 %d 个过期缓存任务", expiredCount)
		// 异步保存
		go func() {
			if err := cm.saveCache(); err != nil {
				cm.logger.Printf("保存清理后的缓存失败: %v", err)
			}
		}()
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

	return stats
}

// ClearCache 清空所有缓存
func (cm *CacheManager) ClearCache() error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	cm.cachedTasks = make(map[string]*TaskCache)

	if err := os.Remove(cm.cacheFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除缓存文件失败: %w", err)
	}

	cm.logger.Printf("已清空所有缓存")
	return nil
}

// GetCacheDir 获取缓存目录路径
func (cm *CacheManager) GetCacheDir() string {
	return cm.cacheDir
}