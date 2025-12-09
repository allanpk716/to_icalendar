package testing

import "time"

// DifyConfig Dify 服务配置结构
type DifyConfig struct {
	APIEndpoint string `yaml:"api_endpoint"`
	APIKey      string `yaml:"api_key"`
	Timeout     int    `yaml:"timeout"`
}

// MicrosoftTodoConfig Microsoft Todo 服务配置结构
type MicrosoftTodoConfig struct {
	TenantID     string `yaml:"tenant_id"`
	ClientID     string `yaml:"client_id"`
	ClientSecret string `yaml:"client_secret"`
	Timezone     string `yaml:"timezone"`
}

// TestItemResult 测试项结果结构
type TestItemResult struct {
	Name      string                 `json:"name"`
	Success   bool                   `json:"success"`
	Message   string                 `json:"message"`
	Error     string                 `json:"error,omitempty"`
	ErrorType string                 `json:"error_type,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Duration  time.Duration          `json:"duration"`
}

// TestResult 测试结果结构
type TestResult struct {
	Items    []*TestItemResult `json:"items"`
	TestedAt time.Time         `json:"tested_at"`
}

// ConnectionTester 连接测试器接口
type ConnectionTester interface {
	TestConnection() error
}