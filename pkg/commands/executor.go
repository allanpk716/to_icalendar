package commands

import (
	"context"
	"fmt"

	"github.com/allanpk716/to_icalendar/internal/app"
	"github.com/allanpk716/to_icalendar/internal/commands"
	"github.com/allanpk716/to_icalendar/internal/services"
)

// DefaultCommandExecutor 默认命令执行器实现
type DefaultCommandExecutor struct {
	app *app.Application
}

// NewCommandExecutor 创建命令执行器
func NewCommandExecutor() CommandExecutor {
	return &DefaultCommandExecutor{
		app: app.NewApplication(),
	}
}

// InitConfig 初始化配置
func (e *DefaultCommandExecutor) InitConfig() (*ConfigResult, error) {
	ctx := context.Background()

	// 初始化应用
	if err := e.app.Initialize(ctx); err != nil {
		return nil, fmt.Errorf("初始化应用失败: %w", err)
	}

	// 获取服务容器
	container := e.app.GetServiceContainer()

	// 创建初始化命令
	initCmd := commands.NewInitCommand(container)

	// 执行命令
	req := &commands.CommandRequest{
		Command: "init",
		Args:    make(map[string]interface{}),
	}

	resp, err := initCmd.Execute(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("执行初始化命令失败: %w", err)
	}

	if !resp.Success {
		return nil, fmt.Errorf("初始化失败: %s", resp.Error)
	}

	// 转换响应数据
	data, ok := resp.Data.(*services.ConfigResult)
	if !ok {
		return nil, fmt.Errorf("无效的响应数据格式")
	}

	// 转换为公共接口格式
	result := &ConfigResult{
		Success:          data.Success,
		ConfigDir:        data.ConfigDir,
		ServerConfig:     data.ServerConfig,
		ReminderTemplate: data.ReminderTemplate,
		Message:          data.Message,
	}

	return result, nil
}

// CleanCache 清理缓存
func (e *DefaultCommandExecutor) CleanCache(options *CleanupOptions) (*CleanupResult, error) {
	ctx := context.Background()

	// 初始化应用
	if err := e.app.Initialize(ctx); err != nil {
		return nil, fmt.Errorf("初始化应用失败: %w", err)
	}

	// 获取服务容器
	container := e.app.GetServiceContainer()

	// 创建清理命令
	cleanCmd := commands.NewCleanCommand(container)

	// 转换选项格式
	servicesOptions := &services.CleanupOptions{
		All:         options.All,
		Tasks:       options.Tasks,
		Images:      options.Images,
		ImageHashes: options.ImageHashes,
		Temp:        options.Temp,
		Generated:   options.Generated,
		DryRun:      options.DryRun,
		Force:       options.Force,
		OlderThan:   options.OlderThan,
		ClearAll:    options.ClearAll,
	}

	// 执行命令
	req := &commands.CommandRequest{
		Command: "clean",
		Args: map[string]interface{}{
			"options": servicesOptions,
		},
	}

	resp, err := cleanCmd.Execute(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("执行清理命令失败: %w", err)
	}

	if !resp.Success {
		return nil, fmt.Errorf("清理失败: %s", resp.Error)
	}

	// 转换响应数据
	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("无效的响应数据格式")
	}

	// 转换为公共接口格式
	result := &CleanupResult{
		Success: resp.Success,
	}

	if totalFiles, ok := data["total_files"].(int64); ok {
		result.TotalFiles = totalFiles
	}
	if totalSize, ok := data["total_size"].(int64); ok {
		result.TotalSize = totalSize
	}
	if filesByType, ok := data["files_by_type"].(map[string]int64); ok {
		result.FilesByType = filesByType
	}
	if skipped, ok := data["skipped"].(bool); ok {
		result.Skipped = skipped
	}
	if message, ok := data["message"].(string); ok {
		result.Message = message
	}

	return result, nil
}

// ProcessClipboard 处理剪贴板并上传
func (e *DefaultCommandExecutor) ProcessClipboard() (*ProcessClipboardResult, error) {
	ctx := context.Background()

	// 初始化应用
	if err := e.app.Initialize(ctx); err != nil {
		return nil, fmt.Errorf("初始化应用失败: %w", err)
	}

	// 获取服务容器
	container := e.app.GetServiceContainer()

	// 创建剪贴板上传命令
	clipCmd := commands.NewClipUploadCommand(container)

	// 执行命令
	req := &commands.CommandRequest{
		Command: "clip-upload",
		Args:    make(map[string]interface{}),
	}

	resp, err := clipCmd.Execute(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("执行剪贴板上传命令失败: %w", err)
	}

	if !resp.Success {
		return nil, fmt.Errorf("剪贴板上传失败: %s", resp.Error)
	}

	// 转换响应数据
	data, ok := resp.Data.(*services.ProcessClipboardResult)
	if !ok {
		return nil, fmt.Errorf("无效的响应数据格式")
	}

	// 转换为公共接口格式
	result := &ProcessClipboardResult{
		Success:     data.Success,
		Title:       data.Title,
		Description: data.Description,
		Message:     data.Message,
	}

	return result, nil
}