# Quick Start Guide: System Tray Implementation

**Date**: 2025-01-25
**Feature**: System Tray Background Running

## Prerequisites

### Development Environment

确保以下依赖已安装：

```bash
# 检查Go版本 (需要1.23.4+)
go version

# 检查Wails版本 (需要v2.11.0+)
wails version

# 检查Node.js版本 (需要18+)
node --version

# 检查npm版本
npm --version
```

### Required Dependencies

- ✅ **Go 1.23.4+**: 已安装
- ✅ **Wails v2.11.0+**: 已安装
- ✅ **Node.js 18+**: 已安装
- ✅ **WebView2**: 已安装 (Windows)

## Project Setup

### 1. Initialize Wails Project

在`cmd/to_icalendar_tray/`目录中初始化Wails项目：

```bash
cd cmd/to_icalendar_tray

# 初始化Wails项目
wails init -n to_icalendar_tray -t vanilla

# 安装依赖
npm install
go mod tidy
```

### 2. Update wails.json Configuration

创建或更新`wails.json`配置文件：

```json
{
  "$schema": "https://wails.io/schemas/config.v2.json",
  "name": "to_icalendar_tray",
  "outputfilename": "to_icalendar_tray",
  "frontend": {
    "dir": "./frontend",
    "install": "npm install",
    "build": "npm run build",
    "dev:watcher": "npm run dev",
    "dev:serverUrl": "auto"
  },
  "backend": {
    "dir": "./backend"
  },
  "author": {
    "name": "to_icalendar",
    "email": "support@to_icalendar.com"
  },
  "info": {
    "productName": "to_icalendar",
    "productVersion": "1.0.0",
    "copyright": "Copyright © 2025 to_icalendar",
    "comments": "Microsoft Todo任务提醒工具"
  },
  "nsisType": "multiple",
  "obfuscated": false,
  "garbleargs": ""
}
```

### 3. Project Structure

```
cmd/to_icalendar_tray/
├── backend/
│   ├── app.go              # 主应用结构
│   ├── tray/               # 托盘功能包
│   │   ├── manager.go      # 托盘管理器
│   │   ├── menu.go         # 菜单管理
│   │   └── icon.go         # 图标管理
│   └── main.go             # 后端入口
├── frontend/
│   ├── dist/               # 构建输出
│   ├── src/
│   │   └── main.js         # 前端主文件
│   ├── package.json
│   └── wailsjs/            # Wails生成的绑定
├── embed.go                # 嵌入文件
├── wails.json              # Wails配置
└── main.go                 # 应用入口
```

## Core Implementation

### 1. Main Application Structure

创建`backend/app.go`：

```go
package backend

import (
    "context"
    "embed"
    "fmt"
    "log"
    "time"

    "github.com/wailsapp/wails/v2/pkg/menu"
    "github.com/wailsapp/wails/v2/pkg/menu/keys"
    "github.com/wailsapp/wails/v2/pkg/options"
    "github.com/wailsapp/wails/v2/pkg/options/assetserver"
    "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

// App 结构体
type App struct {
    ctx     context.Context
    runtime *runtime.Runtime
    trayMgr *TrayManager
}

// NewApp 创建新的应用实例
func NewApp() *App {
    return &App{
        trayMgr: NewTrayManager(),
    }
}

// OnStartup 应用启动时的回调
func (a *App) OnStartup(ctx context.Context) {
    a.ctx = ctx
    a.runtime = runtime.New(ctx)

    log.Println("Application starting up...")

    // 初始化系统托盘
    if err := a.trayMgr.Initialize(a.ctx, a.runtime); err != nil {
        log.Printf("Failed to initialize tray: %v", err)
        return
    }

    // 启动时隐藏主窗口
    a.runtime.Window.Hide()

    log.Println("Application started successfully")
}

// OnDomReady DOM加载完成时的回调
func (a *App) OnDomReady(ctx context.Context) {
    // DOM加载完成后处理
}

// OnBeforeClose 窗口关闭前的回调
func (a *App) OnBeforeClose(ctx context.Context) (prevent bool) {
    // 隐藏窗口而不是退出应用
    a.runtime.Window.Hide()
    return true // 阻止窗口关闭
}

// OnShutdown 应用关闭时的回调
func (a *App) OnShutdown(ctx context.Context) {
    log.Println("Application shutting down...")

    // 清理托盘资源
    if a.trayMgr != nil {
        a.trayMgr.Cleanup()
    }
}

// GetOptions 获取应用选项
func (a *App) GetOptions() *options.App {
    return &options.App{
        Title:  "to_icalendar",
        Width:  800,
        Height: 600,
        AssetServer: &assetserver.Options{
            Assets: assets,
        },
        BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
        OnStartup:        a.OnStartup,
        OnDomReady:       a.OnDomReady,
        OnBeforeClose:    a.OnBeforeClose,
        OnShutdown:       a.OnShutdown,
        Windows: &options.Windows{
            WebviewIsTransparent: false,
            WindowIsTranslucent:  false,
            DisableWindowIcon:    false,
            StartHidden:          true, // 启动时隐藏窗口
        },
    }
}
```

