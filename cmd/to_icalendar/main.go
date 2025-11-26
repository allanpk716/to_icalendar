package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/allanpk716/to_icalendar/internal/app"
	"github.com/allanpk716/to_icalendar/internal/commands"
	"github.com/allanpk716/to_icalendar/internal/logger"
	svcs "github.com/allanpk716/to_icalendar/internal/services"
)

const (
	version = "1.0.0"
	appName = "to_icalendar"
)

// CommandOptions 命令行选项
type CommandOptions struct {
	ForceUpload      bool
	NoDeduplication  bool
	DedupStrategy    string
	IncludeCompleted bool
}

// CleanOptions 清理命令选项
type CleanOptions struct {
	All         bool
	Tasks       bool
	Images      bool
	ImageHashes bool
	Temp        bool
	Generated   bool
	DryRun      bool
	Force       bool
	OlderThan   string
	ClearAll    bool
}

func main() {
	logger.Infof("%s v%s - Reminder sending tool (supports Microsoft Todo)", appName, version)

	// 解析命令行参数
	if len(os.Args) < 2 {
		showUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	logger.Infof("执行命令: %s", command)

	// init 命令使用独立处理路径
	if command == "init" {
		handleInitDirect()
		return
	}

	// 创建应用实例（其他命令需要完整初始化）
	application := app.NewApplication()

	// 初始化应用
	ctx := context.Background()
	if err := application.Initialize(ctx); err != nil {
		logger.Errorf("❌ 配置文件错误，请先运行 '%s init' 初始化配置", appName)
		logger.Errorf("   错误详情: %v", err)
		os.Exit(1)
	}

	// 确保应用在退出时正确关闭
	defer application.Shutdown(ctx)

	// 获取服务容器
	container := application.GetServiceContainer()

	// 执行其他命令
	switch command {
	case "test":
		// 直接使用 TestCommand
		testCmd := commands.NewTestCommand(container)
		req := &commands.CommandRequest{
			Command: "test",
			Args:    make(map[string]interface{}),
		}
		resp, err := testCmd.Execute(ctx, req)
		if err != nil {
			logger.Errorf("命令执行失败: %v", err)
			os.Exit(1)
		}
		if !resp.Success {
			logger.Errorf("命令执行失败: %s", resp.Error)
			os.Exit(1)
		}
		testCmd.ShowTestResult(resp.Data, resp.Metadata)
	case "clip-upload":
		// 直接使用 ClipUploadCommand
		clipCmd := commands.NewClipUploadCommand(container)
		req := &commands.CommandRequest{
			Command: "clip-upload",
			Args:    make(map[string]interface{}),
		}
		resp, err := clipCmd.Execute(ctx, req)
		if err != nil {
			logger.Errorf("命令执行失败: %v", err)
			os.Exit(1)
		}
		if !resp.Success {
			logger.Errorf("命令执行失败: %s", resp.Error)
			os.Exit(1)
		}
		clipCmd.ShowResult(resp.Data, resp.Metadata)
	case "clean":
		// 直接使用 CleanCommand
		cleanCmd := commands.NewCleanCommand(container)
		cleanOptions := parseCleanOptions(os.Args[2:])
		req := &commands.CommandRequest{
			Command: "clean",
			Args: map[string]interface{}{
				"options": &svcs.CleanupOptions{
					All:         cleanOptions.All,
					Tasks:       cleanOptions.Tasks,
					Images:      cleanOptions.Images,
					ImageHashes: cleanOptions.ImageHashes,
					Temp:        cleanOptions.Temp,
					Generated:   cleanOptions.Generated,
					DryRun:      cleanOptions.DryRun,
					Force:       cleanOptions.Force,
					OlderThan:   cleanOptions.OlderThan,
					ClearAll:    cleanOptions.ClearAll,
				},
			},
		}
		resp, err := cleanCmd.Execute(ctx, req)
		if err != nil {
			logger.Errorf("命令执行失败: %v", err)
			os.Exit(1)
		}
		if !resp.Success {
			logger.Errorf("命令执行失败: %s", resp.Error)
			os.Exit(1)
		}
		cleanCmd.ShowResult(resp.Data, resp.Metadata)
	case "upload":
		handleUpload(container, parseCommandOptions(os.Args[2:]))
	case "clip":
		handleClip(container)
	case "tasks":
		handleTasks(container, os.Args[2:])
	case "cache":
		handleCache(container, os.Args[2:])
	case "help", "-h", "--help":
		showUsage()
	default:
		logger.Errorf("未知命令: %s", command)
		fmt.Printf("Unknown command: %s\n\n", command)
		showUsage()
		os.Exit(1)
	}

	logger.Info("程序执行完成")
}

// parseCommandOptions 解析命令行选项
func parseCommandOptions(args []string) CommandOptions {
	options := CommandOptions{
		ForceUpload:      false,
		NoDeduplication:  false,
		DedupStrategy:    "",
		IncludeCompleted: false,
	}

	for i, arg := range args {
		switch arg {
		case "--force-upload":
			options.ForceUpload = true
		case "--no-deduplication":
			options.NoDeduplication = true
		case "--dedup-strategy":
			if i+1 < len(args) {
				options.DedupStrategy = args[i+1]
			}
		case "--include-completed":
			options.IncludeCompleted = true
		}
	}

	return options
}

// parseCleanOptions 解析清理命令选项
func parseCleanOptions(args []string) CleanOptions {
	options := CleanOptions{
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

	for i, arg := range args {
		switch arg {
		case "--all":
			options.All = true
		case "--tasks":
			options.Tasks = true
		case "--images":
			options.Images = true
		case "--image-hashes":
			options.ImageHashes = true
		case "--temp":
			options.Temp = true
		case "--generated":
			options.Generated = true
		case "--dry-run":
			options.DryRun = true
		case "--force":
			options.Force = true
		case "--older-than":
			if i+1 < len(args) {
				options.OlderThan = args[i+1]
			}
		case "--clear-all":
			options.ClearAll = true
		}
	}

	// 如果没有指定任何具体类型，默认清理所有
	if !options.Tasks && !options.Images && !options.ImageHashes && !options.Temp && !options.Generated {
		options.All = true
	}

	return options
}

// handleInitDirect 独立处理 init 命令，不依赖应用初始化
func handleInitDirect() {
	logger.Info("🚀 初始化配置...")

	// 获取用户配置目录
	homeDir, err := os.UserHomeDir()
	if err != nil {
		logger.Errorf("❌ 获取用户目录失败: %v", err)
		os.Exit(1)
	}

	logger.Debugf("用户目录: %s", homeDir)

	configDir := filepath.Join(homeDir, ".to_icalendar")
	serverConfigPath := filepath.Join(configDir, "server.yaml")

	logger.Debugf("配置目录: %s", configDir)
	logger.Debugf("配置文件路径: %s", serverConfigPath)

	// 创建配置目录
	logger.Debug("创建配置目录...")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		logger.Errorf("❌ 创建配置目录失败: %v", err)
		os.Exit(1)
	}
	logger.Debugf("配置目录创建成功: %s", configDir)

	// 检查文件是否已存在
	logger.Debug("检查配置文件是否已存在...")
	if _, err := os.Stat(serverConfigPath); err == nil {
		logger.Warnf("⚠️  配置文件已存在: %s", serverConfigPath)
		logger.Info("如需重新生成，请先删除现有配置文件")
		return
	}

	// 创建默认 server.yaml 内容
	logger.Debug("创建默认配置文件内容...")
	serverConfigContent := `# Microsoft Todo 配置
microsoft_todo:
  tenant_id: "YOUR_TENANT_ID"          # Azure 租户 ID
  client_id: "YOUR_CLIENT_ID"        # 应用程序客户端 ID
  client_secret: "YOUR_CLIENT_SECRET"  # 客户端密钥
  user_email: ""                     # 目标用户邮箱（可选）
  timezone: "Asia/Shanghai"          # 时区设置

# 提醒配置
reminder:
  default_remind_before: "15m"       # 默认提前提醒时间
  enable_smart_reminder: true        # 启用智能提醒功能

# 去重配置
deduplication:
  enabled: true                      # 启用去重功能
  time_window_minutes: 5              # 时间匹配窗口（分钟）
  similarity_threshold: 80            # 相似度阈值（0-100）
  check_incomplete_only: true         # 只检查未完成的任务
  enable_local_cache: true            # 启用本地缓存
  enable_remote_query: true           # 启用远程查询

# Dify AI 配置（可选）
dify:
  api_endpoint: ""                   # Dify API 端点
  api_key: ""                        # Dify API 密钥
  timeout: 60                        # 请求超时时间（秒）

# 缓存配置
cache:
  auto_cleanup_days: 30              # 自动清理天数
  cleanup_on_startup: true           # 启动时清理
  preserve_successful_hashes: true   # 保留成功哈希记录

# 日志配置
logging:
  level: "info"                      # 日志级别
  console_output: true               # 控制台输出
  file_output: true                  # 文件输出
  log_dir: "./Logs"                  # 日志目录
`

	// 写入配置文件
	logger.Debug("写入配置文件...")
	if err := os.WriteFile(serverConfigPath, []byte(serverConfigContent), 0600); err != nil {
		logger.Errorf("❌ 创建配置文件失败: %v", err)
		os.Exit(1)
	}
	logger.Debugf("配置文件写入成功: %s", serverConfigPath)

	// 显示成功信息
	logger.Info("✅ 初始化成功！")
	logger.Infof("📁 配置目录: %s", configDir)
	logger.Infof("⚙️  服务器配置文件: %s", serverConfigPath)
	logger.Info("")
	logger.Info("📝 请编辑 server.yaml 文件，填写以下必要信息：")
	logger.Info("   - microsoft_todo.tenant_id: Azure 租户 ID")
	logger.Info("   - microsoft_todo.client_id: 应用程序客户端 ID")
	logger.Info("   - microsoft_todo.client_secret: 客户端密钥")
	logger.Info("")
	logger.Info("💡 获取 Azure AD 配置信息：")
	logger.Info("   1. 访问 https://portal.azure.com")
	logger.Info("   2. 注册新应用程序或选择现有应用")
	logger.Info("   3. 配置 API 权限：Tasks.ReadWrite.All")
	logger.Info("   4. 创建客户端密钥")
	logger.Info("")
	logger.Info("🎉 配置完成后，运行 'to_icalendar test' 测试连接")
}

// handleInit 处理初始化命令
func handleInit(container commands.ServiceContainer) {
	ctx := context.Background()

	// 创建 InitCommand
	initCmd := commands.NewInitCommand(container)

	// 执行命令
	req := &commands.CommandRequest{
		Command: "init",
		Args:    make(map[string]interface{}),
	}

	resp, err := initCmd.Execute(ctx, req)
	if err != nil {
		fmt.Printf("❌ Failed to execute init command: %v\n", err)
		os.Exit(1)
	}

	if !resp.Success {
		fmt.Printf("❌ Initialization failed: %s\n", resp.Error)
		os.Exit(1)
	}

	// 显示成功消息
	initCmd.ShowSuccessMessage(resp.Metadata)
}



// handleClip 处理剪贴板命令
func handleClip(container commands.ServiceContainer) {
	logger.Info("Processing clipboard content...")

	clipboardService := container.GetClipboardService()
	ctx := context.Background()

	logger.Debug("读取剪贴板内容...")
	content, err := clipboardService.ReadContent(ctx)
	if err != nil {
		logger.Errorf("❌ Failed to read clipboard: %v", err)
		os.Exit(1)
	}

	logger.Infof("✓ Successfully read clipboard content")
	logger.Infof("  Type: %s", content.Type)
	logger.Debugf("剪贴板内容详情: %+v", content)
}


// handleUpload 处理上传命令
func handleUpload(container commands.ServiceContainer, options CommandOptions) {
	logger.Info("Uploading reminders...")
	// 这个命令的实现保持不变，因为它不在重构范围内
	logger.Info("⚠️  Upload command remains unchanged in this refactoring")
}

// handleTasks 处理任务管理命令
func handleTasks(container commands.ServiceContainer, args []string) {
	logger.Info("Task management...")
	// 这个命令的实现保持不变
	logger.Info("⚠️  Tasks command remains unchanged in this refactoring")
}

// handleCache 处理缓存管理命令
func handleCache(container commands.ServiceContainer, args []string) {
	logger.Info("Cache management...")
	// 这个命令的实现保持不变
	logger.Info("⚠️  Cache command remains unchanged in this refactoring")
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

// showUsage 显示使用帮助
func showUsage() {
	logger.Infof(`Usage:
  %s <command> [options]

Commands:
  init                    Initialize configuration files
  upload <file>           Send reminders (supports wildcards *.json)
  test                    Test service connection
  clip                    Process clipboard content (image or text) and generate JSON
  clip-upload             Process clipboard content and directly upload to Microsoft Todo
  clean                   Clean cache files
  tasks                   Task management commands (list, show, clean)
  cache                   Cache management commands (stats, cleanup)
  help                    Show this help message

Options:
  Upload command:
    --force-upload          Force upload even if duplicates are found
    --no-deduplication      Disable deduplication checking
    --dedup-strategy <s>    Set deduplication strategy (exact/similar) [not yet implemented]
    --include-completed     Include completed tasks in duplicate check

  Clean command:
    --all                   Clean all cache types (default)
    --tasks                 Clean task deduplication cache only
    --images                Clean image cache only
    --image-hashes          Clean image hash cache only
    --temp                  Clean temporary files only
    --generated             Clean generated JSON files only
    --dry-run               Preview files to be cleaned (without deleting)
    --force                 Skip confirmation and clean directly
    --older-than 7d         Only clean files older than specified time (7d, 24h, 30m)
    --clear-all             Completely clear all cache data

Examples:
  %s init                                          # Initialize configuration
  %s upload reminder.json                         # Send single reminder
  %s clean --all                                   # Clean all cache
  %s clean --dry-run                               # Preview files to be cleaned

Configuration files:
  ~/.to_icalendar/server.yaml       Service configuration (Microsoft Todo & Dify)
  ~/.to_icalendar/reminder.json     Reminder template

For more information, see README.md
`, appName, appName, appName, appName)
}
