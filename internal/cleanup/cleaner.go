package cleanup

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/allanpk716/to_icalendar/internal/config"
	"github.com/allanpk716/to_icalendar/internal/deduplication"
	"github.com/allanpk716/to_icalendar/internal/image"
)

// Cleaner 清理器结构
type Cleaner struct {
	configManager *config.ConfigManager
	cacheManager  *deduplication.CacheManager
	imageConfig   *image.ConfigManager
	logger        *log.Logger
}

// CleanOptions 清理选项
type CleanOptions struct {
	All           bool   // 清理所有缓存
	Tasks         bool   // 清理任务缓存
	Images        bool   // 清理图片缓存
	ImageHashes   bool   // 清理图片哈希缓存
	Temp          bool   // 清理临时文件
	Generated     bool   // 清理生成的JSON文件
	DryRun        bool   // 预览模式，不实际删除
	Force         bool   // 强制清理，跳过确认
	OlderThan     string // 时间过滤，如 "7d", "24h"
	ClearAll      bool   // 完全清空所有缓存数据
}

// CleanResult 清理结果
type CleanResult struct {
	CacheType    string    // 缓存类型
	FilesCount   int       // 删除的文件数量
	SizeBytes    int64     // 删除的文件大小（字节）
	Files        []string  // 删除的文件列表（预览模式下使用）
	Duration     time.Duration // 清理耗时
	Error        error     // 错误信息
}

// NewCleaner 创建新的清理器
func NewCleaner() *Cleaner {
	return &Cleaner{
		logger: log.Default(),
	}
}

// SetConfig 设置配置管理器
func (c *Cleaner) SetConfig(configManager *config.ConfigManager) {
	c.configManager = configManager
}

// SetCacheManager 设置缓存管理器
func (c *Cleaner) SetCacheManager(cacheManager *deduplication.CacheManager) {
	c.cacheManager = cacheManager
}

// SetImageConfig 设置图片配置
func (c *Cleaner) SetImageConfig(imageConfig *image.ConfigManager) {
	c.imageConfig = imageConfig
}

// Clean 执行清理操作
func (c *Cleaner) Clean(options CleanOptions) (*CleanSummary, error) {
	startTime := time.Now()
	summary := &CleanSummary{
		Results: make([]CleanResult, 0),
	}

	// 解析时间过滤条件
	olderThanTime, err := c.parseOlderThan(options.OlderThan)
	if err != nil {
		return nil, fmt.Errorf("解析时间参数失败: %v", err)
	}

	// 根据选项执行相应的清理操作
	if options.All || options.Tasks {
		result := c.cleanTasksCache(options.DryRun, olderThanTime)
		summary.Results = append(summary.Results, result)
	}

	if options.All || options.Images || options.ImageHashes {
		result := c.cleanImagesCache(options.DryRun, olderThanTime)
		summary.Results = append(summary.Results, result)
	}

	if options.ImageHashes {
		result := c.cleanImageHashCache(options.DryRun)
		summary.Results = append(summary.Results, result)
	}

	if options.All || options.Temp {
		result := c.cleanTempFiles(options.DryRun, olderThanTime)
		summary.Results = append(summary.Results, result)
	}

	if options.All || options.Generated {
		result := c.cleanGeneratedFiles(options.DryRun, olderThanTime)
		summary.Results = append(summary.Results, result)
	}

	// 如果设置了完全清空选项
	if options.ClearAll && c.cacheManager != nil && !options.DryRun {
		if err := c.cacheManager.ClearCache(); err != nil {
			c.logger.Printf("清空所有缓存失败: %v", err)
		} else {
			c.logger.Printf("已清空所有缓存数据")
		}
	}

	summary.Duration = time.Since(startTime)
	summary.TotalFiles = summary.getTotalFiles()
	summary.TotalSize = summary.getTotalSize()

	return summary, nil
}

// CleanSummary 清理摘要
type CleanSummary struct {
	Results    []CleanResult  // 清理结果列表
	Duration   time.Duration  // 总耗时
	TotalFiles int            // 总文件数
	TotalSize  int64          // 总大小（字节）
}

// getTotalFiles 计算总文件数
func (s *CleanSummary) getTotalFiles() int {
	total := 0
	for _, result := range s.Results {
		total += result.FilesCount
	}
	return total
}

// getTotalSize 计算总大小
func (s *CleanSummary) getTotalSize() int64 {
	var total int64
	for _, result := range s.Results {
		total += result.SizeBytes
	}
	return total
}

