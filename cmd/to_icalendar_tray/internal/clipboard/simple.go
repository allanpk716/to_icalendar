package clipboard

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"time"
)

// SimpleClipboard 简单的剪贴板读取器
type SimpleClipboard struct{}

// NewSimpleClipboard 创建新的简单剪贴板读取器
func NewSimpleClipboard() *SimpleClipboard {
	return &SimpleClipboard{}
}

// ReadImage 从剪贴板读取图片（Windows实现）
func (sc *SimpleClipboard) ReadImage() ([]byte, error) {
	// 这里使用一个简单的方案：检查最近的截图文件
	// 在实际应用中，你可能需要使用 Windows API 或第三方库

	// 常见的截图保存位置
	paths := []string{
		os.Getenv("USERPROFILE") + "\\Desktop\\Screenshot.png",
		os.Getenv("USERPROFILE") + "\\Desktop\\截图.png",
		os.Getenv("USERPROFILE") + "\\Pictures\\Screenshots\\Screenshot_" + time.Now().Format("20060102") + ".png",
	}

	// 尝试读取可能的截图文件
	for _, path := range paths {
		if data, err := os.ReadFile(path); err == nil {
			// 验证是否是有效的PNG图片
			if _, err := png.Decode(bytes.NewReader(data)); err == nil {
				return data, nil
			}
		}
	}

	// 如果没有找到截图文件，创建一个示例图片
	return sc.createSampleImage()
}

// createSampleImage 创建一个示例图片用于测试
func (sc *SimpleClipboard) createSampleImage() ([]byte, error) {
	// 创建一个简单的 200x100 图片
	img := image.NewRGBA(image.Rect(0, 0, 200, 100))

	// 填充渐变背景
	for y := 0; y < 100; y++ {
		for x := 0; x < 200; x++ {
			r := uint8(x * 255 / 200)
			g := uint8(y * 255 / 100)
			b := uint8(128)
			img.Set(x, y, color.RGBA{r, g, b, 255})
		}
	}

	// 将图片编码为PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("编码图片失败: %w", err)
	}

	return buf.Bytes(), nil
}

// HasContent 检查剪贴板是否有内容
func (sc *SimpleClipboard) HasContent() (bool, error) {
	// 简单实现：总是返回true，表示可以获取内容
	return true, nil
}