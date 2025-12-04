# Dify 图片处理模块

这个模块提供了专门用于处理图片输入的Dify集成功能，可以从截图中识别任务信息并转换为标准格式。

## 功能特性

- 🖼️ **图片处理**: 支持PNG、JPG、JPEG、BMP、GIF等主流图片格式
- 🤖 **AI识别**: 通过Dify workflow智能识别截图中的任务信息
- ✅ **数据验证**: 完整的输入验证和数据格式检查
- 🔧 **独立模块**: 可以独立使用，不依赖其他处理模块
- 📊 **错误处理**: 详细的错误信息和处理状态反馈

## 核心组件

### 1. ScreenshotProcessor 接口

定义了截图处理的标准接口：

```go
type ScreenshotProcessor interface {
    ProcessScreenshot(ctx context.Context, screenshot *ScreenshotInput) (*models.Reminder, error)
    ValidateInput(screenshot *ScreenshotInput) error
    GetProcessorInfo() *ProcessorInfo
}
```

### 2. ScreenshotInput 输入结构

```go
type ScreenshotInput struct {
    Data      []byte `json:"data"`       // 图片二进制数据
    FileName  string `json:"file_name"`  // 文件名
    Format    string `json:"format"`     // 图片格式 (png, jpg, etc.)
}
```

### 3. 配置结构

简化的三字段配置：

```go
type DifyConfig struct {
    APIEndpoint string `yaml:"api_endpoint"` // Dify API 端点
    APIKey      string `yaml:"api_key"`      // Dify API 密钥
    Timeout     int    `yaml:"timeout"`      // 请求超时时间（秒）
}
```

## 使用方法

### 1. 基本使用

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/allanpk716/to_icalendar/internal/dify"
    "github.com/allanpk716/to_icalendar/internal/models"
)

func main() {
    // 配置Dify
    config := &models.DifyConfig{
        APIEndpoint: "https://api.dify.ai/v1",
        APIKey:      "your-api-key-here",
        Timeout:     30,
    }

    // 创建处理器
    processor, err := dify.NewScreenshotProcessor(config)
    if err != nil {
        panic(err)
    }

    // 读取图片
    imageData, err := os.ReadFile("screenshot.png")
    if err != nil {
        panic(err)
    }

    // 创建输入
    screenshot := &dify.ScreenshotInput{
        Data:     imageData,
        FileName: "screenshot.png",
        Format:   "png",
    }

    // 处理截图
    ctx := context.Background()
    reminder, err := processor.ProcessScreenshot(ctx, screenshot)
    if err != nil {
        panic(err)
    }

    // 使用结果
    fmt.Printf("任务标题: %s\n", reminder.Title)
    fmt.Printf("任务日期: %s\n", reminder.Date)
    fmt.Printf("任务时间: %s\n", reminder.Time)
}
```

### 2. 独立验证输入

```go
// 只验证输入，不处理
err := processor.ValidateInput(screenshot)
if err != nil {
    fmt.Printf("输入验证失败: %v\n", err)
    return
}
fmt.Println("输入验证通过")
```

### 3. 获取处理器信息

```go
info := processor.GetProcessorInfo()
fmt.Printf("处理器: %s v%s\n", info.Name, info.Version)
fmt.Printf("支持格式: %v\n", info.SupportedFormats)
fmt.Printf("最大文件大小: %d MB\n", info.MaxFileSize/(1024*1024))
```

## 运行示例

```bash
# 编译并运行示例
cd examples
go run dify_screenshot_example.go test_screenshot.png
```

## 测试

运行单元测试：

```bash
go test ./internal/dify/...
```

运行特定测试：

```bash
go test ./internal/dify/ -run TestScreenshotProcessor
go test ./internal/dify/ -run TestResponseParser
```

## 支持的图片格式

- **PNG**: .png
- **JPEG**: .jpg, .jpeg
- **BMP**: .bmp
- **GIF**: .gif

## 文件大小限制

- 默认最大文件大小：10MB
- 可通过修改 `ScreenshotProcessorImpl` 中的 `maxSize` 调整

## 错误处理

模块提供了详细的错误信息：

```go
reminder, err := processor.ProcessScreenshot(ctx, screenshot)
if err != nil {
    switch {
    case strings.Contains(err.Error(), "input validation failed"):
        fmt.Println("输入验证失败")
    case strings.Contains(err.Error(), "dify processing failed"):
        fmt.Println("Dify处理失败")
    case strings.Contains(err.Error(), "response parsing failed"):
        fmt.Println("响应解析失败")
    default:
        fmt.Printf("未知错误: %v\n", err)
    }
}
```

## 输出格式

处理成功后返回标准的 `models.Reminder` 结构：

```go
type Reminder struct {
    Title        string   `json:"title"`         // 任务标题
    Description  string   `json:"description"`   // 任务描述
    Date         string   `json:"date"`          // 任务日期 (YYYY-MM-DD)
    Time         string   `json:"time"`          // 任务时间 (HH:MM)
    RemindBefore string   `json:"remind_before"` // 提前提醒时间
    Priority     Priority `json:"priority"`      // 优先级 (low/medium/high)
    List         string   `json:"list"`          // 任务列表
}
```

## 配置要求

确保您的 `config/server.yaml` 包含正确的Dify配置：

```yaml
dify:
  api_endpoint: "http://dify.urithub.com/v1"
  api_key: "your-dify-api-key"
  timeout: 30
```

## Workflow 要求

您的Dify workflow应该：

1. **输入字段**: 接收 `screenshot` 字段（图片文件）
2. **输出格式**: 返回JSON格式的任务信息，包含以下字段：
   - `title`: 任务标题（必需）
   - `date`: 任务日期，格式YYYY-MM-DD（必需）
   - `time`: 任务时间，格式HH:MM（必需）
   - `description`: 任务描述（可选）
   - `priority`: 优先级（可选）
   - `remind_before`: 提前提醒时间（可选）
   - `list`: 任务列表（可选）

示例输出：
```json
{
  "title": "团队周会",
  "description": "讨论本周项目进度",
  "date": "2025-11-15",
  "time": "14:00",
  "priority": "high",
  "remind_before": "15m",
  "list": "Work"
}
```