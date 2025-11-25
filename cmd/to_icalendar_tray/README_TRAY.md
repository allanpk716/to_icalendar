# to_icalendar 系统托盘应用程序

这是一个使用 Wails v2.11.0 构建的 Windows 系统托盘应用程序，为 to_icalendar 项目提供后台运行能力。

## 功能特性

### ✅ 已完成功能 (用户故事 1)

1. **系统托盘图标显示**
   - 应用程序启动后自动最小化到系统托盘
   - 在托盘区域显示应用程序图标
   - 用户可以看到程序在后台运行

2. **窗口管理**
   - 支持显示/隐藏主窗口
   - 启动时默认隐藏窗口
   - 关闭窗口时最小化到托盘而非退出

3. **前端界面**
   - 简洁的 Web 界面用于测试和控制
   - 实时显示窗口状态
   - 提供显示窗口、隐藏到托盘、退出应用的操作按钮

## 项目结构

```
cmd/to_icalendar_tray/
├── main.go                    # 应用程序入口点
├── app.go                     # 主要应用程序逻辑
├── app_test.go                # 应用程序单元测试
├── integration_test.go        # 集成测试
├── wails.json                 # Wails 配置文件
├── build/                     # 构建输出目录
│   └── bin/
│       └── to_icalendar_tray.exe  # 可执行文件
├── frontend/                  # 前端代码
│   └── src/
│       ├── main.js           # 主前端逻辑
│       └── style.css         # 样式文件
└── assets/
    └── icons/                 # 托盘图标资源
        ├── tray-icon.svg      # SVG 图标源文件
        ├── tray-16.png        # 16x16 PNG 图标
        ├── tray-32.png        # 32x32 PNG 图标
        └── tray-48.png        # 48x48 PNG 图标

internal/tray/                 # 托盘功能核心库
├── models.go                  # 数据模型定义
├── models_test.go             # 模型单元测试
├── manager.go                 # 托盘管理器
├── menu.go                    # 菜单功能
├── menu_test.go               # 菜单单元测试
├── icon.go                    # 图标加载功能
├── icon_test.go               # 图标测试
├── errors.go                  # 错误定义
└── logger.go                  # 日志功能
```

## 技术栈

- **后端**: Go 1.24+ with Wails v2.11.0
- **前端**: HTML5 + CSS3 + Vanilla JavaScript
- **构建工具**: Wails CLI v2.11.0
- **测试**: Go 标准测试库 + testify

## 构建和运行

### 前置要求

- Go 1.24 或更高版本
- Wails v2.11.0 CLI
- Node.js 14+ (用于前端构建)
- Windows 10 或更高版本

### 构建步骤

```bash
# 进入项目目录
cd cmd/to_icalendar_tray

# 开发模式运行
wails dev

# 生产模式构建
wails build

# 构建完成后，可执行文件位于：
# build/bin/to_icalendar_tray.exe
```

### 运行应用程序

1. 双击 `to_icalendar_tray.exe` 启动应用程序
2. 应用程序会自动最小化到系统托盘
3. 右键点击托盘图标可以显示菜单（当前只有退出选项）
4. 双击托盘图标可以显示主窗口

## 开发状态

### ✅ 已完成的任务

**阶段 1: 项目设置**
- [x] T001 创建 Wails 项目结构
- [x] T002 初始化 Wails v2.11.0 项目
- [x] T003 创建目录结构
- [x] T004 配置 wails.json
- [x] T005 初始化 Go 模块依赖

**阶段 2: 基础架构**
- [x] T006 创建托盘图标资源目录
- [x] T007 添加占位符托盘图标
- [x] T008 创建内部托盘包结构
- [x] T009 设置托盘管理器基础结构
- [x] T010 配置 Windows 特定构建设置
- [x] T011 设置错误处理和日志基础设施

**用户故事 1: 系统托盘图标显示**
- [x] T012.1 - T014.1 模型单元测试
- [x] T015.1 - T017.1 应用程序单元测试
- [x] T012 - T014 创建数据模型
- [x] T015 实现核心托盘管理器
- [x] T016 创建图标加载功能
- [x] T017 - T022 实现主应用程序结构
- [x] T023 创建前端界面
- [x] T024 测试托盘图标显示和窗口隐藏行为

### 🚧 进行中的任务

**用户故事 2: 右键菜单退出功能**
- [x] T025.1 - T033.1 菜单相关测试
- [ ] 实现完整的右键菜单功能
- [ ] 集成菜单与托盘管理器
- [ ] 清理资源并退出应用程序

### 📋 待开始的任务

**用户故事 3: 后台持续运行**
- [ ] 后台任务管理器
- [ ] 与现有 Microsoft Todo 功能集成
- [ ] 资源监控
- [ ] 优雅关闭处理

**阶段 6: 完善和交叉关注点**
- [ ] 错误日志记录
- [ ] 图标加载和内存优化
- [ ] 配置验证
- [ ] 文档完善
- [ ] 构建脚本
- [ ] 性能测试
- [ ] 崩溃恢复功能

## 配置

应用程序使用 Wails 配置文件 `wails.json` 进行配置：

```json
{
  "$schema": "https://wails.io/schemas/config.v2.json",
  "name": "to_icalendar_tray",
  "outputfilename": "to_icalendar_tray",
  "author": {
    "name": "allan716",
    "email": "525223688@qq.com"
  },
  "info": {
    "companyName": "to_icalendar",
    "productName": "to_icalendar Tray",
    "productVersion": "1.0.0"
  },
  "windowStartState": "hidden"
}
```

## 测试

```bash
# 运行所有测试
go test -v

# 运行特定测试
go test -v -run TestNewApp

# 运行性能测试
go test -v -bench=.
```

## 性能目标

- 启动时间: < 3 秒
- 内存使用: < 50MB
- CPU 空闲使用: < 1%
- 应用包大小: 10-15MB

## 故障排除

### 常见问题

1. **构建失败**: 确保安装了所有必需的依赖（Go, Node.js, Wails CLI）
2. **托盘图标不显示**: 检查图标文件是否存在于 `assets/icons/` 目录
3. **应用程序无法启动**: 检查 Windows 版本兼容性（需要 Windows 10+）

### 日志

应用程序日志位于 `logs/` 目录下，文件名格式为 `tray_YYYY-MM-DD.log`。

## 贡献

请参考项目根目录的 `specs/001-system-tray/` 目录下的规格文档进行开发。

## 许可证

本项目的许可证与主 to_icalendar 项目保持一致。