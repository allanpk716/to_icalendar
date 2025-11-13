package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/allanpk716/to_icalendar/internal/clipboard"
	"github.com/allanpk716/to_icalendar/internal/config"
	"github.com/allanpk716/to_icalendar/internal/dify"
	"github.com/allanpk716/to_icalendar/internal/microsofttodo"
	"github.com/allanpk716/to_icalendar/internal/models"
	"github.com/allanpk716/to_icalendar/internal/processors"
)

const (
	version         = "1.0.0"
	appName         = "to_icalendar"
	configDir       = "config"
	serverConfigFile = "server.yaml"
	reminderTemplateFile = "reminder.json"
)

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
		handleUpload()
	case "test":
		handleTest()
	case "clip":
		handleClip()
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

	configManager := config.NewConfigManager()

	// Create server configuration file
	serverConfigPath := filepath.Join(configDir, serverConfigFile)
	if _, err := os.Stat(serverConfigPath); os.IsNotExist(err) {
		err = configManager.CreateServerConfigTemplate(serverConfigPath)
		if err != nil {
			log.Fatalf("Failed to create server config file: %v", err)
		}
		fmt.Printf("✓ Created server config file: %s\n", serverConfigPath)
		fmt.Println("  Please edit this file to configure Microsoft Todo:")
		fmt.Println("  - Fill in Tenant ID, Client ID, and Client Secret")
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
	fmt.Printf("1. Edit %s to configure Microsoft Todo:\n", serverConfigPath)
	fmt.Println("   - Configure Azure AD application information")
	fmt.Printf("2. Modify %s or create new reminder files\n", reminderTemplatePath)
	fmt.Println("3. Run 'to_icalendar test' to test connection")
	fmt.Println("4. Run 'to_icalendar upload config/reminder.json' to send reminders")
}

// handleUpload handles the upload command by sending reminders to Microsoft Todo.
// It loads reminder files, validates configuration, and processes each reminder.
func handleUpload() {
	if len(os.Args) < 3 {
		fmt.Println("Please specify reminder file path")
		fmt.Println("Usage: to_icalendar upload <reminder_file.json>")
		os.Exit(1)
	}

	reminderPath := os.Args[2]

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

	// Process reminders
	handleMicrosoftTodoUpload(serverConfig, reminders)
}

// handleMicrosoftTodoUpload handles uploading reminders to Microsoft Todo.
// It creates a Todo client, tests connection, and processes each reminder.
func handleMicrosoftTodoUpload(serverConfig *models.ServerConfig, reminders []*models.Reminder) {
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

	// Process reminders
	successCount := 0
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

		parsedReminder, err := models.ParseReminderTime(*reminder, timezone)
		if err != nil {
			fmt.Printf("  ❌ Failed to parse time: %v\n", err)
			continue
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
		successCount++
	}

	fmt.Printf("\nUpload completed! Success: %d/%d\n", successCount, len(reminders))
}


// handleTest handles the test command by validating Microsoft Todo configuration.
// It loads server configuration and tests the Microsoft Graph connection.
func handleTest() {
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
  help                    Show this help message

Examples:
  %s init                                          # Initialize configuration
  %s upload config/reminder.json                  # Send single reminder
  %s upload reminders/*.json                      # Send batch reminders
  %s test                                          # Test connection
  %s clip                                          # Process clipboard and generate JSON

Configuration files:
  config/server.yaml       Service configuration (Microsoft Todo & Dify)
  config/reminder.json     Reminder template

Supported services:
  1. Microsoft Todo:
     - Register application in Azure AD
     - Configure API permissions (Tasks.ReadWrite.All)
     - Fill in Tenant ID, Client ID and Client Secret

Instructions:
  1. Run 'to_icalendar init' to initialize configuration files
  2. Edit config/server.yaml to configure Microsoft Todo and Dify API
  3. Run 'to_icalendar test' to test connection
  4. Run 'to_icalendar upload' to send reminders
  5. Run 'to_icalendar clip' to process clipboard content

For more information, see README.md
`, appName, appName, appName, appName, appName, appName)
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

	// Load configuration
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

	// Initialize Dify processor
	difyProcessor := dify.NewProcessor(difyClient, "clipboard-user", dify.DefaultProcessingOptions())

	// Initialize image processor
	imageProcessor, err := processors.NewImageProcessor(difyProcessor)
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

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
