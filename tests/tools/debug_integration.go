// +build tools

package main

import (
	"fmt"
	"log"
	"time"

	"github.com/allanpk716/to_icalendar/internal/clipboard"
	"github.com/allanpk716/to_icalendar/internal/config"
	"github.com/allanpk716/to_icalendar/internal/errors"
	"github.com/allanpk716/to_icalendar/internal/models"
	"github.com/allanpk716/to_icalendar/internal/processors"
	"github.com/allanpk716/to_icalendar/internal/validators"
)

func main() {
	fmt.Println("=== to_icalendar 集成测试程序 ===")
	fmt.Println("包含错误处理、配置验证和性能监控\n")

	// 测试配置验证
	fmt.Println("1. 测试配置验证功能...")
	testConfigurationValidation()

	// 测试剪贴板功能
	fmt.Println("\n2. 测试剪贴板功能...")
	testClipboardWithValidation()

	// 测试处理流程
	fmt.Println("\n3. 测试处理流程...")
	testProcessingPipeline()

	// 测试错误处理
	fmt.Println("\n4. 测试错误处理机制...")
	testErrorHandling()

	// 测试性能监控
	fmt.Println("\n5. 测试性能监控...")
	testPerformanceMonitoring()

	fmt.Println("\n=== 集成测试完成 ===")
}

func testConfigurationValidation() {
	// 创建验证器
	validator := validators.NewContentValidator()

	// 测试配置加载
	configManager := config.NewConfigManager()
	serverConfig, err := configManager.LoadServerConfig("../../config/server.yaml")
	if err != nil {
		log.Printf("❌ 配置加载失败: %v", err)
		return
	}

	// 验证配置
	if err := serverConfig.Dify.Validate(); err != nil {
		fmt.Printf("⚠️  Dify配置验证失败: %v\n", err)
		fmt.Printf("   这可能会导致AI功能不可用\n")
	} else {
		fmt.Printf("✅ Dify配置验证通过\n")
	}

	// 验证API端点
	endpointValidation := validator.ValidateAPIEndpoint(serverConfig.Dify.APIEndpoint)
	if !endpointValidation.IsValid {
		fmt.Printf("❌ API端点验证失败: %s\n", endpointValidation.Message)
	} else {
		fmt.Printf("✅ API端点验证通过\n")
	}

	// 验证API密钥
	keyValidation := validator.ValidateAPIKey(serverConfig.Dify.APIKey)
	if !keyValidation.IsValid {
		fmt.Printf("⚠️  API密钥验证失败: %s\n", keyValidation.Message)
	} else {
		fmt.Printf("✅ API密钥验证通过\n")
	}
}

func testClipboardWithValidation() {
	// 创建剪贴板管理器
	manager, err := clipboard.NewManager()
	if err != nil {
		log.Printf("❌ 剪贴板管理器创建失败: %v", err)
		return
	}

	// 检查剪贴板内容
	hasContent, err := manager.HasContent()
	if err != nil {
		log.Printf("❌ 检查剪贴板内容失败: %v", err)
		return
	}

	if !hasContent {
		fmt.Printf("ℹ️  剪贴板当前为空，请复制一些文字后重新测试\n")
		return
	}

	// 读取内容
	content, err := manager.Read()
	if err != nil {
		log.Printf("❌ 读取剪贴板内容失败: %v", err)
		return
	}

	fmt.Printf("✅ 剪贴板内容读取成功，类型: %s\n", content.Type)

	// 验证内容
	validator := validators.NewContentValidator()

	if content.Type == models.ContentTypeText {
		validation := validator.ValidateText(content.Text)
		if validation.IsValid {
			fmt.Printf("✅ 文字内容验证通过\n")
			fmt.Printf("   内容长度: %d 字符\n", len(content.Text))
		} else {
			fmt.Printf("❌ 文字内容验证失败: %s\n", validation.Message)
		}
	} else if content.Type == models.ContentTypeImage {
		validation := validator.ValidateImage(content.Image, content.FileName)
		if validation.IsValid {
			fmt.Printf("✅ 图片内容验证通过\n")
			fmt.Printf("   图片大小: %d bytes\n", len(content.Image))
		} else {
			fmt.Printf("❌ 图片内容验证失败: %s\n", validation.Message)
		}
	}
}

