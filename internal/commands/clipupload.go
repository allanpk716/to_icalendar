package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/allanpk716/to_icalendar/internal/logger"
	"github.com/allanpk716/to_icalendar/internal/models"
	"github.com/allanpk716/to_icalendar/internal/services"
)

// ClipUploadCommand 剪贴板上传命令
type ClipUploadCommand struct {
	*BaseCommand
	clipboardService services.ClipboardService
	todoService     services.TodoService
	difyService     services.DifyService
}

// NewClipUploadCommand 创建剪贴板上传命令
func NewClipUploadCommand(container ServiceContainer) *ClipUploadCommand {
	return &ClipUploadCommand{
		BaseCommand:     NewBaseCommand("clip-upload", "处理剪贴板内容并上传到 Microsoft Todo"),
		clipboardService: container.GetClipboardService(),
		todoService:     container.GetTodoService(),
		difyService:     container.GetDifyService(),
	}
}

// Execute 执行剪贴板上传命令
func (c *ClipUploadCommand) Execute(ctx context.Context, req *CommandRequest) (*CommandResponse, error) {
	logger.Info("开始执行 clip-upload 命令")

	// 1. 检查剪贴板是否有内容
	logger.Info("检查剪贴板内容...")
	hasContent, err := c.clipboardService.HasContent()
	if err != nil {
		logger.Error("检查剪贴板内容失败: %v", err)
		return ErrorResponse(fmt.Errorf("检查剪贴板内容失败: %w", err)), nil
	}

	if !hasContent {
		logger.Error("剪贴板没有内容")
		return ErrorResponse(fmt.Errorf("剪贴板没有内容")), nil
	}

	logger.Info("发现剪贴板内容，开始读取...")

	// 2. 读取剪贴板内容
	clipboardContent, err := c.clipboardService.ReadContent(ctx)
	if err != nil {
		logger.Error("读取剪贴板内容失败: %v", err)
		return ErrorResponse(fmt.Errorf("读取剪贴板内容失败: %w", err)), nil
	}

	logger.Info("成功读取剪贴板内容，类型: %s", clipboardContent.Type)

	// 3. 根据内容类型调用 Dify 服务处理
	var difyResponse *models.DifyResponse
	var originalContent string

	switch clipboardContent.Type {
	case models.ContentTypeText:
		originalContent = clipboardContent.Text
		logger.Info("调用 Dify 服务处理文本内容...")
		difyResponse, err = c.difyService.ProcessText(ctx, clipboardContent.Text)
	case models.ContentTypeImage:
		originalContent = "[图片内容]"
		logger.Info("调用 Dify 服务处理图像内容...")
		difyResponse, err = c.difyService.ProcessImage(ctx, clipboardContent.Image)
	default:
		err = fmt.Errorf("不支持的剪贴板内容类型: %s", clipboardContent.Type)
	}

	if err != nil {
		logger.Error("Dify 服务处理失败: %v", err)
		return ErrorResponse(fmt.Errorf("Dify 服务处理失败: %w", err)), nil
	}

	logger.Info("Dify 服务处理成功")

	// 4. 解析 Dify 响应为 Reminder 对象
	reminder, err := ParseDifyResponseToReminder(difyResponse, string(clipboardContent.Type), originalContent)
	if err != nil {
		logger.Error("解析 Dify 响应失败: %v", err)
		return ErrorResponse(fmt.Errorf("解析 Dify 响应失败: %w", err)), nil
	}

	logger.Info("成功解析任务信息: %s", reminder.Title)

	// 5. 创建 Microsoft Todo 任务
	logger.Info("开始创建 Microsoft Todo 任务...")
	err = c.todoService.CreateTask(ctx, reminder)
	if err != nil {
		logger.Error("创建 Microsoft Todo 任务失败: %v", err)
		return ErrorResponse(fmt.Errorf("创建 Microsoft Todo 任务失败: %w", err)), nil
	}

	logger.Info("成功创建 Microsoft Todo 任务")

	// 6. 构建成功响应
	responseData := &services.ProcessClipboardResult{
		Success:     true,
		Title:       reminder.Title,
		Description: reminder.Description,
		Message:     "剪贴板内容已成功处理并创建到 Microsoft Todo",
	}

	// 添加元数据
	metadata := map[string]interface{}{
		"content_type":  clipboardContent.Type,
		"task_title":   reminder.Title,
		"task_list":    reminder.List,
		"task_priority": reminder.Priority,
		"processed_at": time.Now(),
	}

	// 根据内容类型添加额外信息
	switch clipboardContent.Type {
	case models.ContentTypeImage:
		if size, ok := clipboardContent.Metadata["size"].(int); ok {
			metadata["content_size"] = size
		}
	case models.ContentTypeText:
		if length, ok := clipboardContent.Metadata["length"].(int); ok {
			metadata["content_size"] = length
		}
	}

	logger.Info("clip-upload 命令执行完成")
	return SuccessResponse(responseData, metadata), nil
}

// Validate 验证命令参数
func (c *ClipUploadCommand) Validate(args []string) error {
	// clip-upload 命令通常不需要参数
	return nil
}

// ShowResult 显示处理结果（用于CLI调用）
func (c *ClipUploadCommand) ShowResult(data interface{}, metadata map[string]interface{}) {
	result, ok := data.(*services.ProcessClipboardResult)
	if !ok {
		fmt.Println("❌ 无效的结果数据")
		return
	}

	if result.Success {
		fmt.Println("✓ 剪贴板内容处理成功")
		fmt.Println()
		fmt.Printf("📝 任务标题: %s\n", result.Title)
		if result.Description != "" && result.Description != result.Title {
			fmt.Printf("📄 任务描述: %s\n", result.Description)
		}
		if result.Message != "" {
			fmt.Printf("✅ %s\n", result.Message)
		}
		fmt.Println()

		// 显示详细的任务信息
		if taskTitle, ok := metadata["task_title"].(string); ok && taskTitle != "" {
			fmt.Printf("🎯 创建的任务: %s\n", taskTitle)
		}
		if taskList, ok := metadata["task_list"].(string); ok {
			fmt.Printf("📋 任务列表: %s\n", taskList)
		}
		if taskPriority, ok := metadata["task_priority"].(string); ok {
			priorityIcon := "🔵"
			switch taskPriority {
			case "high":
				priorityIcon = "🔴"
			case "medium":
				priorityIcon = "🟡"
			case "low":
				priorityIcon = "🟢"
			}
			fmt.Printf("⭐ 优先级: %s %s\n", priorityIcon, taskPriority)
		}
	} else {
		fmt.Printf("❌ 剪贴板内容处理失败: %s\n", result.Message)
		fmt.Println("💡 请检查剪贴板内容或相关服务配置")
	}

	fmt.Println()
	// 显示元数据
	if contentType, ok := metadata["content_type"].(string); ok {
		fmt.Printf("📂 内容类型: %s\n", contentType)
	}
	if contentSize, ok := metadata["content_size"].(int); ok && contentSize > 0 {
		fmt.Printf("📏 内容大小: %d\n", contentSize)
	}

	fmt.Println("\n💡 提示:")
	fmt.Println("  - 任务已创建到您的 Microsoft Todo")
	fmt.Println("  - 您可以在 Microsoft Todo 应用中查看和管理此任务")
	fmt.Println("  - 支持 AI 智能解析剪贴板内容并生成任务")
}