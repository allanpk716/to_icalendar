package dify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/allanpk716/to_icalendar/internal/models"
	"github.com/go-resty/resty/v2"
)

// Client represents a Dify API client
type Client struct {
	apiEndpoint string
	apiKey      string
	timeout     time.Duration
	httpClient  *resty.Client
}

// NewDifyClient creates a new Dify API client
func NewDifyClient(config *models.DifyConfig) *Client {
	timeout := time.Duration(config.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second // Default timeout
	}

	client := resty.New().
		SetTimeout(timeout).
		SetRetryCount(3).
		SetRetryWaitTime(2*time.Second).
		SetRetryMaxWaitTime(10*time.Second)

	return &Client{
		apiEndpoint: config.APIEndpoint,
		apiKey:      config.APIKey,
		timeout:     timeout,
		httpClient:  client,
	}
}

// ProcessText processes text content using Dify API
func (c *Client) ProcessText(ctx context.Context, text string, userID string) (*models.DifyResponse, error) {
	if text == "" {
		return nil, fmt.Errorf("text content cannot be empty")
	}

	// Create request payload
	request := models.DifyRequest{
		Inputs: map[string]interface{}{
			"text": text,
		},
		Query:            "请分析以下文本内容，提取任务信息（标题、描述、时间、优先级等），并按照JSON格式返回结构化的任务数据。如果无法识别为任务，请返回分析结果。",
		ResponseMode:     "blocking",
		User:             userID,
		AutoGenerateName: false,
	}

	// Make API request
	resp, err := c.makeRequest(ctx, "/chat-messages", request)
	if err != nil {
		return nil, fmt.Errorf("failed to process text: %w", err)
	}

	// Parse response
	var difyResp models.DifyResponse
	if err := json.Unmarshal(resp, &difyResp); err != nil {
		return nil, fmt.Errorf("failed to parse Dify response: %w", err)
	}

	return &difyResp, nil
}

// ProcessImage processes image content using Dify workflow API
func (c *Client) ProcessImage(ctx context.Context, imageData []byte, fileName string, userID string) (*models.DifyResponse, error) {
	log.Printf("[DifyClient] 开始处理图片: %s, 大小: %d bytes, 用户ID: %s", fileName, len(imageData), userID)
	log.Printf("[DifyClient] API 端点: %s", c.apiEndpoint)

	if len(imageData) == 0 {
		return nil, fmt.Errorf("image data cannot be empty")
	}

	// 按照正确的流程：先上传文件，再运行工作流
	log.Printf("[DifyClient] 开始上传文件...")
	fileID, err := c.uploadFile(ctx, imageData, fileName, userID)
	if err != nil {
		return nil, fmt.Errorf("文件上传失败: %w", err)
	}

	log.Printf("[DifyClient] 文件上传成功，ID: %s", fileID)
	log.Printf("[DifyClient] 开始运行工作流...")

	// 使用文件ID运行工作流
	difyResp, err := c.runWorkflowWithFile(ctx, fileID, userID)
	if err != nil {
		return nil, fmt.Errorf("工作流运行失败: %w", err)
	}

	log.Printf("[DifyClient] 工作流运行成功")
	return difyResp, nil
}

