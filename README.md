# to_icalendar

一个支持多种服务的提醒事项发送工具，可将提醒事项发送到 Microsoft Todo 或通过 Pushcut 发送到 iOS 提醒事项应用。

## ✨ 功能特性

- 🎯 **多服务支持**：支持 Microsoft Todo 和 Pushcut（iOS）
- 📝 **简单配置**：JSON 格式配置提醒事项
- ⏰ **智能提醒**：支持自定义提醒时间
- 📦 **批量处理**：支持批量发送提醒事项
- 🎛️ **优先级管理**：支持设置优先级和分类
- 🌐 **跨平台**：支持 Windows、Linux、macOS
- 🔄 **自动同步**：支持云端自动同步
- 🔧 **自动检测**：根据配置自动选择使用的服务

## 🏗️ 系统架构

### Microsoft Todo 模式
```
Go程序 (Windows/Linux/macOS)
    ↓ Microsoft Graph API
Microsoft 365 云端服务
    ↓ 数据同步
Microsoft To Do 应用
    ↓ 跨设备同步
所有设备 (Windows、Web、iOS、Android)
```

### Pushcut 模式
```
Go程序 (Windows/Linux/macOS)
    ↓ HTTP POST
Pushcut API (云端服务)
    ↓ 推送通知
Pushcut应用 (iOS设备)
    ↓ 自动触发
iOS快捷指令
    ↓ 创建提醒
iOS Reminders应用
    ↓ iCloud同步
所有iOS设备
```

## 🚀 安装

### 从源码编译

```bash
git clone https://github.com/allanpk716/to_icalendar.git
cd to_icalendar
go mod tidy
go build -o to_icalendar main.go
```

## 📋 使用指南

### 1. 初始化配置

```bash
./to_icalendar init
```

这将创建配置文件模板：
- `config/server.yaml` - 服务配置文件
- `config/reminder.json` - 提醒事项模板

### 2. 选择并配置服务

编辑 `config/server.yaml`，根据需要选择其中一种服务：

#### 选项一：Microsoft Todo（推荐）

```yaml
microsoft_todo:
  tenant_id: "您的Azure租户ID"
  client_id: "您的应用程序客户端ID"
  client_secret: "您的客户端密钥"
  timezone: "Asia/Shanghai"

pushcut:
  api_key: ""
  webhook_id: ""
  timezone: "Asia/Shanghai"
```

#### 选项二：Pushcut（iOS）

```yaml
pushcut:
  api_key: "您的Pushcut API密钥"
  webhook_id: "您的Webhook ID"
  timezone: "Asia/Shanghai"

microsoft_todo:
  tenant_id: ""
  client_id: ""
  client_secret: ""
  timezone: "Asia/Shanghai"
```

### 3. 创建提醒事项

编辑 `config/reminder.json` 或创建新的 JSON 文件：

```json
{
  "title": "会议提醒",
  "description": "参加产品评审会议",
  "date": "2024-12-25",
  "time": "14:30",
  "remind_before": "15m",
  "priority": "medium",
  "list": "工作"
}
```

#### 支持的字段

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `title` | string | ✅ | 提醒标题 |
| `description` | string | ❌ | 详细备注 |
| `date` | string | ✅ | 日期 (YYYY-MM-DD) |
| `time` | string | ✅ | 时间 (HH:MM) |
| `remind_before` | string | ❌ | 提前提醒时间 (如: 15m, 1h, 1d) |
| `priority` | string | ❌ | 优先级: low/medium/high |
| `list` | string | ❌ | 任务列表名称 |

### 4. 测试连接

```bash
./to_icalendar test
```

### 5. 发送提醒事项

```bash
# 发送单个提醒事项
./to_icalendar upload config/reminder.json

# 批量发送提醒事项
./to_icalendar upload reminders/*.json
```

### 6. 其他命令

```bash
# 显示帮助
./to_icalendar help
```

## 🔧 Microsoft Todo 设置步骤

### 1. 在 Azure AD 中注册应用程序

