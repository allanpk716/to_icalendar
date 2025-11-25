package tray

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// IconLoader 图标加载器
type IconLoader struct {
	icons map[string]*TrayIcon
}

// NewIconLoader 创建新的图标加载器
func NewIconLoader() *IconLoader {
	return &IconLoader{
		icons: make(map[string]*TrayIcon),
	}
}

// LoadIcon 加载图标文件
func (il *IconLoader) LoadIcon(filePath string, size int) (*TrayIcon, error) {
	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, NewTrayError(ErrCodeIconLoad, fmt.Sprintf("图标文件不存在: %s", filePath), err)
	}

	// 检查文件扩展名
	ext := strings.ToLower(filepath.Ext(filePath))
	if !il.isSupportedFormat(ext) {
		return nil, NewTrayError(ErrCodeIconLoad, fmt.Sprintf("不支持的图标格式: %s", ext), nil)
	}

	// 创建图标对象
	icon := NewTrayIcon(filePath, size)

	// 设置关联的应用程序ID
	icon.SetAppID("to_icalendar_tray")

	// 验证图标配置
	if err := icon.Validate(); err != nil {
		return nil, NewTrayError(ErrCodeIconLoad, "图标验证失败", err)
	}

	// 缓存图标
	il.icons[filePath] = icon

	LogInfo("图标加载成功: %s (%dx%d)", filePath, size, size)
	return icon, nil
}

// LoadIconSet 加载图标集合（16x16, 32x32, 48x48）
func (il *IconLoader) LoadIconSet(basePath string) (map[int]*TrayIcon, error) {
	iconSet := make(map[int]*TrayIcon)
	sizes := []int{16, 32, 48}

	for _, size := range sizes {
		filePath := fmt.Sprintf("%s-%d.png", basePath, size)
		icon, err := il.LoadIcon(filePath, size)
		if err != nil {
			LogError("加载图标失败: %v", err)
			continue
		}
		iconSet[size] = icon
	}

	if len(iconSet) == 0 {
		return nil, NewTrayError(ErrCodeIconLoad, "没有成功加载任何图标", nil)
	}

	LogInfo("图标集合加载成功: %d 个图标", len(iconSet))
	return iconSet, nil
}

// GetIcon 从缓存中获取图标
func (il *IconLoader) GetIcon(filePath string) (*TrayIcon, bool) {
	icon, exists := il.icons[filePath]
	return icon, exists
}

// GetIconBySize 按尺寸获取图标
func (il *IconLoader) GetIconBySize(size int) (*TrayIcon, bool) {
	for _, icon := range il.icons {
		if icon.Size == size {
			return icon, true
		}
	}
	return nil, false
}

// GetAllIcons 获取所有已加载的图标
func (il *IconLoader) GetAllIcons() map[string]*TrayIcon {
	result := make(map[string]*TrayIcon)
	for k, v := range il.icons {
		result[k] = v
	}
	return result
}

// RemoveIcon 从缓存中移除图标
func (il *IconLoader) RemoveIcon(filePath string) bool {
	if _, exists := il.icons[filePath]; exists {
		delete(il.icons, filePath)
		LogInfo("图标已从缓存中移除: %s", filePath)
		return true
	}
	return false
}

// ClearCache 清空图标缓存
func (il *IconLoader) ClearCache() {
	il.icons = make(map[string]*TrayIcon)
	LogInfo("图标缓存已清空")
}

// GetCacheSize 获取缓存大小
func (il *IconLoader) GetCacheSize() int {
	return len(il.icons)
}

// ValidateIconFile 验证图标文件
func (il *IconLoader) ValidateIconFile(filePath string) error {
	// 检查文件是否存在
	info, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return fmt.Errorf("文件不存在: %s", filePath)
	}
	if err != nil {
		return fmt.Errorf("无法访问文件: %s", err)
	}

	// 检查是否为文件
	if info.IsDir() {
		return fmt.Errorf("路径指向目录而非文件: %s", filePath)
	}

	// 检查文件扩展名
	ext := strings.ToLower(filepath.Ext(filePath))
	if !il.isSupportedFormat(ext) {
		return fmt.Errorf("不支持的文件格式: %s", ext)
	}

	// 检查文件大小（限制为1MB）
	if info.Size() > 1024*1024 {
		return fmt.Errorf("文件过大: %d bytes (最大1MB)", info.Size())
	}

	return nil
}

// isSupportedFormat 检查是否为支持的格式
func (il *IconLoader) isSupportedFormat(ext string) bool {
	supportedFormats := []string{".png", ".ico", ".jpg", ".jpeg", ".bmp"}
	for _, format := range supportedFormats {
		if ext == format {
			return true
		}
	}
	return false
}

// GetSupportedFormats 获取支持的图标格式列表
func GetSupportedFormats() []string {
	return []string{".png", ".ico", ".jpg", ".jpeg", ".bmp"}
}

// ConvertToPNG 将图标转换为PNG格式（占位符实现）
func ConvertToPNG(inputPath, outputPath string) error {
	// TODO: 实现图标格式转换功能
	// 这里可以使用第三方库如 github.com/disintegration/imaging
	LogInfo("图标格式转换功能尚未实现: %s -> %s", inputPath, outputPath)
	return nil
}

// CreateIconSizes 为指定图标创建不同尺寸的版本
func CreateIconSizes(sourcePath string, targetDir string) error {
	// TODO: 实现图标尺寸调整功能
	// 这里可以使用第三方库如 github.com/nfnt/resize
	LogInfo("图标尺寸调整功能尚未实现: %s -> %s", sourcePath, targetDir)
	return nil
}