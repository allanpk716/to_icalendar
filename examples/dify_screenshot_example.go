package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/allanpk716/to_icalendar/internal/dify"
	"github.com/allanpk716/to_icalendar/internal/models"
)

func main() {
	fmt.Println("=== Dify截图处理示例 ===")

	// 1. 配置Dify
	config := &models.DifyConfig{
		APIEndpoint: "http://dify.urithub.com/v1",
		APIKey:      "app-m51AZqIDX3FdklmHTLyG6Teg",
		Timeout:     30,
	}

	// 2. 创建截图处理器
	processor, err := dify.NewScreenshotProcessor(config)
	if err != nil {
		log.Fatalf("创建截图处理器失败: %v", err)
	}

	// 3. 获取命令行参数中的图片文件路径
	if len(os.Args) < 2 {
		fmt.Println("使用方法: go run dify_screenshot_example.go <图片文件路径>")
		fmt.Println("示例: go run dify_screenshot_example.go test_screenshot.png")
		os.Exit(1)
	}

	imagePath := os.Args[1]

	// 4. 读取图片文件
	imageData, err := os.ReadFile(imagePath)
	if err != nil {
		log.Fatalf("读取图片文件失败: %v", err)
	}

	// 5. 创建输入数据
	screenshot := &dify.ScreenshotInput{
		Data:     imageData,
		FileName: filepath.Base(imagePath),
		Format:   dify.ExtractImageFormat(imagePath),
	}

	fmt.Printf("处理图片: %s\n", screenshot.FileName)
	fmt.Printf("图片大小: %d bytes\n", len(screenshot.Data))
	fmt.Printf("图片格式: %s\n", screenshot.Format)
	fmt.Println()

	// 6. 显示处理器信息
	processorInfo := processor.GetProcessorInfo()
	fmt.Printf("处理器信息:\n")
	fmt.Printf("  名称: %s\n", processorInfo.Name)
	fmt.Printf("  版本: %s\n", processorInfo.Version)
	fmt.Printf("  支持格式: %v\n", processorInfo.SupportedFormats)
	fmt.Printf("  最大文件大小: %d MB\n", processorInfo.MaxFileSize/(1024*1024))
	fmt.Println()

	// 7. 验证输入
	fmt.Println("验证输入数据...")
	if err := processor.ValidateInput(screenshot); err != nil {
		log.Fatalf("输入验证失败: %v", err)
	}
	fmt.Println("✅ 输入验证通过")
	fmt.Println()

	// 8. 处理截图
	fmt.Println("开始处理截图...")
	ctx := context.Background()
	reminder, err := processor.ProcessScreenshot(ctx, screenshot)
	if err != nil {
		log.Fatalf("处理截图失败: %v", err)
	}
	fmt.Println("✅ 截图处理完成")
	fmt.Println()

	// 9. 输出结果
	fmt.Println("=== 处理结果 ===")
	fmt.Printf("标题: %s\n", reminder.Title)
	fmt.Printf("描述: %s\n", reminder.Description)
	fmt.Printf("日期: %s\n", reminder.Date)
	fmt.Printf("时间: %s\n", reminder.Time)
	fmt.Printf("提前提醒: %s\n", reminder.RemindBefore)
	fmt.Printf("优先级: %s\n", reminder.Priority)
	fmt.Printf("任务列表: %s\n", reminder.List)
	fmt.Println()

	// 10. 生成JSON格式的reminder文件
	jsonData, err := generateReminderJSON(reminder)
	if err != nil {
		log.Printf("生成JSON失败: %v", err)
	} else {
		outputFile := "generated_reminder.json"
		err = os.WriteFile(outputFile, jsonData, 0644)
		if err != nil {
			log.Printf("保存JSON文件失败: %v", err)
		} else {
			fmt.Printf("📄 已生成reminder文件: %s\n", outputFile)
		}
	}

	fmt.Println("=== 示例完成 ===")
}

// generateReminderJSON 生成reminder格式的JSON数据
func generateReminderJSON(reminder *models.Reminder) ([]byte, error) {
	jsonData, err := json.MarshalIndent(reminder, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("序列化reminder失败: %w", err)
	}
	return jsonData, nil
}