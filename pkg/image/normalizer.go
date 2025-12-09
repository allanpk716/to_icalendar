package image

import (
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nfnt/resize"
	"github.com/sirupsen/logrus"
)

// NormalizationConfig 图片标准化配置
type NormalizationConfig struct {
	// 最大分辨率
	MaxWidth  int
	MaxHeight int
	// 压缩质量
	PNGCompressionLevel png.CompressionLevel
	JPEGQuality         int
	// 输出格式
	OutputFormat string
	// 文件大小限制 (字节)
	MaxFileSize int64
	// 是否保持宽高比
	KeepAspectRatio bool
}

// DefaultNormalizationConfig 默认配置
func DefaultNormalizationConfig() *NormalizationConfig {
	return &NormalizationConfig{
		MaxWidth:           1920,
		MaxHeight:          1080,
		PNGCompressionLevel: png.DefaultCompression,
		JPEGQuality:        85,
		OutputFormat:       "png",
		MaxFileSize:        5 * 1024 * 1024, // 5MB
		KeepAspectRatio:    true,
	}
}

// DocumentNormalizationConfig 文档类图片配置
func DocumentNormalizationConfig() *NormalizationConfig {
	return &NormalizationConfig{
		MaxWidth:           800,
		MaxHeight:          600,
		PNGCompressionLevel: png.BestCompression,
		JPEGQuality:        90,
		OutputFormat:       "png",
		MaxFileSize:        2 * 1024 * 1024, // 2MB
		KeepAspectRatio:    true,
	}
}

// ImageNormalizer 图片标准化器
type ImageNormalizer struct {
	config *NormalizationConfig
	logger *logrus.Logger
}

// NewImageNormalizer 创建新的图片标准化器
func NewImageNormalizer(config *NormalizationConfig, logger *logrus.Logger) *ImageNormalizer {
	if config == nil {
		config = DefaultNormalizationConfig()
	}
	if logger == nil {
		logger = logrus.New()
	}

	return &ImageNormalizer{
		config: config,
		logger: logger,
	}
}

// NormalizeImage 标准化图片
func (n *ImageNormalizer) NormalizeImage(img image.Image) (image.Image, error) {
	n.logger.Debug("开始图片标准化处理")

	// 1. 尺寸标准化
	normalizedImg := n.resizeImage(img)

	// 2. 色彩空间标准化 (确保为RGBA格式)
	rgbaImg := n.toRGBA(normalizedImg)

	n.logger.Debugf("图片标准化完成，尺寸: %dx%d", rgbaImg.Bounds().Dx(), rgbaImg.Bounds().Dy())

	return rgbaImg, nil
}

// NormalizeFile 标准化图片文件
func (n *ImageNormalizer) NormalizeFile(inputPath, outputPath string) error {
	n.logger.Debugf("开始标准化图片文件: %s -> %s", inputPath, outputPath)

	// 打开输入文件
	inputFile, err := os.Open(inputPath)
	if err != nil {
		return err
	}
	defer inputFile.Close()

	// 解码图片
	img, format, err := image.Decode(inputFile)
	if err != nil {
		return err
	}

	n.logger.Debugf("原始图片格式: %s, 尺寸: %dx%d", format, img.Bounds().Dx(), img.Bounds().Dy())

	// 标准化图片
	normalizedImg, err := n.NormalizeImage(img)
	if err != nil {
		return err
	}

	// 创建输出文件
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer outputFile.Close()

	// 编码并保存
	return n.encodeImage(normalizedImg, outputFile)
}

// resizeImage 调整图片尺寸
func (n *ImageNormalizer) resizeImage(img image.Image) image.Image {
	bounds := img.Bounds()
	originalWidth := bounds.Dx()
	originalHeight := bounds.Dy()

	// 检查是否需要调整尺寸
	if originalWidth <= n.config.MaxWidth && originalHeight <= n.config.MaxHeight {
		n.logger.Debug("图片尺寸符合要求，无需调整")
		return img
	}

	var newWidth, newHeight uint

	if n.config.KeepAspectRatio {
		// 保持宽高比计算新尺寸
		ratioX := float64(n.config.MaxWidth) / float64(originalWidth)
		ratioY := float64(n.config.MaxHeight) / float64(originalHeight)
		ratio := ratioX

		if ratioY < ratioX {
			ratio = ratioY
		}

		newWidth = uint(float64(originalWidth) * ratio)
		newHeight = uint(float64(originalHeight) * ratio)
	} else {
		// 直接使用最大尺寸
		newWidth = uint(n.config.MaxWidth)
		newHeight = uint(n.config.MaxHeight)
	}

	n.logger.Debugf("调整图片尺寸: %dx%d -> %dx%d", originalWidth, originalHeight, newWidth, newHeight)

	// 使用高质量的双三次插值
	return resize.Resize(newWidth, newHeight, img, resize.Lanczos3)
}

// toRGBA 转换为RGBA格式
func (n *ImageNormalizer) toRGBA(img image.Image) *image.RGBA {
	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			rgba.Set(x, y, img.At(x, y))
		}
	}

	return rgba
}

// encodeImage 编码图片
func (n *ImageNormalizer) encodeImage(img image.Image, writer io.Writer) error {
	switch strings.ToLower(n.config.OutputFormat) {
	case "png":
		return png.Encode(writer, img)
	case "jpg", "jpeg":
		return jpeg.Encode(writer, img, &jpeg.Options{Quality: n.config.JPEGQuality})
	default:
		// 默认使用PNG格式
		return png.Encode(writer, img)
	}
}

// GenerateStandardizedFilename 生成标准化的文件名
func (n *ImageNormalizer) GenerateStandardizedFilename(originalName string) string {
	ext := filepath.Ext(originalName)
	name := strings.TrimSuffix(originalName, ext)
	timestamp := time.Now().Format("20060102_150405")
	return name + "_normalized_" + timestamp + ext
}

// ValidateImage 验证图片是否符合标准
func (n *ImageNormalizer) ValidateImage(img image.Image) []string {
	var issues []string

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// 检查尺寸
	if width > n.config.MaxWidth {
		issues = append(issues, "图片宽度超过限制")
	}
	if height > n.config.MaxHeight {
		issues = append(issues, "图片高度超过限制")
	}

	// 检查色彩模型
	if img.ColorModel() != color.RGBAModel && img.ColorModel() != color.NRGBAModel {
		issues = append(issues, "图片色彩格式不是标准RGBA格式")
	}

	return issues
}

// GetImageInfo 获取图片信息
func (n *ImageNormalizer) GetImageInfo(img image.Image) map[string]interface{} {
	bounds := img.Bounds()

	return map[string]interface{}{
		"width":        bounds.Dx(),
		"height":       bounds.Dy(),
		"color_model":  fmt.Sprintf("%T", img.ColorModel()),
		"bounds":       bounds.String(),
	}
}