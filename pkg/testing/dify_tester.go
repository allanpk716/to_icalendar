package testing

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// DifyTester Dify 服务测试器
type DifyTester struct {
	client *http.Client
}

// NewDifyTester 创建新的 Dify 测试器
func NewDifyTester() *DifyTester {
	return &DifyTester{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// ValidateConfig 验证 Dify 配置
func (dt *DifyTester) ValidateConfig(config *DifyConfig) error {
	if config.APIEndpoint == "" {
		return fmt.Errorf("Dify API endpoint is required")
	}
	if config.APIKey == "" {
		return fmt.Errorf("Dify API key is required")
	}
	return nil
}

// TestConnection 测试 Dify 服务连接
func (dt *DifyTester) TestConnection(config *DifyConfig) error {
	// 验证配置
	if err := dt.ValidateConfig(config); err != nil {
		return fmt.Errorf("Dify 配置验证失败: %w", err)
	}

	// 测试 API 端点的连通性
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 直接测试API端点，不添加/health路径（与主项目保持一致）
	testURL := strings.TrimSuffix(config.APIEndpoint, "/")

	// 创建HEAD请求测试端点可达性
	req, err := http.NewRequestWithContext(ctx, "HEAD", testURL, nil)
	if err != nil {
		return fmt.Errorf("无法构造测试请求: %w", err)
	}

	// 添加API密钥到请求头
	if config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+config.APIKey)
	}
	req.Header.Set("User-Agent", "to_icalendar_test/1.0")

	// 发送请求
	resp, err := dt.client.Do(req)
	if err != nil {
		return fmt.Errorf("网络连接失败: %w", err)
	}
	defer resp.Body.Close()

	// 关键：检查响应状态码，允许 2xx 和 404（可能不支持 HEAD 方法）
	// 这与主项目的行为完全一致
	if resp.StatusCode >= 300 && resp.StatusCode != 404 {
		return fmt.Errorf("Dify API 端点返回错误状态码: %d", resp.StatusCode)
	}

	return nil
}

// TestDifyService 完整的 Dify 服务测试（包含配置验证和连接测试）
func (dt *DifyTester) TestDifyService(config *DifyConfig) *TestItemResult {
	startTime := time.Now()
	result := &TestItemResult{
		Name:     "Dify 服务测试",
		Success:  false,
		Duration: 0,
	}

	// 检查Dify配置的完整性
	if config.APIEndpoint == "" && config.APIKey == "" {
		// Dify未配置不算失败，返回nil表示跳过
		result.Success = true
		result.Message = "Dify 服务未配置，跳过测试"
		result.Details = "Dify AI 服务为可选配置\n如需使用，请在配置文件中设置:\n- api_endpoint: Dify API端点地址\n- api_key: API密钥"
		result.Duration = time.Since(startTime)
		return result
	}

	// 检查部分配置缺失的情况
	missingFields := []string{}
	if config.APIEndpoint == "" {
		missingFields = append(missingFields, "APIEndpoint")
	}
	if config.APIKey == "" {
		missingFields = append(missingFields, "APIKey")
	}

	if len(missingFields) > 0 {
		result.Success = true // 部分配置也不算失败，只是提醒
		result.Message = "Dify 服务配置不完整，跳过测试"
		result.Details = "以下字段缺失:\n" + strings.Join(missingFields, "\n") + "\nDify为可选配置"
		result.Duration = time.Since(startTime)
		return result
	}

	// 检查占位符
	placeholderFields := []string{}
	if strings.Contains(strings.ToLower(config.APIEndpoint), "your_") ||
		strings.Contains(config.APIEndpoint, "YOUR_") {
		placeholderFields = append(placeholderFields, "APIEndpoint")
	}
	if strings.Contains(strings.ToLower(config.APIKey), "your_") ||
		strings.Contains(config.APIKey, "YOUR_") {
		placeholderFields = append(placeholderFields, "APIKey")
	}

	if len(placeholderFields) > 0 {
		result.Success = true // 占位符也不算失败，只是提醒
		result.Message = "Dify 服务包含占位符，跳过测试"
		result.Details = "以下字段仍为占位符:\n" + strings.Join(placeholderFields, "\n")
		result.Duration = time.Since(startTime)
		return result
	}

	// 执行实际的连接测试
	if err := dt.TestConnection(config); err != nil {
		result.Error = "Dify 服务连接测试失败：" + err.Error()
		result.Duration = time.Since(startTime)
		return result
	}

	result.Success = true
	result.Message = "Dify 服务连接测试成功"
	result.Details = "API端点: " + config.APIEndpoint + "\n连通性测试通过"
	result.Duration = time.Since(startTime)
	return result
}