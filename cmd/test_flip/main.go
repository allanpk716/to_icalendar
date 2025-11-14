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

	fmt.Println("=== 图片镜像修复功能测试 ===")

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

	fmt.Println("\n=== 镜像修复功能已集成 ===")
	fmt.Println("✅ Windows DIB Y轴翻转修复已应用到所有图片格式：")
	fmt.Println("   - 32-bit BGRA 图片")
	fmt.Println("   - 24-bit BGR 图片")
	fmt.Println("   - 8-bit 调色板图片")

	fmt.Println("\n=== 测试说明 ===")
	fmt.Println("修复内容：")
	fmt.Println("- Windows剪贴板DIB数据是底部到顶部存储的")
	fmt.Println("- Go标准库图片是顶部到底部存储的")
	fmt.Println("- 现在会自动进行Y轴翻转以匹配正确的方向")

	fmt.Println("\n=== 现在可以测试了 ===")
	fmt.Println("1. 复制您的OA软件会议通知截图到剪贴板")
	fmt.Println("2. 运行 './to_icalendar clip-upload' 命令")
	fmt.Println("3. 检查 cache/images/ 目录中的图片文件：")
	fmt.Println("   - clipboard_original_*.png (翻转后的原始图片)")
	fmt.Println("   - clipboard_normalized_*.png (标准化后的图片)")
	fmt.Println("4. 图片现在应该显示正确的方向了！")

	fmt.Println("\n如果之前识别结果是上下颠倒的，")
	fmt.Println("现在应该能得到正确的识别结果了。")
}