// cleanTasksCache 清理任务缓存
func (c *Cleaner) cleanTasksCache(dryRun bool, olderThanTime time.Time) CleanResult {
	result := CleanResult{
		CacheType: "任务缓存",
		Files:     make([]string, 0),
	}

	startTime := time.Now()
	defer func() {
		result.Duration = time.Since(startTime)
	}()

	if c.cacheManager == nil {
		result.Error = fmt.Errorf("缓存管理器未初始化")
		return result
	}

	// 获取缓存目录路径
	cacheDir := c.cacheManager.GetCacheDir()
	if cacheDir == "" {
		result.Error = fmt.Errorf("无法获取缓存目录路径")
		return result
	}

	// 遍历缓存目录
	err := filepath.Walk(cacheDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 跳过目录本身
		if info.IsDir() {
			return nil
		}

		// 检查文件时间
		if !olderThanTime.IsZero() && info.ModTime().After(olderThanTime) {
			return nil
		}

		// 收集文件信息
		result.FilesCount++
		result.SizeBytes += info.Size()
		result.Files = append(result.Files, path)

		// 如果不是预览模式，则删除文件
		if !dryRun {
			if err := os.Remove(path); err != nil {
				result.Error = fmt.Errorf("删除文件 %s 失败: %v", path, err)
				return err
			}
			c.logger.Printf("已删除任务缓存文件: %s", path)
		}

		return nil
	})

	if err != nil {
		result.Error = fmt.Errorf("清理任务缓存失败: %v", err)
	}

	return result
}

// cleanImagesCache 清理图片缓存
func (c *Cleaner) cleanImagesCache(dryRun bool, olderThanTime time.Time) CleanResult {
	result := CleanResult{
		CacheType: "图片缓存",
		Files:     make([]string, 0),
	}

	startTime := time.Now()
	defer func() {
		result.Duration = time.Since(startTime)
	}()

	// 1. 清理图片文件缓存
	cacheDir := c.getImageCacheDir()
	if cacheDir != "" {
		err := filepath.Walk(cacheDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// 跳过目录本身
			if info.IsDir() {
				return nil
			}

			// 只处理图片文件
			if !c.isImageFile(path) {
				return nil
			}

			// 检查文件时间
			if !olderThanTime.IsZero() && info.ModTime().After(olderThanTime) {
				return nil
			}

			// 收集文件信息
			result.FilesCount++
			result.SizeBytes += info.Size()
			result.Files = append(result.Files, path)

			// 如果不是预览模式，则删除文件
			if !dryRun {
				if err := os.Remove(path); err != nil {
					result.Error = fmt.Errorf("删除图片文件 %s 失败: %v", path, err)
					return err
				}
				c.logger.Printf("已删除图片缓存文件: %s", path)
			}

			return nil
		})

		if err != nil {
			result.Error = fmt.Errorf("清理图片文件缓存失败: %v", err)
			return result
		}
	}

	// 2. 清理图片哈希缓存文件
	if c.cacheManager != nil {
		cacheDir = c.cacheManager.GetCacheDir()
		if cacheDir != "" {
			imageHashFile := filepath.Join(cacheDir, "image_hashes.json")

			// 检查文件是否存在
			if info, err := os.Stat(imageHashFile); err == nil {
				// 检查文件时间
				if olderThanTime.IsZero() || info.ModTime().Before(olderThanTime) || dryRun {
					result.FilesCount++
					result.SizeBytes += info.Size()
					result.Files = append(result.Files, imageHashFile)

					// 如果不是预览模式，则清理图片哈希缓存
					if !dryRun {
						// 使用缓存管理器的清理方法，这样会处理过期数据
						if err := c.cacheManager.CleanupExpiredImages(); err != nil {
							result.Error = fmt.Errorf("清理图片哈希缓存失败: %v", err)
							return result
						}
						c.logger.Printf("已清理图片哈希缓存: %s", imageHashFile)
					}
				}
			}
		}
	}

	return result
}

