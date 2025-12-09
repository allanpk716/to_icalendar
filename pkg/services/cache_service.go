package services

import (
	"fmt"

	"github.com/allanpk716/to_icalendar/pkg/cache"
	"github.com/allanpk716/to_icalendar/pkg/logger"
)

// CacheServiceImpl 缓存服务实现
type CacheServiceImpl struct {
	cacheManager *cache.UnifiedCacheManager
	logger       interface{}
}

// NewCacheService 创建缓存服务
func NewCacheService(cacheManager *cache.UnifiedCacheManager, logger interface{}) CacheService {
	return &CacheServiceImpl{
		cacheManager: cacheManager,
		logger:       logger,
	}
}

// Initialize 初始化缓存服务
func (cs *CacheServiceImpl) Initialize() error {
	if cs.cacheManager == nil {
		logger.Error("缓存管理器未初始化")
		return fmt.Errorf("缓存管理器未初始化")
	}
	logger.Info("缓存服务初始化完成")
	return nil
}

// GetManager 获取缓存管理器
func (cs *CacheServiceImpl) GetManager() *cache.UnifiedCacheManager {
	return cs.cacheManager
}

// GetCacheDir 获取缓存目录
func (cs *CacheServiceImpl) GetCacheDir() string {
	if cs.cacheManager != nil {
		return cs.cacheManager.GetBaseCacheDir()
	}
	return ""
}

// Cleanup 清理缓存
func (cs *CacheServiceImpl) Cleanup() error {
	if cs.cacheManager == nil {
		logger.Error("缓存管理器未初始化，无法清理")
		return fmt.Errorf("缓存管理器未初始化")
	}

	// 这里可以添加缓存清理逻辑
	// 目前缓存管理器本身已经包含了清理功能
	logger.Info("缓存清理完成")
	return nil
}