package cache

import (
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"sync"
)

// CacheType 缓存类型枚举
type CacheType string

const (
	CacheTypeImages      CacheType = "images"      // 图片文件缓存
	CacheTypeTasks       CacheType = "tasks"       // 任务缓存
	CacheTypeGlobal      CacheType = "global"      // 全局缓存
	CacheTypeTemp        CacheType = "temp"        // 临时缓存
	CacheTypeConfig      CacheType = "config"      // 配置缓存
	CacheTypeSubmitted   CacheType = "submitted"   // 已提交任务缓存
	CacheTypeHashes      CacheType = "hashes"      // 哈希索引缓存
)

// UnifiedCacheManager 统一缓存管理器
type UnifiedCacheManager struct {
	baseCacheDir string           // 基础缓存目录
	subDirs      map[CacheType]string // 子目录映射
	mutex        sync.RWMutex     // 读写锁
	logger       *log.Logger      // 日志记录器
}

// NewUnifiedCacheManager 创建统一缓存管理器
func NewUnifiedCacheManager(baseDir string, logger *log.Logger) (*UnifiedCacheManager, error) {
	if logger == nil {
		logger = log.Default()
	}

	// 如果没有指定基础目录，使用默认位置
	if baseDir == "" {
		baseDir = getDefaultCacheDir()
	}

	// 确保基础缓存目录存在
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("创建基础缓存目录失败: %w", err)
	}

	ucm := &UnifiedCacheManager{
		baseCacheDir: baseDir,
		subDirs:      make(map[CacheType]string),
		logger:       logger,
	}

	// 初始化子目录
	ucm.initializeSubDirs()

	return ucm, nil
}

// initializeSubDirs 初始化缓存子目录
func (ucm *UnifiedCacheManager) initializeSubDirs() {
	ucm.mutex.Lock()
	defer ucm.mutex.Unlock()

	// 定义各类缓存的子目录
	ucm.subDirs = map[CacheType]string{
		CacheTypeImages:    "images",    // 图片文件缓存
		CacheTypeTasks:     "tasks",     // 任务缓存
		CacheTypeGlobal:    "global",    // 全局缓存
		CacheTypeTemp:      "temp",      // 临时缓存
		CacheTypeConfig:    "config",    // 配置缓存
		CacheTypeSubmitted: "submitted", // 已提交任务缓存
		CacheTypeHashes:    "hashes",    // 哈希索引缓存
	}

	// 创建所有子目录
	for cacheType, subDir := range ucm.subDirs {
		fullPath := filepath.Join(ucm.baseCacheDir, subDir)
		if err := os.MkdirAll(fullPath, 0755); err != nil {
			ucm.logger.Printf("创建缓存子目录失败: %s -> %s: %v", cacheType, fullPath, err)
		} else {
			ucm.logger.Printf("缓存子目录已创建: %s -> %s", cacheType, fullPath)
		}
	}
}

// GetCacheDir 获取指定类型的缓存目录
func (ucm *UnifiedCacheManager) GetCacheDir(cacheType CacheType) string {
	ucm.mutex.RLock()
	defer ucm.mutex.RUnlock()

	if subDir, exists := ucm.subDirs[cacheType]; exists {
		return filepath.Join(ucm.baseCacheDir, subDir)
	}

	// 如果类型不存在，返回基础缓存目录
	return ucm.baseCacheDir
}

// GetCacheFilePath 获取指定类型的缓存文件路径
func (ucm *UnifiedCacheManager) GetCacheFilePath(cacheType CacheType, filename string) string {
	cacheDir := ucm.GetCacheDir(cacheType)
	return filepath.Join(cacheDir, filename)
}

// GetBaseCacheDir 获取基础缓存目录
func (ucm *UnifiedCacheManager) GetBaseCacheDir() string {
	ucm.mutex.RLock()
	defer ucm.mutex.RUnlock()
	return ucm.baseCacheDir
}

// SetBaseCacheDir 设置基础缓存目录（用于动态切换）
func (ucm *UnifiedCacheManager) SetBaseCacheDir(newBaseDir string) error {
	ucm.mutex.Lock()
	defer ucm.mutex.Unlock()

	// 确保新目录存在
	if err := os.MkdirAll(newBaseDir, 0755); err != nil {
		return fmt.Errorf("创建新的基础缓存目录失败: %w", err)
	}

	oldBaseDir := ucm.baseCacheDir
	ucm.baseCacheDir = newBaseDir

	// 重新初始化子目录
	ucm.initializeSubDirs()

	ucm.logger.Printf("缓存基础目录已更改: %s -> %s", oldBaseDir, newBaseDir)
	return nil
}

