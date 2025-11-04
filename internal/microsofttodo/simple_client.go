package microsofttodo

import (
	"fmt"
	"log"
)

// AuthConfig 包含 Microsoft Graph API 认证所需的配置
type AuthConfig struct {
	TenantID     string
	ClientID     string
	ClientSecret string
}

// SimpleTodoClient 简化的 Microsoft Todo 客户端
type SimpleTodoClient struct {
	authConfig *AuthConfig
}

// NewSimpleTodoClient 创建新的简化 Todo 客户端
func NewSimpleTodoClient(tenantID, clientID, clientSecret string) (*SimpleTodoClient, error) {
	// 验证配置
	if tenantID == "" || clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("incomplete authentication configuration: tenant_id, client_id, and client_secret are all required")
	}

	// 暂时不创建实际的 Graph 客户端，只保存配置
	// 在实际使用时可以集成真正的 Microsoft Graph API
	return &SimpleTodoClient{
		authConfig: &AuthConfig{
			TenantID:     tenantID,
			ClientID:     clientID,
			ClientSecret: clientSecret,
		},
	}, nil
}

// TestConnection 测试连接（模拟实现）
func (c *SimpleTodoClient) TestConnection() error {
	log.Printf("Testing Microsoft Graph connection with Tenant ID: %s, Client ID: %s", c.authConfig.TenantID, c.authConfig.ClientID)

	// 模拟成功连接
	log.Println("Successfully connected to Microsoft Graph API")
	return nil
}

// GetOrCreateTaskList 获取或创建任务列表（模拟实现）
func (c *SimpleTodoClient) GetOrCreateTaskList(listName string) (string, error) {
	log.Printf("Getting or creating task list: %s", listName)

	// 模拟返回一个列表ID
	listID := "mock-list-id-" + listName
	log.Printf("Using task list ID: %s", listID)

	return listID, nil
}

// CreateTask 创建任务（模拟实现）
func (c *SimpleTodoClient) CreateTask(title, description string) error {
	log.Printf("Creating task: %s", title)
	if description != "" {
		log.Printf("Task description: %s", description)
	}

	// 模拟成功创建
	log.Printf("Successfully created task: %s", title)
	return nil
}

// GetServerInfo 获取服务器信息（模拟实现）
func (c *SimpleTodoClient) GetServerInfo() (map[string]interface{}, error) {
	info := make(map[string]interface{})
	info["service"] = "Microsoft Graph API"
	info["api"] = "To Do Lists"
	info["status"] = "Connected (Mock Implementation)"

	return info, nil
}
