package caldav

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/allanpk716/to_icalendar/internal/ical"
	"github.com/allanpk716/to_icalendar/internal/models"
)

// CalDAVClient CalDAV客户端
type CalDAVClient struct {
	serverURL string
	username  string
	password  string
	client    *http.Client
}

// NewCalDAVClient 创建CalDAV客户端
func NewCalDAVClient(serverURL, username, password string) *CalDAVClient {
	return &CalDAVClient{
		serverURL: serverURL,
		username:  username,
		password:  password,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// UploadReminder 上传提醒事项到CalDAV服务器
func (c *CalDAVClient) UploadReminder(cal *ical.Calendar, reminder *models.ParsedReminder) error {
	// 获取iCalendar数据
	icalCreator := ical.NewICalCreator()
	icalData, err := icalCreator.GetICalString(cal)
	if err != nil {
		return fmt.Errorf("failed to serialize iCalendar: %w", err)
	}

	// 生成文件名
	filename := c.generateFilename(reminder)
	path := filepath.Join("/principal/reminders/", filename)

	// 确保路径以/开头
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	// 构建完整的URL
	url := c.serverURL + path

	// 创建HTTP请求
	req, err := http.NewRequest("PUT", url, strings.NewReader(icalData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "text/calendar; charset=utf-8")
	req.SetBasicAuth(c.username, c.password)

	// 发送请求
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to upload reminder: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// TestConnection 测试CalDAV连接
func (c *CalDAVClient) TestConnection() error {
	// 发送PROPFIND请求测试连接
	path := "/principal/"
	url := c.serverURL + path

	req, err := http.NewRequest("PROPFIND", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create test request: %w", err)
	}

	req.Header.Set("Depth", "0")
	req.SetBasicAuth(c.username, c.password)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to CalDAV server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("CalDAV server returned status %d", resp.StatusCode)
	}

	return nil
}

// ListReminders 列出现有的提醒事项
func (c *CalDAVClient) ListReminders() ([]string, error) {
	path := "/principal/reminders/"
	url := c.serverURL + path

	req, err := http.NewRequest("PROPFIND", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create list request: %w", err)
	}

	req.Header.Set("Depth", "1")
	req.Header.Set("Content-Type", "application/xml; charset=utf-8")
	req.SetBasicAuth(c.username, c.password)

	// 简单的PROPFIND请求体
	propfindBody := `<?xml version="1.0" encoding="utf-8" ?>
<D:propfind xmlns:D="DAV:">
    <D:prop>
        <D:displayname/>
        <D:getcontenttype/>
    </D:prop>
</D:propfind>`

	req.Body = io.NopCloser(strings.NewReader(propfindBody))
	req.ContentLength = int64(len(propfindBody))

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list reminders: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("list reminders failed with status %d", resp.StatusCode)
	}

	// 简单解析响应，提取.ics文件
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var reminders []string
	lines := strings.Split(string(body), "\n")
	for _, line := range lines {
		if strings.Contains(line, ".ics") {
			// 提取文件名
			start := strings.Index(line, ">")
			end := strings.Index(line, ".ics<")
			if start != -1 && end != -1 && end > start {
				filename := line[start+1 : end+4] // 包含.ics
				if strings.HasSuffix(filename, ".ics") {
					reminders = append(reminders, filename)
				}
			}
		}
	}

	return reminders, nil
}

// DeleteReminder 删除指定的提醒事项
func (c *CalDAVClient) DeleteReminder(filename string) error {
	path := filepath.Join("/principal/reminders/", filename)
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	url := c.serverURL + path

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete request: %w", err)
	}

	req.SetBasicAuth(c.username, c.password)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete reminder: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("delete failed with status %d", resp.StatusCode)
	}

	return nil
}

// generateFilename 为提醒事项生成文件名
func (c *CalDAVClient) generateFilename(reminder *models.ParsedReminder) string {
	// 使用时间戳和标题生成文件名
	timestamp := reminder.DueTime.Unix()
	// 清理文件名中的非法字符
	title := strings.ReplaceAll(reminder.Original.Title, " ", "_")
	title = strings.ReplaceAll(title, "/", "_")
	title = strings.ReplaceAll(title, "\\", "_")

	// 限制文件名长度
	if len(title) > 50 {
		title = title[:50]
	}

	filename := fmt.Sprintf("%s_%d.ics", title, timestamp)
	return filename
}

// GetRemindersList 获取所有提醒事项的详细信息
func (c *CalDAVClient) GetRemindersList() ([]*ReminderInfo, error) {
	filenames, err := c.ListReminders()
	if err != nil {
		return nil, err
	}

	var reminders []*ReminderInfo
	for _, filename := range filenames {
		info := &ReminderInfo{
			Filename: filename,
			ModTime:  time.Now(), // 简化实现，实际应从服务器获取
		}
		reminders = append(reminders, info)
	}

	return reminders, nil
}

// ReminderInfo 提醒事项信息
type ReminderInfo struct {
	Filename string    `json:"filename"`
	Size     int64     `json:"size"`
	ModTime  time.Time `json:"mod_time"`
}

// UploadMultipleReminders 批量上传多个提醒事项
func (c *CalDAVClient) UploadMultipleReminders(reminders [](*ReminderUpload)) error {
	for _, upload := range reminders {
		err := c.UploadReminder(upload.Calendar, upload.ParsedReminder)
		if err != nil {
			return fmt.Errorf("failed to upload reminder '%s': %w", upload.ParsedReminder.Original.Title, err)
		}
	}
	return nil
}

// ReminderUpload 上传提醒事项的数据结构
type ReminderUpload struct {
	Calendar       *ical.Calendar
	ParsedReminder *models.ParsedReminder
}

// ValidateServerConfig 验证服务器配置
func (c *CalDAVClient) ValidateServerConfig() error {
	// 检查服务器URL格式
	if !strings.HasPrefix(c.serverURL, "http://") && !strings.HasPrefix(c.serverURL, "https://") {
		return fmt.Errorf("invalid server URL format")
	}

	// 检查用户名
	if c.username == "" {
		return fmt.Errorf("username is required")
	}

	// 检查密码
	if c.password == "" {
		return fmt.Errorf("password is required")
	}

	return nil
}

// GetServerInfo 获取服务器信息
func (c *CalDAVClient) GetServerInfo() (*ServerInfo, error) {
	// 发送OPTIONS请求获取服务器信息
	req, err := http.NewRequest("OPTIONS", c.serverURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(c.username, c.password)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get server info: %w", err)
	}
	defer resp.Body.Close()

	info := &ServerInfo{
		ServerURL:        c.serverURL,
		StatusCode:       resp.StatusCode,
		SupportedMethods: resp.Header.Get("Allow"),
		DavCapabilities:  resp.Header.Get("DAV"),
	}

	return info, nil
}

// ServerInfo 服务器信息
type ServerInfo struct {
	ServerURL        string `json:"server_url"`
	StatusCode       int    `json:"status_code"`
	SupportedMethods string `json:"supported_methods"`
	DavCapabilities  string `json:"dav_capabilities"`
}