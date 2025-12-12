package main

import (
	"fmt"
	"log"

	"github.com/allanpk716/to_icalendar/pkg/app"
	"github.com/allanpk716/to_icalendar/pkg/cache"
	"github.com/allanpk716/to_icalendar/pkg/logger"
	"github.com/allanpk716/to_icalendar/pkg/models"
)

func main() {
	// 初始化日志
	logger.InitLogger()

	// 创建配置目录（使用当前目录）
	configDir := "."

	// 创建默认配置
	serverConfig := &models.ServerConfig{
		MicrosoftTodo: models.MicrosoftTodoConfig{
			TenantID:    "test-tenant",
			ClientID:    "test-client",
			ClientSecret: "test-secret",
			UserEmail:   "test@example.com",
			Timezone:    "Asia/Shanghai",
		},
	}

	// 创建统一缓存管理器
	cacheManager, err := cache.NewUnifiedCacheManager(configDir)
	if err != nil {
		log.Fatalf("创建缓存管理器失败: %v", err)
	}

	// 创建服务容器
	serviceContainer := app.NewServiceContainer(configDir, serverConfig, cacheManager, logger.GetLogger())

	// 获取剪贴板服务
	clipboardService := serviceContainer.GetClipboardService()

	// 测试读取剪贴板内容
	fmt.Println("正在测试剪贴板读取功能...")

	content, err := clipboardService.ReadContent(nil)
	if err != nil {
		fmt.Printf("读取剪贴板失败: %v\n", err)
		return
	}

	// 显示结果
	if content != nil {
		fmt.Printf("成功读取剪贴板内容！\n")
		fmt.Printf("类型: %s\n", content.Type)

		switch content.Type {
		case models.ContentTypeImage:
			fmt.Printf("图片大小: %d bytes\n", len(content.Image))
			fmt.Printf("文件名: %s\n", content.FileName)
		case models.ContentTypeText:
			fmt.Printf("文本内容: %s\n", content.Text)
			fmt.Printf("文本长度: %d\n", len(content.Text))
		}
	} else {
		fmt.Println("剪贴板内容为空")
	}
}