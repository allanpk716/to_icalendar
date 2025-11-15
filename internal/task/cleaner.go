package task

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/allanpk716/to_icalendar/internal/models"
)

// TaskCleaner 任务清理器
type TaskCleaner struct {
	taskManager *TaskManager
	logger      *log.Logger
}

// NewTaskCleaner 创建新的任务清理器
func NewTaskCleaner(taskManager *TaskManager, logger *log.Logger) *TaskCleaner {
	if logger == nil {
		logger = log.Default()
	}

	return &TaskCleaner{
		taskManager: taskManager,
		logger:      logger,
	}
}

// SetConfig 设置清理配置
func (tc *TaskCleaner) SetConfig(config models.CacheConfig) {
	tc.taskManager.cacheConfig = config
}

// AutoCleanup 执行自动清理
func (tc *TaskCleaner) AutoCleanup() (*CleanupResult, error) {
	if !tc.taskManager.cacheConfig.CleanupOnStartup {
		tc.logger.Println("Auto cleanup is disabled, skipping cleanup")
		return &CleanupResult{Skipped: true, Reason: "Auto cleanup is disabled"}, nil
	}

	days := tc.taskManager.cacheConfig.AutoCleanupDays
	if days <= 0 {
		tc.logger.Println("Invalid cleanup days setting, skipping cleanup")
		return &CleanupResult{Skipped: true, Reason: "Invalid cleanup days setting"}, nil
	}

	tc.logger.Printf("Starting cleanup of %d days old task data", days)
	return tc.CleanupOlderThan(days)
}

// CleanupOlderThan 清理指定天数前的数据
func (tc *TaskCleaner) CleanupOlderThan(days int) (*CleanupResult, error) {
	result := &CleanupResult{
		StartTime: time.Now(),
	}

	cutoffTime := time.Now().AddDate(0, 0, -days)

	// 清理任务目录
	if err := tc.cleanupTaskDirectories(cutoffTime, result); err != nil {
		return nil, fmt.Errorf("failed to cleanup task directories: %w", err)
	}

	// 清理任务索引
	if err := tc.cleanupTaskIndex(cutoffTime, result); err != nil {
		return nil, fmt.Errorf("failed to cleanup task index: %w", err)
	}

	// 清理全局缓存文件
	if err := tc.cleanupGlobalCache(cutoffTime, result); err != nil {
		return nil, fmt.Errorf("failed to cleanup global cache: %w", err)
	}

	// 清理孤儿文件
	if err := tc.cleanupOrphanedFiles(result); err != nil {
		return nil, fmt.Errorf("failed to cleanup orphaned files: %w", err)
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	tc.logger.Printf("Cleanup completed: removed %d task directories, freed %.2f MB space, took %v",
		result.TasksCleaned, float64(result.BytesFreed)/(1024*1024), result.Duration)

	return result, nil
}

// CleanupStatistics 获取清理统计信息
func (tc *TaskCleaner) CleanupStatistics() (*CleanupStats, error) {
	stats := &CleanupStats{}

	// 统计任务目录
	if err := tc.statsTaskDirectories(stats); err != nil {
		return nil, fmt.Errorf("failed to statistics task directories: %w", err)
	}

	// 统计全局缓存
	if err := tc.statsGlobalCache(stats); err != nil {
		return nil, fmt.Errorf("failed to statistics global cache: %w", err)
	}

	return stats, nil
}

// 以下为私有方法

// cleanupTaskDirectories 清理任务目录
func (tc *TaskCleaner) cleanupTaskDirectories(cutoffTime time.Time, result *CleanupResult) error {
	entries, err := os.ReadDir(tc.taskManager.tasksDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // 目录不存在，无需清理
		}
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		taskID := entry.Name()
		taskDir := filepath.Join(tc.taskManager.tasksDir, taskID)

		// 尝试加载任务信息以获取准确的任务时间
		shouldCleanup := false

		// 尝试从任务信息文件获取时间
		taskInfoFile := filepath.Join(taskDir, "task_info.json")
		if data, err := os.ReadFile(taskInfoFile); err == nil {
			var session TaskSession
			if err := json.Unmarshal(data, &session); err == nil {
				// 检查任务的开始时间或结束时间
				if session.EndTime.IsZero() {
					// 如果没有结束时间，使用开始时间
					shouldCleanup = session.StartTime.Before(cutoffTime)
				} else {
					// 使用结束时间
					shouldCleanup = session.EndTime.Before(cutoffTime)
				}
			} else {
				tc.logger.Printf("Failed to parse task info file %s: %v", taskInfoFile, err)
			}
		} else {
			// 如果无法读取任务信息文件，使用目录的修改时间作为备选
			tc.logger.Printf("Failed to read task info file %s: %v, using directory mod time", taskInfoFile, err)
			if info, err := entry.Info(); err == nil {
				shouldCleanup = info.ModTime().Before(cutoffTime)
			} else {
				tc.logger.Printf("Failed to get directory info %s: %v", taskDir, err)
				continue
			}
		}

		// 检查是否需要清理
		if shouldCleanup {
			if err := tc.cleanupTaskDirectory(taskDir, result); err != nil {
				tc.logger.Printf("Failed to cleanup task directory %s: %v", taskDir, err)
				result.Errors = append(result.Errors, err.Error())
			}
		}
	}

	return nil
}

