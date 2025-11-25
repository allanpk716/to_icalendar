package tray

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// TrayApplication 托盘应用程序的主要配置和状态管理实体
type TrayApplication struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	IconPath    string    `json:"icon_path"`
	Tooltip     string    `json:"tooltip"`
	StartHidden bool      `json:"start_hidden"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// NewTrayApplication 创建新的托盘应用程序实例
func NewTrayApplication(id, name, version string) *TrayApplication {
	now := time.Now()
	if id == "" {
		id = generateID()
	}

	return &TrayApplication{
		ID:          id,
		Name:        name,
		Version:     version,
		IconPath:    "",
		Tooltip:     "",
		StartHidden: false,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// Validate 验证托盘应用程序配置
func (ta *TrayApplication) Validate() error {
	if ta.Name == "" {
		return fmt.Errorf("应用名称不能为空")
	}
	if len(ta.Name) > 50 {
		return fmt.Errorf("应用名称不能超过50个字符")
	}
	if ta.IconPath == "" {
		return fmt.Errorf("图标路径不能为空")
	}
	return nil
}

// SetTooltip 设置提示文本
func (ta *TrayApplication) SetTooltip(tooltip string) {
	ta.Tooltip = tooltip
	ta.UpdateTimestamp()
}

// SetStartHidden 设置启动时是否隐藏
func (ta *TrayApplication) SetStartHidden(hidden bool) {
	ta.StartHidden = hidden
	ta.UpdateTimestamp()
}

// UpdateTimestamp 更新时间戳
func (ta *TrayApplication) UpdateTimestamp() {
	ta.UpdatedAt = time.Now()
}

// ToJSON 转换为JSON字符串
func (ta *TrayApplication) ToJSON() (string, error) {
	data, err := json.Marshal(ta)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// FromJSON 从JSON字符串创建实例
func (ta *TrayApplication) FromJSON(jsonStr string) error {
	return json.Unmarshal([]byte(jsonStr), ta)
}

// TrayIcon 托盘图标管理
type TrayIcon struct {
	ID         string    `json:"id"`
	AppID      string    `json:"app_id"`
	Size       int       `json:"size"`
	FilePath   string    `json:"file_path"`
	Format     string    `json:"format"`
	IsActive   bool      `json:"is_active"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// NewTrayIcon 创建新的托盘图标实例
func NewTrayIcon(filePath string, size int) *TrayIcon {
	now := time.Now()
	format := extractFormat(filePath)

	return &TrayIcon{
		ID:        generateID(),
		AppID:     "",
		Size:      size,
		FilePath:  filePath,
		Format:    format,
		IsActive:  false,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// Validate 验证托盘图标配置
func (ti *TrayIcon) Validate() error {
	if ti.FilePath == "" {
		return fmt.Errorf("图标文件路径不能为空")
	}
	if ti.Size <= 0 {
		return fmt.Errorf("图标尺寸必须大于0")
	}
	if ti.Size > 256 {
		return fmt.Errorf("图标尺寸不能超过256像素")
	}
	if ti.Format == "" {
		return fmt.Errorf("图标格式不能为空")
	}
	if !IsValidIconFormat(ti.Format) {
		return fmt.Errorf("不支持的图标格式: %s", ti.Format)
	}
	return nil
}

// SetActive 设置是否为激活状态
func (ti *TrayIcon) SetActive(active bool) {
	ti.IsActive = active
	ti.UpdatedAt = time.Now()
}

// SetAppID 设置关联的应用程序ID
func (ti *TrayIcon) SetAppID(appID string) {
	ti.AppID = appID
	ti.UpdatedAt = time.Now()
}

// IsValidIconSize 检查图标尺寸是否有效
func IsValidIconSize(size int) bool {
	validSizes := []int{16, 32, 48, 64, 128, 256}
	for _, validSize := range validSizes {
		if size == validSize {
			return true
		}
	}
	return false
}

// IsValidIconFormat 检查图标格式是否有效
func IsValidIconFormat(format string) bool {
	validFormats := []string{"PNG", "ICO", "JPG", "JPEG", "BMP"}
	upperFormat := strings.ToUpper(format)
	for _, validFormat := range validFormats {
		if upperFormat == validFormat {
			return true
		}
	}
	return false
}

// ApplicationState 应用程序运行时状态
type ApplicationState struct {
	AppID         string    `json:"app_id"`
	IsRunning     bool      `json:"is_running"`
	IsVisible     bool      `json:"is_visible"`
	LastActivity  time.Time `json:"last_activity"`
	ProcessID     int       `json:"process_id,omitempty"`
	MemoryUsage   int64     `json:"memory_usage,omitempty"`
	CPUUsage      float64   `json:"cpu_usage,omitempty"`
	StartUpTime   time.Time `json:"start_up_time"`
}

// NewApplicationState 创建新的应用程序状态实例
func NewApplicationState(appID string) *ApplicationState {
	now := time.Now()
	return &ApplicationState{
		AppID:        appID,
		IsRunning:    false,
		IsVisible:    false,
		LastActivity: now,
		StartUpTime:  now,
	}
}

// SetRunning 设置运行状态
func (as *ApplicationState) SetRunning(running bool) {
	as.IsRunning = running
	if running {
		as.StartUpTime = time.Now()
	}
	as.LastActivity = time.Now()
}

// SetVisible 设置可见状态
func (as *ApplicationState) SetVisible(visible bool) {
	as.IsVisible = visible
	as.LastActivity = time.Now()
}

// UpdateActivity 更新活动时间
func (as *ApplicationState) UpdateActivity() {
	as.LastActivity = time.Now()
}

// UpdateResourceUsage 更新资源使用情况
func (as *ApplicationState) UpdateResourceUsage(processID int, memoryUsage int64, cpuUsage float64) {
	as.ProcessID = processID
	as.MemoryUsage = memoryUsage
	as.CPUUsage = cpuUsage
	as.LastActivity = time.Now()
}

// 辅助函数

// generateID 生成唯一ID
func generateID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// extractFormat 从文件路径提取格式
func extractFormat(filePath string) string {
	parts := strings.Split(filePath, ".")
	if len(parts) > 1 {
		return strings.ToUpper(parts[len(parts)-1])
	}
	return "UNKNOWN"
}