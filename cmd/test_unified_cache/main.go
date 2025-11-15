package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/allanpk716/to_icalendar/internal/cache"
)

func main() {
	fmt.Println("=== 统一缓存管理器测试 ===")

	// 1. 创建统一缓存管理器
	fmt.Println("\n1. 创建统一缓存管理器...")
	unifiedCacheMgr, err := cache.NewUnifiedCacheManager("", log.Default())
	if err != nil {
		log.Fatalf("创建统一缓存管理器失败: %v", err)
	}

	fmt.Printf("   ✓ 缓存基础目录: %s\n", unifiedCacheMgr.GetBaseCacheDir())

	// 2. 显示支持的缓存类型
	fmt.Println("\n2. 支持的缓存类型:")
	cacheTypes := unifiedCacheMgr.ListCacheTypes()
	for _, cacheType := range cacheTypes {
		cacheDir := unifiedCacheMgr.GetCacheDir(cacheType)
		fmt.Printf("   ✓ %s: %s\n", cacheType, cacheDir)
	}

	// 3. 显示缓存统计信息
	fmt.Println("\n3. 缓存统计信息:")
	stats := unifiedCacheMgr.GetCacheStats()
	statsJSON, _ := json.MarshalIndent(stats, "   ", "  ")
	fmt.Printf("   %s\n", string(statsJSON))

	// 4. 测试缓存文件操作
	fmt.Println("\n4. 测试缓存文件操作...")
	testCacheFiles(unifiedCacheMgr)

	// 5. 检查旧版缓存
	fmt.Println("\n5. 检查旧版缓存...")
	if unifiedCacheMgr.IsLegacyCacheExists() {
		fmt.Println("   ⚠️  检测到旧版缓存")
		legacyPaths := unifiedCacheMgr.GetLegacyCachePaths()
		for _, path := range legacyPaths {
			fmt.Printf("   - %s\n", path)
		}

		// 创建迁移管理器
		migrationMgr := cache.NewMigrationManager(unifiedCacheMgr, log.Default())
		plan := migrationMgr.GetMigrationPlan()

		if plan.MigrationRequired {
			fmt.Printf("   ✓ 需要迁移 %d 个项目，总大小: %.2f MB\n",
				len(plan.Migrations), float64(plan.TotalSize)/(1024*1024))

			// 试运行迁移
			options := &cache.MigrationOptions{
				DryRun:        true,
				Backup:        false,
				DeleteSource:  false,
				SkipExisting:  true,
				ForceOverwrite: false,
			}

			result, err := migrationMgr.ExecuteMigration(plan, options)
			if err != nil {
				fmt.Printf("   ✗ 迁移试运行失败: %v\n", err)
			} else {
				fmt.Printf("   ✓ 迁移试运行成功，将迁移 %d 个项目\n", len(result.Migrated))
			}
		}
	} else {
		fmt.Println("   ✓ 未检测到旧版缓存")
	}

	// 6. 测试缓存清理
	fmt.Println("\n6. 测试缓存清理...")
	for _, cacheType := range []cache.CacheType{
		cache.CacheTypeTemp,
		cache.CacheTypeConfig,
	} {
		if err := unifiedCacheMgr.ClearCache(cacheType); err != nil {
			fmt.Printf("   ✗ 清理缓存失败 %s: %v\n", cacheType, err)
		} else {
			fmt.Printf("   ✓ 已清理缓存: %s\n", cacheType)
		}
	}

	fmt.Println("\n=== 测试完成 ===")
}

// testCacheFiles 测试缓存文件操作
func testCacheFiles(ucm *cache.UnifiedCacheManager) {
	// 测试图片缓存
	testCacheType(ucm, cache.CacheTypeImages, "test_image.jpg", "test image content")

	// 测试任务缓存
	testCacheType(ucm, cache.CacheTypeTasks, "test_task.json", `{"title": "测试任务", "status": "pending"}`)

	// 测试全局缓存
	testCacheType(ucm, cache.CacheTypeGlobal, "test_data.txt", "test global data")
}

// testCacheType 测试特定类型的缓存操作
func testCacheType(ucm *cache.UnifiedCacheManager, cacheType cache.CacheType, filename, content string) {
	// 获取缓存文件路径
	filePath := ucm.GetCacheFilePath(cacheType, filename)

	// 写入测试文件
	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		fmt.Printf("   ✗ 写入文件失败 %s: %v\n", cacheType, err)
		return
	}

	fmt.Printf("   ✓ 已写入测试文件: %s\n", filePath)

	// 读取文件验证
	data, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("   ✗ 读取文件失败 %s: %v\n", cacheType, err)
		return
	}

	if string(data) == content {
		fmt.Printf("   ✓ 文件内容验证通过: %s\n", cacheType)
	} else {
		fmt.Printf("   ✗ 文件内容验证失败: %s\n", cacheType)
	}

	// 显示文件信息
	if info, err := os.Stat(filePath); err == nil {
		fmt.Printf("   - 文件大小: %d bytes, 修改时间: %s\n",
			info.Size(), info.ModTime().Format(time.RFC3339))
	}
}