// cleanupTaskDirectory 清理单个任务目录
func (tc *TaskCleaner) cleanupTaskDirectory(taskDir string, result *CleanupResult) error {
	// 计算目录大小
	size, err := tc.calculateDirectorySize(taskDir)
	if err != nil {
		return fmt.Errorf("failed to calculate directory size: %w", err)
	}

	// 删除目录
	if err := os.RemoveAll(taskDir); err != nil {
		return fmt.Errorf("failed to remove directory: %w", err)
	}

	result.TasksCleaned++
	result.BytesFreed += size

	tc.logger.Printf("Removed task directory: %s (%.2f MB)",
		filepath.Base(taskDir), float64(size)/(1024*1024))

	return nil
}

// cleanupTaskIndex 清理任务索引
func (tc *TaskCleaner) cleanupTaskIndex(cutoffTime time.Time, result *CleanupResult) error {
	tc.taskManager.mutex.Lock()
	defer tc.taskManager.mutex.Unlock()

	// 过滤出需要保留的索引
	var keptIndexes []*TaskIndex
	var removedIndexes []*TaskIndex

	for _, index := range tc.taskManager.index {
		if index.StartTime.Before(cutoffTime) {
			removedIndexes = append(removedIndexes, index)
		} else {
			keptIndexes = append(keptIndexes, index)
		}
	}

	// 保留成功的图片哈希记录（如果配置要求）
	if tc.taskManager.cacheConfig.PreserveSuccessfulHashes {
		keptIndexes = tc.preserveSuccessfulHashes(removedIndexes, keptIndexes)
	}

	tc.taskManager.index = keptIndexes
	result.IndexEntriesRemoved = len(removedIndexes)

	// 保存更新后的索引
	if err := tc.taskManager.saveIndex(); err != nil {
		return fmt.Errorf("failed to save updated index: %w", err)
	}

	tc.logger.Printf("Cleaned up %d task index entries", result.IndexEntriesRemoved)

	return nil
}

// preserveSuccessfulHashes 保留成功的图片哈希记录
func (tc *TaskCleaner) preserveSuccessfulHashes(removedIndexes, keptIndexes []*TaskIndex) []*TaskIndex {
	for _, index := range removedIndexes {
		if index.ImageHash != "" && index.DifySuccess && index.TodoSuccess {
			// 创建一个简化的索引记录只保留哈希信息
			preserved := &TaskIndex{
				TaskID:      index.TaskID,
				StartTime:   index.StartTime,
				Status:      index.Status,
				ImageHash:   index.ImageHash,
				Title:       index.Title,
				DifySuccess: index.DifySuccess,
				TodoSuccess: index.TodoSuccess,
			}
			keptIndexes = append(keptIndexes, preserved)
		}
	}
	return keptIndexes
}

// cleanupGlobalCache 清理全局缓存文件
func (tc *TaskCleaner) cleanupGlobalCache(cutoffTime time.Time, result *CleanupResult) error {
	globalFiles := []string{
		filepath.Join(tc.taskManager.globalDir, "submitted_tasks.json"),
		filepath.Join(tc.taskManager.globalDir, "image_hashes.json"),
	}

	for _, file := range globalFiles {
		if err := tc.cleanupCacheFile(file, cutoffTime, result); err != nil {
			tc.logger.Printf("Failed to cleanup cache file %s: %v", file, err)
		}
	}

	return nil
}

// cleanupCacheFile 清理缓存文件
func (tc *TaskCleaner) cleanupCacheFile(filePath string, cutoffTime time.Time, result *CleanupResult) error {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil // 文件不存在
	}

	// 读取文件内容
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// 根据文件类型处理清理逻辑
	var entries []json.RawMessage
	if err := json.Unmarshal(data, &entries); err != nil {
		return fmt.Errorf("failed to parse file: %w", err)
	}

	var keptEntries []json.RawMessage
	var removedCount int

	for _, entry := range entries {
		var entryMap map[string]interface{}
		if err := json.Unmarshal(entry, &entryMap); err != nil {
			continue // 无法解析的条目，跳过
		}

		// 检查创建时间
		if createdAtStr, ok := entryMap["created_at"].(string); ok {
			if createdAt, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
				if createdAt.After(cutoffTime) {
					keptEntries = append(keptEntries, entry)
					continue
				}
			}
		} else if createdAtStr, ok := entryMap["created_at"].(float64); ok {
			// 处理Unix时间戳格式
			createdAt := time.Unix(int64(createdAtStr), 0)
			if createdAt.After(cutoffTime) {
				keptEntries = append(keptEntries, entry)
				continue
			}
		}

		// 如果配置要求保留成功的记录，检查成功状态
		if tc.taskManager.cacheConfig.PreserveSuccessfulHashes {
			if success, ok := entryMap["success"].(bool); ok && success {
				keptEntries = append(keptEntries, entry)
				continue
			}
		}

		removedCount++
	}

	// 如果有删除的条目，更新文件
	if removedCount > 0 {
		newData, err := json.MarshalIndent(keptEntries, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to serialize new data: %w", err)
		}

		if err := os.WriteFile(filePath, newData, 0644); err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}

		tc.logger.Printf("Cleaned up %d entries from cache file %s", removedCount, filepath.Base(filePath))
		result.CacheEntriesRemoved += removedCount
	}

	return nil
}

