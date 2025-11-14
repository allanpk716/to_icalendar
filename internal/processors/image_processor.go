package processors

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/allanpk716/to_icalendar/internal/dify"
	"github.com/allanpk716/to_icalendar/internal/image"
	"github.com/allanpk716/to_icalendar/internal/models"
	"github.com/sirupsen/logrus"
)

// ImageProcessor handles image content processing workflow
type ImageProcessor struct {
	difyProcessor  *dify.Processor
	tempDir        string
	normalizer     *image.ImageNormalizer
	configManager  *image.ConfigManager
	logger         *logrus.Logger
}

// NewImageProcessor creates a new image processor
func NewImageProcessor(difyProcessor *dify.Processor) (*ImageProcessor, error) {
	// 创建临时目录
	tempDir := filepath.Join(os.TempDir(), "to_icalendar_images")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	logger := logrus.New()
	configManager := image.NewConfigManager(".", logger)
	if err := configManager.LoadConfig(); err != nil {
		log.Printf("加载图片处理配置失败: %v", err)
	}

	return &ImageProcessor{
		difyProcessor: difyProcessor,
		tempDir:       tempDir,
		configManager: configManager,
		logger:        logger,
	}, nil
}

// NewImageProcessorWithNormalizer creates a new image processor with image normalizer
func NewImageProcessorWithNormalizer(difyProcessor *dify.Processor, normalizer *image.ImageNormalizer) (*ImageProcessor, error) {
	// 创建临时目录
	tempDir := filepath.Join(os.TempDir(), "to_icalendar_images")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	logger := logrus.New()
	configManager := image.NewConfigManager(".", logger)
	if err := configManager.LoadConfig(); err != nil {
		log.Printf("加载图片处理配置失败: %v", err)
	}

	return &ImageProcessor{
		difyProcessor: difyProcessor,
		tempDir:       tempDir,
		normalizer:    normalizer,
		configManager: configManager,
		logger:        logger,
	}, nil
}

// ProcessClipboardImage processes an image from clipboard
func (ip *ImageProcessor) ProcessClipboardImage(ctx context.Context, imageData []byte) (*models.ProcessingResult, error) {
	startTime := time.Now()

	log.Printf("开始处理剪贴板图片，大小: %d bytes", len(imageData))

	// 验证图片数据
	if len(imageData) == 0 {
		return &models.ProcessingResult{
			Success:      false,
			ErrorMessage: "图片数据为空",
			ProcessingTime: time.Since(startTime),
		}, fmt.Errorf("image data is empty")
	}

	// 生成临时文件名
	fileName := fmt.Sprintf("clipboard_%s.png", time.Now().Format("20060102_150405"))
	tempFilePath := filepath.Join(ip.tempDir, fileName)

	// 应用图片标准化（如果可用）
	processedImageData := imageData
	if ip.normalizer != nil {
		log.Printf("应用图片标准化处理")
		normalizedData, err := ip.normalizeImageData(imageData, tempFilePath)
		if err != nil {
			log.Printf("图片标准化失败，使用原始图片: %v", err)
		} else {
			processedImageData = normalizedData
			log.Printf("图片标准化完成，处理后大小: %d bytes", len(processedImageData))

			// 缓存标准化后的图片
			if ip.configManager != nil && ip.configManager.IsCacheEnabled() {
				timestamp := time.Now().Format("20060102_150405_000000")
				normalizedFilename := fmt.Sprintf("clipboard_normalized_%s.png", timestamp)

				cachePath, err := ip.configManager.SaveCacheImage(normalizedData, normalizedFilename)
				if err != nil {
					log.Printf("缓存标准化图片失败: %v", err)
				} else {
					log.Printf("标准化图片已缓存: %s", cachePath)
				}
			}
		}
	}

	// 保存图片到临时文件
	if err := ip.saveImageToTempFile(processedImageData, tempFilePath); err != nil {
		return &models.ProcessingResult{
			Success:      false,
			ErrorMessage: fmt.Sprintf("保存临时文件失败: %v", err),
			ProcessingTime: time.Since(startTime),
		}, err
	}

	// 确保清理临时文件
	defer func() {
		if err := os.Remove(tempFilePath); err != nil {
			log.Printf("清理临时文件失败: %v", err)
		}
	}()

	log.Printf("图片已保存到临时文件: %s", tempFilePath)

	// 使用Dify处理器处理图片
	response, err := ip.difyProcessor.ProcessImage(ctx, processedImageData, fileName)
	if err != nil {
		return &models.ProcessingResult{
			Success:        false,
			ErrorMessage:   fmt.Sprintf("Dify处理失败: %v", err),
			ProcessingTime: time.Since(startTime),
		}, err
	}

	// 转换为处理结果格式
	result := &models.ProcessingResult{
		Success:        response.Success,
		Reminder:       response.Reminder,
		ParsedInfo:     response.ParsedInfo,
		ErrorMessage:   response.ErrorMessage,
		ProcessingTime: time.Since(startTime),
	}

	log.Printf("图片处理完成，成功: %v", result.Success)

	return result, nil
}

