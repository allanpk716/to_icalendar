package image

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/allanpk716/to_icalendar/internal/cache"
	"github.com/sirupsen/logrus"
)

// ImageProcessingConfig 图片处理配置
type ImageProcessingConfig struct {
	// 标准化配置
	Normalization *NormalizationConfig `json:"normalization"`
	// 是否启用标准化
	EnableNormalization bool `json:"enable_normalization"`
	// 是否保存中间结果用于调试
	DebugMode bool `json:"debug_mode"`
	// 调试文件保存目录
	DebugOutputDir string `json:"debug_output_dir"`
	// 是否启用图片缓存
	EnableCache bool `json:"enable_cache"`
	// 图片缓存目录
	CacheDir string `json:"cache_dir"`
	// 最大缓存文件数量
	MaxCacheFiles int `json:"max_cache_files"`
}

// DefaultImageProcessingConfig 默认图片处理配置
func DefaultImageProcessingConfig() *ImageProcessingConfig {
	return &ImageProcessingConfig{
		Normalization:       DefaultNormalizationConfig(),
		EnableNormalization: true,
		DebugMode:           false,
		DebugOutputDir:      "debug/images",
		EnableCache:         true,
		CacheDir:            "", // 将由统一缓存管理器设置
		MaxCacheFiles:       50,
	}
}

