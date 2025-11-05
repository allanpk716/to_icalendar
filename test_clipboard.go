package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/allanpk716/to_icalendar/internal/clipboard"
	"github.com/allanpk716/to_icalendar/internal/config"
	"github.com/allanpk716/to_icalendar/internal/dify"
	"github.com/allanpk716/to_icalendar/internal/models"
	"github.com/allanpk716/to_icalendar/internal/processors"
)

func main() {
	fmt.Println("=== to_icalendar 剪贴板功能测试 ===\n")

	// 检查是否有剪贴板内容
	manager, err := clipboard.NewManager()
	if err != nil {
		log.Fatalf("创建剪贴板管理器失败: %v", err)
	}

	hasContent, err := manager.HasContent()
	if err != nil {
		log.Fatalf("检查剪贴板内容失败: %v", err)
	}

	if !hasContent {
		fmt.Println("❌ 剪贴板为空，请先复制一些文字或截图，然后重新运行此测试")
		fmt.Println("\n建议测试内容:")
		fmt.Println("1. 文字测试: 复制 '明天下午3点开会讨论新产品发布'")
		fmt.Println("2. 截图测试: 截取包含时间和会议信息的界面")
		os.Exit(1)
	}

	// 读取剪贴板内容
	content, err := manager.Read()
	if err != nil {
		log.Fatalf("读取剪贴板内容失败: %v", err)
	}

	fmt.Printf("✅ 成功读取剪贴板内容\n")
	fmt.Printf("   内容类型: %s\n", content.Type)

	if content.Type == models.ContentTypeText {
		testTextProcessing(content.Text)
	} else if content.Type == models.ContentTypeImage {
		testImageProcessing(content.Image, content.FileName)
	} else {
		fmt.Printf("❌ 不支持的内容类型: %s\n", content.Type)
	}
}

func testTextProcessing(text string) {
	fmt.Printf("   文字内容: %s\n", truncateString(text, 100))

	// 加载配置
	configManager := config.NewConfigManager()
	serverConfig, err := configManager.LoadServerConfig("config/server.yaml")
	if err != nil {
		log.Printf("配置加载失败，使用基础文字处理: %v", err)
		basicTextProcessing(text)
		return
	}

	// 如果有Dify配置，使用Dify处理
	if serverConfig.Dify.APIKey != "" && serverConfig.Dify.APIKey != "YOUR_DIFY_API_KEY" {
		difyTextProcessing(text, serverConfig)
	} else {
		fmt.Println("   ⚠️  未配置Dify API，使用基础文字处理")
		basicTextProcessing(text)
	}
}

func basicTextProcessing(text string) {
	// 创建文字处理器
	processor, err := processors.NewTextProcessor(nil)
	if err != nil {
		log.Printf("文字处理器创建失败: %v", err)
		return
	}

	// 快速分析
	analysis := processor.QuickAnalyze(text)
	fmt.Printf("   ✅ 快速分析结果:\n")
	fmt.Printf("      包含日期: %v\n", analysis.HasDate)
	fmt.Printf("      包含时间: %v\n", analysis.HasTime)
	fmt.Printf("      是否紧急: %v\n", analysis.IsUrgent)
	fmt.Printf("      是否会议: %v\n", analysis.IsMeeting)
	fmt.Printf("      置信度: %.2f\n", analysis.Confidence)

	// 任务解析
	parser := processors.NewTaskParser()
	taskInfo, err := parser.ParseFromText(text)
	if err != nil {
		log.Printf("任务解析失败: %v", err)
		return
	}

	fmt.Printf("   ✅ 任务解析结果:\n")
	fmt.Printf("      标题: %s\n", taskInfo.Title)
	fmt.Printf("      日期: %s\n", taskInfo.Date)
	fmt.Printf("      时间: %s\n", taskInfo.Time)
	fmt.Printf("      优先级: %s\n", taskInfo.Priority)
	fmt.Printf("      列表: %s\n", taskInfo.List)
	fmt.Printf("      置信度: %.2f\n", taskInfo.Confidence)

	// 生成JSON
	if taskInfo.Confidence > 0.5 {
		generateJSONFromTaskInfo(taskInfo)
	} else {
		fmt.Printf("   ⚠️  置信度太低(%.2f)，不生成JSON文件\n", taskInfo.Confidence)
	}
}

