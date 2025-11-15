package cache

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// MigrationManager 缓存迁移管理器
type MigrationManager struct {
	unifiedCacheMgr *UnifiedCacheManager
	logger          *log.Logger
}

// NewMigrationManager 创建迁移管理器
func NewMigrationManager(unifiedCacheMgr *UnifiedCacheManager, logger *log.Logger) *MigrationManager {
	if logger == nil {
		logger = log.Default()
	}

	return &MigrationManager{
		unifiedCacheMgr: unifiedCacheMgr,
		logger:          logger,
	}
}

// LegacyCachePaths 旧版缓存路径信息
type LegacyCachePaths struct {
	ProgramRootCache string   // 程序根目录缓存
	ImageCache       string   // 旧版图片缓存
	UserConfigCache  string   // 用户配置缓存
	AllPaths         []string // 所有检测到的旧缓存路径
}

// DetectLegacyCache 检测旧版缓存位置
func (mm *MigrationManager) DetectLegacyCache() *LegacyCachePaths {
	paths := &LegacyCachePaths{
		AllPaths: make([]string, 0),
	}

	// 检测程序根目录缓存
	if cacheDir := "./cache"; mm.pathExists(cacheDir) {
		paths.ProgramRootCache = cacheDir
		paths.AllPaths = append(paths.AllPaths, cacheDir)
	}

	// 检测图片缓存
	if imgCache := "./cache/images"; mm.pathExists(imgCache) {
		paths.ImageCache = imgCache
		paths.AllPaths = append(paths.AllPaths, imgCache)
	}

	// 检测用户配置目录下的旧缓存
	if usr, err := os.UserHomeDir(); err == nil {
		userCache := filepath.Join(usr, ".to_icalendar", "cache")
		if mm.pathExists(userCache) {
			paths.UserConfigCache = userCache
			paths.AllPaths = append(paths.AllPaths, userCache)
		}
	}

	return paths
}

// HasLegacyCache 检查是否存在需要迁移的旧缓存
func (mm *MigrationManager) HasLegacyCache() bool {
	legacyPaths := mm.DetectLegacyCache()
	return len(legacyPaths.AllPaths) > 0
}

// GetMigrationPlan 获取迁移计划
func (mm *MigrationManager) GetMigrationPlan() *MigrationPlan {
	legacyPaths := mm.DetectLegacyCache()
	plan := &MigrationPlan{
		LegacyPaths:       legacyPaths,
		TargetBaseDir:     mm.unifiedCacheMgr.GetBaseCacheDir(),
		MigrationRequired: len(legacyPaths.AllPaths) > 0,
		Migrations:        make([]*MigrationItem, 0),
	}

	if !plan.MigrationRequired {
		return plan
	}

	// 分析每个旧缓存路径的迁移方案
	for _, legacyPath := range legacyPaths.AllPaths {
		items := mm.analyzeLegacyPath(legacyPath)
		plan.Migrations = append(plan.Migrations, items...)
	}

	// 计算总大小和文件数量
	for _, item := range plan.Migrations {
		plan.TotalSize += item.Size
		plan.TotalFiles += item.FileCount
	}

	return plan
}

// MigrationPlan 迁移计划
type MigrationPlan struct {
	LegacyPaths       *LegacyCachePaths // 旧版缓存路径
	TargetBaseDir     string            // 目标基础目录
	MigrationRequired bool              // 是否需要迁移
	Migrations        []*MigrationItem  // 具体的迁移项目
	TotalSize         int64             // 总大小（字节）
	TotalFiles        int               // 总文件数量
}

// MigrationItem 迁移项目
type MigrationItem struct {
	SourcePath      string     // 源路径
	TargetPath      string     // 目标路径
	CacheType       CacheType  // 缓存类型
	Size            int64      // 文件大小
	FileCount       int        // 文件数量
	Description     string     // 描述
	MigrationAction string     // 迁移动作（move/copy/skip）
}

