package app

import (
	"github.com/allanpk716/to_icalendar/internal/cache"
	"github.com/allanpk716/to_icalendar/internal/services"
)

// CacheServiceImpl 缓存服务实现
type CacheServiceImpl struct {
	cacheManager *cache.UnifiedCacheManager
}

// NewCacheService 创建缓存服务
func NewCacheService(cacheManager *cache.UnifiedCacheManager) services.CacheService {
	return &CacheServiceImpl{
		cacheManager: cacheManager,
	}
}

// Initialize 初始化缓存服务
func (s *CacheServiceImpl) Initialize() error {
	// 缓存管理器会在应用初始化时处理
	return nil
}

// GetManager 获取缓存管理器
func (s *CacheServiceImpl) GetManager() *cache.UnifiedCacheManager {
	return s.cacheManager
}

// GetCacheDir 获取缓存目录
func (s *CacheServiceImpl) GetCacheDir() string {
	return s.cacheManager.GetBaseCacheDir()
}

// Cleanup 清理缓存
func (s *CacheServiceImpl) Cleanup() error {
	// 简单实现，实际的清理逻辑在 CleanupService 中
	return nil
}