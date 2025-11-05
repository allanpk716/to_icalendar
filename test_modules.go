package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/allanpk716/to_icalendar/internal/clipboard"
	"github.com/allanpk716/to_icalendar/internal/config"
	"github.com/allanpk716/to_icalendar/internal/dify"
	"github.com/allanpk716/to_icalendar/internal/models"
	"github.com/allanpk716/to_icalendar/internal/processors"
)

func main() {
	fmt.Println("=== to_icalendar 模块功能测试 ===\n")

	// 测试配置加载
	fmt.Println("1. 测试配置加载功能...")
	testConfigLoading()

	// 测试剪贴板读取
	fmt.Println("\n2. 测试剪贴板读取功能...")
	testClipboardReading()

	// 测试Dify API连接（如果配置了API密钥）
	fmt.Println("\n3. 测试Dify API连接...")
	testDifyConnection()

	// 测试处理功能
	fmt.Println("\n4. 测试内容处理功能...")
	testProcessing()

	// 测试JSON生成
	fmt.Println("\n5. 测试JSON生成功能...")
	testJSONGeneration()

	fmt.Println("\n=== 测试完成 ===")
}

func testConfigLoading() {
	// 尝试加载配置
	configManager := config.NewConfigManager()
	serverConfig, err := configManager.LoadServerConfig("config/server.yaml")
	if err != nil {
		log.Printf("❌ 配置加载失败: %v", err)
		return
	}

	fmt.Println("✅ 配置加载成功")
	fmt.Printf("   Microsoft Todo配置: 已配置\n")
	fmt.Printf("   Dify API端点: %s\n", serverConfig.Dify.APIEndpoint)
	fmt.Printf("   Dify模型: %s\n", serverConfig.Dify.Model)

	if serverConfig.Dify.APIKey == "" || serverConfig.Dify.APIKey == "YOUR_DIFY_API_KEY" {
		fmt.Println("   ⚠️  Dify API密钥未配置，将跳过API测试")
	}
}

func testClipboardReading() {
	// 创建剪贴板管理器
	manager, err := clipboard.NewManager()
	if err != nil {
		log.Printf("❌ 剪贴板管理器创建失败: %v", err)
		return
	}

	// 检查剪贴板是否有内容
	hasContent, err := manager.HasContent()
	if err != nil {
		log.Printf("❌ 检查剪贴板内容失败: %v", err)
		return
	}

	if !hasContent {
		fmt.Println("ℹ️  剪贴板当前为空，请复制一些文字或截图后再测试")
		return
	}

	// 获取内容类型
	contentType, err := manager.GetContentType()
	if err != nil {
		log.Printf("❌ 获取内容类型失败: %v", err)
		return
	}

	fmt.Printf("✅ 剪贴板管理器创建成功\n")
	fmt.Printf("   剪贴板有内容: %v\n", hasContent)
	fmt.Printf("   内容类型: %s\n", contentType)

	// 尝试读取内容
	content, err := manager.Read()
	if err != nil {
		log.Printf("❌ 读取剪贴板内容失败: %v", err)
		return
	}

	fmt.Printf("   内容读取成功，类型: %s\n", content.Type)
	if content.Type == models.ContentTypeText {
		fmt.Printf("   文字内容预览: %s\n", truncateString(content.Text, 50))
	} else if content.Type == models.ContentTypeImage {
		fmt.Printf("   图片大小: %d bytes\n", len(content.Image))
	}
}

