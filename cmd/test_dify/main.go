package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/allanpk716/to_icalendar/internal/config"
	"github.com/allanpk716/to_icalendar/internal/dify"
	"github.com/allanpk716/to_icalendar/internal/models"
)

const (
	version        = "1.0.0"
 programName    = "Dify 图片识别测试程序"
	defaultTimeout = 30 * time.Second
)

var (
	verbose       = flag.Bool("verbose", false, "详细输出模式")
	outputFile    = flag.String("output", "", "输出结果到指定文件")
	downloadURL   = flag.String("url", "", "从指定URL下载测试图片")
	configFile    = flag.String("config", "config/server.yaml", "配置文件路径")
	showVersion   = flag.Bool("version", false, "显示版本信息")
	showHelp      = flag.Bool("help", false, "显示帮助信息")
)

func main() {
	flag.Parse()

	if *showVersion {
		printVersion()
		return
	}

	if *showHelp || flag.NArg() == 0 && *downloadURL == "" {
		printHelp()
		return
	}

	log.Printf("=== %s v%s ===", programName, version)

	// 初始化配置
	cfg, err := loadConfiguration(*configFile)
	if err != nil {
		log.Fatalf("❌ 配置加载失败: %v", err)
	}

	log.Printf("✅ 配置加载成功")
	if *verbose {
		log.Printf("📋 Dify API 端点: %s", cfg.Dify.APIEndpoint)
		log.Printf("📋 超时设置: %d秒", cfg.Dify.Timeout)
	}

	// 处理图片文件
	imageFile, err := prepareImageFile()
	if err != nil {
		log.Fatalf("❌ 图片准备失败: %v", err)
	}

	defer func() {
		if *downloadURL != "" && imageFile != "" {
			// 清理下载的临时文件
			if removeErr := os.Remove(imageFile); removeErr != nil {
				log.Printf("⚠️  清理临时文件失败: %v", removeErr)
			}
		}
	}()

	// 执行测试
	testResult, err := executeImageTest(cfg, imageFile)
	if err != nil {
		log.Fatalf("❌ 测试执行失败: %v", err)
	}

	// 输出结果
	if err := outputResults(testResult, *outputFile); err != nil {
		log.Fatalf("❌ 结果输出失败: %v", err)
	}

	log.Printf("🎉 测试完成！")
}

// loadConfiguration 加载配置文件
func loadConfiguration(configPath string) (*models.ServerConfig, error) {
	if *verbose {
		log.Printf("📖 正在加载配置文件: %s", configPath)
	}

	configManager := config.NewConfigManager()
	cfg, err := configManager.LoadServerConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("加载服务器配置失败: %w", err)
	}

	// 验证 Dify 配置
	if err := cfg.Dify.Validate(); err != nil {
		return nil, fmt.Errorf("Dify 配置验证失败: %w", err)
	}

	return cfg, nil
}

// prepareImageFile 准备图片文件
func prepareImageFile() (string, error) {
	if *downloadURL != "" {
		return downloadImageFromURL(*downloadURL)
	}

	if flag.NArg() > 0 {
		imagePath := flag.Arg(0)
		if err := validateImageFile(imagePath); err != nil {
			return "", fmt.Errorf("图片文件验证失败: %w", err)
		}
		return imagePath, nil
	}

	return "", fmt.Errorf("请指定图片文件路径或使用 -url 参数下载图片")
}