// LoadFromFile 从文件加载配置
func (c *ImageProcessingConfig) LoadFromFile(configPath string) error {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// 配置文件不存在，使用默认配置并保存
		c = DefaultImageProcessingConfig()
		return c.SaveToFile(configPath)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	if err := json.Unmarshal(data, c); err != nil {
		return fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 验证配置
	if err := c.Validate(); err != nil {
		return fmt.Errorf("配置验证失败: %w", err)
	}

	return nil
}

// SaveToFile 保存配置到文件
func (c *ImageProcessingConfig) SaveToFile(configPath string) error {
	// 确保目录存在
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	return nil
}

// Validate 验证配置
func (c *ImageProcessingConfig) Validate() error {
	if c.Normalization == nil {
		return fmt.Errorf("标准化配置不能为空")
	}

	if c.Normalization.MaxWidth <= 0 || c.Normalization.MaxHeight <= 0 {
		return fmt.Errorf("图片最大尺寸必须大于0")
	}

	if c.Normalization.JPEGQuality < 1 || c.Normalization.JPEGQuality > 100 {
		return fmt.Errorf("JPEG质量必须在1-100之间")
	}

	if c.Normalization.MaxFileSize <= 0 {
		return fmt.Errorf("文件大小限制必须大于0")
	}

	validFormats := map[string]bool{"png": true, "jpg": true, "jpeg": true}
	if !validFormats[c.Normalization.OutputFormat] {
		return fmt.Errorf("不支持的输出格式: %s", c.Normalization.OutputFormat)
	}

	return nil
}

// GetConfigPath 获取配置文件路径
func GetConfigPath(configDir string) string {
	return filepath.Join(configDir, "image_processing.json")
}

// LoadOrCreateConfig 加载或创建配置
func LoadOrCreateConfig(configDir string, logger *logrus.Logger) (*ImageProcessingConfig, error) {
	configPath := GetConfigPath(configDir)
	config := DefaultImageProcessingConfig()

	// 尝试加载现有配置
	if err := config.LoadFromFile(configPath); err != nil {
		logger.Warnf("加载图片处理配置失败，使用默认配置: %v", err)

		// 使用默认配置
		config = DefaultImageProcessingConfig()

		// 尝试保存默认配置
		if saveErr := config.SaveToFile(configPath); saveErr != nil {
			logger.Warnf("保存默认配置失败: %v", saveErr)
		} else {
			logger.Infof("已创建默认图片处理配置: %s", configPath)
		}
	}

	logger.Infof("图片标准化功能: %s", map[bool]string{true: "启用", false: "禁用"}[config.EnableNormalization])

	if config.EnableNormalization {
		logger.Infof("图片尺寸限制: %dx%d", config.Normalization.MaxWidth, config.Normalization.MaxHeight)
		logger.Infof("输出格式: %s, 最大文件大小: %d MB",
			config.Normalization.OutputFormat, config.Normalization.MaxFileSize/(1024*1024))
	}

	return config, nil
}

// EnsureDebugDir 确保调试目录存在
func (c *ImageProcessingConfig) EnsureDebugDir() error {
	if !c.DebugMode {
		return nil
	}

	return os.MkdirAll(c.DebugOutputDir, 0755)
}

// ConfigManager 配置管理器
type ConfigManager struct {
	config            *ImageProcessingConfig
	logger            *logrus.Logger
	configDir         string
	unifiedCacheMgr   *cache.UnifiedCacheManager // 统一缓存管理器
}

// NewConfigManager 创建配置管理器
func NewConfigManager(configDir string, logger *logrus.Logger) *ConfigManager {
	return &ConfigManager{
		configDir: configDir,
		logger:    logger,
	}
}

// NewConfigManagerWithUnifiedCache 创建带有统一缓存管理器的配置管理器
func NewConfigManagerWithUnifiedCache(configDir string, logger *logrus.Logger) (*ConfigManager, error) {
	// 创建统一缓存管理器
	unifiedCacheMgr, err := cache.NewUnifiedCacheManager("", nil) // 使用默认缓存目录
	if err != nil {
		return nil, fmt.Errorf("创建统一缓存管理器失败: %w", err)
	}

	cm := &ConfigManager{
		configDir:       configDir,
		logger:          logger,
		unifiedCacheMgr: unifiedCacheMgr,
	}

	return cm, nil
}

// LoadConfig 加载配置
func (cm *ConfigManager) LoadConfig() error {
	config, err := LoadOrCreateConfig(cm.configDir, cm.logger)
	if err != nil {
		return err
	}

	cm.config = config

	// 如果有统一缓存管理器，更新缓存目录配置
	if cm.unifiedCacheMgr != nil {
		cm.config.CacheDir = cm.unifiedCacheMgr.GetCacheDir(cache.CacheTypeImages)
		cm.logger.Infof("使用统一缓存管理器，图片缓存目录: %s", cm.config.CacheDir)
	}

	// 确保调试目录存在
	if err := cm.config.EnsureDebugDir(); err != nil {
		cm.logger.Warnf("创建调试目录失败: %v", err)
	}

	return nil
}

// GetConfig 获取配置
func (cm *ConfigManager) GetConfig() *ImageProcessingConfig {
	return cm.config
}

// UpdateConfig 更新配置
func (cm *ConfigManager) UpdateConfig(newConfig *ImageProcessingConfig) error {
	if err := newConfig.Validate(); err != nil {
		return fmt.Errorf("配置验证失败: %w", err)
	}

	cm.config = newConfig

	configPath := GetConfigPath(cm.configDir)
	return cm.config.SaveToFile(configPath)
}

// IsNormalizationEnabled 检查是否启用了标准化
func (cm *ConfigManager) IsNormalizationEnabled() bool {
	return cm.config != nil && cm.config.EnableNormalization
}

// GetNormalizer 获取图片标准化器
func (cm *ConfigManager) GetNormalizer() *ImageNormalizer {
	if !cm.IsNormalizationEnabled() {
		return nil
	}

	return NewImageNormalizer(cm.config.Normalization, cm.logger)
}

// SaveDebugImage 保存调试图片
func (cm *ConfigManager) SaveDebugImage(imgData []byte, filename string) error {
	if !cm.config.DebugMode {
		return nil
	}

	if err := cm.config.EnsureDebugDir(); err != nil {
		return err
	}

	filePath := filepath.Join(cm.config.DebugOutputDir, filename)
	return os.WriteFile(filePath, imgData, 0644)
}

// EnsureCacheDir 确保缓存目录存在
func (c *ImageProcessingConfig) EnsureCacheDir() error {
	if !c.EnableCache {
		return nil
	}

	return os.MkdirAll(c.CacheDir, 0755)
}

// SaveCacheImage 保存缓存图片
func (cm *ConfigManager) SaveCacheImage(imgData []byte, filename string) (string, error) {
	if !cm.config.EnableCache {
		return "", nil
	}

	if err := cm.config.EnsureCacheDir(); err != nil {
		return "", err
	}

	// 清理过多的缓存文件
	cm.cleanupCache()

	filePath := filepath.Join(cm.config.CacheDir, filename)
	err := os.WriteFile(filePath, imgData, 0644)
	if err != nil {
		return "", err
	}

	cm.logger.Debugf("图片已缓存到: %s", filePath)
	return filePath, nil
}

// cleanupCache 清理过多的缓存文件
func (cm *ConfigManager) cleanupCache() {
	if cm.config.MaxCacheFiles <= 0 {
		return
	}

	files, err := os.ReadDir(cm.config.CacheDir)
	if err != nil {
		return
	}

	if len(files) <= cm.config.MaxCacheFiles {
		return
	}

	// 按修改时间排序，删除最旧的文件
	fileInfos := make([]os.FileInfo, 0, len(files))
	for _, file := range files {
		if info, err := file.Info(); err == nil {
			fileInfos = append(fileInfos, info)
		}
	}

	// 简单的按时间排序（最早修改的在前）
	for i := 0; i < len(fileInfos); i++ {
		for j := i + 1; j < len(fileInfos); j++ {
			if fileInfos[i].ModTime().After(fileInfos[j].ModTime()) {
				fileInfos[i], fileInfos[j] = fileInfos[j], fileInfos[i]
			}
		}
	}

	// 删除多余的文件
	filesToDelete := len(fileInfos) - cm.config.MaxCacheFiles
	for i := 0; i < filesToDelete; i++ {
		filePath := filepath.Join(cm.config.CacheDir, fileInfos[i].Name())
		if err := os.Remove(filePath); err != nil {
			cm.logger.Warnf("删除缓存文件失败: %s, 错误: %v", filePath, err)
		}
	}

	if filesToDelete > 0 {
		cm.logger.Debugf("已清理 %d 个旧缓存文件", filesToDelete)
	}
}

// GetCacheDir 获取缓存目录
func (cm *ConfigManager) GetCacheDir() string {
	if cm.unifiedCacheMgr != nil {
		return cm.unifiedCacheMgr.GetCacheDir(cache.CacheTypeImages)
	}
	return cm.config.CacheDir
}

// GetUnifiedCacheManager 获取统一缓存管理器
func (cm *ConfigManager) GetUnifiedCacheManager() *cache.UnifiedCacheManager {
	return cm.unifiedCacheMgr
}

// SetUnifiedCacheManager 设置统一缓存管理器
func (cm *ConfigManager) SetUnifiedCacheManager(ucm *cache.UnifiedCacheManager) {
	cm.unifiedCacheMgr = ucm
	if cm.config != nil && ucm != nil {
		cm.config.CacheDir = ucm.GetCacheDir(cache.CacheTypeImages)
		cm.logger.Infof("更新图片缓存目录: %s", cm.config.CacheDir)
	}
}

// IsCacheEnabled 检查是否启用了缓存
func (cm *ConfigManager) IsCacheEnabled() bool {
	return cm.config.EnableCache
}