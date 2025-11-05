package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/allanpk716/to_icalendar/internal/config"
	"github.com/allanpk716/to_icalendar/internal/microsofttodo"
	"github.com/allanpk716/to_icalendar/internal/models"
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

		// Send to Microsoft Todo
		err = todoClient.CreateTask(parsedReminder.Original.Title, parsedReminder.Description, listID)
		if err != nil {
			fmt.Printf("  ❌ Failed to create task: %v\n", err)
			continue
		}

		fmt.Printf("  ✓ Created successfully (due: %s)\n", parsedReminder.DueTime.Format("2006-01-02 15:04"))
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
  help                    Show this help message

Examples:
  %s init                                          # Initialize configuration
  %s upload config/reminder.json                  # Send single reminder
  %s upload reminders/*.json                      # Send batch reminders
  %s test                                          # Test connection

Configuration files:
  config/server.yaml       Service configuration (Microsoft Todo)
  config/reminder.json     Reminder template

Supported services:
  1. Microsoft Todo:
     - Register application in Azure AD
     - Configure API permissions (Tasks.ReadWrite.All)
     - Fill in Tenant ID, Client ID and Client Secret

Instructions:
  1. Run 'to_icalendar init' to initialize configuration files
  2. Edit config/server.yaml to configure Microsoft Todo
  3. Run 'to_icalendar test' to test connection
  4. Run 'to_icalendar upload' to send reminders

For more information, see README.md
`, appName, appName, appName, appName, appName)
}