// ExecuteMigration 执行缓存迁移
func (mm *MigrationManager) ExecuteMigration(plan *MigrationPlan, options *MigrationOptions) (*MigrationResult, error) {
	result := &MigrationResult{
		Plan:       plan,
		StartTime:  time.Now(),
		Success:    true,
		Migrated:   make([]*MigrationItem, 0),
		Skipped:    make([]*MigrationItem, 0),
		Failed:     make([]*FailedMigration, 0),
	}

	mm.logger.Printf("开始缓存迁移，共 %d 个项目", len(plan.Migrations))

	for i, item := range plan.Migrations {
		mm.logger.Printf("迁移项目 %d/%d: %s", i+1, len(plan.Migrations), item.Description)

		if options.DryRun {
			mm.logger.Printf("[DRY RUN] 将迁移: %s -> %s", item.SourcePath, item.TargetPath)
			result.Migrated = append(result.Migrated, item)
			continue
		}

		err := mm.migrateItem(item, options)
		if err != nil {
			mm.logger.Printf("迁移失败: %s: %v", item.Description, err)
			result.Success = false
			result.Failed = append(result.Failed, &FailedMigration{
				Item:  item,
				Error: err.Error(),
			})
		} else {
			mm.logger.Printf("迁移成功: %s", item.Description)
			result.Migrated = append(result.Migrated, item)
		}
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	mm.logger.Printf("缓存迁移完成，耗时: %v", result.Duration)
	mm.logger.Printf("成功: %d, 跳过: %d, 失败: %d", len(result.Migrated), len(result.Skipped), len(result.Failed))

	return result, nil
}

// MigrationOptions 迁移选项
type MigrationOptions struct {
	DryRun        bool // 试运行
	Backup        bool // 是否备份
	DeleteSource  bool // 是否删除源文件
	SkipExisting  bool // 跳过已存在的文件
	ForceOverwrite bool // 强制覆盖
}

// MigrationResult 迁移结果
type MigrationResult struct {
	Plan       *MigrationPlan       // 迁移计划
	StartTime  time.Time            // 开始时间
	EndTime    time.Time            // 结束时间
	Duration   time.Duration        // 耗时
	Success    bool                 // 是否成功
	Migrated   []*MigrationItem     // 成功迁移的项目
	Skipped    []*MigrationItem     // 跳过的项目
	Failed     []*FailedMigration   // 失败的项目
}

// FailedMigration 失败的迁移
type FailedMigration struct {
	Item  *MigrationItem // 迁移项目
	Error string         // 错误信息
}

// analyzeLegacyPath 分析旧版缓存路径
func (mm *MigrationManager) analyzeLegacyPath(legacyPath string) []*MigrationItem {
	items := make([]*MigrationItem, 0)

	// 根据路径类型确定缓存类型和目标路径
	switch {
	case strings.Contains(legacyPath, "images"):
		// 图片缓存
		targetDir := mm.unifiedCacheMgr.GetCacheDir(CacheTypeImages)
		size, count := mm.calculatePathSize(legacyPath)

		items = append(items, &MigrationItem{
			SourcePath:      legacyPath,
			TargetPath:      targetDir,
			CacheType:       CacheTypeImages,
			Size:            size,
			FileCount:       count,
			Description:     fmt.Sprintf("图片缓存: %s", legacyPath),
			MigrationAction: "move",
		})

	case strings.Contains(legacyPath, "submitted_tasks.json"):
		// 已提交任务缓存
		targetFile := mm.unifiedCacheMgr.GetCacheFilePath(CacheTypeSubmitted, "submitted_tasks.json")
		size := mm.getFileSize(legacyPath)

		items = append(items, &MigrationItem{
			SourcePath:      legacyPath,
			TargetPath:      targetFile,
			CacheType:       CacheTypeSubmitted,
			Size:            size,
			FileCount:       1,
			Description:     fmt.Sprintf("已提交任务缓存: %s", legacyPath),
			MigrationAction: "copy",
		})

	case strings.Contains(legacyPath, "image_hashes.json"):
		// 图片哈希缓存
		targetFile := mm.unifiedCacheMgr.GetCacheFilePath(CacheTypeHashes, "image_hashes.json")
		size := mm.getFileSize(legacyPath)

		items = append(items, &MigrationItem{
			SourcePath:      legacyPath,
			TargetPath:      targetFile,
			CacheType:       CacheTypeHashes,
			Size:            size,
			FileCount:       1,
			Description:     fmt.Sprintf("图片哈希缓存: %s", legacyPath),
			MigrationAction: "copy",
		})

	default:
		// 其他缓存文件，移动到全局缓存
		targetDir := mm.unifiedCacheMgr.GetCacheDir(CacheTypeGlobal)
		size, count := mm.calculatePathSize(legacyPath)

		items = append(items, &MigrationItem{
			SourcePath:      legacyPath,
			TargetPath:      targetDir,
			CacheType:       CacheTypeGlobal,
			Size:            size,
			FileCount:       count,
			Description:     fmt.Sprintf("其他缓存: %s", legacyPath),
			MigrationAction: "move",
		})
	}

	return items
}

// migrateItem 执行单个项目的迁移
func (mm *MigrationManager) migrateItem(item *MigrationItem, options *MigrationOptions) error {
	// 检查源路径是否存在
	if !mm.pathExists(item.SourcePath) {
		return fmt.Errorf("源路径不存在: %s", item.SourcePath)
	}

	// 确保目标目录存在
	targetDir := item.TargetPath
	if item.FileCount == 1 {
		targetDir = filepath.Dir(item.TargetPath)
	}

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("创建目标目录失败: %w", err)
	}

	// 检查目标是否已存在
	if mm.pathExists(item.TargetPath) && !options.ForceOverwrite {
		if options.SkipExisting {
			mm.logger.Printf("目标已存在，跳过: %s", item.TargetPath)
			return nil
		}
		return fmt.Errorf("目标已存在: %s", item.TargetPath)
	}

	// 执行迁移
	switch item.MigrationAction {
	case "copy":
		return mm.copyPath(item.SourcePath, item.TargetPath, item.FileCount > 1)
	case "move":
		return mm.movePath(item.SourcePath, item.TargetPath, item.FileCount > 1)
	default:
		return fmt.Errorf("不支持的迁移动作: %s", item.MigrationAction)
	}
}

