package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/allanpk716/to_icalendar/internal/image"
	"github.com/allanpk716/to_icalendar/internal/processors"
	"github.com/allanpk716/to_icalendar/internal/dify"
	"github.com/sirupsen/logrus"
)

func main() {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	fmt.Println("=== 图片缓存功能测试 ===")

	// 测试配置管理器
	fmt.Println("\n1. 测试配置管理器...")
	configManager := image.NewConfigManager(".", logger)
	err := configManager.LoadConfig()
	if err != nil {
		log.Printf("加载配置失败: %v", err)
		return
	}

	fmt.Printf("缓存功能状态: %s\n", map[bool]string{true: "启用", false: "禁用"}[configManager.IsCacheEnabled()])
	fmt.Printf("缓存目录: %s\n", configManager.GetCacheDir())

	// 创建测试图片数据
	fmt.Println("\n2. 创建测试图片数据...")
	testImageData := []byte("fake-image-data-for-testing") // 简化的测试数据

	// 测试缓存保存
	fmt.Println("\n3. 测试缓存保存...")
	timestamp := time.Now().Format("20060102_150405_000000")
	testFilename := fmt.Sprintf("test_image_%s.png", timestamp)

	cachePath, err := configManager.SaveCacheImage(testImageData, testFilename)
	if err != nil {
		log.Printf("保存缓存图片失败: %v", err)
		return
	}

	fmt.Printf("✅ 测试图片已缓存到: %s\n", cachePath)

	// 验证文件是否存在
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		log.Printf("❌ 缓存文件不存在: %s", cachePath)
		return
	} else {
		fmt.Printf("✅ 缓存文件验证成功\n")
	}

	// 读取并显示文件信息
	fileInfo, err := os.Stat(cachePath)
	if err == nil {
		fmt.Printf("   文件大小: %d bytes\n", fileInfo.Size())
		fmt.Printf("   修改时间: %s\n", fileInfo.ModTime().Format("2006-01-02 15:04:05"))
	}

	// 测试图片处理器
	fmt.Println("\n4. 测试图片处理器...")
	// 创建一个简单的 Dify 处理器（模拟）
	difyProcessor := &dify.Processor{} // 简化，仅用于测试

	imageProcessor, err := processors.NewImageProcessor(difyProcessor)
	if err != nil {
		log.Printf("创建图片处理器失败: %v", err)
		return
	}

	fmt.Printf("图片处理器缓存状态: %s\n", map[bool]string{true: "启用", false: "禁用"}[imageProcessor.IsCacheEnabled()])
	fmt.Printf("图片处理器缓存目录: %s\n", imageProcessor.GetCacheDir())

	// 列出缓存目录中的所有文件
	fmt.Println("\n5. 列出缓存文件...")
	cacheDir := configManager.GetCacheDir()
	if cacheDir != "" {
		files, err := os.ReadDir(cacheDir)
		if err != nil {
			log.Printf("读取缓存目录失败: %v", err)
		} else {
			fmt.Printf("缓存目录 '%s' 中的文件:\n", cacheDir)
			for i, file := range files {
				info, _ := file.Info()
				fmt.Printf("   %d. %s (大小: %d bytes, 修改时间: %s)\n",
					i+1, file.Name(), info.Size(),
					info.ModTime().Format("2006-01-02 15:04:05"))
			}
		}
	}

	fmt.Println("\n=== 缓存功能测试完成 ===")
	fmt.Println("✅ 配置管理器正常工作")
	fmt.Println("✅ 图片缓存保存成功")
	fmt.Println("✅ 图片处理器集成成功")
	fmt.Println("✅ 文件系统验证通过")

	fmt.Println("\n现在您可以:")
	fmt.Println("1. 复制任何图片到剪贴板")
	fmt.Println("2. 运行 './to_icalendar clip-upload' 命令")
	fmt.Println("3. 检查缓存目录中的图片文件")
	fmt.Printf("4. 缓存目录位置: %s\n", cacheDir)
}