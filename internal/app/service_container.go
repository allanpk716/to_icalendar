package app

import (
	"github.com/allanpk716/to_icalendar/internal/cache"
	"github.com/allanpk716/to_icalendar/internal/models"
	"github.com/allanpk716/to_icalendar/internal/services"
)

// ServiceContainer 服务容器实现
type ServiceContainer struct {
	configDir       string
	config          *models.ServerConfig
	cacheManager    *cache.UnifiedCacheManager
	logger          interface{}
	configService   services.ConfigService
	cacheService    services.CacheService
	cleanupService  services.CleanupService
	clipboardService services.ClipboardService
	todoService     services.TodoService
	difyService     services.DifyService
}

// NewServiceContainer 创建服务容器
func NewServiceContainer(configDir string, config *models.ServerConfig, cacheManager *cache.UnifiedCacheManager, logger interface{}) *ServiceContainer {
	container := &ServiceContainer{
		configDir:      configDir,
		config:         config,
		cacheManager:   cacheManager,
		logger:         logger,
	}

	// 初始化所有服务
	container.initializeServices()

	return container
}

// initializeServices 初始化所有服务
func (sc *ServiceContainer) initializeServices() {
	// 初始化配置服务
	sc.configService = NewConfigService()

	// 初始化缓存服务
	sc.cacheService = NewCacheService(sc.cacheManager)

	// 初始化清理服务
	sc.cleanupService = NewCleanupService(sc.cacheManager)

	// 初始化剪贴板服务
	sc.clipboardService = services.NewClipboardService(sc.logger)

	// 初始化 Todo 服务（延迟初始化）
	sc.todoService = nil

	// 初始化 Dify 服务（延迟初始化）
	sc.difyService = nil
}

// GetConfigService 获取配置服务
func (sc *ServiceContainer) GetConfigService() services.ConfigService {
	return sc.configService
}

// GetCacheService 获取缓存服务
func (sc *ServiceContainer) GetCacheService() services.CacheService {
	return sc.cacheService
}

// GetCleanupService 获取清理服务
func (sc *ServiceContainer) GetCleanupService() services.CleanupService {
	return sc.cleanupService
}

// GetClipboardService 获取剪贴板服务
func (sc *ServiceContainer) GetClipboardService() services.ClipboardService {
	return sc.clipboardService
}

// GetTodoService 获取 Todo 服务
func (sc *ServiceContainer) GetTodoService() services.TodoService {
	if sc.todoService == nil {
		sc.todoService = NewTodoService(sc.config, sc.logger)
	}
	return sc.todoService
}

// GetDifyService 获取 Dify 服务
func (sc *ServiceContainer) GetDifyService() services.DifyService {
	if sc.difyService == nil {
		sc.difyService = NewDifyService(sc.config, sc.logger)
	}
	return sc.difyService
}

// GetLogger 获取日志器
func (sc *ServiceContainer) GetLogger() interface{} {
	return sc.logger
}

// GetConfigDir 获取配置目录
func (sc *ServiceContainer) GetConfigDir() string {
	return sc.configDir
}

// GetConfig 获取配置
func (sc *ServiceContainer) GetConfig() *models.ServerConfig {
	return sc.config
}

// GetCacheManager 获取缓存管理器
func (sc *ServiceContainer) GetCacheManager() *cache.UnifiedCacheManager {
	return sc.cacheManager
}