func testProcessingPipeline() {
	// 创建验证器
	validator := validators.NewContentValidator()

	// 模拟文字处理
	testText := "2025-11-06 15:00 重要会议讨论项目进展"

	// 验证文字内容
	textValidation := validator.ValidateText(testText)
	if !textValidation.IsValid {
		fmt.Printf("❌ 测试文字验证失败: %s\n", textValidation.Message)
		return
	}

	// 创建文字处理器
	processor, err := processors.NewTextProcessor(nil)
	if err != nil {
		log.Printf("❌ 文字处理器创建失败: %v", err)
		return
	}

	// 执行处理
	startTime := time.Now()
	analysis := processor.QuickAnalyze(testText)
	processingTime := time.Since(startTime)

	fmt.Printf("✅ 文字处理完成\n")
	fmt.Printf("   处理时间: %v\n", processingTime)
	fmt.Printf("   包含日期: %v\n", analysis.HasDate)
	fmt.Printf("   包含时间: %v\n", analysis.HasTime)
	fmt.Printf("   是否紧急: %v\n", analysis.IsUrgent)
	fmt.Printf("   是否会议: %v\n", analysis.IsMeeting)
	fmt.Printf("   置信度: %.2f\n", analysis.Confidence)

	// 任务解析
	parser := processors.NewTaskParser()
	taskInfo, err := parser.ParseFromText(testText)
	if err != nil {
		log.Printf("❌ 任务解析失败: %v", err)
		return
	}

	fmt.Printf("✅ 任务解析完成\n")
	fmt.Printf("   解析标题: %s\n", taskInfo.Title)
	fmt.Printf("   解析日期: %s\n", taskInfo.Date)
	fmt.Printf("   解析时间: %s\n", taskInfo.Time)
	fmt.Printf("   解析置信度: %.2f\n", taskInfo.Confidence)

	// JSON生成
	if taskInfo.Confidence > 0.5 {
		generator, err := processors.NewJSONGenerator("../../config/drafts")
		if err != nil {
			log.Printf("❌ JSON生成器创建失败: %v", err)
			return
		}

		jsonStartTime := time.Now()
		filePath, err := generator.GenerateFromParsedInfo(taskInfo)
		jsonProcessingTime := time.Since(jsonStartTime)

		if err != nil {
			log.Printf("❌ JSON生成失败: %v", err)
			return
		}

		fmt.Printf("✅ JSON生成完成\n")
		fmt.Printf("   生成时间: %v\n", jsonProcessingTime)
		fmt.Printf("   文件路径: %s\n", filePath)
	} else {
		fmt.Printf("⚠️  置信度太低(%.2f)，跳过JSON生成\n", taskInfo.Confidence)
	}
}

func testErrorHandling() {
	fmt.Println("   测试结构化错误处理...")

	// 测试验证错误
	validator := validators.NewContentValidator()

	// 测试空文字
	emptyValidation := validator.ValidateText("")
	if !emptyValidation.IsValid {
		fmt.Printf("✅ 空文字错误处理正确: %s\n", emptyValidation.Message)
	}

	// 测试过长文字
	longText := string(make([]byte, 11000))
	longValidation := validator.ValidateText(longText)
	if !longValidation.IsValid {
		fmt.Printf("✅ 过长文字错误处理正确: %s\n", longValidation.Message)
	}

	// 测试空图片
	emptyImageValidation := validator.ValidateImage([]byte{}, "test.png")
	if !emptyImageValidation.IsValid {
		fmt.Printf("✅ 空图片错误处理正确: %s\n", emptyImageValidation.Message)
	}

	// 测试过大图片
	largeImage := make([]byte, 11*1024*1024) // 11MB
	largeImageValidation := validator.ValidateImage(largeImage, "large.png")
	if !largeImageValidation.IsValid {
		fmt.Printf("✅ 过大图片错误处理正确: %s\n", largeImageValidation.Message)
	}

	// 测试应用错误
	appErr := errors.NewValidationError("test_validation", "Test validation error", "Additional details")
	fmt.Printf("✅ 应用错误创建成功: %s\n", appErr.Error())
	fmt.Printf("   错误类型: %s\n", errors.GetErrorType(appErr))
	fmt.Printf("   错误代码: %s\n", errors.GetErrorCode(appErr))
	fmt.Printf("   是否可重试: %v\n", errors.IsRetryable(appErr))

	// 测试错误包装
	wrappedErr := errors.WrapError(appErr, errors.ErrorTypeProcessing, "processing_failed", "Processing step failed")
	fmt.Printf("✅ 错误包装成功: %s\n", wrappedErr.Error())
}

func testPerformanceMonitoring() {
	fmt.Println("   测试性能监控...")

	// 模拟多次处理以测试性能
	processor, err := processors.NewTextProcessor(nil)
	if err != nil {
		log.Printf("❌ 文字处理器创建失败: %v", err)
		return
	}

	testText := "明天下午2点开会讨论项目进展，非常重要，请准时参加"
	iterations := 100

	totalStartTime := time.Now()
	var totalProcessingTime time.Duration

	for i := 0; i < iterations; i++ {
		startTime := time.Now()
		analysis := processor.QuickAnalyze(testText)
		processingTime := time.Since(startTime)
		totalProcessingTime += processingTime

		if i == 0 {
			fmt.Printf("   首次处理时间: %v\n", processingTime)
			fmt.Printf("   置信度: %.2f\n", analysis.Confidence)
		}
	}

	totalTime := time.Since(totalStartTime)
	avgTime := totalProcessingTime / time.Duration(iterations)

	fmt.Printf("✅ 性能监控完成\n")
	fmt.Printf("   总处理次数: %d\n", iterations)
	fmt.Printf("   总耗时: %v\n", totalTime)
	fmt.Printf("   平均处理时间: %v\n", avgTime)
	fmt.Printf("   处理速度: %.2f 次/秒\n", float64(iterations)/totalTime.Seconds())

	// 内存使用情况模拟
	fmt.Printf("   内存监控: 需要集成runtime.MemStats\n")
	fmt.Printf("   建议: 添加内存使用监控和GC统计\n")
}

func init() {
	// 设置日志格式
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}