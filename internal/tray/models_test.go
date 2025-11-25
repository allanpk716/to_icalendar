package tray

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTrayApplication(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		appName string
		version string
		want    *TrayApplication
	}{
		{
			name:    "创建基本托盘应用程序",
			id:      "test-app",
			appName: "Test App",
			version: "1.0.0",
			want: &TrayApplication{
				ID:          "test-app",
				Name:        "Test App",
				Version:     "1.0.0",
				IconPath:    "",
				Tooltip:     "",
				StartHidden: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewTrayApplication(tt.id, tt.appName, tt.version)

			// 检查基本字段
			assert.Equal(t, tt.want.ID, got.ID)
			assert.Equal(t, tt.want.Name, got.Name)
			assert.Equal(t, tt.want.Version, got.Version)
			assert.Equal(t, tt.want.IconPath, got.IconPath)
			assert.Equal(t, tt.want.Tooltip, got.Tooltip)
			assert.Equal(t, tt.want.StartHidden, got.StartHidden)

			// 检查时间戳
			assert.NotZero(t, got.CreatedAt)
			assert.NotZero(t, got.UpdatedAt)
		})
	}
}

func TestTrayApplication_Validate(t *testing.T) {
	tests := []struct {
		name        string
		app         *TrayApplication
		expectError bool
		errorMsg    string
	}{
		{
			name: "有效的应用程序配置",
			app: &TrayApplication{
				ID:       "valid-app",
				Name:     "Valid App",
				IconPath: "assets/icons/tray-32.png",
			},
			expectError: false,
		},
		{
			name: "空名称应该失败",
			app: &TrayApplication{
				ID:       "empty-name",
				Name:     "",
				IconPath: "assets/icons/tray-32.png",
			},
			expectError: true,
			errorMsg:    "应用名称不能为空",
		},
		{
			name: "名称过长应该失败",
			app: &TrayApplication{
				ID:       "long-name",
				Name:     "This is a very long application name that exceeds the maximum allowed length of fifty characters",
				IconPath: "assets/icons/tray-32.png",
			},
			expectError: true,
			errorMsg:    "应用名称不能超过50个字符",
		},
		{
			name: "空图标路径应该失败",
			app: &TrayApplication{
				ID:       "empty-icon",
				Name:     "Valid App",
				IconPath: "",
			},
			expectError: true,
			errorMsg:    "图标路径不能为空",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.app.Validate()

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestTrayApplication_SetTooltip(t *testing.T) {
	app := NewTrayApplication("test", "Test App", "1.0.0")

	tooltip := "这是一个测试提示"
	app.SetTooltip(tooltip)

	assert.Equal(t, tooltip, app.Tooltip)
}

func TestTrayApplication_SetStartHidden(t *testing.T) {
	app := NewTrayApplication("test", "Test App", "1.0.0")

	app.SetStartHidden(true)
	assert.True(t, app.StartHidden)

	app.SetStartHidden(false)
	assert.False(t, app.StartHidden)
}

func TestTrayApplication_UpdateTimestamp(t *testing.T) {
	app := NewTrayApplication("test", "Test App", "1.0.0")
	originalTime := app.UpdatedAt

	// 等待一小段时间确保时间戳不同
	time.Sleep(time.Millisecond * 10)

	app.UpdateTimestamp()

	assert.True(t, app.UpdatedAt.After(originalTime))
}

func TestTrayApplication_ToJSON(t *testing.T) {
	app := NewTrayApplication("test", "Test App", "1.0.0")
	app.SetTooltip("Test Tooltip")
	app.SetStartHidden(true)

	json, err := app.ToJSON()
	require.NoError(t, err)
	require.NotEmpty(t, json)

	// 验证JSON包含必要字段
	assert.Contains(t, json, "test")
	assert.Contains(t, json, "Test App")
	assert.Contains(t, json, "Test Tooltip")
}

func TestTrayApplication_FromJSON(t *testing.T) {
	jsonStr := `{
		"id": "json-test",
		"name": "JSON Test App",
		"version": "2.0.0",
		"tooltip": "JSON Tooltip",
		"start_hidden": true
	}`

	app := &TrayApplication{}
	err := app.FromJSON(jsonStr)
	require.NoError(t, err)

	assert.Equal(t, "json-test", app.ID)
	assert.Equal(t, "JSON Test App", app.Name)
	assert.Equal(t, "2.0.0", app.Version)
	assert.Equal(t, "JSON Tooltip", app.Tooltip)
	assert.True(t, app.StartHidden)
}

func TestNewTrayIcon(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		size     int
		want     *TrayIcon
	}{
		{
			name:     "创建基本托盘图标",
			filePath: "assets/icons/tray-32.png",
			size:     32,
			want: &TrayIcon{
				ID:       "",
				AppID:    "",
				Size:     32,
				FilePath: "assets/icons/tray-32.png",
				Format:   "PNG",
				IsActive: false,
			},
		},
		{
			name:     "创建16x16图标",
			filePath: "assets/icons/tray-16.png",
			size:     16,
			want: &TrayIcon{
				ID:       "",
				AppID:    "",
				Size:     16,
				FilePath: "assets/icons/tray-16.png",
				Format:   "PNG",
				IsActive: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewTrayIcon(tt.filePath, tt.size)

			assert.Equal(t, tt.want.Size, got.Size)
			assert.Equal(t, tt.want.FilePath, got.FilePath)
			assert.Equal(t, tt.want.Format, got.Format)
			assert.Equal(t, tt.want.IsActive, got.IsActive)

			// 检查时间戳
			assert.NotZero(t, got.CreatedAt)
			assert.NotZero(t, got.UpdatedAt)

			// 检查ID已生成
			assert.NotEmpty(t, got.ID)
		})
	}
}

func TestTrayIcon_Validate(t *testing.T) {
	tests := []struct {
		name        string
		icon        *TrayIcon
		expectError bool
		errorMsg    string
	}{
		{
			name: "有效的图标配置",
			icon: &TrayIcon{
				ID:       "valid-icon",
				FilePath: "assets/icons/tray-32.png",
				Size:     32,
				Format:   "PNG",
			},
			expectError: false,
		},
		{
			name: "空文件路径应该失败",
			icon: &TrayIcon{
				ID:       "empty-path",
				FilePath: "",
				Size:     32,
				Format:   "PNG",
			},
			expectError: true,
			errorMsg:    "图标文件路径不能为空",
		},
		{
			name: "无效尺寸应该失败",
			icon: &TrayIcon{
				ID:       "invalid-size",
				FilePath: "assets/icons/tray-32.png",
				Size:     -1,
				Format:   "PNG",
			},
			expectError: true,
			errorMsg:    "图标尺寸必须大于0",
		},
		{
			name: "过大尺寸应该失败",
			icon: &TrayIcon{
				ID:       "oversized",
				FilePath: "assets/icons/tray-32.png",
				Size:     1000,
				Format:   "PNG",
			},
			expectError: true,
			errorMsg:    "图标尺寸不能超过256像素",
		},
		{
			name: "空格式应该失败",
			icon: &TrayIcon{
				ID:       "empty-format",
				FilePath: "assets/icons/tray-32.png",
				Size:     32,
				Format:   "",
			},
			expectError: true,
			errorMsg:    "图标格式不能为空",
		},
		{
			name: "无效格式应该失败",
			icon: &TrayIcon{
				ID:       "invalid-format",
				FilePath: "assets/icons/tray-32.png",
				Size:     32,
				Format:   "INVALID",
			},
			expectError: true,
			errorMsg:    "不支持的图标格式",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.icon.Validate()

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestTrayIcon_SetActive(t *testing.T) {
	icon := NewTrayIcon("test.png", 32)

	// 默认应该是非激活状态
	assert.False(t, icon.IsActive)

	icon.SetActive(true)
	assert.True(t, icon.IsActive)

	icon.SetActive(false)
	assert.False(t, icon.IsActive)
}

func TestTrayIcon_SetAppID(t *testing.T) {
	icon := NewTrayIcon("test.png", 32)

	appID := "test-app-id"
	icon.SetAppID(appID)

	assert.Equal(t, appID, icon.AppID)
}

func TestTrayIcon_IsValidSize(t *testing.T) {
	tests := []struct {
		name string
		size int
		want bool
	}{
		{"16x16", 16, true},
		{"32x32", 32, true},
		{"48x48", 48, true},
		{"64x64", 64, true},
		{"128x128", 128, true},
		{"256x256", 256, true},
		{"负数", -1, false},
		{"零", 0, false},
		{"过大", 512, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidIconSize(tt.size)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTrayIcon_IsValidFormat(t *testing.T) {
	tests := []struct {
		name   string
		format string
		want   bool
	}{
		{"PNG格式", "PNG", true},
		{"png格式", "png", true},
		{"ICO格式", "ICO", true},
		{"ico格式", "ico", true},
		{"JPG格式", "JPG", true},
		{"jpg格式", "jpg", true},
		{"BMP格式", "BMP", true},
		{"bmp格式", "bmp", true},
		{"无效格式", "INVALID", false},
		{"空格式", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidIconFormat(tt.format)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNewApplicationState(t *testing.T) {
	appID := "test-app"
	state := NewApplicationState(appID)

	assert.Equal(t, appID, state.AppID)
	assert.False(t, state.IsRunning)
	assert.False(t, state.IsVisible)
	assert.NotZero(t, state.LastActivity)
	assert.NotZero(t, state.StartUpTime)
}

func TestApplicationState_SetRunning(t *testing.T) {
	state := NewApplicationState("test-app")

	// 设置为运行状态
	state.SetRunning(true)
	assert.True(t, state.IsRunning)

	// 检查启动时间已更新
	originalStartTime := state.StartUpTime

	// 再次设置为运行状态，启动时间不应该改变
	state.SetRunning(true)
	assert.Equal(t, originalStartTime, state.StartUpTime)
}

func TestApplicationState_SetVisible(t *testing.T) {
	state := NewApplicationState("test-app")

	state.SetVisible(true)
	assert.True(t, state.IsVisible)

	state.SetVisible(false)
	assert.False(t, state.IsVisible)
}

func TestApplicationState_UpdateActivity(t *testing.T) {
	state := NewApplicationState("test-app")
	originalActivity := state.LastActivity

	// 等待一小段时间
	time.Sleep(time.Millisecond * 10)

	state.UpdateActivity()
	assert.True(t, state.LastActivity.After(originalActivity))
}

func TestApplicationState_UpdateResourceUsage(t *testing.T) {
	state := NewApplicationState("test-app")

	processID := 1234
	memoryUsage := int64(1024 * 1024) // 1MB
	cpuUsage := 15.5

	state.UpdateResourceUsage(processID, memoryUsage, cpuUsage)

	assert.Equal(t, processID, state.ProcessID)
	assert.Equal(t, memoryUsage, state.MemoryUsage)
	assert.Equal(t, cpuUsage, state.CPUUsage)
}