# to_icalendar

一个提醒事项发送工具，可将提醒事项发送到 Microsoft Todo。

## ✨ 功能特性

- 🎯 **Microsoft Todo 支持**：支持将提醒事项发送到 Microsoft Todo
- 📝 **简单配置**：JSON 格式配置提醒事项
- ⏰ **智能提醒**：支持自定义提醒时间
- 📦 **批量处理**：支持批量发送提醒事项
- 🎛️ **优先级管理**：支持设置优先级和分类
- 🌐 **跨平台**：支持 Windows、Linux、macOS
- 🔄 **自动同步**：支持云端自动同步

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

### 2. 配置 Microsoft Todo

编辑 `config/server.yaml`：

```yaml
microsoft_todo:
  tenant_id: "您的Azure租户ID"
  client_id: "您的应用程序客户端ID"
  client_secret: "您的客户端密钥"
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
3. 填写应用程序信息：
   - **Name**: `to_icalendar Personal`
   - **Supported account types**: `Accounts in any organizational directory (Any Azure AD directory - Multitenant) and personal Microsoft accounts`
   - **Redirect URI**: `http://localhost:8080/callback`
4. 点击 **注册**

### 2. 配置身份验证

1. 在应用注册页面，转到 **身份验证**
2. 在 **高级设置** 中找到 **允许公共客户端流**
3. 将 **允许公共客户端流** 设置为 **是**
4. 点击 **保存**

### 3. 配置 API 权限

1. 在应用注册页面，转到 **API 权限**
2. 点击 **添加权限** → **Microsoft Graph**
3. 选择 **委托的权限**
4. 搜索并添加以下权限：
   - `Tasks.ReadWrite` - 读写Microsoft Todo任务
   - `User.Read` - 读取用户基本信息
   - `offline_access` - 获取刷新token以实现长期访问
5. 点击 **添加权限**
6. 点击 **授予管理员同意**（需要管理员权限）

### 4. 创建客户端密钥

1. 转到 **证书和密钥** 页面
2. 点击 **新客户端密钥**
3. 输入密钥描述（如 "to_icalendar_secret"）
4. 选择过期时间（建议选择 12 个月或更长）
5. 点击 **添加**
6. **重要**：立即复制密钥值，此值只显示一次

### 5. 获取必要信息

在应用注册的 **概述** 页面找到：
- **应用程序（客户端）ID** → 对应配置文件中的 `client_id`
- **目录（租户）ID** → 对应配置文件中的 `tenant_id` # 建议使用 consumers

### 6. 配置应用程序

将获取的信息填入 `config/server.yaml`：

```yaml
microsoft_todo:
  tenant_id: "consumers"  # 个人Microsoft账户使用consumers，建议使用这个即可
  client_id: "您的应用程序客户端ID"
  client_secret: "您的客户端密钥"
  user_email: "您的个人Microsoft邮箱"
  timezone: "Asia/Shanghai"
```

### 7. 验证登录

运行以下命令进行交互式验证登录：

```bash
./to_icalendar test
```

程序会：
1. 自动打开浏览器进行Microsoft账户登录
2. 授权应用程序访问Microsoft Todo
3. 缓存访问令牌和刷新令牌以实现长期自动访问
4. 显示连接成功消息

**重要说明**：
- 首次登录后，程序会缓存令牌，90天内无需重复登录
- 令牌会自动刷新，提供无缝的长期访问体验
- 如果缓存过期，只需重新运行 `test` 命令重新登录

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
- ✅ 简洁的 CLI 接口
- ✅ JSON 格式配置支持
- ✅ 批量处理功能

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
    └── microsoft-todo/       # Microsoft Todo 客户端
```

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

**免责声明**：本工具使用第三方服务（Microsoft Graph API），请确保遵守相关服务的使用条款和隐私政策。