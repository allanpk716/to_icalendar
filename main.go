package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/allanpk716/to_icalendar/internal/caldav"
	"github.com/allanpk716/to_icalendar/internal/config"
	"github.com/allanpk716/to_icalendar/internal/crypto"
	"github.com/allanpk716/to_icalendar/internal/ical"
	"github.com/allanpk716/to_icalendar/internal/models"
)

const (
	version = "1.0.0"
	appName = "to_icalendar"
)

func main() {
	fmt.Printf("%s v%s - CalDAV提醒事项发送工具\n", appName, version)

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
	case "list":
		handleList()
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
		fmt.Println("  请编辑此文件，填入您的Apple ID和CalDAV服务器地址")
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
	fmt.Println("1. 编辑 config/server.yaml 配置您的Apple ID")
	fmt.Println("2. 修改 config/reminder.json 或创建新的提醒事项文件")
	fmt.Println("3. 运行 'to_icalendar upload config/reminder.json' 上传提醒事项")
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

	// 加载密码
	password, err := loadPassword(serverConfig.CalDAV.Username)
	if err != nil {
		log.Fatalf("密码加载失败: %v", err)
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

	fmt.Printf("准备上传 %d 个提醒事项...\n", len(reminders))

	// 创建CalDAV客户端
	caldavClient := caldav.NewCalDAVClient(serverConfig.CalDAV.ServerURL, serverConfig.CalDAV.Username, password)

	// 测试连接
	fmt.Println("测试CalDAV连接...")
	err = caldavClient.TestConnection()
	if err != nil {
		log.Fatalf("CalDAV连接测试失败: %v", err)
	}
	fmt.Println("✓ CalDAV连接成功")

	// 处理提醒事项
	icalCreator := ical.NewICalCreator()
	successCount := 0

	for i, reminder := range reminders {
		fmt.Printf("\n处理提醒事项 %d/%d: %s\n", i+1, len(reminders), reminder.Title)

		// 解析时间
		timezone, err := time.LoadLocation(serverConfig.CalDAV.Timezone)
		if err != nil {
			fmt.Printf("  ⚠️ 时区加载失败，使用UTC: %v\n", err)
			timezone = time.UTC
		}

		parsedReminder, err := models.ParseReminderTime(*reminder, timezone)
		if err != nil {
			fmt.Printf("  ❌ 时间解析失败: %v\n", err)
			continue
		}

		// 验证提醒事项
		err = icalCreator.ValidateReminder(parsedReminder)
		if err != nil {
			fmt.Printf("  ❌ 提醒事项验证失败: %v\n", err)
			continue
		}

		// 创建iCalendar
		cal, err := icalCreator.CreateVTODO(parsedReminder)
		if err != nil {
			fmt.Printf("  ❌ iCalendar创建失败: %v\n", err)
			continue
		}

		// 上传到CalDAV服务器
		err = caldavClient.UploadReminder(cal, parsedReminder)
		if err != nil {
			fmt.Printf("  ❌ 上传失败: %v\n", err)
			continue
		}

		fmt.Printf("  ✓ 上传成功 (截止时间: %s)\n", parsedReminder.DueTime.Format("2006-01-02 15:04"))
		successCount++
	}

	fmt.Printf("\n上传完成！成功: %d/%d\n", successCount, len(reminders))
}

// handleTest 处理测试命令
func handleTest() {
	fmt.Println("测试CalDAV连接...")

	// 加载服务器配置
	configManager := config.NewConfigManager()
	serverConfig, err := configManager.LoadServerConfig("config/server.yaml")
	if err != nil {
		log.Fatalf("加载服务器配置失败: %v", err)
	}

	// 加载密码
	password, err := loadPassword(serverConfig.CalDAV.Username)
	if err != nil {
		log.Fatalf("密码加载失败: %v", err)
	}

	// 创建CalDAV客户端
	caldavClient := caldav.NewCalDAVClient(serverConfig.CalDAV.ServerURL, serverConfig.CalDAV.Username, password)

	// 测试连接
	err = caldavClient.TestConnection()
	if err != nil {
		log.Fatalf("CalDAV连接测试失败: %v", err)
	}

	fmt.Println("✓ CalDAV连接成功")

	// 获取服务器信息
	serverInfo, err := caldavClient.GetServerInfo()
	if err != nil {
		fmt.Printf("⚠️ 获取服务器信息失败: %v\n", err)
	} else {
		fmt.Printf("✓ 服务器状态: %d\n", serverInfo.StatusCode)
		fmt.Printf("✓ 支持的方法: %s\n", serverInfo.SupportedMethods)
	}

	// 列出现有提醒事项
	reminders, err := caldavClient.ListReminders()
	if err != nil {
		fmt.Printf("⚠️ 列出提醒事项失败: %v\n", err)
	} else {
		fmt.Printf("✓ 现有提醒事项数量: %d\n", len(reminders))
	}
}

// handleList 处理列表命令
func handleList() {
	fmt.Println("获取提醒事项列表...")

	// 加载服务器配置
	configManager := config.NewConfigManager()
	serverConfig, err := configManager.LoadServerConfig("config/server.yaml")
	if err != nil {
		log.Fatalf("加载服务器配置失败: %v", err)
	}

	// 加载密码
	password, err := loadPassword(serverConfig.CalDAV.Username)
	if err != nil {
		log.Fatalf("密码加载失败: %v", err)
	}

	// 创建CalDAV客户端
	caldavClient := caldav.NewCalDAVClient(serverConfig.CalDAV.ServerURL, serverConfig.CalDAV.Username, password)

	// 获取提醒事项列表
	reminders, err := caldavClient.GetRemindersList()
	if err != nil {
		log.Fatalf("获取提醒事项列表失败: %v", err)
	}

	if len(reminders) == 0 {
		fmt.Println("没有找到提醒事项")
		return
	}

	fmt.Printf("找到 %d 个提醒事项:\n\n", len(reminders))
	for i, reminder := range reminders {
		fmt.Printf("%d. %s\n", i+1, reminder.Filename)
		fmt.Printf("   大小: %d 字节\n", reminder.Size)
		fmt.Printf("   修改时间: %s\n", reminder.ModTime.Format("2006-01-02 15:04:05"))
		fmt.Println()
	}
}

// loadPassword 加载或获取密码
func loadPassword(username string) (string, error) {
	dataDir := "data"
	passwordManager, err := crypto.NewPasswordManager(dataDir)
	if err != nil {
		return "", fmt.Errorf("创建密码管理器失败: %w", err)
	}
	defer passwordManager.ClearSensitiveData()

	// 检查是否有已保存的密码
	if passwordManager.HasSavedPassword() {
		fmt.Println("使用已保存的密码...")
		password, err := passwordManager.LoadPassword()
		if err != nil {
			return "", fmt.Errorf("加载已保存密码失败: %w", err)
		}
		return password, nil
	}

	// 首次运行，需要输入密码
	fmt.Printf("首次运行，请输入Apple ID '%s' 的APP专用密码:\n", username)
	fmt.Print("密码: ")
	password, err := readPassword()
	if err != nil {
		return "", fmt.Errorf("读取密码失败: %w", err)
	}

	if password == "" {
		return "", fmt.Errorf("密码不能为空")
	}

	// 保存密码
	err = passwordManager.SavePassword(password)
	if err != nil {
		fmt.Printf("⚠️ 密码保存失败: %v\n", err)
		fmt.Println("密码将不会被保存，下次运行时需要重新输入")
	} else {
		fmt.Println("✓ 密码已加密保存")
	}

	return password, nil
}

// readPassword 安全地读取密码
func readPassword() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	password, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(password), nil
}

// showUsage 显示使用说明
func showUsage() {
	fmt.Printf(`
用法:
  %s <command> [options]

命令:
  init                    初始化配置文件
  upload <file>           上传提醒事项 (支持通配符 *.json)
  test                    测试CalDAV连接
  list                    列出所有提醒事项
  help                    显示此帮助信息

示例:
  %s init                                          # 初始化配置
  %s upload config/reminder.json                  # 上传单个提醒事项
  %s upload reminders/*.json                      # 批量上传提醒事项
  %s test                                          # 测试连接
  %s list                                          # 列出提醒事项

配置文件:
  config/server.yaml       服务器配置 (Apple ID, CalDAV地址)
  config/reminder.json     提醒事项模板

更多信息请参考 README.md
`, appName, appName, appName, appName, appName, appName)
}