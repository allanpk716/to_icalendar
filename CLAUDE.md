# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

这是一个提醒事项发送工具，支持将提醒事项发送到 Microsoft Todo。项目包含两个主要版本：
1. **CLI 版本** (`cmd/to_icalendar`) - 命令行工具
2. **GUI 版本** (`cmd/to_icalendar_tray`) - 基于 Wails 的桌面应用，支持剪贴板图片处理和 AI 智能识别

## Architecture

项目采用模块化架构，核心功能位于 `pkg` 目录中，两个版本共享相同的底层服务：

### 核心模块 (pkg/)
- **app/** - 应用程序核心逻辑和服务容器
- **commands/** - CLI 命令实现（init, test, clip-upload, clean）
- **services/** - 核心服务层（clipboard, config, todo, dify, cache, cleanup）
- **models/** - 数据模型定义（reminder, server config, dify config）
- **config/** - 配置管理
- **logger/** - 使用 github.com/WQGroup/logger 的日志管理
- **clipboard/** - 剪贴板内容读取（支持普通格式和 MSTSC 增强格式）
- **microsofttodo/** - Microsoft Todo API 客户端
- **dify/** - Dify AI 服务集成，用于图片内容识别和任务信息提取
- **cache/** - 统一缓存管理
- **cleanup/** - 缓存清理服务
- **timezone/** - 时区转换工具
- **wails/** - Wails 桌面应用的绑定接口

### 应用程序流程
1. **CLI 版本**: 解析命令 → 初始化服务容器 → 执行命令
2. **GUI 版本**: Wails 启动 → 初始化系统托盘 → 前后端通信 → 异步处理任务

### 关键数据流
- **剪贴板处理**: 读取剪贴板 → 图片标准化 → Dify AI 分析 → 解析响应 → 创建 Todo 任务
- **缓存管理**: 图片哈希缓存 → 任务去重 → 统一缓存清理
- **时区处理**: 本地时间 → UTC 转换 → Microsoft Graph API 格式

## Common Development Commands

### 构建和测试

```bash
# 更新依赖
go mod tidy

# 构建 CLI 版本
cd cmd/to_icalendar
go build -o to_icalendar.exe main.go

# 构建 GUI 版本
cd cmd/to_icalendar_tray
wails build  # 生产构建
wails dev    # 开发模式

# 运行前端开发服务器
cd cmd/to_icalendar_tray/frontend
npm install
npm run dev

# 运行测试
go test ./pkg/models -v     # 模型测试
go test ./pkg/timezone -v   # 时区测试
go test ./tests/unit/... -v # 单元测试
```

### CLI 使用命令

```bash
# 初始化配置
to_icalendar init

# 测试 Microsoft Todo 连接
to_icalendar test

# 处理剪贴板内容并上传
to_icalendar clip-upload

# 清理缓存
to_icalendar clean --all
to_icalendar clean --dry-run  # 预览
```

### 配置文件结构

**~/.to_icalendar/server.yaml**:
```yaml
microsoft_todo:
  tenant_id: "YOUR_TENANT_ID"
  client_id: "YOUR_CLIENT_ID"
  client_secret: "YOUR_CLIENT_SECRET"
  user_email: ""
  timezone: "Asia/Shanghai"

reminder:
  default_remind_before: "15m"
  enable_smart_reminder: true

deduplication:
  enabled: true
  time_window_minutes: 5
  similarity_threshold: 80

dify:
  api_endpoint: "YOUR_DIFY_ENDPOINT"
  api_key: "YOUR_DIFY_API_KEY"
  timeout: 60

cache:
  auto_cleanup_days: 30
  cleanup_on_startup: true

logging:
  level: "info"
  console_output: true
  file_output: true
  log_dir: "./Logs"
```

## Integration Details

### Microsoft Todo 集成
- 使用 Microsoft Graph API (OAuth 2.0)
- 需要 Tasks.ReadWrite 权限
- 支持任务列表、优先级、提醒时间
- 自动处理令牌刷新

### Dify AI 集成
- 支持图片内容识别和任务提取
- 自动解析会议时间、地点、参与者
- 智能识别任务优先级和分类
- 响应解析为标准 Reminder 格式

### 剪贴板处理
- 支持标准图片格式（PNG, JPG, BMP）
- 支持 MSTSC 远程桌面剪贴板格式
- 自动图片标准化和压缩
- 图片哈希去重

## Testing Strategy

测试分为三层：
1. **单元测试** (`tests/unit/`) - 测试单个组件
2. **集成测试** (`tests/integration/`) - 测试服务间协作
3. **包测试** (`pkg/*/test.go`) - 测试包功能

运行特定测试：
```bash
go test -run TestSpecificFunction ./pkg/path/to/test
```

## Development Notes

### 日志系统
- 统一使用 `github.com/WQGroup/logger`
- 日志级别：debug, info, warn, error
- 支持文件轮转和控制台输出
- Windows 下注意 symlink 权限问题

### 错误处理
- 所有服务返回明确的错误类型
- 使用 Go 1.23+ 的错误处理模式
- 错误信息包含足够的上下文

### 剪贴板权限
- Windows 需要管理员权限创建某些类型的符号链接
- 开发时注意权限相关的问题

### 时区处理
- 所有内部时间使用 UTC
- 用户界面显示本地时间
- 时区转换使用 `time.LoadLocation`

## Architecture Patterns

### 服务容器模式
使用依赖注入的 ServiceContainer 管理所有服务：
```go
container := app.NewServiceContainer(configDir, serverConfig, cacheManager, logger)
```

### 命令模式
所有 CLI 操作实现 Command 接口：
```go
type Command interface {
    Execute(ctx context.Context, req *CommandRequest) (*CommandResponse, error)
}
```

### 异步任务处理
GUI 版本使用 TaskManager 管理异步操作：
```go
taskManager := NewTaskManager(ctx)
taskID := generateTaskID()
// 发射事件更新前端进度
wailsRuntime.EventsEmit(ctx, "taskStatusChange", status)
```

## Security Considerations

- Azure AD 凭证存储在配置文件中（权限 0600）
- 所有 HTTPS 通信
- 图片处理在内存中进行，不留临时文件
- Dify API 密钥本地存储

## Performance Notes

- 图片处理有大小和尺寸限制
- 缓存系统避免重复处理
- 批量操作支持并发处理
- 日志文件自动轮转

## Debugging Tips

1. **查看日志**: `Logs/` 目录下的日志文件
2. **测试连接**: 使用 `to_icalendar test` 验证配置
3. **剪贴板诊断**: 查看剪贴板格式和内容
4. **缓存状态**: 使用 `clean --dry-run` 查看缓存文件
5. **Wails 开发**: 使用 `wails dev` 查看前后端通信日志