func difyTextProcessing(text string, serverConfig *models.ServerConfig) {
	fmt.Println("   🤖 使用Dify AI进行智能分析...")

	// 创建Dify客户端和处理器
	client := dify.NewClient(serverConfig.Dify)
	processor := dify.NewProcessor(client, "test_user", dify.DefaultProcessingOptions())

	// 处理文字
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	response, err := processor.ProcessText(ctx, text)
	if err != nil {
		log.Printf("Dify处理失败，回退到基础处理: %v", err)
		basicTextProcessing(text)
		return
	}

	fmt.Printf("   ✅ Dify处理完成:\n")
	fmt.Printf("      处理成功: %v\n", response.Success)
	fmt.Printf("      处理时间: %v\n", response.ProcessingTime)

	if response.ParsedInfo != nil {
		fmt.Printf("      AI置信度: %.2f\n", response.ParsedInfo.Confidence)
		fmt.Printf("      解析标题: %s\n", response.ParsedInfo.Title)
		fmt.Printf("      解析日期: %s\n", response.ParsedInfo.Date)
		fmt.Printf("      解析时间: %s\n", response.ParsedInfo.Time)
	}

	// 生成JSON
	if response.Success && response.Reminder != nil {
		generateJSONFromReminder(response.Reminder)
	} else if response.ParsedInfo != nil && response.ParsedInfo.Confidence > 0.5 {
		generateJSONFromTaskInfo(response.ParsedInfo)
	} else {
		fmt.Printf("   ⚠️  处理结果不满足生成条件\n")
	}
}

func testImageProcessing(imageData []byte, fileName string) {
	fmt.Printf("   图片文件: %s\n", fileName)
	fmt.Printf("   图片大小: %d bytes\n", len(imageData))

	// 加载配置
	configManager := config.NewConfigManager()
	serverConfig, err := configManager.LoadServerConfig("config/server.yaml")
	if err != nil {
		log.Printf("配置加载失败，无法处理图片: %v", err)
		return
	}

	// 检查Dify配置
	if serverConfig.Dify.APIKey == "" || serverConfig.Dify.APIKey == "YOUR_DIFY_API_KEY" {
		fmt.Println("   ❌ 未配置Dify API，无法处理图片内容")
		fmt.Println("   💡 请在 config/server.yaml 中配置您的Dify API密钥")
		return
	}

	// 创建处理器
	client := dify.NewClient(serverConfig.Dify)
	difyProcessor := dify.NewProcessor(client, "test_user", dify.DefaultProcessingOptions())
	imageProcessor, err := processors.NewImageProcessor(difyProcessor)
	if err != nil {
		log.Printf("图片处理器创建失败: %v", err)
		return
	}

	// 处理图片
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	result, err := imageProcessor.ProcessClipboardImage(ctx, imageData)
	if err != nil {
		log.Printf("图片处理失败: %v", err)
		return
	}

	fmt.Printf("   ✅ 图片处理完成:\n")
	fmt.Printf("      处理成功: %v\n", result.Success)
	fmt.Printf("      处理时间: %v\n", result.ProcessingTime)

	if result.ErrorMessage != "" {
		fmt.Printf("      错误信息: %s\n", result.ErrorMessage)
	}

	if result.ParsedInfo != nil {
		fmt.Printf("      AI置信度: %.2f\n", result.ParsedInfo.Confidence)
		fmt.Printf("      识别文字: %s\n", truncateString(result.ParsedInfo.OriginalText, 200))
		fmt.Printf("      解析标题: %s\n", result.ParsedInfo.Title)
		fmt.Printf("      解析日期: %s\n", result.ParsedInfo.Date)
		fmt.Printf("      解析时间: %s\n", result.ParsedInfo.Time)
	}

	// 生成JSON
	if result.Success && result.Reminder != nil {
		generateJSONFromReminder(result.Reminder)
	} else if result.ParsedInfo != nil && result.ParsedInfo.Confidence > 0.5 {
		generateJSONFromTaskInfo(result.ParsedInfo)
	} else {
		fmt.Printf("   ⚠️  图片处理结果不满足生成条件\n")
	}

	// 清理临时文件
	defer imageProcessor.Cleanup()
}

func generateJSONFromReminder(reminder *models.Reminder) {
	generator, err := processors.NewJSONGenerator("config/drafts")
	if err != nil {
		log.Printf("JSON生成器创建失败: %v", err)
		return
	}

	filePath, err := generator.GenerateFromReminder(reminder)
	if err != nil {
		log.Printf("JSON生成失败: %v", err)
		return
	}

	fmt.Printf("   ✅ JSON提醒文件已生成: %s\n", filePath)
}

func generateJSONFromTaskInfo(taskInfo *models.ParsedTaskInfo) {
	generator, err := processors.NewJSONGenerator("config/drafts")
	if err != nil {
		log.Printf("JSON生成器创建失败: %v", err)
		return
	}

	filePath, err := generator.GenerateFromParsedInfo(taskInfo)
	if err != nil {
		log.Printf("JSON生成失败: %v", err)
		return
	}

	fmt.Printf("   ✅ JSON草稿文件已生成: %s\n", filePath)
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}