// ProcessImageFile processes an image file
func (ip *ImageProcessor) ProcessImageFile(ctx context.Context, filePath string) (*models.ProcessingResult, error) {
	startTime := time.Now()

	log.Printf("开始处理图片文件: %s", filePath)

	// 验证文件存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return &models.ProcessingResult{
			Success:      false,
			ErrorMessage: fmt.Sprintf("文件不存在: %s", filePath),
			ProcessingTime: time.Since(startTime),
		}, fmt.Errorf("file not found: %s", filePath)
	}

	// 读取图片文件
	imageData, err := os.ReadFile(filePath)
	if err != nil {
		return &models.ProcessingResult{
			Success:      false,
			ErrorMessage: fmt.Sprintf("读取文件失败: %v", err),
			ProcessingTime: time.Since(startTime),
		}, err
	}

	// 获取文件名
	fileName := filepath.Base(filePath)

	// 使用Dify处理器处理图片
	response, err := ip.difyProcessor.ProcessImage(ctx, imageData, fileName)
	if err != nil {
		return &models.ProcessingResult{
			Success:        false,
			ErrorMessage:   fmt.Sprintf("Dify处理失败: %v", err),
			ProcessingTime: time.Since(startTime),
		}, err
	}

	// 转换为处理结果格式
	result := &models.ProcessingResult{
		Success:        response.Success,
		Reminder:       response.Reminder,
		ParsedInfo:     response.ParsedInfo,
		ErrorMessage:   response.ErrorMessage,
		ProcessingTime: time.Since(startTime),
	}

	log.Printf("图片文件处理完成，成功: %v", result.Success)

	return result, nil
}

// BatchProcessImages processes multiple images
func (ip *ImageProcessor) BatchProcessImages(ctx context.Context, filePaths []string) ([]*models.ProcessingResult, error) {
	results := make([]*models.ProcessingResult, 0, len(filePaths))

	log.Printf("开始批量处理 %d 个图片文件", len(filePaths))

	for i, filePath := range filePaths {
		log.Printf("处理第 %d/%d 个文件: %s", i+1, len(filePaths), filePath)

		result, err := ip.ProcessImageFile(ctx, filePath)
		if err != nil {
			log.Printf("处理文件失败: %s, 错误: %v", filePath, err)
			// 继续处理其他文件，不中断批量处理
			result = &models.ProcessingResult{
				Success:      false,
				ErrorMessage: fmt.Sprintf("处理文件失败: %v", err),
			}
		}

		results = append(results, result)

		// 检查上下文是否被取消
		select {
		case <-ctx.Done():
			log.Printf("批量处理被取消")
			return results, ctx.Err()
		default:
			// 继续处理
		}
	}

	successCount := 0
	for _, result := range results {
		if result.Success {
			successCount++
		}
	}

	log.Printf("批量处理完成，成功: %d/%d", successCount, len(filePaths))

	return results, nil
}

