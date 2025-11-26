package main

import (
	"fmt"

	"github.com/allanpk716/to_icalendar/pkg/commands"
)

func main() {
	fmt.Println("=== to_icalendar Go 包调用示例 ===")

	// 创建命令执行器
	executor := commands.NewCommandExecutor()

	// 示例1: 初始化配置
	fmt.Println("\n1. 初始化配置...")
	configResult, err := executor.InitConfig()
	if err != nil {
		fmt.Printf("❌ 初始化失败: %v\n", err)
		return
	}

	if configResult.Success {
		fmt.Printf("✅ 配置初始化成功\n")
		fmt.Printf("   配置目录: %s\n", configResult.ConfigDir)
		fmt.Printf("   服务器配置: %s\n", configResult.ServerConfig)
		fmt.Printf("   提醒模板: %s\n", configResult.ReminderTemplate)
	} else {
		fmt.Printf("❌ 配置初始化失败: %s\n", configResult.Message)
	}

	// 示例2: 清理缓存
	fmt.Println("\n2. 清理缓存...")
	cleanOptions := &commands.CleanupOptions{
		All:     true,
		DryRun:  true, // 预览模式
		Force:   false,
	}

	cleanResult, err := executor.CleanCache(cleanOptions)
	if err != nil {
		fmt.Printf("❌ 清理失败: %v\n", err)
		return
	}

	if cleanResult.Success {
		fmt.Printf("✅ 缓存清理预览完成\n")
		fmt.Printf("   将删除文件数: %d\n", cleanResult.TotalFiles)
		fmt.Printf("   将释放空间: %s\n", formatBytes(cleanResult.TotalSize))
	} else {
		fmt.Printf("❌ 缓存清理失败: %s\n", cleanResult.Message)
	}

	// 示例3: 处理剪贴板（这个可能需要剪贴板内容）
	fmt.Println("\n3. 处理剪贴板...")
	// 注意：在实际环境中，这需要剪贴板中有内容
	fmt.Printf("⚠️  剪贴板处理需要剪贴板中有内容\n")

	fmt.Println("\n=== 示例完成 ===")
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