1. 访问 [Azure Portal](https://portal.azure.com)
2. 转到 **Azure Active Directory** → **应用注册** → **新注册**
3. 输入应用程序名称（如 "to_icalendar"）
4. 选择支持的账户类型（通常选择"仅此组织目录中的账户"）
5. 点击 **注册**

### 2. 配置 API 权限

1. 在应用注册页面，转到 **API 权限**
2. 点击 **添加权限** → **Microsoft Graph**
3. 选择 **应用程序权限**
4. 搜索并添加 `Tasks.ReadWrite.All` 权限
5. 点击 **添加权限**
6. 点击 **授予管理员同意**（需要管理员权限）

### 3. 创建客户端密钥

1. 转到 **证书和密钥** 页面
2. 点击 **新客户端密钥**
3. 输入密钥描述（如 "to_icalendar_secret"）
4. 选择过期时间（建议选择 12 个月或更长）
5. 点击 **添加**
6. **重要**：立即复制密钥值，此值只显示一次

### 4. 获取必要信息

在应用注册的 **概述** 页面找到：
- **应用程序（客户端）ID** → 对应配置文件中的 `client_id`
- **目录（租户）ID** → 对应配置文件中的 `tenant_id`

客户端密钥值对应配置文件中的 `client_secret`。

## 🍎 Pushcut 设置步骤（iOS）

### 1. 安装 Pushcut 应用

- 在 App Store 中搜索并安装 "Pushcut"
- 打开应用并完成基本设置

### 2. 创建快捷指令

- 在 Pushcut 中创建新的快捷指令
- 配置快捷指令接收提醒事项数据并创建 iOS 提醒事项
- 获取 Webhook API 端点信息

### 3. 获取 API 密钥

- 在 Pushcut 设置中找到 API 密钥
- 记录 Webhook ID 用于配置

### 4. 快捷指令配置

确保快捷指令能够：
1. **接收输入数据**：配置快捷指令接收 HTTP 请求中的 JSON 数据
2. **解析提醒信息**：从 JSON 中提取 title、description、date、time 等字段
3. **创建提醒事项**：使用"添加新提醒事项"动作创建提醒
4. **设置提醒时间**：根据 date 和 time 字段设置提醒时间
5. **配置优先级**：根据 priority 字段设置提醒优先级

## 📅 时间格式说明

### 提醒时间格式

`remind_before` 字段支持以下格式：

- `15m` - 提前 15 分钟
- `1h` - 提前 1 小时
- `2d` - 提前 2 天
- `30m` - 提前 30 分钟

### 优先级说明

- `high` - 高优先级
- `medium` - 中等优先级（默认）
- `low` - 低优先级

## 🔍 服务对比

| 功能 | Microsoft Todo | Pushcut (iOS) |
|------|----------------|---------------|
| 跨平台支持 | ✅ 全平台 | ❌ 仅 iOS |
| 设置复杂度 | 中等（需要 Azure AD） | 简单 |
| 同步速度 | 快 | 中等 |
| 成本 | 免费 | 部分免费 |
| 离线支持 | ✅ | ❌ |
| 团队协作 | ✅ | ❌ |

## 💡 使用示例

### 创建工作会议提醒

```json
{
  "title": "团队周会",
  "description": "讨论本周工作进展和下周计划",
  "date": "2024-12-25",
  "time": "09:30",
  "remind_before": "30m",
  "priority": "high",
  "list": "工作"
}
```

### 创建个人事务提醒

```json
{
  "title": "缴纳水电费",
  "description": "本月水电费账单",
  "date": "2024-12-28",
  "time": "18:00",
  "remind_before": "1h",
  "priority": "medium",
  "list": "生活"
}
```

### 批量创建提醒事项

创建多个 JSON 文件：
- `meetings/meeting1.json`
- `meetings/meeting2.json`
- `tasks/task1.json`

然后批量上传：
```bash
./to_icalendar upload meetings/*.json tasks/*.json
```

## 🛠️ 故障排除

### Microsoft Todo 相关问题

#### 连接失败
1. 确认 Azure AD 配置正确
2. 检查 API 权限是否已授予管理员同意
3. 验证 Tenant ID、Client ID 和 Client Secret 是否正确
4. 确认网络连接正常

#### 任务创建失败
1. 检查 Microsoft Graph API 权限配置
2. 确认用户账户有访问 Microsoft Todo 的权限
3. 验证 JSON 数据格式是否正确

### Pushcut 相关问题

#### 连接失败
1. 确认 Pushcut API 密钥和 Webhook ID 正确
2. 检查网络连接
3. 确认 iOS 设备上的 Pushcut 应用正常运行
4. 检查快捷指令配置是否正确

#### 提醒事项未创建
1. 检查快捷指令是否正确配置
2. 确认 iOS 设备上的提醒事项应用权限
3. 验证 JSON 数据格式是否正确
4. 检查提醒时间是否在未来

### 通用问题

#### 配置文件错误
1. 确认 YAML 格式正确
2. 检查必填字段是否完整
3. 验证时间格式是否正确

#### 时间同步问题
1. 检查系统时区设置
2. 确认配置文件中的时区设置正确
3. 验证日期时间格式

## 🔄 版本历史

### v1.0.0
- ✅ 新增 Microsoft Todo 支持
- ✅ 保持 Pushcut 兼容性
- ✅ 自动服务检测
- ✅ 统一的 CLI 接口
- ✅ 扩展的配置选项

## 📝 开发说明

### 项目结构

```
to_icalendar/
├── main.go                    # 主程序入口
├── config/
│   ├── server.yaml           # 服务配置
│   └── reminder.json         # 提醒事项模板
└── internal/
    ├── config/               # 配置管理
    ├── models/               # 数据结构
    ├── microsoft-todo/       # Microsoft Todo 客户端
    └── pushcut/              # Pushcut 客户端
```

### 添加新的服务支持

1. 在 `internal/` 目录下创建新的服务包
2. 实现服务的客户端接口
3. 更新配置结构以支持新服务
4. 修改主程序以支持新服务检测和处理

## 📄 许可证

MIT License

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

### 贡献指南

1. Fork 本项目
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启 Pull Request

## 📞 支持

如果您遇到问题或有建议，请：

1. 查看 [故障排除](#故障排除) 部分
2. 搜索现有的 [Issues](https://github.com/allanpk716/to_icalendar/issues)
3. 如果没有找到解决方案，请创建新的 Issue

---

**免责声明**：本工具使用第三方服务（Microsoft Graph API 和 Pushcut），请确保遵守相关服务的使用条款和隐私政策。