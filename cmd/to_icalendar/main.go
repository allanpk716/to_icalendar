package main

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/allanpk716/to_icalendar/internal/cache"
	"github.com/allanpk716/to_icalendar/internal/clipboard"
	"github.com/allanpk716/to_icalendar/internal/cleanup"
	"github.com/allanpk716/to_icalendar/internal/config"
	"github.com/allanpk716/to_icalendar/internal/deduplication"
	"github.com/allanpk716/to_icalendar/internal/dify"
	"github.com/allanpk716/to_icalendar/internal/logger"
	"github.com/allanpk716/to_icalendar/internal/microsofttodo"
	"github.com/allanpk716/to_icalendar/internal/models"
	"github.com/allanpk716/to_icalendar/internal/processors"
	"github.com/allanpk716/to_icalendar/internal/task"
)

const (
	version         = "1.0.0"
	appName         = "to_icalendar"
	configDirName   = ".to_icalendar"
	serverConfigFile = "server.yaml"
	reminderTemplateFile = "reminder.json"
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
	All          bool
	Tasks        bool
	Images       bool
	ImageHashes  bool
	Temp         bool
	Generated    bool
	DryRun       bool
	Force        bool
	OlderThan    string
	ClearAll     bool
}

// 全局变量
var unifiedCacheMgr *cache.UnifiedCacheManager

// getConfigDir 获取配置文件目录路径
func getConfigDir() (string, error) {
	// 尝试获取用户主目录
	usr, err := user.Current()
	if err != nil {
		// 如果无法获取用户目录，使用当前目录的子目录
		return configDirName, nil
	}

	configDir := filepath.Join(usr.HomeDir, configDirName)
	return configDir, nil
}

// initializeCacheSystem 初始化统一缓存系统
func initializeCacheSystem() (*cache.UnifiedCacheManager, error) {
	// 获取配置目录
	configDir, err := getConfigDir()
	if err != nil {
		return nil, fmt.Errorf("获取配置目录失败: %w", err)
	}

	// 创建统一缓存管理器
	unifiedCacheMgr, err := cache.NewUnifiedCacheManager(filepath.Join(configDir, "cache"), log.Default())
	if err != nil {
		return nil, fmt.Errorf("创建统一缓存管理器失败: %w", err)
	}

	// 检查是否需要迁移
	if err := performCacheMigration(unifiedCacheMgr); err != nil {
		log.Printf("缓存迁移失败: %v", err)
		// 迁移失败不应该阻止程序启动，只记录日志
	}

	log.Printf("缓存系统初始化完成，缓存目录: %s", unifiedCacheMgr.GetBaseCacheDir())
	return unifiedCacheMgr, nil
}

// performCacheMigration 执行缓存迁移
func performCacheMigration(unifiedCacheMgr *cache.UnifiedCacheManager) error {
	// 创建迁移管理器
	migrationMgr := cache.NewMigrationManager(unifiedCacheMgr, log.Default())

	// 检查是否需要迁移
	if !migrationMgr.HasLegacyCache() {
		return nil // 无需迁移
	}

	// 检查是否已经完成迁移
	if isMigrationCompleted(unifiedCacheMgr.GetBaseCacheDir()) {
		log.Println("检测到缓存已完成迁移，跳过")
		return nil
	}

	log.Println("🚀 检测到旧版缓存，开始自动迁移...")

	// 获取迁移计划
	plan := migrationMgr.GetMigrationPlan()
	if !plan.MigrationRequired {
		return nil
	}

	log.Printf("📦 发现 %d 个旧版缓存项目，总大小: %.2f MB",
		len(plan.Migrations), float64(plan.TotalSize)/(1024*1024))

	// 执行迁移
	options := &cache.MigrationOptions{
		DryRun:        false,
		Backup:        false, // 不需要备份，直接迁移
		DeleteSource:  true,
		SkipExisting:  true,
		ForceOverwrite: false,
	}

	result, err := migrationMgr.ExecuteMigration(plan, options)
	if err != nil {
		return fmt.Errorf("执行缓存迁移失败: %w", err)
	}

	if result.Success {
		log.Printf("✅ 缓存迁移完成，共迁移 %d 个项目", len(result.Migrated))

		// 标记迁移完成
		markMigrationCompleted(unifiedCacheMgr.GetBaseCacheDir())

		// 强制清理旧缓存目录
		forceCleanupLegacyDirs(plan.LegacyPaths)

	} else {
		log.Printf("⚠️  缓存迁移部分失败，成功: %d, 失败: %d",
			len(result.Migrated), len(result.Failed))
	}

	return nil
}

// isMigrationCompleted 检查是否已经完成迁移
func isMigrationCompleted(cacheBaseDir string) bool {
	migrationFile := filepath.Join(cacheBaseDir, ".migration_completed")
	_, err := os.Stat(migrationFile)
	return err == nil
}

// markMigrationCompleted 标记迁移完成
func markMigrationCompleted(cacheBaseDir string) error {
	migrationFile := filepath.Join(cacheBaseDir, ".migration_completed")
	return os.WriteFile(migrationFile, []byte(time.Now().Format(time.RFC3339)), 0644)
}

// cleanupEmptyLegacyDirs 清理空的旧版缓存目录
func cleanupEmptyLegacyDirs(legacyPaths *cache.LegacyCachePaths) {
	if legacyPaths == nil {
		return
	}

	// 要清理的目录列表
	dirsToCheck := []string{
		legacyPaths.ProgramRootCache,
		legacyPaths.ImageCache,
	}

	for _, dir := range dirsToCheck {
		if dir == "" {
			continue
		}

		if isEmpty, err := isDirEmpty(dir); err == nil && isEmpty {
			if err := os.RemoveAll(dir); err != nil {
				log.Printf("清理空目录失败: %s: %v", dir, err)
			} else {
				log.Printf("🧹 已清理空目录: %s", dir)
			}
		}
	}
}

// forceCleanupLegacyDirs 强制清理旧版缓存目录（即使非空）
func forceCleanupLegacyDirs(legacyPaths *cache.LegacyCachePaths) {
	if legacyPaths == nil {
		return
	}

	// 要强制清理的目录列表
	dirsToClean := []string{
		legacyPaths.ProgramRootCache,
		legacyPaths.ImageCache,
	}

	for _, dir := range dirsToClean {
		if dir == "" {
			continue
		}

		// 检查目录是否存在
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue // 目录不存在，无需清理
		}

		// 尝试删除目录
		if err := os.RemoveAll(dir); err != nil {
			log.Printf("⚠️  强制清理目录失败: %s: %v", dir, err)
		} else {
			log.Printf("🧹 强制清理旧缓存目录: %s", dir)
		}
	}

	// 也清理可能的旧缓存文件
	oldCacheFiles := []string{
		"./cache/submitted_tasks.json",
		"./cache/image_hashes.json",
	}

	for _, file := range oldCacheFiles {
		if _, err := os.Stat(file); err == nil {
			if err := os.Remove(file); err != nil {
				log.Printf("⚠️  清理旧缓存文件失败: %s: %v", file, err)
			} else {
				log.Printf("🧹 已清理旧缓存文件: %s", file)
			}
		}
	}
}

// isDirEmpty 检查目录是否为空
func isDirEmpty(dirPath string) (bool, error) {
	file, err := os.Open(dirPath)
	if err != nil {
		return false, err
	}
	defer file.Close()

	_, err = file.Readdirnames(1)
	if err == nil {
		return false, nil // 目录不为空
	}
	if err.Error() == "EOF" {
		return true, nil // 目录为空
	}
	return false, err // 其他错误
}

// ensureConfigDir 确保配置目录存在
func ensureConfigDir() (string, error) {
	configDir, err := getConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get config directory: %w", err)
	}

	// 创建配置目录（如果不存在）
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	return configDir, nil
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