// downloadImageFromURL 从URL下载图片
func downloadImageFromURL(imageURL string) (string, error) {
	if *verbose {
		log.Printf("🌐 正在从URL下载图片: %s", imageURL)
	}

	parsedURL, err := url.Parse(imageURL)
	if err != nil {
		return "", fmt.Errorf("无效的URL: %w", err)
	}

	// 解析文件名
	fileName := filepath.Base(parsedURL.Path)
	if fileName == "." || fileName == "/" || fileName == "" {
		fileName = fmt.Sprintf("downloaded_image_%d.jpg", time.Now().Unix())
	}

	// 确保文件有扩展名
	ext := strings.ToLower(filepath.Ext(fileName))
	if ext == "" {
		fileName += ".jpg"
	}

	// 创建测试图片目录
	testDir := "cmd/test_dify/test-images"
	if err := os.MkdirAll(testDir, 0755); err != nil {
		return "", fmt.Errorf("创建测试目录失败: %w", err)
	}

	filePath := filepath.Join(testDir, fileName)

	// 检查文件是否已存在
	if _, err := os.Stat(filePath); err == nil {
		log.Printf("📁 图片文件已存在: %s", filePath)
		return filePath, nil
	}

	log.Printf("⬇️  正在下载图片到: %s", filePath)

	// 这里应该实现实际的下载逻辑，为简化暂时返回错误
	// 在实际使用中，可以使用 http.Get 下载文件
	return "", fmt.Errorf("URL下载功能暂未实现，请手动下载图片到 %s", filePath)
}

// validateImageFile 验证图片文件
func validateImageFile(imagePath string) error {
	if *verbose {
		log.Printf("🔍 正在验证图片文件: %s", imagePath)
	}

	// 检查文件是否存在
	fileInfo, err := os.Stat(imagePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("文件不存在: %s", imagePath)
		}
		return fmt.Errorf("文件访问错误: %w", err)
	}

	// 检查文件大小
	if fileInfo.Size() == 0 {
		return fmt.Errorf("文件为空")
	}

	maxSize := int64(10 * 1024 * 1024) // 10MB
	if fileInfo.Size() > maxSize {
		return fmt.Errorf("文件大小 %d 超过最大限制 %d bytes", fileInfo.Size(), maxSize)
	}

	// 检查文件扩展名
	ext := strings.ToLower(filepath.Ext(imagePath))
	supportedExts := map[string]bool{
		".png":  true,
		".jpg":  true,
		".jpeg": true,
		".bmp":  true,
		".gif":  true,
	}

	if !supportedExts[ext] {
		return fmt.Errorf("不支持的文件格式: %s", ext)
	}

	if *verbose {
		log.Printf("✅ 图片文件验证通过")
		log.Printf("📊 文件大小: %s", formatBytes(fileInfo.Size()))
		log.Printf("📊 文件格式: %s", ext[1:])
	}

	return nil
}

// executeImageTest 执行图片识别测试
func executeImageTest(cfg *models.ServerConfig, imagePath string) (*TestResult, error) {
	startTime := time.Now()
	result := &TestResult{
		ImagePath:    imagePath,
		StartTime:    startTime,
		Configuration: cfg.Dify,
	}

	if *verbose {
		log.Printf("🚀 开始执行图片识别测试")
	}

	// 创建截图处理器
	processor, err := dify.NewScreenshotProcessor(&cfg.Dify)
	if err != nil {
		return nil, fmt.Errorf("创建截图处理器失败: %w", err)
	}

	if *verbose {
		info := processor.GetProcessorInfo()
		log.Printf("🔧 处理器信息: %s v%s", info.Name, info.Version)
		log.Printf("🔧 支持格式: %v", info.SupportedFormats)
		log.Printf("🔧 最大文件大小: %s", formatBytes(info.MaxFileSize))
	}

	// 读取图片文件
	imageData, err := os.ReadFile(imagePath)
	if err != nil {
		return nil, fmt.Errorf("读取图片文件失败: %w", err)
	}

	// 准备输入数据
	screenshotInput := &dify.ScreenshotInput{
		Data:     imageData,
		FileName: filepath.Base(imagePath),
		Format:   dify.ExtractImageFormat(filepath.Base(imagePath)),
	}

	// 设置上下文和超时
	ctx := context.Background()
	if cfg.Dify.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(cfg.Dify.Timeout)*time.Second)
		defer cancel()
	} else {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, defaultTimeout)
		defer cancel()
	}

	// 处理截图
	reminder, err := processor.ProcessScreenshot(ctx, screenshotInput)
	if err != nil {
		result.Error = err.Error()
		result.Success = false
		return result, nil // 返回结果而不是错误，以便显示详细信息
	}

	// 记录成功的处理结果
	result.Reminder = reminder
	result.Success = true
	result.ProcessingTime = time.Since(startTime)

	if *verbose {
		log.Printf("✅ 图片处理完成")
		log.Printf("⏱️  处理耗时: %v", result.ProcessingTime)
	}

	return result, nil
}

