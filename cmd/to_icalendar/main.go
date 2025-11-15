package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/allanpk716/to_icalendar/internal/clipboard"
	"github.com/allanpk716/to_icalendar/internal/cleanup"
	"github.com/allanpk716/to_icalendar/internal/config"
	"github.com/allanpk716/to_icalendar/internal/deduplication"
	"github.com/allanpk716/to_icalendar/internal/dify"
	"github.com/allanpk716/to_icalendar/internal/microsofttodo"
	"github.com/allanpk716/to_icalendar/internal/models"
	"github.com/allanpk716/to_icalendar/internal/processors"
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

	// 解析命令行参数
	if len(os.Args) < 2 {
		showUsage()
		os.Exit(1)
	}

	command := os.Args[1]
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
	case "help", "-h", "--help":
		showUsage()
	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		showUsage()
		os.Exit(1)
	}
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
		log.Fatalf("Failed to load server configuration: %v", err)
	}

	// Validate configuration
	if serverConfig == nil {
		log.Fatalf("Server configuration is nil")
	}

	if !validateMicrosoftTodoConfig(serverConfig) {
		log.Fatalf("No valid Microsoft Todo configuration found")
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

		// Initialize cache manager
		configDir, _ := getConfigDir()
		cacheDir := filepath.Join(configDir, "cache")
		cacheManager = deduplication.NewCacheManager(cacheDir, nil)

		// Initialize deduplicator (简化版 - 仅本地缓存)
		deduplicator = deduplication.NewDeduplicator(&dedupConfig, cacheManager)

		fmt.Printf("  - Local cache: %t\n", dedupConfig.EnableLocalCache)
		fmt.Printf("  - Remote query: 已禁用\n")
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
	fmt.Printf(`
Usage:
  %s <command> [options]

Commands:
  init                    Initialize configuration files
  upload <file>           Send reminders (supports wildcards *.json)
  test                    Test service connection
  clip                    Process clipboard content (image or text) and generate JSON
  clip-upload             Process clipboard content and directly upload to Microsoft Todo
  clean                   Clean cache files
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
`, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName)
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

	// Process based on content type
	switch contentType {
	case models.ContentTypeImage:
		fmt.Println("Processing image from clipboard...")
		imageData, err := clipboardManager.ReadImage()
		if err != nil {
			log.Fatalf("Failed to read image from clipboard: %v", err)
		}

		// Initialize Dify client and processor
		difyClient := dify.NewDifyClient(&serverConfig.Dify)

		// Create processing options with configuration from server config
		processingOptions := dify.DefaultProcessingOptions()
		processingOptions.DefaultRemindBefore = serverConfig.Reminder.DefaultRemindBefore

		difyProcessor := dify.NewProcessor(difyClient, "clip-upload-user", processingOptions)

		// Initialize image processor with deduplication
		imageProcessor, err := processors.NewImageProcessorWithDeduplication(difyProcessor, deduplicator)
		if err != nil {
			log.Fatalf("Failed to create image processor with deduplication: %v", err)
		}
		defer imageProcessor.Cleanup()

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
		log.Fatalf("Microsoft Graph connection test failed: %v", err)
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
		log.Fatalf("Failed to create task: %v", err)
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