func main() {
	fmt.Printf("%s v%s - Reminder sending tool (supports Microsoft Todo)\n", appName, version)

	// 初始化日志系统（使用默认配置）
	logger.Initialize(&models.LoggingConfig{
		Level:         "info",
		ConsoleOutput: true,
		FileOutput:    true,
		LogDir:        "config",
	})

	logger.Info("程序启动，版本: %s", version)
	logger.Debugf("命令行参数: %v", os.Args)

	// 初始化统一缓存系统（所有命令都需要的初始化）
	var err error
	unifiedCacheMgr, err = initializeCacheSystem()
	if err != nil {
		logger.Errorf("缓存系统初始化失败: %v", err)
		// 缓存初始化失败不应该阻止程序运行，只记录错误
	}

	// 解析命令行参数
	if len(os.Args) < 2 {
		showUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	logger.Infof("执行命令: %s", command)

	switch command {
	case "init":
		handleInit()
	case "upload":
		handleUpload(parseCommandOptions(os.Args[2:]))
	case "test":
		handleTest()
	case "clip":
		handleClip()
	case "clip-upload":
		handleClipUpload(parseCommandOptions(os.Args[2:]))
	case "clean":
		handleClean(parseCleanOptions(os.Args[2:]))
	case "tasks":
		handleTasks(os.Args[2:])
	case "cache":
		handleCache(os.Args[2:])
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

// validateMicrosoftTodoConfig validates Microsoft Todo configuration by checking
// if all required fields (TenantID, ClientID, ClientSecret, UserEmail) are present.
// Returns true if configuration is valid, false otherwise.
func validateMicrosoftTodoConfig(config *models.ServerConfig) bool {
	return config.MicrosoftTodo.TenantID != "" && config.MicrosoftTodo.ClientID != "" && config.MicrosoftTodo.ClientSecret != "" && config.MicrosoftTodo.UserEmail != ""
}

// handleInit handles the init command by creating configuration template files.
// It creates server.yaml and reminder.json templates if they don't exist.
func handleInit() {
	fmt.Println("Initializing configuration files...")

	// Ensure config directory exists
	configDir, err := ensureConfigDir()
	if err != nil {
		log.Fatalf("Failed to create config directory: %v", err)
	}

	fmt.Printf("✓ Config directory: %s\n", configDir)

	configManager := config.NewConfigManager()

	// Create server configuration file
	serverConfigPath := filepath.Join(configDir, serverConfigFile)
	if _, err := os.Stat(serverConfigPath); os.IsNotExist(err) {
		err = configManager.CreateServerConfigTemplate(serverConfigPath)
		if err != nil {
			log.Fatalf("Failed to create server config file: %v", err)
		}
		fmt.Printf("✓ Created server config file: %s\n", serverConfigPath)
		fmt.Println("  Please edit this file to configure Microsoft Todo and Dify:")
		fmt.Println("  - Fill in Tenant ID, Client ID, and Client Secret for Microsoft Todo")
		fmt.Println("  - Fill in Dify API endpoint and API key")
	} else {
		fmt.Printf("✓ Server config file already exists: %s\n", serverConfigPath)
	}

	// Create reminder template file
	reminderTemplatePath := filepath.Join(configDir, reminderTemplateFile)
	if _, err := os.Stat(reminderTemplatePath); os.IsNotExist(err) {
		err = configManager.CreateReminderTemplate(reminderTemplatePath)
		if err != nil {
			log.Fatalf("Failed to create reminder template: %v", err)
		}
		fmt.Printf("✓ Created reminder template: %s\n", reminderTemplatePath)
		fmt.Println("  You can create reminder JSON files based on this template")
	} else {
		fmt.Printf("✓ Reminder template already exists: %s\n", reminderTemplatePath)
	}

	fmt.Println("\nInitialization completed!")
	fmt.Println("Next steps:")
	fmt.Printf("1. Edit %s to configure Microsoft Todo and Dify:\n", serverConfigPath)
	fmt.Println("   - Configure Azure AD application information")
	fmt.Println("   - Configure Dify API settings")
	fmt.Printf("2. Modify %s or create new reminder files\n", reminderTemplatePath)
	fmt.Println("3. Run 'to_icalendar test' to test connection")
	fmt.Println("4. Run 'to_icalendar upload <reminder-file.json>' to send reminders")
	fmt.Println("5. Run 'to_icalendar clip-upload' to process clipboard content")
}

// handleUpload handles the upload command by sending reminders to Microsoft Todo.
// It loads reminder files, validates configuration, and processes each reminder.
func handleUpload(options CommandOptions) {
	// 过滤掉选项参数，找到实际的文件路径参数
	args := os.Args[2:]
	var reminderPath string
	for i, arg := range args {
		if !strings.HasPrefix(arg, "--") {
			reminderPath = arg
			// 移除选项参数，保留文件路径
			if i > 0 {
				args = args[i:]
			}
			break
		}
	}

	if reminderPath == "" {
		fmt.Println("Please specify reminder file path")
		fmt.Println("Usage: to_icalendar upload <reminder_file.json> [options]")
		fmt.Println("Options:")
		fmt.Println("  --force-upload         Force upload even if duplicates are found")
		fmt.Println("  --no-deduplication     Disable deduplication checking")
		fmt.Println("  --dedup-strategy <s>   Set deduplication strategy (exact/similar)")
		fmt.Println("  --include-completed    Include completed tasks in duplicate check")
		os.Exit(1)
	}

	// Validate and sanitize input path
	if strings.TrimSpace(reminderPath) == "" {
		log.Fatalf("Reminder file path cannot be empty")
	}

	// Clean the path to prevent directory traversal
	reminderPath = filepath.Clean(reminderPath)

	// Additional validation for dangerous patterns
	if strings.Contains(reminderPath, "..") {
		log.Fatalf("Invalid file path: directory traversal not allowed")
	}

	// Get config directory
	configDir, err := getConfigDir()
	if err != nil {
		log.Fatalf("Failed to get config directory: %v", err)
	}

	configManager := config.NewConfigManager()

	// Load server configuration
	serverConfigPath := filepath.Join(configDir, serverConfigFile)
	serverConfig, err := configManager.LoadServerConfig(serverConfigPath)
	if err != nil {
		logger.Fatalf("加载服务器配置失败: %v", err)
	}

	// 重新初始化日志系统（使用配置文件中的设置）
	if err := logger.Initialize(&serverConfig.Logging); err != nil {
		logger.Errorf("初始化日志系统失败: %v", err)
		// 继续使用默认配置
	}

	logger.Info("服务器配置加载成功")
	logger.Debugf("日志配置: level=%s, console=%t, file=%t",
		serverConfig.Logging.Level, serverConfig.Logging.ConsoleOutput, serverConfig.Logging.FileOutput)

	// Validate configuration
	if serverConfig == nil {
		logger.Fatalf("服务器配置为空")
	}

	if !validateMicrosoftTodoConfig(serverConfig) {
		logger.Fatalf("无效的 Microsoft Todo 配置")
	}

	// Load reminders
	var reminders []*models.Reminder
	if strings.Contains(reminderPath, "*") {
		// Batch processing
		reminders, err = configManager.LoadRemindersFromPattern(reminderPath)
		if err != nil {
			log.Fatalf("Failed to load reminders from pattern: %v", err)
		}
	} else {
		// Single file
		reminder, err := configManager.LoadReminder(reminderPath)
		if err != nil {
			log.Fatalf("Failed to load reminder: %v", err)
		}
		if reminder == nil {
			log.Fatalf("Loaded reminder is nil")
		}
		reminders = append(reminders, reminder)
	}

	if len(reminders) == 0 {
		log.Fatalf("No reminders found to process")
	}

	fmt.Printf("Preparing to send %d reminders...\n", len(reminders))

	// Display active options
	if options.ForceUpload {
		fmt.Println("⚠️ Force upload enabled - duplicates will be ignored")
	}
	if options.NoDeduplication {
		fmt.Println("⚠️ Deduplication disabled by command line option")
	}
	if options.DedupStrategy != "" {
		fmt.Printf("📊 Deduplication strategy: %s\n", options.DedupStrategy)
	}
	if options.IncludeCompleted {
		fmt.Println("📋 Including completed tasks in duplicate check")
	}

	// Process reminders
	handleMicrosoftTodoUpload(serverConfig, reminders, options)
}

// handleMicrosoftTodoUpload handles uploading reminders to Microsoft Todo.
// It creates a Todo client, tests connection, and processes each reminder.
func handleMicrosoftTodoUpload(serverConfig *models.ServerConfig, reminders []*models.Reminder, options CommandOptions) {
	fmt.Println("Using Microsoft Todo service...")

	// Create simplified Todo client
	todoClient, err := microsofttodo.NewSimpleTodoClient(
		serverConfig.MicrosoftTodo.TenantID,
		serverConfig.MicrosoftTodo.ClientID,
		serverConfig.MicrosoftTodo.ClientSecret,
		serverConfig.MicrosoftTodo.UserEmail,
	)
	if err != nil {
		log.Fatalf("Failed to create Microsoft Todo client: %v", err)
	}

	// Test connection
	fmt.Println("Testing Microsoft Graph connection...")
	err = todoClient.TestConnection()
	if err != nil {
		log.Fatalf("Microsoft Graph connection test failed: %v", err)
	}
	fmt.Println("✓ Microsoft Graph connection successful")

	// Initialize deduplication service
	var deduplicator *deduplication.Deduplicator
	var cacheManager *deduplication.CacheManager
	var unifiedCacheMgr *cache.UnifiedCacheManager

	// Apply command line options to configuration
	dedupConfig := serverConfig.Deduplication
	if options.NoDeduplication {
		dedupConfig.Enabled = false
	}
	if options.ForceUpload {
		dedupConfig.Enabled = false
	}
	if options.DedupStrategy != "" {
		// This would require modifying the deduplication logic to use different strategies
		// For now, we just log it
		fmt.Printf("  📊 Strategy override: %s (not yet implemented)\n", options.DedupStrategy)
	}
	if options.IncludeCompleted {
		dedupConfig.CheckIncompleteOnly = false
	}

	if dedupConfig.Enabled {
		fmt.Println("✓ Deduplication enabled")

		// Use the already initialized unified cache manager from main()
		if unifiedCacheMgr == nil {
			// 如果主函数初始化失败，创建一个新的
			configDir, _ := getConfigDir()
			var err error
			unifiedCacheMgr, err = cache.NewUnifiedCacheManager(filepath.Join(configDir, "cache"), log.Default())
			if err != nil {
				log.Fatalf("创建统一缓存管理器失败: %v", err)
			}
		}

		// Initialize cache manager with unified cache
		cacheManager = deduplication.NewCacheManager(unifiedCacheMgr.GetBaseCacheDir(), log.Default())

		// Initialize deduplicator (简化版 - 仅本地缓存)
		deduplicator = deduplication.NewDeduplicator(&dedupConfig, cacheManager)

		fmt.Printf("  - Local cache: %t\n", dedupConfig.EnableLocalCache)
		fmt.Printf("  - Remote query: 已禁用\n")
		fmt.Printf("  - 缓存目录: %s\n", unifiedCacheMgr.GetBaseCacheDir())
	} else {
		if options.NoDeduplication {
			fmt.Println("  ⚠️ Deduplication disabled by command line option")
		} else if options.ForceUpload {
			fmt.Println("  ⚠️ Deduplication disabled due to force upload")
		} else {
			fmt.Println("  ⚠️ Deduplication disabled in configuration")
		}
	}

	// Process reminders
	successCount := 0
	skippedCount := 0
	for i, reminder := range reminders {
		// Validate reminder data
		if reminder == nil {
			fmt.Printf("\nSkipping reminder %d/%d: reminder is nil\n", i+1, len(reminders))
			continue
		}

		// Validate required fields
		if strings.TrimSpace(reminder.Title) == "" {
			fmt.Printf("\nSkipping reminder %d/%d: title is empty\n", i+1, len(reminders))
			continue
		}

		if strings.TrimSpace(reminder.Date) == "" {
			fmt.Printf("\nSkipping reminder %d/%d: date is empty\n", i+1, len(reminders))
			continue
		}

		if strings.TrimSpace(reminder.Time) == "" {
			fmt.Printf("\nSkipping reminder %d/%d: time is empty\n", i+1, len(reminders))
			continue
		}

		fmt.Printf("\nProcessing reminder %d/%d: %s\n", i+1, len(reminders), reminder.Title)

		// Parse time with timezone validation
		var timezone *time.Location
		if serverConfig.MicrosoftTodo.Timezone == "" {
			fmt.Printf("  ⚠️ Timezone not configured, using UTC\n")
			timezone = time.UTC
		} else {
			timezone, err = time.LoadLocation(serverConfig.MicrosoftTodo.Timezone)
			if err != nil {
				fmt.Printf("  ⚠️ Failed to load timezone '%s', using UTC: %v\n", serverConfig.MicrosoftTodo.Timezone, err)
				timezone = time.UTC
			}
		}

		// 添加调试日志
		if reminder.RemindBefore != "" {
			fmt.Printf("  📝 用户设置的提醒时间: %s\n", reminder.RemindBefore)
		} else {
			fmt.Printf("  ⚠️  用户未设置提醒时间，将使用默认值\n")
		}

		parsedReminder, err := models.ParseReminderTimeWithConfig(*reminder, timezone, &serverConfig.Reminder)
		if err != nil {
			fmt.Printf("  ❌ Failed to parse time: %v\n", err)
			continue
		}

		// 添加结果日志
		fmt.Printf("  ✅ 最终提醒时间: %s (截止: %s)\n",
			parsedReminder.AlarmTime.Format("2006-01-02 15:04"),
			parsedReminder.DueTime.Format("2006-01-02 15:04"))

		// Deduplication check
		if deduplicator != nil {
			fmt.Printf("  🔍 Checking for duplicates...\n")
			dupResult, err := deduplicator.CheckDuplicate(parsedReminder)
		if err != nil {
			fmt.Printf("  ⚠️ Deduplication check failed: %v\n", err)
		} else if dupResult.IsDuplicate {
			fmt.Printf("  🚫 Duplicate detected: %s\n", dupResult.SkipReason)
			if dupResult.DuplicateType == "cache" {
				fmt.Printf("    → Skipping (found in local cache)\n")
				skippedCount++
				continue
			}
		} else {
			fmt.Printf("  ✅ No duplicates found\n")
		}
		}

		// Get or create task list
		listName := parsedReminder.Original.List
		if listName == "" {
			listName = "Default" // 使用默认列表名称
		}

		listID, err := todoClient.GetOrCreateTaskList(listName)
		if err != nil {
			fmt.Printf("  ❌ Failed to get or create task list '%s': %v\n", listName, err)
			continue
		}

		// Send to Microsoft Todo with full details
		err = todoClient.CreateTaskWithDetails(
			parsedReminder.Original.Title,
			parsedReminder.Description,
			listID,
			parsedReminder.DueTime,
			parsedReminder.AlarmTime,
			parsedReminder.Priority,
			serverConfig.MicrosoftTodo.Timezone,
		)
		if err != nil {
			fmt.Printf("  ❌ Failed to create task: %v\n", err)
			continue
		}

		fmt.Printf("  ✓ Created successfully (due: %s, reminder: %s)\n",
			parsedReminder.DueTime.Format("2006-01-02 15:04"),
			parsedReminder.AlarmTime.Format("2006-01-02 15:04"))

		// Record successful submission to cache
		if deduplicator != nil {
			if err := deduplicator.RecordSubmittedTask(parsedReminder, ""); err != nil {
				fmt.Printf("  ⚠️ Failed to record task to cache: %v\n", err)
			}
		}

		successCount++
	}

	// Show deduplication statistics
	if deduplicator != nil {
		stats := deduplicator.GetStats()
		fmt.Printf("\n📊 Deduplication Statistics:\n")
		fmt.Printf("  - Enabled: %t\n", stats["deduplication_enabled"])
		if cacheStats, ok := stats["cache_stats"].(map[string]interface{}); ok {
			fmt.Printf("  - Cached tasks: %v\n", cacheStats["total_tasks"])
			fmt.Printf("  - Recent tasks (24h): %v\n", cacheStats["recent_tasks_24h"])
		}
	}

	fmt.Printf("\nUpload completed! Success: %d/%d", successCount, len(reminders))
	if skippedCount > 0 {
		fmt.Printf(" (Skipped: %d duplicates)", skippedCount)
	}
	fmt.Printf("\n")
}


// handleTest handles the test command by validating Microsoft Todo configuration.
// It loads server configuration and tests the Microsoft Graph connection.
func handleTest() {
	// Get config directory
	configDir, err := getConfigDir()
	if err != nil {
		log.Fatalf("Failed to get config directory: %v", err)
	}

	// Load server configuration
	configManager := config.NewConfigManager()
	serverConfigPath := filepath.Join(configDir, serverConfigFile)
	serverConfig, err := configManager.LoadServerConfig(serverConfigPath)
	if err != nil {
		log.Fatalf("Failed to load server configuration: %v", err)
	}

	// Validate configuration
	if !validateMicrosoftTodoConfig(serverConfig) {
		log.Fatalf("No valid Microsoft Todo configuration found")
	}

	// Test Microsoft Graph connection
	fmt.Println("Testing Microsoft Graph connection...")
	testMicrosoftTodoConnection(serverConfig)
}

// testMicrosoftTodoConnection tests the Microsoft Graph API connection.
// It creates a Todo client and validates the connection, then displays server info.
func testMicrosoftTodoConnection(serverConfig *models.ServerConfig) {
	// Create simplified Todo client
	todoClient, err := microsofttodo.NewSimpleTodoClient(
		serverConfig.MicrosoftTodo.TenantID,
		serverConfig.MicrosoftTodo.ClientID,
		serverConfig.MicrosoftTodo.ClientSecret,
		serverConfig.MicrosoftTodo.UserEmail,
	)
	if err != nil {
		log.Fatalf("Failed to create Microsoft Todo client: %v", err)
	}

	// Test connection
	err = todoClient.TestConnection()
	if err != nil {
		log.Fatalf("Microsoft Graph connection test failed: %v", err)
	}

	fmt.Println("✓ Microsoft Graph connection successful")

	// Get server information
	serverInfo, err := todoClient.GetServerInfo()
	if err != nil {
		fmt.Printf("⚠️ Failed to get server info: %v\n", err)
	} else {
		fmt.Printf("✓ Service: %s\n", serverInfo["service"])
		fmt.Printf("✓ API: %s\n", serverInfo["api"])
		if status, ok := serverInfo["status"].(string); ok {
			fmt.Printf("✓ Status: %s\n", status)
		}
	}
}


// showUsage displays the usage information and command examples.
// It prints the help message with all available commands and their usage.
func showUsage() {
	fmt.Printf(`Usage:
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
  %s upload ~/.to_icalendar/reminder.json        # Send single reminder
  %s upload reminders/*.json                      # Send batch reminders
  %s upload reminder.json --force-upload         # Force upload, ignore duplicates
  %s upload reminder.json --no-deduplication     # Disable deduplication
  %s test                                          # Test connection
  %s clip                                          # Process clipboard and generate JSON
  %s clip-upload --force-upload                   # Process clipboard and upload, ignore duplicates
  %s clean --dry-run                               # Preview files to be cleaned
  %s clean --tasks --force                         # Force clean task cache
  %s clean --image-hashes --force                 # Force clean image hash cache
  %s clean --older-than 7d                         # Clean files older than 7 days
  %s clean --clear-all --force                     # Completely clear all cache data
  %s tasks list                                    # List recent tasks
  %s tasks show <task-id>                         # Show task details
  %s tasks clean <task-id>                        # Clean specific task
  %s cache stats                                   # Show cache statistics
  %s cache cleanup 30                              # Clean cache older than 30 days

Configuration files:
  ~/.to_icalendar/server.yaml       Service configuration (Microsoft Todo & Dify)
  ~/.to_icalendar/reminder.json     Reminder template

Deduplication:
  The application supports intelligent deduplication to avoid creating duplicate tasks:
  - Local cache for fast offline checking
  - Image SHA-256 hashing for visual content deduplication
  - Remote query to check Microsoft Todo for existing tasks
  - Similarity matching for near-duplicates
  - Only checks incomplete tasks by default (configurable)

Supported services:
  1. Microsoft Todo:
     - Register application in Azure AD
     - Configure API permissions (Tasks.ReadWrite.All)
     - Fill in Tenant ID, Client ID and Client Secret

Instructions:
  1. Run 'to_icalendar init' to initialize configuration files
  2. Edit ~/.to_icalendar/server.yaml to configure Microsoft Todo and Dify API
  3. Run 'to_icalendar test' to test connection
  4. Run 'to_icalendar upload' to send reminders
  5. Run 'to_icalendar clip' to process clipboard content and generate JSON
  6. Run 'to_icalendar clip-upload' to process clipboard and directly upload to Microsoft Todo

For more information, see README.md
`, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName)
}

// handleClip processes clipboard content (image or text) using Dify API
// and generates a JSON reminder file. It handles the complete workflow:
// 1. Load and validate server configuration
// 2. Initialize clipboard and Dify clients
// 3. Read content from clipboard
// 4. Process content using Dify API
// 5. Generate JSON reminder file
func handleClip() {
	fmt.Println("Starting clipboard processing...")

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Get config directory and load configuration
	configDir, err := getConfigDir()
	if err != nil {
		log.Fatalf("Failed to get config directory: %v", err)
	}

	configManager := config.NewConfigManager()
	serverConfigPath := filepath.Join(configDir, serverConfigFile)
	serverConfig, err := configManager.LoadServerConfig(serverConfigPath)
	if err != nil {
		log.Fatalf("Failed to load server configuration: %v", err)
	}

	// Validate configuration - need both Microsoft Todo and Dify configs
	if !validateMicrosoftTodoConfig(serverConfig) {
		log.Fatalf("No valid Microsoft Todo configuration found")
	}

	// Validate Dify configuration
	if err := serverConfig.Dify.Validate(); err != nil {
		log.Fatalf("Invalid Dify configuration: %v", err)
	}

	fmt.Println("✓ Configuration loaded successfully")

	// Initialize Dify client
	difyClient := dify.NewDifyClient(&serverConfig.Dify)

	// Create processing options with configuration from server config
	processingOptions := dify.DefaultProcessingOptions()
	processingOptions.DefaultRemindBefore = serverConfig.Reminder.DefaultRemindBefore

	// Initialize Dify processor
	difyProcessor := dify.NewProcessor(difyClient, "clipboard-user", processingOptions)

	// Initialize deduplication service (same as clip-upload)
	dedupConfig := serverConfig.Deduplication
	var deduplicator *deduplication.Deduplicator
	var cacheManager *deduplication.CacheManager

	if dedupConfig.Enabled {
		fmt.Println("✓ Deduplication enabled")

		// Initialize cache manager
		cacheDir := filepath.Join(configDir, "cache")
		cacheManager = deduplication.NewCacheManager(cacheDir, nil)

		// Initialize deduplicator (简化版 - 仅本地缓存)
		deduplicator = deduplication.NewDeduplicator(&dedupConfig, cacheManager)
	}

	// Initialize image processor with deduplication
	var imageProcessor *processors.ImageProcessor
	if deduplicator != nil {
		imageProcessor, err = processors.NewImageProcessorWithDeduplication(difyProcessor, deduplicator)
	} else {
		imageProcessor, err = processors.NewImageProcessor(difyProcessor)
	}
	if err != nil {
		log.Fatalf("Failed to create image processor: %v", err)
	}
	defer imageProcessor.Cleanup()

	// Initialize clipboard manager
	clipboardManager, err := clipboard.NewManager()
	if err != nil {
		log.Fatalf("Failed to initialize clipboard manager: %v", err)
	}

	// Read clipboard content
	fmt.Println("Reading clipboard content...")
	hasContent, err := clipboardManager.HasContent()
	if err != nil {
		log.Fatalf("Failed to check clipboard content: %v", err)
	}

	if !hasContent {
		log.Fatalf("No content found in clipboard")
	}

	// Get content type
	contentType, err := clipboardManager.GetContentType()
	if err != nil {
		log.Fatalf("Failed to determine clipboard content type: %v", err)
	}

	fmt.Printf("✓ Detected content type: %s\n", contentType)

	var processingResult *models.ProcessingResult

	// Process based on content type
	switch contentType {
	case models.ContentTypeImage:
		fmt.Println("Processing image from clipboard...")
		imageData, err := clipboardManager.ReadImage()
		if err != nil {
			log.Fatalf("Failed to read image from clipboard: %v", err)
		}

		result, err := imageProcessor.ProcessClipboardImage(ctx, imageData)
		if err != nil {
			log.Fatalf("Failed to process clipboard image: %v", err)
		}

		processingResult = result

	case models.ContentTypeText:
		fmt.Println("Processing text from clipboard...")
		text, err := clipboardManager.ReadText()
		if err != nil {
			log.Fatalf("Failed to read text from clipboard: %v", err)
		}

		if strings.TrimSpace(text) == "" {
			log.Fatalf("Clipboard text is empty")
		}

		fmt.Printf("Text content (first 100 chars): %s...\n", strings.TrimSpace(text)[:min(100, len(text))])

		// Process text using Dify
		difyResponse, err := difyProcessor.ProcessText(ctx, text)
		if err != nil {
			log.Fatalf("Failed to process text: %v", err)
		}

		// Convert to processing result
		processingResult = &models.ProcessingResult{
			Success:      difyResponse.Success,
			Reminder:     difyResponse.Reminder,
			ParsedInfo:   difyResponse.ParsedInfo,
			ErrorMessage: difyResponse.ErrorMessage,
		}

	default:
		log.Fatalf("Unsupported content type: %s", contentType)
	}

	// Check processing result
	if !processingResult.Success {
		log.Fatalf("Processing failed: %s", processingResult.ErrorMessage)
	}

	if processingResult.Reminder == nil {
		log.Fatalf("No reminder data generated from processing")
	}

	fmt.Println("\n✓ Content processed successfully")
	fmt.Printf("  Title: %s\n", processingResult.Reminder.Title)
	if processingResult.Reminder.Description != "" {
		fmt.Printf("  Description: %s\n", processingResult.Reminder.Description)
	}
	fmt.Printf("  Date: %s\n", processingResult.Reminder.Date)
	fmt.Printf("  Time: %s\n", processingResult.Reminder.Time)
	if processingResult.Reminder.RemindBefore != "" {
		fmt.Printf("  Remind Before: %s\n", processingResult.Reminder.RemindBefore)
	}
	fmt.Printf("  List: %s\n", processingResult.Reminder.List)

	// Generate JSON file
	fmt.Println("\nGenerating JSON file...")

	outputDir := "generated"
	jsonGenerator, err := processors.NewJSONGenerator(outputDir)
	if err != nil {
		log.Fatalf("Failed to create JSON generator: %v", err)
	}

	jsonFilePath, err := jsonGenerator.GenerateFromReminder(processingResult.Reminder)
	if err != nil {
		log.Fatalf("Failed to generate JSON file: %v", err)
	}

	fmt.Printf("\n✓ JSON file generated: %s\n", jsonFilePath)
	fmt.Println("\nNext steps:")
	fmt.Printf("1. Review the generated JSON file: %s\n", jsonFilePath)
	fmt.Println("2. Run 'to_icalendar upload " + jsonFilePath + "' to send to Microsoft Todo")
	fmt.Println("   OR manually upload to your todo application")
}

// handleClipUpload processes clipboard content and directly uploads to Microsoft Todo
// It handles the complete workflow: clipboard → Dify AI → Microsoft Todo upload
func handleClipUpload(options CommandOptions) {
	fmt.Println("Starting clipboard upload to Microsoft Todo...")

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Get config directory and load configuration
	configDir, err := getConfigDir()
	if err != nil {
		log.Fatalf("Failed to get config directory: %v", err)
	}

	configManager := config.NewConfigManager()
	serverConfigPath := filepath.Join(configDir, serverConfigFile)
	serverConfig, err := configManager.LoadServerConfig(serverConfigPath)
	if err != nil {
		log.Fatalf("Failed to load server configuration: %v", err)
	}

	// Validate configuration - need both Microsoft Todo and Dify configs
	if !validateMicrosoftTodoConfig(serverConfig) {
		log.Fatalf("No valid Microsoft Todo configuration found")
	}

	if err := serverConfig.Dify.Validate(); err != nil {
		log.Fatalf("Invalid Dify configuration: %v", err)
	}

	fmt.Println("✓ Configuration loaded successfully")

	// Initialize task manager and perform auto cleanup
	taskManager, err := task.NewTaskManager(configDir, serverConfig.Cache, log.Default())
	if err != nil {
		log.Fatalf("Failed to initialize task manager: %v", err)
	}

	// Perform automatic cleanup if enabled
	if serverConfig.Cache.CleanupOnStartup {
		fmt.Println("🧹 Performing automatic cache cleanup...")
		taskCleaner := task.NewTaskCleaner(taskManager, log.Default())
		cleanupResult, err := taskCleaner.AutoCleanup()
		if err != nil {
			log.Printf("Warning: Automatic cleanup failed: %v", err)
		} else if !cleanupResult.Skipped {
			fmt.Printf("✓ Cleanup completed: removed %d tasks, freed %.2f MB\n",
				cleanupResult.TasksCleaned, float64(cleanupResult.BytesFreed)/(1024*1024))
		}
	}

	// Create new task session for this upload
	taskSession, err := taskManager.CreateTaskSession()
	if err != nil {
		log.Fatalf("Failed to create task session: %v", err)
	}
	fmt.Printf("📝 Created task session: %s\n", taskSession.TaskID)

	// Initialize clipboard manager
	clipboardManager, err := clipboard.NewManager()
	if err != nil {
		log.Fatalf("Failed to initialize clipboard manager: %v", err)
	}

	// Read clipboard content
	fmt.Println("Reading clipboard content...")
	hasContent, err := clipboardManager.HasContent()
	if err != nil {
		log.Fatalf("Failed to check clipboard content: %v", err)
	}

	if !hasContent {
		log.Fatalf("No content found in clipboard")
	}

	// Get content type
	contentType, err := clipboardManager.GetContentType()
	if err != nil {
		log.Fatalf("Failed to determine clipboard content type: %v", err)
	}

	fmt.Printf("✓ Detected content type: %s\n", contentType)

	// Initialize deduplication service for clip-upload (before content processing)
	var deduplicator *deduplication.Deduplicator
	var cacheManager *deduplication.CacheManager

	// Apply command line options to configuration
	dedupConfig := serverConfig.Deduplication
	if options.NoDeduplication {
		dedupConfig.Enabled = false
	}
	if options.ForceUpload {
		dedupConfig.Enabled = false
	}
	if options.IncludeCompleted {
		dedupConfig.CheckIncompleteOnly = false
	}

	if dedupConfig.Enabled {
		fmt.Println("✓ Deduplication enabled")

		// Initialize cache manager
		configDir, _ := getConfigDir()
		cacheDir := filepath.Join(configDir, "cache")
		cacheManager = deduplication.NewCacheManager(cacheDir, nil)

		// Initialize deduplicator (简化版 - 仅本地缓存)
		deduplicator = deduplication.NewDeduplicator(&dedupConfig, cacheManager)
	} else {
		if options.NoDeduplication {
			fmt.Println("  ⚠️ Deduplication disabled by command line option")
		} else if options.ForceUpload {
			fmt.Println("  ⚠️ Deduplication disabled due to force upload")
		} else {
			fmt.Println("  ⚠️ Deduplication disabled in configuration")
		}
	}

	var processingResult *models.ProcessingResult
	var imageData []byte // Declare imageData outside switch to make it accessible later
	var imageProcessor *processors.ImageProcessor // Declare imageProcessor outside switch to make it accessible later

	// Process based on content type
	switch contentType {
	case models.ContentTypeImage:
		fmt.Println("Processing image from clipboard...")
		imageData, err = clipboardManager.ReadImage()
		if err != nil {
			// Update task session with error
			taskManager.UpdateTaskStatus(taskSession, task.TaskStatusFailed, fmt.Sprintf("Failed to read image from clipboard: %v", err))
			log.Fatalf("Failed to read image from clipboard: %v", err)
		}

		// Save original clipboard image to task session
		if err := taskManager.SaveFileToTask(taskSession, task.FileTypeClipboardOriginal, imageData); err != nil {
			log.Printf("Warning: Failed to save original image to task session: %v", err)
		}

		// Generate and set image hash in task session
		imageHash := generateImageHash(imageData)
		taskManager.SetImageHash(taskSession, imageHash)

		// Initialize Dify client and processor
		difyClient := dify.NewDifyClient(&serverConfig.Dify)

		// Create processing options with configuration from server config
		processingOptions := dify.DefaultProcessingOptions()
		processingOptions.DefaultRemindBefore = serverConfig.Reminder.DefaultRemindBefore

		difyProcessor := dify.NewProcessor(difyClient, "clip-upload-user", processingOptions)

		// Initialize image processor with deduplication
		imageProcessor, err = processors.NewImageProcessorWithDeduplication(difyProcessor, deduplicator)
		if err != nil {
			log.Fatalf("Failed to create image processor with deduplication: %v", err)
		}
		defer imageProcessor.Cleanup()

		// Process image without recording cache initially
		result, err := imageProcessor.ProcessClipboardImageWithCacheControl(ctx, imageData, false)
		if err != nil {
			log.Fatalf("Failed to process clipboard image: %v", err)
		}

		processingResult = result

	case models.ContentTypeText:
		fmt.Println("Processing text from clipboard...")
		text, err := clipboardManager.ReadText()
		if err != nil {
			log.Fatalf("Failed to read text from clipboard: %v", err)
		}

		if strings.TrimSpace(text) == "" {
			log.Fatalf("Clipboard text is empty")
		}

		fmt.Printf("Text content (first 100 chars): %s...\n", strings.TrimSpace(text)[:min(100, len(text))])

		// Initialize Dify client and processor
		difyClient := dify.NewDifyClient(&serverConfig.Dify)

		// Create processing options with configuration from server config
		processingOptions := dify.DefaultProcessingOptions()
		processingOptions.DefaultRemindBefore = serverConfig.Reminder.DefaultRemindBefore

		difyProcessor := dify.NewProcessor(difyClient, "clip-upload-user", processingOptions)

		// Process text using Dify
		difyResponse, err := difyProcessor.ProcessText(ctx, text)
		if err != nil {
			log.Fatalf("Failed to process text: %v", err)
		}

		// Convert to processing result
		processingResult = &models.ProcessingResult{
			Success:      difyResponse.Success,
			Reminder:     difyResponse.Reminder,
			ParsedInfo:   difyResponse.ParsedInfo,
			ErrorMessage: difyResponse.ErrorMessage,
		}

	default:
		log.Fatalf("Unsupported content type: %s", contentType)
	}

	// Check processing result
	if !processingResult.Success {
		log.Fatalf("Processing failed: %s", processingResult.ErrorMessage)
	}

	if processingResult.Reminder == nil {
		log.Fatalf("No reminder data generated from processing")
	}

	fmt.Println("\n✓ Content processed successfully")
	fmt.Printf("  Title: %s\n", processingResult.Reminder.Title)
	if processingResult.Reminder.Description != "" {
		fmt.Printf("  Description: %s\n", processingResult.Reminder.Description)
	}
	fmt.Printf("  Date: %s\n", processingResult.Reminder.Date)
	fmt.Printf("  Time: %s\n", processingResult.Reminder.Time)
	if processingResult.Reminder.RemindBefore != "" {
		fmt.Printf("  Remind Before: %s\n", processingResult.Reminder.RemindBefore)
	}
	fmt.Printf("  List: %s\n", processingResult.Reminder.List)

	// Create Microsoft Todo client and upload directly
	fmt.Println("\nUploading to Microsoft Todo...")

	todoClient, err := microsofttodo.NewSimpleTodoClient(
		serverConfig.MicrosoftTodo.TenantID,
		serverConfig.MicrosoftTodo.ClientID,
		serverConfig.MicrosoftTodo.ClientSecret,
		serverConfig.MicrosoftTodo.UserEmail,
	)
	if err != nil {
		log.Fatalf("Failed to create Microsoft Todo client: %v", err)
	}

	// Test connection
	fmt.Println("Testing Microsoft Graph connection...")
	err = todoClient.TestConnection()
	if err != nil {
		// Handle connection test failure gracefully
		fmt.Printf("\n❌ Microsoft Graph connection test failed: %v\n", err)

		// Provide specific guidance based on error type
		if strings.Contains(err.Error(), "authentication") || strings.Contains(err.Error(), "unauthorized") {
			fmt.Println("\n🔧 Authentication Error Detected:")
			fmt.Println("  Please check your Microsoft Todo configuration in config/server.yaml:")
			fmt.Println("  - tenant_id")
			fmt.Println("  - client_id")
			fmt.Println("  - client_secret")
			fmt.Println("  - user_email")
			fmt.Println("\n💡 You can test your configuration with: ./to_icalendar test")
			fmt.Println("   Or run: ./to_icalendar test-microsoft-todo")
		} else if strings.Contains(err.Error(), "connection") || strings.Contains(err.Error(), "network") {
			fmt.Println("\n🌐 Network Error Detected:")
			fmt.Println("  Please check your internet connection")
			fmt.Println("  Microsoft Graph API may be temporarily unavailable")
			fmt.Println("  Try again in a few minutes")
		} else {
			fmt.Println("\n⚠️ Connection Error:")
			fmt.Println("  Please verify your configuration and network connectivity")
		}

		fmt.Println("\n🔄 Note: Your image has NOT been cached due to this connection failure.")
		fmt.Println("   You can safely retry clip-upload after fixing the issue.")

		// Update task session with error
		taskManager.UpdateTaskStatus(taskSession, task.TaskStatusFailed,
			fmt.Sprintf("Microsoft Graph connection test failed: %v", err))

		// Exit with error code but don't cache the image
		os.Exit(1)
	}
	fmt.Println("✓ Microsoft Graph connection successful")

	// Parse reminder with timezone
	var timezone *time.Location
	if serverConfig.MicrosoftTodo.Timezone == "" {
		fmt.Printf("  ⚠️ Timezone not configured, using UTC\n")
		timezone = time.UTC
	} else {
		timezone, err = time.LoadLocation(serverConfig.MicrosoftTodo.Timezone)
		if err != nil {
			fmt.Printf("  ⚠️ Failed to load timezone '%s', using UTC: %v\n", serverConfig.MicrosoftTodo.Timezone, err)
			timezone = time.UTC
		}
	}

	// 添加调试日志
	if processingResult.Reminder.RemindBefore != "" {
		fmt.Printf("  📝 用户设置的提醒时间: %s\n", processingResult.Reminder.RemindBefore)
	} else {
		fmt.Printf("  ⚠️  用户未设置提醒时间，将使用默认值\n")
	}

	parsedReminder, err := models.ParseReminderTimeWithConfig(*processingResult.Reminder, timezone, &serverConfig.Reminder)
	if err != nil {
		log.Fatalf("Failed to parse reminder time: %v", err)
	}

	// 添加结果日志
	fmt.Printf("  ✅ 最终提醒时间: %s (截止: %s)\n",
		parsedReminder.AlarmTime.Format("2006-01-02 15:04"),
		parsedReminder.DueTime.Format("2006-01-02 15:04"))

	// Get or create task list
	listName := parsedReminder.Original.List
	if listName == "" {
		listName = "Default" // 使用默认列表名称
	}

	listID, err := todoClient.GetOrCreateTaskList(listName)
	if err != nil {
		log.Fatalf("Failed to get or create task list '%s': %v", listName, err)
	}

	// Apply command line options to configuration
	dedupConfig = serverConfig.Deduplication
	if options.NoDeduplication {
		dedupConfig.Enabled = false
	}
	if options.ForceUpload {
		dedupConfig.Enabled = false
	}
	if options.IncludeCompleted {
		dedupConfig.CheckIncompleteOnly = false
	}

	// Check for duplicates
	if deduplicator != nil {
		fmt.Printf("  🔍 Checking for duplicates...\n")
		dupResult, err := deduplicator.CheckDuplicate(parsedReminder)
		if err != nil {
			fmt.Printf("  ⚠️ Deduplication check failed: %v\n", err)
		} else if dupResult.IsDuplicate {
				fmt.Printf("  🚫 Duplicate detected: %s\n", dupResult.SkipReason)
				if dupResult.DuplicateType == "cache" {
					fmt.Printf("    → Skipping (found in local cache)\n")
					fmt.Println("\n❌ Clip-upload skipped due to duplicate task")
					fmt.Println("Use --force-upload to override if needed")
					return
				}
			} else {
				fmt.Printf("  ✅ No duplicates found\n")
			}
		} else {
			if options.NoDeduplication {
				fmt.Println("  ⚠️ Deduplication disabled by command line option")
			} else if options.ForceUpload {
				fmt.Println("  ⚠️ Deduplication disabled due to force upload")
			} else {
				fmt.Println("  ⚠️ Deduplication disabled in configuration")
			}
		}

	// Send to Microsoft Todo with full details
	fmt.Println("Creating Microsoft Todo task...")
	err = todoClient.CreateTaskWithDetails(
		parsedReminder.Original.Title,
		parsedReminder.Description,
		listID,
		parsedReminder.DueTime,
		parsedReminder.AlarmTime,
		parsedReminder.Priority,
		serverConfig.MicrosoftTodo.Timezone,
	)
	if err != nil {
		// Handle Microsoft Todo task creation failure gracefully
		fmt.Printf("\n❌ Failed to create Microsoft Todo task: %v\n", err)

		// Provide specific guidance based on error type
		if strings.Contains(err.Error(), "authentication") || strings.Contains(err.Error(), "unauthorized") {
			fmt.Println("\n🔧 Authentication Error Detected:")
			fmt.Println("  Please check your Microsoft Todo configuration in config/server.yaml:")
			fmt.Println("  - tenant_id")
			fmt.Println("  - client_id")
			fmt.Println("  - client_secret")
			fmt.Println("  - user_email")
			fmt.Println("\n💡 You can test your configuration with: ./to_icalendar test")
		} else if strings.Contains(err.Error(), "connection") || strings.Contains(err.Error(), "network") {
			fmt.Println("\n🌐 Network Error Detected:")
			fmt.Println("  Please check your internet connection")
			fmt.Println("  Microsoft Graph API may be temporarily unavailable")
		} else {
			fmt.Println("\n⚠️ Unknown Error:")
			fmt.Println("  Please check the error message above and try again")
		}

		fmt.Println("\n🔄 Note: Your image has NOT been cached due to this failure.")
		fmt.Println("   You can safely retry clip-upload after fixing the issue.")

		// Update task session with error
		taskManager.UpdateTaskStatus(taskSession, task.TaskStatusFailed,
			fmt.Sprintf("Microsoft Todo task creation failed: %v", err))

		// Exit with error code but don't cache the image
		os.Exit(1)
	}

	// Microsoft Todo task created successfully - now record image cache
	if contentType == models.ContentTypeImage && deduplicator != nil {
		// For images, record the cache now that Microsoft Todo task was created successfully
		if err := imageProcessor.RecordImageCache(imageData, processingResult, ""); err != nil {
			fmt.Printf("  ⚠️ Failed to record image cache: %v\n", err)
		} else {
			fmt.Printf("  ✅ Image cache recorded successfully\n")
		}
	}

	// Record successful submission to cache
	if deduplicator != nil {
		if err := deduplicator.RecordSubmittedTask(parsedReminder, ""); err != nil {
			fmt.Printf("  ⚠️ Failed to record task to cache: %v\n", err)
		}
	}

	fmt.Printf("✓ Successfully created task in Microsoft Todo!\n")
	fmt.Printf("  Title: %s\n", parsedReminder.Original.Title)
	fmt.Printf("  List: %s\n", listName)
	fmt.Printf("  Due: %s\n", parsedReminder.DueTime.Format("2006-01-02 15:04"))
	if parsedReminder.AlarmTime.Before(parsedReminder.DueTime) {
		fmt.Printf("  Reminder: %s\n", parsedReminder.AlarmTime.Format("2006-01-02 15:04"))
	}
	fmt.Printf("  Priority: %d\n", parsedReminder.Priority)

	fmt.Println("\n🎉 Clip-upload completed successfully!")
	fmt.Println("The task has been added to your Microsoft Todo list.")
}

// handleClean 处理清理缓存命令
func handleClean(options CleanOptions) {
	fmt.Println("🧹 开始清理缓存...")

	// 创建清理器
	cleaner := cleanup.NewCleaner()

	// 初始化必要的组件
	configManager := config.NewConfigManager()
	cleaner.SetConfig(configManager)

	// 尝试初始化缓存管理器
	cacheDir, _ := getConfigDir()
	cacheManager := deduplication.NewCacheManager(filepath.Join(cacheDir, "cache"), log.Default())
	cleaner.SetCacheManager(cacheManager)

	// 暂时跳过图片配置初始化，避免空指针问题
	// logger := logrus.New()
	// imageConfig := image.NewConfigManager(cacheDir, logger)
	// cleaner.SetImageConfig(imageConfig)

	// 准备清理选项
	cleanOptions := cleanup.CleanOptions{
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

	// 显示清理信息
	fmt.Printf("清理选项:\n")
	if cleanOptions.All {
		fmt.Printf("  - 清理所有缓存\n")
	} else {
		if cleanOptions.Tasks {
			fmt.Printf("  - 任务去重缓存\n")
		}
		if cleanOptions.Images {
			fmt.Printf("  - 图片处理缓存\n")
		}
		if cleanOptions.ImageHashes {
			fmt.Printf("  - 图片哈希缓存\n")
		}
		if cleanOptions.Temp {
			fmt.Printf("  - 临时文件\n")
		}
		if cleanOptions.Generated {
			fmt.Printf("  - 生成的JSON文件\n")
		}
	}
	if cleanOptions.DryRun {
		fmt.Printf("  - 预览模式（不会实际删除文件）\n")
	}
	if cleanOptions.OlderThan != "" {
		fmt.Printf("  - 仅清理超过 %s 的文件\n", cleanOptions.OlderThan)
	}
	if cleanOptions.ClearAll {
		fmt.Printf("  - 完全清空所有缓存数据\n")
	}

	// 如果不是预览模式且没有强制标志，询问确认
	if !cleanOptions.DryRun && !cleanOptions.Force {
		fmt.Printf("\n⚠️  这将删除缓存文件，是否继续？ [y/N]: ")
		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalf("读取用户输入失败: %v", err)
		}
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("清理操作已取消")
			return
		}
	}

	// 执行清理
	summary, err := cleaner.Clean(cleanOptions)
	if err != nil {
		log.Fatalf("清理失败: %v", err)
	}

	// 显示结果
	if cleanOptions.DryRun {
		summary.PrintPreview()
	} else {
		summary.PrintSummary()
		if summary.TotalFiles > 0 {
			fmt.Printf("\n✅ 清理完成！共删除 %d 个文件，释放 %s 空间\n",
				summary.TotalFiles, formatBytes(summary.TotalSize))
		} else {
			fmt.Printf("\nℹ️  没有找到需要清理的文件\n")
		}
	}
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

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// generateImageHash 生成图片数据的SHA256哈希
func generateImageHash(imageData []byte) string {
	hash := sha256.Sum256(imageData)
	return hex.EncodeToString(hash[:])
}

// handleTasks 处理任务管理命令
func handleTasks(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: to_icalendar tasks <command>")
		fmt.Println("Commands:")
		fmt.Println("  list [limit]     - List recent tasks")
		fmt.Println("  show <task-id>   - Show task details")
		fmt.Println("  clean <task-id>  - Clean specific task")
		return
	}

	command := args[0]
	configDir, err := getConfigDir()
	if err != nil {
		log.Fatalf("Failed to get config directory: %v", err)
	}

	// Load configuration
	configManager := config.NewConfigManager()
	serverConfigPath := filepath.Join(configDir, serverConfigFile)
	serverConfig, err := configManager.LoadServerConfig(serverConfigPath)
	if err != nil {
		log.Fatalf("Failed to load server configuration: %v", err)
	}

	// Create task manager
	taskManager, err := task.NewTaskManager(configDir, serverConfig.Cache, log.Default())
	if err != nil {
		log.Fatalf("Failed to create task manager: %v", err)
	}

	switch command {
	case "list":
		limit := 10 // default limit
		if len(args) > 1 {
			if l, err := fmt.Sscanf(args[1], "%d", &limit); err != nil || l != 1 {
				log.Fatalf("Invalid limit: %s", args[1])
			}
		}

		tasks, err := taskManager.GetRecentTasks(limit)
		if err != nil {
			log.Fatalf("Failed to get recent tasks: %v", err)
		}

		if len(tasks) == 0 {
			fmt.Println("No tasks found")
			return
		}

		fmt.Printf("Recent %d tasks:\n", len(tasks))
		fmt.Println("=====================================")
		for _, taskItem := range tasks {
			status := "✓"
			if taskItem.Status != task.TaskStatusSuccess {
				status = "✗"
			}
			fmt.Printf("%s %s - %s (%s)\n", taskItem.TaskID[:8], taskItem.Title, status, taskItem.StartTime.Format("2006-01-02 15:04:05"))
		}

	case "show":
		if len(args) < 2 {
			log.Fatalf("Task ID is required")
		}
		taskID := args[1]

		session, err := taskManager.GetTaskSession(taskID)
		if err != nil {
			log.Fatalf("Failed to get task session: %v", err)
		}

		fmt.Printf("Task Details: %s\n", session.TaskID)
		fmt.Println("=====================================")
		fmt.Printf("Status: %s\n", session.Status)
		fmt.Printf("Start Time: %s\n", session.StartTime.Format("2006-01-02 15:04:05"))
		if !session.EndTime.IsZero() {
			fmt.Printf("End Time: %s\n", session.EndTime.Format("2006-01-02 15:04:05"))
		}
		if session.Title != "" {
			fmt.Printf("Title: %s\n", session.Title)
		}
		if session.ImageHash != "" {
			fmt.Printf("Image Hash: %s\n", session.ImageHash[:16])
		}
		fmt.Printf("Dify Success: %t\n", session.DifySuccess)
		fmt.Printf("Todo Success: %t\n", session.TodoSuccess)
		if session.ErrorMessage != "" {
			fmt.Printf("Error: %s\n", session.ErrorMessage)
		}
		fmt.Printf("Files: %d\n", len(session.Files))

	case "clean":
		if len(args) < 2 {
			log.Fatalf("Task ID is required")
		}
		taskID := args[1]

		session, err := taskManager.GetTaskSession(taskID)
		if err != nil {
			log.Fatalf("Failed to get task session: %v", err)
		}

		// Remove task directory
		if err := os.RemoveAll(session.TaskDir); err != nil {
			log.Fatalf("Failed to remove task directory: %v", err)
		}

		fmt.Printf("Task %s cleaned successfully\n", taskID)

	default:
		fmt.Printf("Unknown tasks command: %s\n", command)
	}
}

// handleCache 处理缓存管理命令
func handleCache(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: to_icalendar cache <command>")
		fmt.Println("Commands:")
		fmt.Println("  stats            - Show cache statistics")
		fmt.Println("  cleanup [days]   - Manually cleanup cache")
		return
	}

	command := args[0]
	configDir, err := getConfigDir()
	if err != nil {
		log.Fatalf("Failed to get config directory: %v", err)
	}

	// Load configuration
	configManager := config.NewConfigManager()
	serverConfigPath := filepath.Join(configDir, serverConfigFile)
	serverConfig, err := configManager.LoadServerConfig(serverConfigPath)
	if err != nil {
		log.Fatalf("Failed to load server configuration: %v", err)
	}

	// Create task manager
	taskManager, err := task.NewTaskManager(configDir, serverConfig.Cache, log.Default())
	if err != nil {
		log.Fatalf("Failed to create task manager: %v", err)
	}

	// Create cleaner
	cleaner := task.NewTaskCleaner(taskManager, log.Default())

	switch command {
	case "stats":
		stats, err := cleaner.CleanupStatistics()
		if err != nil {
			log.Fatalf("Failed to get cache statistics: %v", err)
		}

		fmt.Println("Cache Statistics")
		fmt.Println("==================")
		fmt.Printf("Total Tasks: %d\n", stats.TaskCount)
		fmt.Printf("Recent Tasks (7 days): %d\n", stats.RecentTasks7Days)
		fmt.Printf("Recent Tasks (30 days): %d\n", stats.RecentTasks30Days)
		fmt.Printf("Total Size: %.2f MB\n", stats.GetSizeMB())
		fmt.Printf("Cache Files: %d\n", stats.CacheFiles)
		fmt.Printf("Cache Size: %.2f MB\n", stats.GetCacheSizeMB())

	case "cleanup":
		days := 30 // default
		if len(args) > 1 {
			if d, err := fmt.Sscanf(args[1], "%d", &days); err != nil || d != 1 {
				log.Fatalf("Invalid days: %s", args[1])
			}
		}

		fmt.Printf("Cleaning cache older than %d days...\n", days)
		result, err := cleaner.CleanupOlderThan(days)
		if err != nil {
			log.Fatalf("Failed to cleanup cache: %v", err)
		}

		if result.Skipped {
			fmt.Printf("Cleanup skipped: %s\n", result.Reason)
			return
		}

		fmt.Printf("Cleanup completed:\n")
		fmt.Printf("  Tasks cleaned: %d\n", result.TasksCleaned)
		fmt.Printf("  Space freed: %.2f MB\n", float64(result.BytesFreed)/(1024*1024))
		fmt.Printf("  Cache entries removed: %d\n", result.CacheEntriesRemoved)
		fmt.Printf("  Orphaned files cleaned: %d\n", result.OrphanedFilesCleaned)
		if len(result.Errors) > 0 {
			fmt.Printf("  Errors: %d\n", len(result.Errors))
		}

	default:
		fmt.Printf("Unknown cache command: %s\n", command)
	}
}