// ListCacheTypes 列出所有支持的缓存类型
func (ucm *UnifiedCacheManager) ListCacheTypes() []CacheType {
	ucm.mutex.RLock()
	defer ucm.mutex.RUnlock()

	types := make([]CacheType, 0, len(ucm.subDirs))
	for cacheType := range ucm.subDirs {
		types = append(types, cacheType)
	}
	return types
}

// GetCacheStats 获取缓存统计信息
func (ucm *UnifiedCacheManager) GetCacheStats() map[string]interface{} {
	stats := make(map[string]interface{})

	ucm.mutex.RLock()
	defer ucm.mutex.RUnlock()

	stats["base_cache_dir"] = ucm.baseCacheDir
	stats["sub_dirs"] = make(map[string]string)

	for cacheType, subDir := range ucm.subDirs {
		stats["sub_dirs"].(map[string]string)[string(cacheType)] = subDir
	}

	// 统计各缓存目录的大小和文件数量
	cacheSizes := make(map[string]interface{})
	for cacheType := range ucm.subDirs {
		cacheDir := ucm.GetCacheDir(cacheType)
		size, count, err := calculateDirSize(cacheDir)
		if err != nil {
			ucm.logger.Printf("计算缓存目录大小失败: %s: %v", cacheDir, err)
			continue
		}

		cacheSizes[string(cacheType)] = map[string]interface{}{
			"size_bytes": size,
			"size_mb":    float64(size) / (1024 * 1024),
			"file_count": count,
		}
	}
	stats["cache_sizes"] = cacheSizes

	return stats
}

// ClearCache 清空指定类型的缓存
func (ucm *UnifiedCacheManager) ClearCache(cacheType CacheType) error {
	cacheDir := ucm.GetCacheDir(cacheType)

	// 检查目录是否存在
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		return nil // 目录不存在，无需清理
	}

	// 删除目录下的所有文件
	err := filepath.Walk(cacheDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if path == cacheDir {
			return nil // 跳过根目录
		}

		if info.IsDir() {
			return os.RemoveAll(path) // 删除子目录
		}

		return os.Remove(path) // 删除文件
	})

	if err != nil {
		return fmt.Errorf("清空缓存失败: %s: %w", cacheType, err)
	}

	// 重新创建目录
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("重新创建缓存目录失败: %s: %w", cacheType, err)
	}

	ucm.logger.Printf("缓存已清空: %s -> %s", cacheType, cacheDir)
	return nil
}

// ClearAllCache 清空所有缓存
func (ucm *UnifiedCacheManager) ClearAllCache() error {
	for cacheType := range ucm.subDirs {
		if err := ucm.ClearCache(cacheType); err != nil {
			ucm.logger.Printf("清空缓存失败: %s: %v", cacheType, err)
			return err
		}
	}

	ucm.logger.Printf("所有缓存已清空")
	return nil
}

// getDefaultCacheDir 获取默认缓存目录
func getDefaultCacheDir() string {
	// 优先检查环境变量
	if customDir := os.Getenv("TO_ICALendar_CACHE_DIR"); customDir != "" {
		return customDir
	}

	// 使用用户配置目录
	if usr, err := user.Current(); err == nil {
		return filepath.Join(usr.HomeDir, ".to_icalendar", "cache")
	}

	// 备用方案：使用系统临时目录
	return filepath.Join(os.TempDir(), "to_icalendar_cache")
}

// calculateDirSize 计算目录大小和文件数量
func calculateDirSize(dirPath string) (int64, int, error) {
	var totalSize int64
	var fileCount int

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			totalSize += info.Size()
			fileCount++
		}

		return nil
	})

	return totalSize, fileCount, err
}

// IsLegacyCacheExists 检查是否存在旧版缓存目录
func (ucm *UnifiedCacheManager) IsLegacyCacheExists() bool {
	legacyLocations := []string{
		"./cache",           // 程序根目录下的缓存
		"./cache/images",    // 旧版图片缓存
	}

	for _, legacyPath := range legacyLocations {
		if _, err := os.Stat(legacyPath); err == nil {
			return true
		}
	}

	return false
}

// GetLegacyCachePaths 获取旧版缓存路径列表
func (ucm *UnifiedCacheManager) GetLegacyCachePaths() []string {
	var paths []string
	legacyLocations := []string{
		"./cache",
		"./cache/images",
	}

	for _, legacyPath := range legacyLocations {
		if _, err := os.Stat(legacyPath); err == nil {
			paths = append(paths, legacyPath)
		}
	}

	return paths
}