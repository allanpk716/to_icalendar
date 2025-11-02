# to_icalendar

一个通过Pushcut发送提醒事项到iOS提醒事项应用的Go程序。

## 功能特性

- 🚀 通过Pushcut直接创建iOS提醒事项
- 📝 JSON格式配置提醒事项
- ⏰ 支持自定义提醒时间
- 📦 支持批量发送提醒事项
- 🎯 支持设置优先级和分类
- 🌐 跨平台支持（Windows、Linux、macOS）
- 🔄 iCloud自动同步到所有iOS设备

## 原理说明

本程序通过Pushcut API将提醒事项数据发送到iOS设备，然后在iOS设备上通过快捷指令自动创建真正的提醒事项。相比CalDAV方案，这种方法能够确保提醒事项出现在iOS Reminders应用中，而不是日历应用中。

## 安装

### 从源码编译

```bash
git clone https://github.com/allanpk716/to_icalendar.git
cd to_icalendar
go mod tidy
go build -o to_icalendar main.go
```

## 使用方法

### 1. iOS设备设置

1. **安装Pushcut应用**
   - 在App Store中搜索并安装"Pushcut"
   - 打开应用并完成基本设置

2. **创建快捷指令**
   - 在Pushcut中创建新的快捷指令
   - 配置快捷指令接收提醒事项数据并创建iOS提醒事项
   - 获取Webhook API端点信息

3. **获取API密钥**
   - 在Pushcut设置中找到API密钥
   - 记录Webhook ID用于配置

### 2. 初始化配置

```bash
./to_icalendar init
```

这将创建配置文件模板：
- `config/server.yaml` - Pushcut配置
- `config/reminder.json` - 提醒事项模板

### 3. 配置Pushcut

编辑 `config/server.yaml`：

```yaml
pushcut:
  api_key: "your_pushcut_api_key"
  webhook_id: "your_webhook_id"
  timezone: "Asia/Shanghai"
```

### 4. 创建提醒事项

编辑提醒事项JSON文件：

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
| `list` | string | ❌ | 提醒事项列表名称 |

### 5. 发送提醒事项

```bash
# 发送单个提醒事项
./to_icalendar upload config/reminder.json

# 批量发送提醒事项
./to_icalendar upload reminders/*.json
```

### 6. 其他命令

```bash
# 测试Pushcut连接
./to_icalendar test

# 显示帮助
./to_icalendar help
```

## iOS快捷指令配置

在Pushcut中创建快捷指令时，确保快捷指令能够：

1. **接收输入数据**：配置快捷指令接收HTTP请求中的JSON数据
2. **解析提醒信息**：从JSON中提取title、description、date、time等字段
3. **创建提醒事项**：使用"添加新提醒事项"动作创建提醒
4. **设置提醒时间**：根据date和time字段设置提醒时间
5. **配置优先级**：根据priority字段设置提醒优先级

快捷指令示例流程：
```
收到Webhook请求
    ↓
解析JSON数据
    ↓
提取提醒信息
    ↓
创建iOS提醒事项
    ↓
设置提醒时间和属性
```

## 提醒时间格式

`remind_before` 字段支持以下格式：

- `15m` - 提前15分钟
- `1h` - 提前1小时
- `2d` - 提前2天
- `30m` - 提前30分钟

## 优先级说明

- `high` - 高优先级
- `medium` - 中等优先级 - 默认
- `low` - 低优先级

## 故障排除

### 连接失败

1. 确认Pushcut API密钥和Webhook ID正确
2. 检查网络连接
3. 确认iOS设备上的Pushcut应用正常运行
4. 检查快捷指令配置是否正确

### 提醒事项未创建

1. 检查快捷指令是否正确配置
2. 确认iOS设备上的提醒事项应用权限
3. 验证JSON数据格式是否正确
4. 检查提醒时间是否在未来

### 同步问题

1. 确认iCloud提醒事项同步已启用
2. 检查iOS设备网络连接
3. 重启iOS设备上的提醒事项应用

## 示例

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

## 技术架构

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

## 许可证

MIT License

## 贡献

欢迎提交Issue和Pull Request！