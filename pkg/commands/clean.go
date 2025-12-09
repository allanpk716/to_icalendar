package commands

import (
	"context"
	"fmt"

	"github.com/allanpk716/to_icalendar/pkg/logger"
	"github.com/allanpk716/to_icalendar/pkg/services"
)

// CleanCommand 清理命令
type CleanCommand struct {
	*BaseCommand
	cleanupService services.CleanupService
}

// NewCleanCommand 创建清理命令
func NewCleanCommand(container ServiceContainer) *CleanCommand {
	return &CleanCommand{
		BaseCommand:    NewBaseCommand("clean", "清理缓存文件"),
		cleanupService: container.GetCleanupService(),
	}
}

// Execute 执行清理命令
func (c *CleanCommand) Execute(ctx context.Context, req *CommandRequest) (*CommandResponse, error) {
	// 获取清理选项
	optionsInterface, exists := req.Args["options"]
	if !exists {
		optionsInterface = &services.CleanupOptions{
			All: true, // 默认清理所有
		}
	}

	var options *services.CleanupOptions
	var err error

	switch opts := optionsInterface.(type) {
	case *services.CleanupOptions:
		options = opts
	case map[string]interface{}:
		options, err = c.parseOptionsFromMap(opts)
		if err != nil {
			return ErrorResponse(fmt.Errorf("解析清理选项失败: %w", err)), nil
		}
	default:
		options = &services.CleanupOptions{All: true}
	}

	// 执行清理
	result, err := c.cleanupService.Cleanup(ctx, options)
	if err != nil {
		return ErrorResponse(fmt.Errorf("清理失败: %w", err)), nil
	}

	// 构建响应数据
	responseData := map[string]interface{}{
		"total_files":  result.TotalFiles,
		"total_size":   result.TotalSize,
		"files_by_type": result.FilesByType,
		"skipped":      result.Skipped,
		"message":      result.Message,
	}

	// 构建元数据
	metadata := map[string]interface{}{
		"cleanup_completed": true,
	}

	// 如果是预览模式，添加额外信息
	if options.DryRun {
		metadata["dry_run"] = true
	}

	return SuccessResponse(responseData, metadata), nil
}

// Validate 验证命令参数
func (c *CleanCommand) Validate(args []string) error {
	// clean 命令可以没有参数（默认清理所有）
	return nil
}

// parseOptionsFromMap 从map解析选项
func (c *CleanCommand) parseOptionsFromMap(optionsMap map[string]interface{}) (*services.CleanupOptions, error) {
	options := &services.CleanupOptions{
		All:         false,
		Tasks:       false,
		Images:      false,
		ImageHashes: false,
		Temp:        false,
		Generated:   false,
		DryRun:      false,
		Force:       false,
		OlderThan:   "",
		ClearAll:    false,
	}

	if all, ok := optionsMap["all"].(bool); ok {
		options.All = all
	}
	if tasks, ok := optionsMap["tasks"].(bool); ok {
		options.Tasks = tasks
	}
	if images, ok := optionsMap["images"].(bool); ok {
		options.Images = images
	}
	if imageHashes, ok := optionsMap["image_hashes"].(bool); ok {
		options.ImageHashes = imageHashes
	}
	if temp, ok := optionsMap["temp"].(bool); ok {
		options.Temp = temp
	}
	if generated, ok := optionsMap["generated"].(bool); ok {
		options.Generated = generated
	}
	if dryRun, ok := optionsMap["dry_run"].(bool); ok {
		options.DryRun = dryRun
	}
	if force, ok := optionsMap["force"].(bool); ok {
		options.Force = force
	}
	if olderThan, ok := optionsMap["older_than"].(string); ok {
		options.OlderThan = olderThan
	}
	if clearAll, ok := optionsMap["clear_all"].(bool); ok {
		options.ClearAll = clearAll
	}

	return options, nil
}

// ShowResult 显示清理结果（用于CLI调用）
func (c *CleanCommand) ShowResult(data interface{}, metadata map[string]interface{}) {
	logger.Debug("开始显示清理结果...")

	resultData, ok := data.(map[string]interface{})
	if !ok {
		logger.Error("❌ Invalid result data")
		logger.Debugf("接收到的数据类型: %T, 数据内容: %+v", data)
		return
	}

	skipped, _ := resultData["skipped"].(bool)
	message, _ := resultData["message"].(string)

	logger.Debugf("清理结果 - 跳过: %t, 消息: %s", skipped, message)

	if skipped {
		logger.Infof("ℹ️  %s", message)
		logger.Debug("清理操作被跳过")
		return
	}

	totalFiles, _ := resultData["total_files"].(int64)
	totalSize, _ := resultData["total_size"].(int64)

	logger.Debugf("清理统计 - 文件数量: %d, 总大小: %d bytes", totalFiles, totalSize)

	logger.Info("✅ Cleanup completed successfully")
	logger.Infof("  Total files: %d", totalFiles)
	logger.Infof("  Total size: %s", formatBytes(totalSize))

	// 如果是预览模式，显示额外信息
	if dryRun, ok := metadata["dry_run"].(bool); ok && dryRun {
		logger.Info("  📋 This was a dry run - no files were actually deleted")
		logger.Debug("这是预览模式，没有实际删除文件")
	}

	logger.Debug("清理结果显示完成")
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