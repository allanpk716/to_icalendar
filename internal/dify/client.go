package dify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
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
	if len(imageData) == 0 {
		return nil, fmt.Errorf("image data cannot be empty")
	}

	// Create multipart form data for workflow
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add inputs as JSON with screenshot field
	inputs := map[string]interface{}{
		"screenshot": fileName,
	}
	inputsJSON, _ := json.Marshal(inputs)
	_ = writer.WriteField("inputs", string(inputsJSON))

	// Add response mode
	_ = writer.WriteField("response_mode", "blocking")

	// Add user
	_ = writer.WriteField("user", userID)

	// Add auto_generate_name
	_ = writer.WriteField("auto_generate_name", "false")

	// Add file
	part, err := writer.CreateFormFile("files", fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

	_, err = io.Copy(part, bytes.NewReader(imageData))
	if err != nil {
		return nil, fmt.Errorf("failed to copy image data: %w", err)
	}

	// Close multipart writer
	err = writer.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Create HTTP request for workflow
	req, err := http.NewRequestWithContext(ctx, "POST", c.apiEndpoint+"/workflows/run", &buf)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	// Make request
	httpClient := &http.Client{Timeout: c.timeout}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make HTTP request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		var errorResp models.DifyErrorResponse
		if err := json.Unmarshal(body, &errorResp); err == nil {
			return nil, fmt.Errorf("Dify API error: %s (code: %s)", errorResp.Message, errorResp.Code)
		}
		return nil, fmt.Errorf("Dify API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var difyResp models.DifyResponse
	if err := json.Unmarshal(body, &difyResp); err != nil {
		return nil, fmt.Errorf("failed to parse Dify response: %w", err)
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
	return nil
}

// GetConfig returns the current configuration
func (c *Client) GetConfig() map[string]interface{} {
	return map[string]interface{}{
		"api_endpoint": c.apiEndpoint,
		"timeout":      c.timeout.String(),
	}
}