// uploadFile 上传文件到 Dify
func (c *Client) uploadFile(ctx context.Context, fileData []byte, fileName string, userID string) (string, error) {
	log.Printf("[DifyClient] 上传文件: %s, 大小: %d bytes", fileName, len(fileData))

	// 创建 multipart form data
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// 添加文件
	part, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		return "", fmt.Errorf("创建文件字段失败: %w", err)
	}

	_, err = io.Copy(part, bytes.NewReader(fileData))
	if err != nil {
		return "", fmt.Errorf("写入文件数据失败: %w", err)
	}

	// 添加其他字段
	if err := writer.WriteField("user", userID); err != nil {
		return "", fmt.Errorf("写入user字段失败: %w", err)
	}

	// 确定文件类型
	ext := strings.ToLower(filepath.Ext(fileName))
	fileType := "IMAGE" // 默认为图像类型
	switch ext {
	case ".txt":
		fileType = "TXT"
	case ".pdf":
		fileType = "PDF"
	case ".doc", ".docx":
		fileType = "WORD"
	}

	if err := writer.WriteField("type", fileType); err != nil {
		return "", fmt.Errorf("写入type字段失败: %w", err)
	}

	// 关闭 multipart writer
	err = writer.Close()
	if err != nil {
		return "", fmt.Errorf("关闭multipart writer失败: %w", err)
	}

	// 创建 HTTP 请求
	req, err := http.NewRequestWithContext(ctx, "POST", c.apiEndpoint+"/files/upload", &buf)
	if err != nil {
		return "", fmt.Errorf("创建上传请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	log.Printf("[DifyClient] 发送文件上传请求...")
	log.Printf("[DifyClient] Content-Type: %s", writer.FormDataContentType())

	// 发送请求
	httpClient := &http.Client{Timeout: c.timeout}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("发送上传请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取上传响应失败: %w", err)
	}

	log.Printf("[DifyClient] 上传响应状态码: %d", resp.StatusCode)
	log.Printf("[DifyClient] 上传响应内容: %s", string(body))

	// 检查状态码
	if resp.StatusCode != 201 { // 201 表示创建成功
		var errorResp models.DifyErrorResponse
		if err := json.Unmarshal(body, &errorResp); err == nil {
			return "", fmt.Errorf("文件上传失败: %s (code: %s)", errorResp.Message, errorResp.Code)
		}
		return "", fmt.Errorf("文件上传失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	// 解析响应获取文件 ID
	var uploadResp struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &uploadResp); err != nil {
		return "", fmt.Errorf("解析上传响应失败: %w", err)
	}

	if uploadResp.ID == "" {
		return "", fmt.Errorf("上传响应中未找到文件ID")
	}

	return uploadResp.ID, nil
}

// runWorkflowWithFile 使用文件ID运行工作流
func (c *Client) runWorkflowWithFile(ctx context.Context, fileID string, userID string) (*models.DifyResponse, error) {
	log.Printf("[DifyClient] 使用文件ID运行工作流: %s", fileID)

	// 构建请求数据，根据错误信息调整格式
	inputs := map[string]interface{}{
		"screenshot": map[string]interface{}{
			"transfer_method": "local_file",
			"upload_file_id":  fileID,
			"type":           "image",
		},
	}

	request := models.DifyImageRequest{
		Inputs:           inputs,
		ResponseMode:     "blocking",
		User:             userID,
		AutoGenerateName: false,
	}

	log.Printf("[DifyClient] 工作流请求数据: %+v", inputs)

	// 发送工作流请求
	response, err := c.httpClient.R().
		SetContext(ctx).
		SetHeader("Authorization", "Bearer "+c.apiKey).
		SetHeader("Content-Type", "application/json").
		SetBody(request).
		Post(c.apiEndpoint + "/workflows/run")

	if err != nil {
		return nil, fmt.Errorf("发送工作流请求失败: %w", err)
	}

	log.Printf("[DifyClient] 工作流响应状态码: %d", response.StatusCode())
	log.Printf("[DifyClient] 工作流响应内容: %s", string(response.Body()))

	// 检查响应状态
	if response.StatusCode() != http.StatusOK {
		var errorResp models.DifyErrorResponse
		if err := json.Unmarshal(response.Body(), &errorResp); err == nil {
			return nil, fmt.Errorf("工作流执行失败: %s (code: %s)", errorResp.Message, errorResp.Code)
		}
		return nil, fmt.Errorf("工作流执行失败，状态码: %d, 响应: %s", response.StatusCode(), string(response.Body()))
	}

	// 解析响应
	var difyResp models.DifyResponse
	if err := json.Unmarshal(response.Body(), &difyResp); err != nil {
		return nil, fmt.Errorf("解析工作流响应失败: %w", err)
	}

	return &difyResp, nil
}

// makeRequest makes a generic API request to Dify
func (c *Client) makeRequest(ctx context.Context, endpoint string, payload interface{}) ([]byte, error) {
	// Prepare request
	request := c.httpClient.R().
		SetContext(ctx).
		SetHeader("Authorization", "Bearer "+c.apiKey).
		SetHeader("Content-Type", "application/json").
		SetBody(payload)

	// Make request
	response, err := request.Post(c.apiEndpoint + endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to make request to %s: %w", endpoint, err)
	}

	// Check response status
	if response.StatusCode() != http.StatusOK {
		var errorResp models.DifyErrorResponse
		if err := json.Unmarshal(response.Body(), &errorResp); err == nil {
			return nil, fmt.Errorf("Dify API error: %s (code: %s)", errorResp.Message, errorResp.Code)
		}
		return nil, fmt.Errorf("Dify API returned status %d: %s", response.StatusCode(), string(response.Body()))
	}

	return response.Body(), nil
}

// UploadFile uploads a file to Dify API
func (c *Client) UploadFile(ctx context.Context, fileData []byte, fileName string, userID string) (string, error) {
	// Create multipart form data
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add file
	part, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		return "", fmt.Errorf("failed to create form file: %w", err)
	}

	_, err = io.Copy(part, bytes.NewReader(fileData))
	if err != nil {
		return "", fmt.Errorf("failed to copy file data: %w", err)
	}

	// Add user
	_ = writer.WriteField("user", userID)

	// Close multipart writer
	err = writer.Close()
	if err != nil {
		return "", fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", c.apiEndpoint+"/files/upload", &buf)
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	// Make request
	httpClient := &http.Client{Timeout: c.timeout}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make HTTP request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("file upload failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response to get file ID
	var uploadResp struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &uploadResp); err != nil {
		return "", fmt.Errorf("failed to parse upload response: %w", err)
	}

	return uploadResp.ID, nil
}

// ValidateConfig validates the Dify client configuration
func (c *Client) ValidateConfig() error {
	if c.apiEndpoint == "" {
		return fmt.Errorf("Dify API endpoint is required")
	}
	if c.apiKey == "" {
		return fmt.Errorf("Dify API key is required")
	}

	// 测试 API 端点的连通性
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 发送 HEAD 请求测试端点可达性
	resp, err := c.httpClient.R().
		SetContext(ctx).
		Head(c.apiEndpoint)

	if err != nil {
		return fmt.Errorf("无法连接到 Dify API 端点 %s: %w", c.apiEndpoint, err)
	}

	// 检查响应状态码，允许 2xx 和 404（可能不支持 HEAD 方法）
	if resp.StatusCode() >= 300 && resp.StatusCode() != 404 {
		return fmt.Errorf("Dify API 端点返回错误状态码: %d", resp.StatusCode())
	}

	return nil
}

// GetConfig returns the current configuration
func (c *Client) GetConfig() map[string]interface{} {
	return map[string]interface{}{
		"api_endpoint": c.apiEndpoint,
		"timeout":      c.timeout.String(),
	}
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}