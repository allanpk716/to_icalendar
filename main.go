package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/allanpk716/to_icalendar/internal/config"
	"github.com/allanpk716/to_icalendar/internal/models"
	"github.com/allanpk716/to_icalendar/internal/pushcut"
)

const (
	version = "1.0.0"
	appName = "to_icalendar"
)

func main() {
	fmt.Printf("%s v%s - iOS提醒事项发送工具 (Pushcut)\n", appName, version)

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
		fmt.Printf("未知命令: %s\n\n", command)
		showUsage()
		os.Exit(1)
	}
}

// handleInit 处理初始化命令
func handleInit() {
	fmt.Println("初始化配置文件...")

	configManager := config.NewConfigManager()

	// 创建服务器配置文件
	serverConfigPath := "config/server.yaml"
	if _, err := os.Stat(serverConfigPath); os.IsNotExist(err) {
		err = configManager.CreateServerConfigTemplate(serverConfigPath)
		if err != nil {
			log.Fatalf("创建服务器配置文件失败: %v", err)
		}
		fmt.Printf("✓ 已创建服务器配置文件: %s\n", serverConfigPath)
		fmt.Println("  请编辑此文件，填入您的Pushcut API密钥和Webhook ID")
	} else {
		fmt.Printf("✓ 服务器配置文件已存在: %s\n", serverConfigPath)
	}

	// 创建提醒事项模板文件
	reminderTemplatePath := "config/reminder.json"
	if _, err := os.Stat(reminderTemplatePath); os.IsNotExist(err) {
		err = configManager.CreateReminderTemplate(reminderTemplatePath)
		if err != nil {
			log.Fatalf("创建提醒事项模板失败: %v", err)
		}
		fmt.Printf("✓ 已创建提醒事项模板: %s\n", reminderTemplatePath)
		fmt.Println("  您可以基于此模板创建提醒事项JSON文件")
	} else {
		fmt.Printf("✓ 提醒事项模板已存在: %s\n", reminderTemplatePath)
	}

	fmt.Println("\n初始化完成！")
	fmt.Println("下一步:")
	fmt.Println("1. 编辑 config/server.yaml 配置您的Pushcut API密钥和Webhook ID")
	fmt.Println("2. 修改 config/reminder.json 或创建新的提醒事项文件")
	fmt.Println("3. 在iOS设备上安装Pushcut并配置快捷指令")
	fmt.Println("4. 运行 'to_icalendar upload config/reminder.json' 发送提醒事项")
}

// handleUpload 处理上传命令
func handleUpload() {
	if len(os.Args) < 3 {
		fmt.Println("请指定提醒事项文件路径")
		fmt.Println("用法: to_icalendar upload <reminder_file.json>")
		os.Exit(1)
	}

	reminderPath := os.Args[2]
	configManager := config.NewConfigManager()

	// 加载服务器配置
	serverConfig, err := configManager.LoadServerConfig("config/server.yaml")
	if err != nil {
		log.Fatalf("加载服务器配置失败: %v", err)
	}

	// 加载提醒事项
	var reminders []*models.Reminder
	if strings.Contains(reminderPath, "*") {
		// 批量处理
		reminders, err = configManager.LoadRemindersFromPattern(reminderPath)
	} else {
		// 单个文件
		reminder, err := configManager.LoadReminder(reminderPath)
		if err != nil {
			log.Fatalf("加载提醒事项失败: %v", err)
		}
		reminders = append(reminders, reminder)
	}

	if err != nil {
		log.Fatalf("加载提醒事项失败: %v", err)
	}

	fmt.Printf("准备发送 %d 个提醒事项到iOS设备...\n", len(reminders))

	// 创建Pushcut客户端
	pushcutClient := pushcut.NewPushcutClient(serverConfig.Pushcut.APIKey, serverConfig.Pushcut.WebhookID)

	// 测试连接
	fmt.Println("测试Pushcut连接...")
	err = pushcutClient.TestConnection()
	if err != nil {
		log.Fatalf("Pushcut连接测试失败: %v", err)
	}
	fmt.Println("✓ Pushcut连接成功")

	// 处理提醒事项
	successCount := 0

	for i, reminder := range reminders {
		fmt.Printf("\n处理提醒事项 %d/%d: %s\n", i+1, len(reminders), reminder.Title)

		// 解析时间
		timezone, err := time.LoadLocation(serverConfig.Pushcut.Timezone)
		if err != nil {
			fmt.Printf("  ⚠️ 时区加载失败，使用UTC: %v\n", err)
			timezone = time.UTC
		}

		parsedReminder, err := models.ParseReminderTime(*reminder, timezone)
		if err != nil {
			fmt.Printf("  ❌ 时间解析失败: %v\n", err)
			continue
		}

		// 发送到Pushcut
		err = pushcutClient.UploadReminder(parsedReminder)
		if err != nil {
			fmt.Printf("  ❌ 发送失败: %v\n", err)
			continue
		}

		fmt.Printf("  ✓ 发送成功 (截止时间: %s)\n", parsedReminder.DueTime.Format("2006-01-02 15:04"))
		successCount++
	}

	fmt.Printf("\n发送完成！成功: %d/%d\n", successCount, len(reminders))
}

// handleTest 处理测试命令
func handleTest() {
	fmt.Println("测试Pushcut连接...")

	// 加载服务器配置
	configManager := config.NewConfigManager()
	serverConfig, err := configManager.LoadServerConfig("config/server.yaml")
	if err != nil {
		log.Fatalf("加载服务器配置失败: %v", err)
	}

	// 创建Pushcut客户端
	pushcutClient := pushcut.NewPushcutClient(serverConfig.Pushcut.APIKey, serverConfig.Pushcut.WebhookID)

	// 测试连接
	err = pushcutClient.TestConnection()
	if err != nil {
		log.Fatalf("Pushcut连接测试失败: %v", err)
	}

	fmt.Println("✓ Pushcut连接成功")

	// 获取服务器信息
	serverInfo, err := pushcutClient.GetServerInfo()
	if err != nil {
		fmt.Printf("⚠️ 获取服务器信息失败: %v\n", err)
	} else {
		fmt.Printf("✓ 服务: %s\n", serverInfo.Service)
		fmt.Printf("✓ 状态码: %d\n", serverInfo.StatusCode)
		fmt.Printf("✓ 支持的方法: %s\n", serverInfo.SupportedMethods)
	}
}


// showUsage 显示使用说明
func showUsage() {
	fmt.Printf(`
用法:
  %s <command> [options]

命令:
  init                    初始化配置文件
  upload <file>           发送提醒事项到iOS (支持通配符 *.json)
  test                    测试Pushcut连接
  help                    显示此帮助信息

示例:
  %s init                                          # 初始化配置
  %s upload config/reminder.json                  # 发送单个提醒事项
  %s upload reminders/*.json                      # 批量发送提醒事项
  %s test                                          # 测试连接

配置文件:
  config/server.yaml       Pushcut配置 (API密钥, Webhook ID)
  config/reminder.json     提醒事项模板

使用说明:
  1. 在iOS设备上安装Pushcut应用
  2. 在Pushcut中创建接收提醒事项的快捷指令
  3. 配置Webhook API端点
  4. 编辑config/server.yaml填入API密钥和Webhook ID
  5. 运行upload命令发送提醒事项

更多信息请参考 README.md
`, appName, appName, appName, appName, appName)
}