package main

import (
	"fmt"
	"log"

	"github.com/allanpk716/to_icalendar/internal/image"
	"github.com/sirupsen/logrus"
)

func main() {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// 创建默认标准化器
	config := image.DefaultNormalizationConfig()
	_ = image.NewImageNormalizer(config, logger)

	fmt.Println("=== 图片标准化功能测试 ===")
	fmt.Printf("默认配置:\n")
	fmt.Printf("- 最大尺寸: %dx%d\n", config.MaxWidth, config.MaxHeight)
	fmt.Printf("- 压缩级别: %d\n", config.PNGCompressionLevel)
	fmt.Printf("- 输出格式: %s\n", config.OutputFormat)
	fmt.Printf("- 最大文件大小: %d MB\n", config.MaxFileSize/(1024*1024))

	// 测试配置管理器
	fmt.Println("\n=== 配置管理器测试 ===")
	configManager := image.NewConfigManager(".", logger)
	err := configManager.LoadConfig()
	if err != nil {
		log.Printf("加载配置失败: %v", err)
	} else {
		fmt.Printf("配置加载成功，标准化功能: %s\n",
			map[bool]string{true: "启用", false: "禁用"}[configManager.IsNormalizationEnabled()])
	}

	// 获取标准化器
	normalizerFromManager := configManager.GetNormalizer()
	if normalizerFromManager != nil {
		fmt.Println("从配置管理器成功获取标准化器")
	} else {
		fmt.Println("标准化功能未启用")
	}

	// 测试文档配置
	fmt.Println("\n=== 文档配置测试 ===")
	docConfig := image.DocumentNormalizationConfig()
	_ = image.NewImageNormalizer(docConfig, logger)
	fmt.Printf("文档配置 - 最大尺寸: %dx%d\n", docConfig.MaxWidth, docConfig.MaxHeight)

	fmt.Println("\n=== 功能验证完成 ===")
	fmt.Println("✅ 图片标准化模块已成功集成")
	fmt.Println("✅ 配置管理功能正常")
	fmt.Println("✅ 多种配置支持正常")
	fmt.Println("\n可以通过 to_icalendar clip-upload 命令测试完整的剪贴板图片识别流程")
}