func testDifyConnection() {
	// 加载配置
	configManager := config.NewConfigManager()
	serverConfig, err := configManager.LoadServerConfig("config/server.yaml")
	if err != nil {
		log.Printf("❌ 配置加载失败: %v", err)
		return
	}

	// 检查API密钥
	if serverConfig.Dify.APIKey == "" || serverConfig.Dify.APIKey == "YOUR_DIFY_API_KEY" {
		fmt.Println("⚠️  Dify API密钥未配置，跳过API连接测试")
		return
	}

	// 创建Dify客户端
	client := dify.NewClient(serverConfig.Dify)

	// 验证配置
	if err := client.ValidateConfig(); err != nil {
		log.Printf("❌ Dify配置验证失败: %v", err)
		return
	}

	fmt.Println("✅ Dify客户端配置验证通过")

	// 创建处理器
	processor := dify.NewProcessor(client, "test_user", dify.DefaultProcessingOptions())

	// 测试文字处理
	testText := "明天下午2点开会讨论项目进度"
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	response, err := processor.ProcessText(ctx, testText)
	if err != nil {
		log.Printf("❌ Dify文字处理测试失败: %v", err)
		return
	}

	fmt.Println("✅ Dify文字处理测试成功")
	fmt.Printf("   处理结果: %v\n", response.Success)
	if response.ParsedInfo != nil {
		fmt.Printf("   解析置信度: %.2f\n", response.ParsedInfo.Confidence)
	}
}

func testProcessing() {
	fmt.Println("   测试文字处理器的快速分析功能...")

	// 创建文字处理器（不需要Dify处理器进行快速分析测试）
	processor, err := processors.NewTextProcessor(nil)
	if err != nil {
		log.Printf("❌ 文字处理器创建失败: %v", err)
		return
	}

	// 测试快速分析
	testText := "明天下午2点开重要的项目评审会议，请准时参加"
	analysis := processor.QuickAnalyze(testText)

	fmt.Printf("   ✅ 快速分析完成\n")
	fmt.Printf("   包含日期: %v\n", analysis.HasDate)
	fmt.Printf("   包含时间: %v\n", analysis.HasTime)
	fmt.Printf("   是否紧急: %v\n", analysis.IsUrgent)
	fmt.Printf("   是否会议: %v\n", analysis.IsMeeting)
	fmt.Printf("   置信度: %.2f\n", analysis.Confidence)

	// 测试任务解析
	parser := processors.NewTaskParser()
	taskInfo, err := parser.ParseFromText(testText)
	if err != nil {
		log.Printf("❌ 任务解析测试失败: %v", err)
		return
	}

	fmt.Printf("   ✅ 任务解析完成\n")
	fmt.Printf("   解析标题: %s\n", taskInfo.Title)
	fmt.Printf("   解析日期: %s\n", taskInfo.Date)
	fmt.Printf("   解析时间: %s\n", taskInfo.Time)
	fmt.Printf("   解析置信度: %.2f\n", taskInfo.Confidence)
}

func testJSONGeneration() {
	// 创建测试用的提醒事项
	reminder := &models.Reminder{
		Title:        "项目评审会议",
		Description:  "讨论Q4项目进度和下一步计划",
		Date:         "2025-11-06",
		Time:         "14:00",
		RemindBefore: "15m",
		Priority:     models.PriorityHigh,
		List:         "会议",
	}

	// 创建JSON生成器
	outputDir := filepath.Join("config", "drafts")
	generator, err := processors.NewJSONGenerator(outputDir)
	if err != nil {
		log.Printf("❌ JSON生成器创建失败: %v", err)
		return
	}

	// 验证提醒事项
	if err := generator.ValidateReminder(reminder); err != nil {
		log.Printf("❌ 提醒事项验证失败: %v", err)
		return
	}

	// 生成JSON文件
	filePath, err := generator.GenerateFromReminder(reminder)
	if err != nil {
		log.Printf("❌ JSON文件生成失败: %v", err)
		return
	}

	fmt.Printf("✅ JSON生成器测试成功\n")
	fmt.Printf("   输出目录: %s\n", outputDir)
	fmt.Printf("   生成文件: %s\n", filePath)

	// 检查文件是否存在
	if _, err := os.Stat(filePath); err == nil {
		fmt.Printf("   文件验证: 存在且可访问\n")
	} else {
		fmt.Printf("   ❌ 文件验证失败: %v\n", err)
	}
}

// truncateString 截断字符串到指定长度
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}