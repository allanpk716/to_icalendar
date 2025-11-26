package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/allanpk716/to_icalendar/internal/cache"
	"github.com/allanpk716/to_icalendar/internal/commands"
	"github.com/allanpk716/to_icalendar/internal/config"
	"github.com/allanpk716/to_icalendar/internal/logger"
	"github.com/allanpk716/to_icalendar/internal/models"
)

// Application 应用主类
type Application struct {
	container        commands.ServiceContainer
	config           *models.ServerConfig
	unifiedCacheMgr  *cache.UnifiedCacheManager
	logger           interface{}
	initialized      bool
	mu               sync.RWMutex
}

// NewApplication 创建应用实例
func NewApplication() *Application {
	return &Application{
		initialized: false,
	}
}

// Initialize 初始化应用
func (app *Application) Initialize(ctx context.Context) error {
	app.mu.Lock()
	defer app.mu.Unlock()

	if app.initialized {
		return nil
	}

	// 初始化日志系统
	logger.Initialize(&models.LoggingConfig{
		Level:         "info",
		ConsoleOutput: true,
		FileOutput:    true,
		LogDir:        "./Logs",
	})
	app.logger = logger.GetLogger()

	// 获取配置目录
	configDir, err := app.getConfigDir()
	if err != nil {
		return fmt.Errorf("获取配置目录失败: %w", err)
	}

	// 初始化统一缓存系统
	app.unifiedCacheMgr, err = app.initializeCacheSystem(configDir)
	if err != nil {
		logger.Errorf("缓存系统初始化失败: %v", err)
		// 缓存初始化失败不应该阻止程序运行，只记录错误
	}

	// 加载服务器配置
	configManager := config.NewConfigManager()
	serverConfigPath := filepath.Join(configDir, "server.yaml")
	serverConfig, err := configManager.LoadServerConfig(serverConfigPath)
	if err != nil {
		return fmt.Errorf("加载服务器配置失败: %w", err)
	}
	app.config = serverConfig

	// 重新初始化日志系统（使用配置文件中的设置）
	if err := logger.Initialize(&serverConfig.Logging); err != nil {
		logger.Errorf("初始化日志系统失败: %v", err)
		// 继续使用默认配置
	}

	// 创建服务容器
	app.container = NewServiceContainer(configDir, serverConfig, app.unifiedCacheMgr, app.logger)

	app.initialized = true
	logger.Info("应用初始化完成")
	return nil
}

// GetServiceContainer 获取服务容器
func (app *Application) GetServiceContainer() commands.ServiceContainer {
	app.mu.RLock()
	defer app.mu.RUnlock()

	if !app.initialized {
		panic("应用未初始化，请先调用 Initialize()")
	}

	return app.container
}

// GetConfig 获取配置
func (app *Application) GetConfig() *models.ServerConfig {
	app.mu.RLock()
	defer app.mu.RUnlock()
	return app.config
}

// GetCacheManager 获取缓存管理器
func (app *Application) GetCacheManager() *cache.UnifiedCacheManager {
	app.mu.RLock()
	defer app.mu.RUnlock()
	return app.unifiedCacheMgr
}

// Shutdown 关闭应用
func (app *Application) Shutdown(ctx context.Context) error {
	app.mu.Lock()
	defer app.mu.Unlock()

	if app.unifiedCacheMgr != nil {
		// 清理缓存资源
	}

	logger.Info("应用已关闭")
	app.initialized = false
	return nil
}

// getConfigDir 获取配置目录
func (app *Application) getConfigDir() (string, error) {
	// 获取用户主目录
	usr, err := os.UserHomeDir()
	if err != nil {
		// 如果无法获取用户目录，使用当前目录的子目录
		return ".to_icalendar", nil
	}

	configDir := filepath.Join(usr, ".to_icalendar")
	return configDir, nil
}

// initializeCacheSystem 初始化统一缓存系统
func (app *Application) initializeCacheSystem(configDir string) (*cache.UnifiedCacheManager, error) {
	// 检查并迁移配置文件
	if err := app.checkAndMigrateConfigFiles(configDir); err != nil {
		logger.Infof("配置文件迁移失败: %v", err)
		// 迁移失败不应该阻止程序启动，只记录日志
	}

	// 创建统一缓存管理器
	unifiedCacheMgr, err := cache.NewUnifiedCacheManager(filepath.Join(configDir, "cache"), logger.GetLogger().GetStdLogger())
	if err != nil {
		return nil, fmt.Errorf("创建统一缓存管理器失败: %w", err)
	}

	// 检查是否需要迁移
	if err := app.performCacheMigration(unifiedCacheMgr); err != nil {
		logger.Infof("缓存迁移失败: %v", err)
		// 迁移失败不应该阻止程序启动，只记录日志
	}

	logger.Infof("缓存系统初始化完成，缓存目录: %s", unifiedCacheMgr.GetBaseCacheDir())
	return unifiedCacheMgr, nil
}

