package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/allanpk716/to_icalendar/internal/cache"
	"github.com/allanpk716/to_icalendar/internal/services"
)

// CleanupServiceImpl 清理服务实现
type CleanupServiceImpl struct {
	cacheManager *cache.UnifiedCacheManager
}

// NewCleanupService 创建清理服务
func NewCleanupService(cacheManager *cache.UnifiedCacheManager) services.CleanupService {
	return &CleanupServiceImpl{
		cacheManager: cacheManager,
	}
}

// Cleanup 执行清理操作
func (s *CleanupServiceImpl) Cleanup(ctx context.Context, options *services.CleanupOptions) (*services.CleanupResult, error) {
	result := &services.CleanupResult{
		FilesByType: make(map[string]int64),
	}

	// 获取缓存目录
	cacheDir := s.cacheManager.GetCacheDir(cache.CacheTypeGlobal)
	if cacheDir == "" {
		return result, fmt.Errorf("无法获取缓存目录")
	}

	// 如果是预览模式，只统计不删除
	if options.DryRun {
		return s.previewCleanup(cacheDir, options)
	}

	// 执行实际清理
	if err := s.performCleanup(ctx, cacheDir, options, result); err != nil {
		return result, err
	}

	result.Message = "清理完成"

	return result, nil
}

// GetCleanupStats 获取清理统计信息
func (s *CleanupServiceImpl) GetCleanupStats(ctx context.Context) (*services.CleanupStats, error) {
	cacheDir := s.cacheManager.GetCacheDir(cache.CacheTypeGlobal)
	if cacheDir == "" {
		return nil, fmt.Errorf("无法获取缓存目录")
	}

	stats := &services.CleanupStats{}

	// 简化统计，只计算基本的缓存信息
	// 这里使用默认值，实际统计可以在后续实现中完善

	return stats, nil
}

// previewCleanup 预览清理操作
func (s *CleanupServiceImpl) previewCleanup(cacheDir string, options *services.CleanupOptions) (*services.CleanupResult, error) {
	result := &services.CleanupResult{
		FilesByType: make(map[string]int64),
		Skipped:     true,
		Message:     "预览模式 - 未实际删除文件",
	}

	err := filepath.Walk(cacheDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // 忽略访问错误
		}

		if !info.IsDir() && s.shouldCleanFile(path, info, options) {
			fileType := s.getFileType(path)
			result.FilesByType[fileType]++
			result.TotalFiles++
			result.TotalSize += info.Size()
		}

		return nil
	})

	if err != nil {
		return result, fmt.Errorf("预览清理时出错: %w", err)
	}

	return result, nil
}

// performCleanup 执行实际清理
func (s *CleanupServiceImpl) performCleanup(ctx context.Context, cacheDir string, options *services.CleanupOptions, result *services.CleanupResult) error {
	return filepath.Walk(cacheDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // 忽略访问错误
		}

		if !info.IsDir() && s.shouldCleanFile(path, info, options) {
			// 删除文件
			if err := os.Remove(path); err != nil {
				fmt.Printf("删除文件失败: %s - %v\n", path, err)
			} else {
				fileType := s.getFileType(path)
				result.FilesByType[fileType]++
				result.TotalFiles++
				result.TotalSize += info.Size()
			}
		}

		return nil
	})
}

// shouldCleanFile 判断文件是否应该被清理
func (s *CleanupServiceImpl) shouldCleanFile(path string, info os.FileInfo, options *services.CleanupOptions) bool {
	// 检查文件时间
	if options.OlderThan != "" {
		duration, err := time.ParseDuration(options.OlderThan)
		if err == nil {
			if time.Since(info.ModTime()) < duration {
				return false
			}
		}
	}

	fileType := s.getFileType(path)

	// 根据选项判断
	if options.All {
		return true
	}

	switch fileType {
	case "tasks":
		return options.Tasks
	case "images":
		return options.Images
	case "image_hashes":
		return options.ImageHashes
	case "temp":
		return options.Temp
	case "generated":
		return options.Generated
	default:
		return false
	}
}

// ParseCleanOptions 解析清理选项参数
func (s *CleanupServiceImpl) ParseCleanOptions(args []string) (*services.CleanupOptions, error) {
	options := &services.CleanupOptions{}

	for _, arg := range args {
		switch arg {
		case "--all", "-a":
			options.All = true
		case "--tasks":
			options.Tasks = true
		case "--images":
			options.Images = true
		case "--image-hashes":
			options.ImageHashes = true
		case "--temp":
			options.Temp = true
		case "--generated":
			options.Generated = true
		case "--dry-run", "-n":
			options.DryRun = true
		case "--force", "-f":
			options.Force = true
		case "--clear-all":
			options.ClearAll = true
		default:
			if strings.HasPrefix(arg, "--older-than=") {
				options.OlderThan = strings.TrimPrefix(arg, "--older-than=")
			}
		}
	}

	return options, nil
}

// getFileType 根据文件路径和名称判断文件类型
func (s *CleanupServiceImpl) getFileType(path string) string {
	base := filepath.Base(path)
	dir := filepath.Dir(path)

	switch {
	case strings.Contains(dir, "tasks"):
		return "tasks"
	case strings.Contains(dir, "images"):
		return "images"
	case strings.Contains(dir, "hashes"):
		return "image_hashes"
	case strings.Contains(dir, "temp"):
		return "temp"
	case strings.HasSuffix(base, ".json") && strings.Contains(path, "generated"):
		return "generated"
	default:
		return "other"
	}
}