package pushcut

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/allanpk716/to_icalendar/internal/models"
)

// PushcutClient Pushcut API客户端
type PushcutClient struct {
	apiKey    string
	webhookID string
	baseURL   string
	client    *http.Client
}

// NewPushcutClient 创建Pushcut客户端
func NewPushcutClient(apiKey, webhookID string) *PushcutClient {
	return &PushcutClient{
		apiKey:    apiKey,
		webhookID: webhookID,
		baseURL:   "https://api.pushcut.io/v1/execute",
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ReminderData 发送给Pushcut的提醒事项数据
type ReminderData struct {
	Title        string `json:"title"`
	Description  string `json:"description,omitempty"`
	Date         string `json:"date"`
	Time         string `json:"time"`
	Priority     string `json:"priority,omitempty"`
	List         string `json:"list,omitempty"`
	RemindBefore string `json:"remind_before,omitempty"`
}

// UploadReminder 上传提醒事项到Pushcut
func (pc *PushcutClient) UploadReminder(reminder *models.ParsedReminder) error {
	// 构建发送数据
	data := ReminderData{
		Title:        reminder.Original.Title,
		Description:  reminder.Original.Description,
		Date:         reminder.Original.Date,
		Time:         reminder.Original.Time,
		Priority:     string(reminder.Original.Priority),
		List:         reminder.Original.List,
		RemindBefore: reminder.Original.RemindBefore,
	}

	// 序列化为JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal reminder data: %w", err)
	}

	log.Printf("DEBUG: Sending reminder data to Pushcut: %s", string(jsonData))

	// 构建请求URL
	url := fmt.Sprintf("%s/%s", pc.baseURL, pc.webhookID)

	// 创建HTTP请求
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+pc.apiKey)
	req.Header.Set("User-Agent", "to_icalendar/1.0")

	log.Printf("DEBUG: Sending request to: %s", url)
	log.Printf("DEBUG: Request headers: %+v", req.Header)

	// 发送请求
	resp, err := pc.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	log.Printf("DEBUG: Response status: %d", resp.StatusCode)
	log.Printf("DEBUG: Response body: %s", string(body))

	// 检查响应状态
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("Pushcut API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	log.Printf("INFO: Successfully sent reminder to Pushcut: %s", reminder.Original.Title)
	return nil
}

// TestConnection 测试Pushcut连接
func (pc *PushcutClient) TestConnection() error {
	// 构建测试数据
	testData := ReminderData{
		Title:        "连接测试",
		Description:  "这是一个来自to_icalendar的连接测试",
		Date:         "2024-12-25",
		Time:         "10:00",
		Priority:     "medium",
		List:         "测试",
		RemindBefore: "5m",
	}

	// 序列化为JSON
	jsonData, err := json.Marshal(testData)
	if err != nil {
		return fmt.Errorf("failed to marshal test data: %w", err)
	}

	log.Printf("DEBUG: Testing Pushcut connection with data: %s", string(jsonData))

	// 构建请求URL
	url := fmt.Sprintf("%s/%s", pc.baseURL, pc.webhookID)

	// 创建HTTP请求
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create test request: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+pc.apiKey)
	req.Header.Set("User-Agent", "to_icalendar/1.0")

	log.Printf("DEBUG: Sending test request to: %s", url)

	// 发送请求
	resp, err := pc.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send test request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read test response: %w", err)
	}

	log.Printf("DEBUG: Test response status: %d", resp.StatusCode)
	log.Printf("DEBUG: Test response body: %s", string(body))

	// 检查响应状态
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("Pushcut connection test failed: status %d, body: %s", resp.StatusCode, string(body))
	}

	log.Printf("INFO: Pushcut connection test successful")
	return nil
}

// GetServerInfo 获取Pushcut服务器信息（简化版本）
func (pc *PushcutClient) GetServerInfo() (*ServerInfo, error) {
	// 构建请求URL
	url := fmt.Sprintf("%s/%s", pc.baseURL, pc.webhookID)

	// 创建HTTP请求
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create info request: %w", err)
	}

	// 设置请求头
	req.Header.Set("Authorization", "Bearer "+pc.apiKey)
	req.Header.Set("User-Agent", "to_icalendar/1.0")

	// 发送请求
	resp, err := pc.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send info request: %w", err)
	}
	defer resp.Body.Close()

	// 返回服务器信息
	return &ServerInfo{
		StatusCode:       resp.StatusCode,
		SupportedMethods: []string{"POST"},
		Service:          "Pushcut API",
	}, nil
}

// ServerInfo 服务器信息
type ServerInfo struct {
	StatusCode       int      `json:"status_code"`
	SupportedMethods []string `json:"supported_methods"`
	Service          string   `json:"service"`
}