// outputResults 输出测试结果
func outputResults(result *TestResult, outputPath string) error {
	if *verbose {
		log.Printf("📤 正在输出测试结果")
	}

	// 打印结果到控制台
	printTestResult(result)

	// 如果指定了输出文件，则保存结果
	if outputPath != "" {
		if err := saveResultsToFile(result, outputPath); err != nil {
			return fmt.Errorf("保存结果到文件失败: %w", err)
		}
		log.Printf("💾 结果已保存到: %s", outputPath)
	}

	return nil
}

// printTestResult 打印测试结果
func printTestResult(result *TestResult) {
	fmt.Println()
	fmt.Println("=== 测试结果 ===")
	fmt.Printf("图片路径: %s\n", result.ImagePath)
	fmt.Printf("测试时间: %s\n", result.StartTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("处理状态: %s\n", getStatusText(result.Success))
	fmt.Printf("处理耗时: %v\n", result.ProcessingTime)

	if result.Success && result.Reminder != nil {
		fmt.Println()
		fmt.Println("=== 识别结果 ===")
		fmt.Printf("标题: %s\n", result.Reminder.Title)
		fmt.Printf("描述: %s\n", result.Reminder.Description)
		fmt.Printf("日期: %s\n", result.Reminder.Date)
		fmt.Printf("时间: %s\n", result.Reminder.Time)
		fmt.Printf("提前提醒: %s\n", result.Reminder.RemindBefore)
		fmt.Printf("优先级: %s\n", result.Reminder.Priority)
		fmt.Printf("任务列表: %s\n", result.Reminder.List)

		// 输出JSON格式预览
		fmt.Println()
		fmt.Println("=== JSON格式预览 ===")
		if jsonData, err := json.MarshalIndent(result.Reminder, "", "  "); err == nil {
			fmt.Println(string(jsonData))
		}
	} else if !result.Success {
		fmt.Println()
		fmt.Println("=== 错误信息 ===")
		fmt.Printf("错误: %s\n", result.Error)
	}

	fmt.Println()
}

// saveResultsToFile 保存结果到文件
func saveResultsToFile(result *TestResult, filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	return encoder.Encode(result)
}

// TestResult 测试结果结构
type TestResult struct {
	ImagePath      string               `json:"image_path"`
	StartTime      time.Time            `json:"start_time"`
	ProcessingTime time.Duration        `json:"processing_time"`
	Success        bool                 `json:"success"`
	Reminder       *models.Reminder     `json:"reminder,omitempty"`
	Error          string               `json:"error,omitempty"`
	Configuration  models.DifyConfig    `json:"configuration"`
}

// 辅助函数
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

func getStatusText(success bool) string {
	if success {
		return "✅ 成功"
	}
	return "❌ 失败"
}

func printVersion() {
	fmt.Printf("%s v%s\n", programName, version)
	fmt.Printf("一个用于测试 Dify 图片识别提醒事项功能的工具\n")
}

func printHelp() {
	fmt.Printf("用法: %s [选项] [图片文件路径]\n\n", os.Args[0])
	fmt.Println("选项:")
	flag.PrintDefaults()
	fmt.Println()
	fmt.Println("示例:")
	fmt.Printf("  %s /path/to/image.jpg\n", os.Args[0])
	fmt.Printf("  %s -verbose /path/to/image.jpg\n", os.Args[0])
	fmt.Printf("  %s -output result.json /path/to/image.jpg\n", os.Args[0])
	fmt.Printf("  %s -url https://example.com/image.jpg\n", os.Args[0])
	fmt.Printf("  %s -config custom.yaml /path/to/image.jpg\n", os.Args[0])
	fmt.Println()
	fmt.Println("支持的图片格式: png, jpg, jpeg, bmp, gif")
	fmt.Println("最大文件大小: 10MB")
}