// performCacheMigration 执行缓存迁移
func (app *Application) performCacheMigration(unifiedCacheMgr *cache.UnifiedCacheManager) error {
	// 创建迁移管理器
	migrationMgr := cache.NewMigrationManager(unifiedCacheMgr, logger.GetLogger().GetStdLogger())

	// 检查是否需要迁移
	if !migrationMgr.HasLegacyCache() {
		return nil // 无需迁移
	}

	// 检查是否已经完成迁移
	if app.isMigrationCompleted(unifiedCacheMgr.GetBaseCacheDir()) {
		logger.Info("检测到缓存已完成迁移，跳过")
		return nil
	}

	logger.Info("🚀 检测到旧版缓存，开始自动迁移...")

	// 获取迁移计划
	plan := migrationMgr.GetMigrationPlan()
	if !plan.MigrationRequired {
		return nil
	}

	logger.Infof("📦 发现 %d 个旧版缓存项目，总大小: %.2f MB",
		len(plan.Migrations), float64(plan.TotalSize)/(1024*1024))

	// 执行迁移
	options := &cache.MigrationOptions{
		DryRun:         false,
		Backup:         false, // 不需要备份，直接迁移
		DeleteSource:   true,
		SkipExisting:   true,
		ForceOverwrite: false,
	}

	result, err := migrationMgr.ExecuteMigration(plan, options)
	if err != nil {
		return fmt.Errorf("执行缓存迁移失败: %w", err)
	}

	if result.Success {
		logger.Infof("✅ 缓存迁移完成，共迁移 %d 个项目", len(result.Migrated))

		// 标记迁移完成
		app.markMigrationCompleted(unifiedCacheMgr.GetBaseCacheDir())

		// 强制清理旧缓存目录
		app.forceCleanupLegacyDirs(plan.LegacyPaths)

	} else {
		logger.Infof("⚠️  缓存迁移部分失败，成功: %d, 失败: %d",
			len(result.Migrated), len(result.Failed))
	}

	return nil
}

// isMigrationCompleted 检查是否已经完成迁移
func (app *Application) isMigrationCompleted(cacheBaseDir string) bool {
	migrationFile := filepath.Join(cacheBaseDir, ".migration_completed")
	_, err := os.Stat(migrationFile)
	return err == nil
}

// markMigrationCompleted 标记迁移完成
func (app *Application) markMigrationCompleted(cacheBaseDir string) error {
	migrationFile := filepath.Join(cacheBaseDir, ".migration_completed")
	return os.WriteFile(migrationFile, []byte("migrated"), 0644)
}

// forceCleanupLegacyDirs 强制清理旧版缓存目录（即使非空）
func (app *Application) forceCleanupLegacyDirs(legacyPaths *cache.LegacyCachePaths) {
	if legacyPaths == nil {
		return
	}

	// 要强制清理的目录列表
	dirsToClean := []string{
		legacyPaths.ProgramRootCache,
		legacyPaths.ImageCache,
	}

	for _, dir := range dirsToClean {
		if dir == "" {
			continue
		}

		// 检查目录是否存在
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue // 目录不存在，无需清理
		}

		// 尝试删除目录
		if err := os.RemoveAll(dir); err != nil {
			logger.Infof("⚠️  强制清理目录失败: %s: %v", dir, err)
		} else {
			logger.Infof("🧹 强制清理旧缓存目录: %s", dir)
		}
	}

	// 也清理可能的旧缓存文件
	oldCacheFiles := []string{
		"./cache/submitted_tasks.json",
		"./cache/image_hashes.json",
	}

	for _, file := range oldCacheFiles {
		if _, err := os.Stat(file); err == nil {
			if err := os.Remove(file); err != nil {
				logger.Infof("⚠️  清理旧缓存文件失败: %s: %v", file, err)
			} else {
				logger.Infof("🧹 已清理旧缓存文件: %s", file)
			}
		}
	}
}

// checkAndMigrateConfigFiles 检查并迁移配置文件到用户配置目录
func (app *Application) checkAndMigrateConfigFiles(configDir string) error {
	// 这里可以添加配置文件迁移逻辑
	// 目前暂时不需要特殊的迁移逻辑
	return nil
}