// ValidateImage validates image data
func (ip *ImageProcessor) ValidateImage(imageData []byte) error {
	if len(imageData) == 0 {
		return fmt.Errorf("image data is empty")
	}

	// 检查文件大小（限制为10MB）
	const maxImageSize = 10 * 1024 * 1024
	if len(imageData) > maxImageSize {
		return fmt.Errorf("image size too large: %d bytes (max: %d bytes)", len(imageData), maxImageSize)
	}

	// 这里可以添加更多的图片格式验证
	// 例如：检查图片头部标识，验证是否为有效的图片格式

	return nil
}

// normalizeImageData 标准化图片数据
func (ip *ImageProcessor) normalizeImageData(imageData []byte, tempFilePath string) ([]byte, error) {
	if ip.normalizer == nil {
		return imageData, nil
	}

	// 先保存原始图片到临时文件
	if err := ip.saveImageToTempFile(imageData, tempFilePath); err != nil {
		return nil, fmt.Errorf("保存原始图片失败: %w", err)
	}

	// 生成标准化后的临时文件路径
	normalizedPath := strings.TrimSuffix(tempFilePath, filepath.Ext(tempFilePath)) + "_normalized.png"

	// 标准化图片文件
	err := ip.normalizer.NormalizeFile(tempFilePath, normalizedPath)
	if err != nil {
		return nil, fmt.Errorf("图片标准化失败: %w", err)
	}

	// 读取标准化后的图片
	normalizedData, err := os.ReadFile(normalizedPath)
	if err != nil {
		return nil, fmt.Errorf("读取标准化图片失败: %w", err)
	}

	// 清理标准化临时文件
	os.Remove(normalizedPath)

	return normalizedData, nil
}

// saveImageToTempFile saves image data to a temporary file
func (ip *ImageProcessor) saveImageToTempFile(imageData []byte, filePath string) error {
	return os.WriteFile(filePath, imageData, 0644)
}

// Cleanup cleans up temporary files with error handling
func (ip *ImageProcessor) Cleanup() error {
	if ip.tempDir == "" {
		return nil
	}

	log.Printf("清理临时目录: %s", ip.tempDir)

	// 检查目录是否存在
	if _, err := os.Stat(ip.tempDir); os.IsNotExist(err) {
		log.Printf("临时目录不存在，无需清理: %s", ip.tempDir)
		return nil
	}

	// 删除临时目录及其内容
	err := os.RemoveAll(ip.tempDir)
	if err != nil {
		log.Printf("清理临时目录失败: %v", err)
		return fmt.Errorf("failed to cleanup temp directory %s: %w", ip.tempDir, err)
	}

	log.Printf("临时目录清理成功: %s", ip.tempDir)
	return nil
}

// GetTempDir returns the temporary directory path
func (ip *ImageProcessor) GetTempDir() string {
	return ip.tempDir
}

// SetTempDir sets a new temporary directory
func (ip *ImageProcessor) SetTempDir(tempDir string) error {
	// 创建新目录
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}

	// 清理旧目录
	if ip.tempDir != "" {
		ip.Cleanup()
	}

	ip.tempDir = tempDir
	return nil
}

// GetSupportedFormats returns supported image formats
func (ip *ImageProcessor) GetSupportedFormats() []string {
	return []string{
		"png",
		"jpg",
		"jpeg",
		"bmp",
		"gif",
		"webp",
	}
}

// IsFormatSupported checks if the format is supported
func (ip *ImageProcessor) IsFormatSupported(format string) bool {
	supportedFormats := ip.GetSupportedFormats()
	for _, supported := range supportedFormats {
		if supported == format {
			return true
		}
	}
	return false
}

// GetCacheDir returns the cache directory path
func (ip *ImageProcessor) GetCacheDir() string {
	if ip.configManager != nil {
		return ip.configManager.GetCacheDir()
	}
	return ""
}

// IsCacheEnabled checks if caching is enabled
func (ip *ImageProcessor) IsCacheEnabled() bool {
	if ip.configManager != nil {
		return ip.configManager.IsCacheEnabled()
	}
	return false
}