// copyPath 复制路径（文件或目录）
func (mm *MigrationManager) copyPath(src, dst string, isDir bool) error {
	if isDir {
		return mm.copyDir(src, dst)
	}
	return mm.copyFile(src, dst)
}

// movePath 移动路径（文件或目录）
func (mm *MigrationManager) movePath(src, dst string, isDir bool) error {
	if isDir {
		// 对于目录，先复制后删除
		if err := mm.copyDir(src, dst); err != nil {
			return err
		}
		return os.RemoveAll(src)
	}
	return os.Rename(src, dst)
}

// copyFile 复制文件
func (mm *MigrationManager) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

// copyDir 复制目录
func (mm *MigrationManager) copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 计算相对路径
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		targetPath := filepath.Join(dst, relPath)

		// 如果是目录，创建目录
		if info.IsDir() {
			return os.MkdirAll(targetPath, info.Mode())
		}

		// 复制文件
		return mm.copyFile(path, targetPath)
	})
}

// Helper methods

func (mm *MigrationManager) pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (mm *MigrationManager) getFileSize(path string) int64 {
	if info, err := os.Stat(path); err == nil {
		return info.Size()
	}
	return 0
}

func (mm *MigrationManager) calculatePathSize(path string) (int64, int) {
	var totalSize int64
	var fileCount int

	filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		totalSize += info.Size()
		fileCount++
		return nil
	})

	return totalSize, fileCount
}