### 2. Tray Manager Implementation

创建`backend/tray/manager.go`：

```go
package tray

import (
    "context"
    "embed"
    "log"

    "github.com/wailsapp/wails/v2/pkg/menu"
    "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:../../../assets/icons
var iconAssets embed.FS

// TrayManager 系统托盘管理器
type TrayManager struct {
    ctx     context.Context
    runtime *runtime.Runtime
    tray    *runtime.SystemTray
}

// NewTrayManager 创建新的托盘管理器
func NewTrayManager() *TrayManager {
    return &TrayManager{}
}

// Initialize 初始化系统托盘
func (tm *TrayManager) Initialize(ctx context.Context, runtime *runtime.Runtime) error {
    tm.ctx = ctx
    tm.runtime = runtime

    // 加载托盘图标
    icon, err := tm.loadTrayIcon()
    if err != nil {
        return fmt.Errorf("failed to load tray icon: %w", err)
    }

    // 创建托盘
    tm.tray = tm.runtime.SystemTray.New(icon)

    // 创建菜单
    menu := tm.createTrayMenu()
    tm.tray.SetMenu(menu)

    // 设置提示
    tm.tray.SetTooltip("to_icalendar - 任务提醒工具")

    log.Println("System tray initialized successfully")
    return nil
}

// loadTrayIcon 加载托盘图标
func (tm *TrayManager) loadTrayIcon() ([]byte, error) {
    // 尝试加载不同尺寸的图标
    iconSizes := []string{"tray-32.png", "tray-16.png", "tray-48.png"}

    for _, size := range iconSizes {
        iconData, err := iconAssets.ReadFile("icons/" + size)
        if err == nil && len(iconData) > 0 {
            log.Printf("Loaded tray icon: %s", size)
            return iconData, nil
        }
    }

    return nil, fmt.Errorf("no valid tray icon found")
}

// createTrayMenu 创建托盘菜单
func (tm *TrayManager) createTrayMenu() *menu.Menu {
    trayMenu := menu.NewMenu()

    // 退出菜单项
    quitItem := menu.Text("退出", keys.CmdOrCtrl("q"), func(_ *menu.CallbackData) {
        log.Println("Quit menu item clicked")
        tm.runtime.Quit(tm.ctx)
    })
    trayMenu.Append(quitItem)

    return trayMenu
}

// Cleanup 清理托盘资源
func (tm *TrayManager) Cleanup() {
    if tm.tray != nil {
        log.Println("Cleaning up system tray...")
        // Wails会自动清理托盘资源
        tm.tray = nil
    }
}

// ShowWindow 显示主窗口
func (tm *TrayManager) ShowWindow() {
    if tm.runtime != nil {
        tm.runtime.Window.Show()
        tm.runtime.Window.SetFocus()
    }
}

// HideWindow 隐藏主窗口
func (tm *TrayManager) HideWindow() {
    if tm.runtime != nil {
        tm.runtime.Window.Hide()
    }
}
```

### 3. Main Entry Point