// cleanImageHashCache 专门清理图片哈希缓存
func (c *Cleaner) cleanImageHashCache(dryRun bool) CleanResult {
	result := CleanResult{
		CacheType: "图片哈希缓存",
		Files:     make([]string, 0),
	}

	startTime := time.Now()
	defer func() {
		result.Duration = time.Since(startTime)
	}()

	if c.cacheManager == nil {
		result.Error = fmt.Errorf("缓存管理器未初始化")
		return result
	}

	// 获取缓存目录路径
	cacheDir := c.cacheManager.GetCacheDir()
	if cacheDir == "" {
		result.Error = fmt.Errorf("无法获取缓存目录路径")
		return result
	}

	// 图片哈希缓存文件路径
	imageHashFile := filepath.Join(cacheDir, "image_hashes.json")

	// 检查文件是否存在
	if info, err := os.Stat(imageHashFile); err == nil {
		result.FilesCount++
		result.SizeBytes += info.Size()
		result.Files = append(result.Files, imageHashFile)

		// 如果不是预览模式，则清空图片哈希缓存
		if !dryRun {
			if err := c.cacheManager.ClearImageCache(); err != nil {
				result.Error = fmt.Errorf("清空图片哈希缓存失败: %v", err)
				return result
			}
			c.logger.Printf("已清空图片哈希缓存: %s", imageHashFile)
		}
	} else if os.IsNotExist(err) {
		c.logger.Printf("图片哈希缓存文件不存在，跳过: %s", imageHashFile)
	} else {
		result.Error = fmt.Errorf("检查图片哈希缓存文件失败: %v", err)
		return result
	}

	return result
}

// cleanTempFiles 清理临时文件
func (c *Cleaner) cleanTempFiles(dryRun bool, olderThanTime time.Time) CleanResult {
	result := CleanResult{
		CacheType: "临时文件",
		Files:     make([]string, 0),
	}

	startTime := time.Now()
	defer func() {
		result.Duration = time.Since(startTime)
	}()

	// 获取临时目录
	tempDir := c.getTempDir()
	if tempDir == "" {
		result.Error = fmt.Errorf("无法获取临时目录路径")
		return result
	}

	// 如果临时目录不存在，则跳过
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		return result
	}

	// 遍历临时目录
	err := filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 跳过目录本身
		if info.IsDir() {
			return nil
		}

		// 检查文件时间
		if !olderThanTime.IsZero() && info.ModTime().After(olderThanTime) {
			return nil
		}

		// 收集文件信息
		result.FilesCount++
		result.SizeBytes += info.Size()
		result.Files = append(result.Files, path)

		// 如果不是预览模式，则删除文件
		if !dryRun {
			if err := os.Remove(path); err != nil {
				result.Error = fmt.Errorf("删除临时文件 %s 失败: %v", path, err)
				return err
			}
			c.logger.Printf("已删除临时文件: %s", path)
		}

		return nil
	})

	if err != nil {
		result.Error = fmt.Errorf("清理临时文件失败: %v", err)
	}

	return result
}

// cleanGeneratedFiles 清理生成的JSON文件
func (c *Cleaner) cleanGeneratedFiles(dryRun bool, olderThanTime time.Time) CleanResult {
	result := CleanResult{
		CacheType: "生成文件",
		Files:     make([]string, 0),
	}

	startTime := time.Now()
	defer func() {
		result.Duration = time.Since(startTime)
	}()

	// 使用当前目录作为输出目录
	outputDir := "."

	// 遍历输出目录，查找生成的JSON文件
	err := filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 跳过目录本身和隐藏目录
		if info.IsDir() {
			if strings.HasPrefix(filepath.Base(path), ".") || path == "." {
				return nil
			}
			return nil
		}

		// 只处理生成的JSON文件（通常是临时生成的）
		if !c.isGeneratedFile(path) {
			return nil
		}

		// 检查文件时间
		if !olderThanTime.IsZero() && info.ModTime().After(olderThanTime) {
			return nil
		}

		// 收集文件信息
		result.FilesCount++
		result.SizeBytes += info.Size()
		result.Files = append(result.Files, path)

		// 如果不是预览模式，则删除文件
		if !dryRun {
			if err := os.Remove(path); err != nil {
				result.Error = fmt.Errorf("删除生成文件 %s 失败: %v", path, err)
				return err
			}
			c.logger.Printf("已删除生成文件: %s", path)
		}

		return nil
	})

	if err != nil {
		result.Error = fmt.Errorf("清理生成文件失败: %v", err)
	}

	return result
}