// cleanupOrphanedFiles 清理孤儿文件
func (tc *TaskCleaner) cleanupOrphanedFiles(result *CleanupResult) error {
	// 扫描任务目录中的文件，检查是否为孤儿文件
	err := filepath.WalkDir(tc.taskManager.tasksDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		// 检查是否在有效的任务目录中
		taskDir := filepath.Dir(path)

		// 检查任务信息文件是否存在
		taskInfoFile := filepath.Join(taskDir, "task_info.json")
		if _, err := os.Stat(taskInfoFile); os.IsNotExist(err) {
			// 这是一个孤儿文件
			if info, err := d.Info(); err == nil {
				if err := os.Remove(path); err == nil {
					result.OrphanedFilesCleaned++
					result.BytesFreed += info.Size()
					tc.logger.Printf("Removed orphaned file: %s", path)
				}
			}
		}

		return nil
	})

	return err
}

// calculateDirectorySize 计算目录大小
func (tc *TaskCleaner) calculateDirectorySize(dirPath string) (int64, error) {
	var size int64

	err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			if info, err := d.Info(); err == nil {
				size += info.Size()
			}
		}

		return nil
	})

	return size, err
}

// statsTaskDirectories 统计任务目录信息
func (tc *TaskCleaner) statsTaskDirectories(stats *CleanupStats) error {
	entries, err := os.ReadDir(tc.taskManager.tasksDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	now := time.Now()
	var totalSize int64

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		totalSize += info.Size()
		stats.TaskCount++

		// 统计最近7天、30天的任务
		age := now.Sub(info.ModTime())
		if age <= 7*24*time.Hour {
			stats.RecentTasks7Days++
		}
		if age <= 30*24*time.Hour {
			stats.RecentTasks30Days++
		}
	}

	stats.TotalSizeBytes = totalSize
	return nil
}

// statsGlobalCache 统计全局缓存信息
func (tc *TaskCleaner) statsGlobalCache(stats *CleanupStats) error {
	globalFiles := []string{
		"submitted_tasks.json",
		"image_hashes.json",
		"task_index.json",
	}

	for _, file := range globalFiles {
		filePath := filepath.Join(tc.taskManager.globalDir, file)
		if info, err := os.Stat(filePath); err == nil {
			stats.CacheFiles++
			stats.CacheSizeBytes += info.Size()
		}
	}

	return nil
}

// CleanupResult 清理结果
type CleanupResult struct {
	StartTime            time.Time     `json:"start_time"`
	EndTime              time.Time     `json:"end_time"`
	Duration             time.Duration `json:"duration"`
	TasksCleaned         int           `json:"tasks_cleaned"`
	BytesFreed           int64         `json:"bytes_freed"`
	IndexEntriesRemoved  int           `json:"index_entries_removed"`
	CacheEntriesRemoved  int           `json:"cache_entries_removed"`
	OrphanedFilesCleaned int           `json:"orphaned_files_cleaned"`
	Errors               []string      `json:"errors,omitempty"`
	Skipped              bool          `json:"skipped,omitempty"`
	Reason               string        `json:"reason,omitempty"`
}

// CleanupStats 清理统计信息
type CleanupStats struct {
	TaskCount          int   `json:"task_count"`
	RecentTasks7Days   int   `json:"recent_tasks_7_days"`
	RecentTasks30Days  int   `json:"recent_tasks_30_days"`
	TotalSizeBytes     int64 `json:"total_size_bytes"`
	CacheFiles         int   `json:"cache_files"`
	CacheSizeBytes     int64 `json:"cache_size_bytes"`
}

// GetSizeMB 获取大小（MB）
func (cs *CleanupStats) GetSizeMB() float64 {
	return float64(cs.TotalSizeBytes) / (1024 * 1024)
}

// GetCacheSizeMB 获取缓存大小（MB）
func (cs *CleanupStats) GetCacheSizeMB() float64 {
	return float64(cs.CacheSizeBytes) / (1024 * 1024)
}