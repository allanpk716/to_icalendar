package commands

import (
	"context"
	"fmt"

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
	// 1. 检查剪贴板是否有内容
	hasContent, err := c.clipboardService.HasContent()
	if err != nil {
		return ErrorResponse(fmt.Errorf("检查剪贴板内容失败: %w", err)), nil
	}

	if !hasContent {
		return ErrorResponse(fmt.Errorf("剪贴板没有内容")), nil
	}

	// 2. 读取剪贴板内容
	clipboardContent, err := c.clipboardService.ReadContent(ctx)
	if err != nil {
		return ErrorResponse(fmt.Errorf("读取剪贴板内容失败: %w", err)), nil
	}

	// 3. 预处理剪贴板内容
	processingResult, err := c.clipboardService.ProcessContent(ctx, clipboardContent)
	if err != nil {
		return ErrorResponse(fmt.Errorf("预处理剪贴板内容失败: %w", err)), nil
	}

	// 4. 这里应该有完整的内容处理逻辑
	// 由于这个命令比较复杂，我们先返回基本信息，完整的实现将在后续步骤中完成

	// 构建响应数据
	responseData := &services.ProcessClipboardResult{
		Success:     true,
		Message:     "剪贴板内容处理完成",
		Data:        processingResult,
		Description: fmt.Sprintf("已读取 %s 类型内容", clipboardContent.Type),
	}

	// 添加元数据
	metadata := map[string]interface{}{
		"content_type": clipboardContent.Type,
		"content_size": 0,
	}

	// 根据内容类型添加额外信息
	switch clipboardContent.Type {
	case "image":
		if size, ok := clipboardContent.Metadata["size"].(int); ok {
			metadata["content_size"] = size
			responseData.Description = fmt.Sprintf("已读取图片，大小: %d 字节", size)
		}
	case "text":
		if length, ok := clipboardContent.Metadata["length"].(int); ok {
			metadata["content_size"] = length
			responseData.Description = fmt.Sprintf("已读取文本，长度: %d 字符", length)
		}
	}

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
		fmt.Println("❌ Invalid result data")
		return
	}

	if result.Success {
		fmt.Println("✓ 剪贴板内容处理成功")
		if result.Title != "" {
			fmt.Printf("  标题: %s\n", result.Title)
		}
		if result.Description != "" {
			fmt.Printf("  描述: %s\n", result.Description)
		}
		if result.Message != "" {
			fmt.Printf("  信息: %s\n", result.Message)
		}
	} else {
		fmt.Printf("❌ 剪贴板内容处理失败: %s\n", result.Message)
	}

	// 显示元数据
	if contentType, ok := metadata["content_type"].(string); ok {
		fmt.Printf("  内容类型: %s\n", contentType)
	}
	if contentSize, ok := metadata["content_size"].(int); ok && contentSize > 0 {
		fmt.Printf("  内容大小: %d\n", contentSize)
	}
}