创建`main.go`：

```go
package main

import (
    "embed"
    "log"

    "github.com/wailsapp/wails/v2"
    "github.com/allanpk716/to_icalendar/cmd/to_icalendar_tray/backend"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
    // 创建应用实例
    app := backend.NewApp()

    // 运行应用
    err := wails.Run(app.GetOptions())
    if err != nil {
        log.Fatal("Error:", err.Error())
    }
}
```

### 4. Frontend Setup

创建`frontend/src/main.js`：

```javascript
// 前端主文件 - 最小化实现
window.addEventListener('DOMContentLoaded', () => {
    console.log('Frontend loaded successfully');

    // 这里可以添加前端逻辑，如果需要的话
    // 对于纯托盘应用，前端可能不需要复杂的功能
});
```

## Build and Run

### Development Mode

```bash
# 开发模式运行（带热重载）
cd cmd/to_icalendar_tray
wails dev
```

### Production Build

```bash
# 构建生产版本
cd cmd/to_icalendar_tray
wails build

# 构建时会生成：
# - build/bin/to_icalendar_tray.exe (Windows可执行文件)
# - build/bin/to_icalendar_tray (其他平台)
```

### Test the Tray Application

1. **运行应用**：
   ```bash
   ./build/bin/to_icalendar_tray.exe
   ```

2. **验证功能**：
   - 应用启动后应该在系统托盘显示图标
   - 右键点击托盘图标应该显示"退出"菜单
   - 点击"退出"应该完全关闭应用程序

3. **检查资源使用**：
   - 内存使用应该在30-50MB范围内
   - CPU使用应该在1%以下（空闲状态）

## Icon Assets

### Required Icons

在`assets/icons/`目录中放置以下图标文件：

```
assets/icons/
├── tray-16.png   # 16x16像素
├── tray-32.png   # 32x32像素 (主要使用)
└── tray-48.png   # 48x48像素
```

### Icon Requirements

- **格式**: PNG (推荐) 或 ICO
- **尺寸**: 16x16, 32x32, 48x48像素
- **透明背景**: 支持透明背景
- **风格**: 简洁，与应用主题一致

## Testing

### Unit Tests

创建`backend/tray/manager_test.go`：

```go
package tray

import (
    "context"
    "testing"
)

func TestNewTrayManager(t *testing.T) {
    tm := NewTrayManager()
    if tm == nil {
        t.Fatal("NewTrayManager() returned nil")
    }
}

func TestTrayManager_Cleanup(t *testing.T) {
    tm := NewTrayManager()

    // 测试清理不会panic
    defer func() {
        if r := recover(); r != nil {
            t.Errorf("Cleanup() panicked: %v", r)
        }
    }()

    tm.Cleanup()
}
```

### Integration Tests

```bash
# 运行所有测试
cd cmd/to_icalendar_tray
go test ./...

# 运行测试并显示覆盖率
go test -cover ./...
```

## Troubleshooting

### Common Issues

1. **托盘图标不显示**：
   - 检查图标文件是否存在
   - 确认图标格式正确（PNG/ICO）
   - 查看控制台错误信息

2. **菜单不显示**：
   - 检查菜单创建逻辑
   - 确认菜单项正确添加
   - 验证事件处理器设置

3. **应用无法退出**：
   - 检查`OnBeforeClose`回调
   - 确认`runtime.Quit()`正确调用
   - 验证清理逻辑

### Debug Mode

```bash
# 启用详细日志
wails dev -debug

# 检查Wails环境
wails doctor
```

## Next Steps

1. **集成现有功能**：连接到现有的Microsoft Todo API
2. **添加菜单选项**：如"发送提醒"、"查看任务"等
3. **配置管理**：集成到现有的配置系统
4. **错误处理**：完善错误处理和用户反馈
5. **打包发布**：创建Windows安装程序

## References

- [Wails v2 Documentation](https://wails.io/docs/)
- [Wails v2 API Reference](https://pkg.go.dev/github.com/wailsapp/wails/v2)
- [Project Specification](./spec.md)
- [Data Model](./data-model.md)