// parseOlderThan 解析时间过滤参数
func (c *Cleaner) parseOlderThan(olderThan string) (time.Time, error) {
	if olderThan == "" {
		return time.Time{}, nil
	}

	now := time.Now()

	// 解析时间格式
	if strings.HasSuffix(olderThan, "d") {
		days := strings.TrimSuffix(olderThan, "d")
		var d int
		_, err := fmt.Sscanf(days, "%d", &d)
		if err != nil {
			return time.Time{}, fmt.Errorf("无效的天数格式: %s", olderThan)
		}
		return now.AddDate(0, 0, -d), nil
	}

	if strings.HasSuffix(olderThan, "h") {
		hours := strings.TrimSuffix(olderThan, "h")
		var h int
		_, err := fmt.Sscanf(hours, "%d", &h)
		if err != nil {
			return time.Time{}, fmt.Errorf("无效的小时格式: %s", olderThan)
		}
		return now.Add(-time.Duration(h) * time.Hour), nil
	}

	if strings.HasSuffix(olderThan, "m") {
		minutes := strings.TrimSuffix(olderThan, "m")
		var m int
		_, err := fmt.Sscanf(minutes, "%d", &m)
		if err != nil {
			return time.Time{}, fmt.Errorf("无效的分钟格式: %s", olderThan)
		}
		return now.Add(-time.Duration(m) * time.Minute), nil
	}

	return time.Time{}, fmt.Errorf("不支持的时间格式: %s (支持: 7d, 24h, 30m)", olderThan)
}

// getImageCacheDir 获取图片缓存目录
func (c *Cleaner) getImageCacheDir() string {
	// 尝试从图片配置获取
	if c.imageConfig != nil {
		if cacheDir := c.imageConfig.GetCacheDir(); cacheDir != "" {
			return cacheDir
		}
	}

	// 尝试从环境变量获取
	if cacheDir := os.Getenv("TO_ICALendar_CACHE_DIR"); cacheDir != "" {
		return cacheDir
	}

	// 使用默认路径
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(homeDir, ".to_icalendar", "cache")
}

// getTempDir 获取临时目录
func (c *Cleaner) getTempDir() string {
	// 尝试从环境变量获取
	if tempDir := os.Getenv("TO_ICALendar_TEMP_DIR"); tempDir != "" {
		return tempDir
	}

	// 使用默认路径
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(homeDir, ".to_icalendar", "temp")
}

// isImageFile 检查是否为图片文件
func (c *Cleaner) isImageFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	imageExts := []string{".png", ".jpg", ".jpeg", ".gif", ".bmp", ".webp"}
	for _, imgExt := range imageExts {
		if ext == imgExt {
			return true
		}
	}
	return false
}

// isGeneratedFile 检查是否为生成的文件
func (c *Cleaner) isGeneratedFile(path string) bool {
	// 检查文件名模式
	filename := filepath.Base(path)

	// 临时生成的JSON文件
	if strings.HasPrefix(filename, "temp_") && strings.HasSuffix(filename, ".json") {
		return true
	}

	// 解析后生成的文件
	if strings.Contains(filename, "_parsed_") && strings.HasSuffix(filename, ".json") {
		return true
	}

	// 其他临时文件模式
	if strings.HasPrefix(filename, "dify_") && strings.HasSuffix(filename, ".json") {
		return true
	}

	return false
}

// PrintSummary 打印清理摘要
func (s *CleanSummary) PrintSummary() {
	fmt.Printf("\n=== 清理完成 ===\n")
	fmt.Printf("总耗时: %v\n", s.Duration)
	fmt.Printf("清理文件数: %d\n", s.TotalFiles)
	fmt.Printf("释放空间: %s\n", formatBytes(s.TotalSize))

	fmt.Printf("\n详细结果:\n")
	for _, result := range s.Results {
		if result.Error != nil {
			fmt.Printf("❌ %s: %v\n", result.CacheType, result.Error)
			continue
		}

		fmt.Printf("✅ %s: %d个文件, %s, 耗时%v\n",
			result.CacheType, result.FilesCount, formatBytes(result.SizeBytes), result.Duration)
	}
}

// PrintPreview 打印预览信息
func (s *CleanSummary) PrintPreview() {
	fmt.Printf("\n=== 清理预览 ===\n")
	fmt.Printf("预计删除文件数: %d\n", s.TotalFiles)
	fmt.Printf("预计释放空间: %s\n", formatBytes(s.TotalSize))

	fmt.Printf("\n将要删除的文件:\n")
	for _, result := range s.Results {
		if result.Error != nil {
			fmt.Printf("❌ %s: %v\n", result.CacheType, result.Error)
			continue
		}

		fmt.Printf("\n📁 %s (%d个文件, %s):\n",
			result.CacheType, result.FilesCount, formatBytes(result.SizeBytes))

		// 显示文件列表（限制数量）
		maxFiles := 10
		for i, file := range result.Files {
			if i >= maxFiles {
				fmt.Printf("  ... 还有 %d 个文件\n", len(result.Files)-maxFiles)
				break
			}
			fmt.Printf("  - %s\n", file)
		}
	}

	fmt.Printf("\n注意：这只是预览，实际不会删除任何文件。使用 --force 参数执行实际清理。\n")
}

// formatBytes 格式化字节数为人类可读格式
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}