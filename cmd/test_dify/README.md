# Dify 图片识别测试程序

这是一个专门用于测试 Dify 图片识别提醒事项功能的独立测试程序。

## 功能特性

- 🖼️ **图片格式支持**: PNG, JPG, JPEG, BMP, GIF
- 📊 **详细日志**: 完整的处理步骤和性能统计
- ⚙️ **配置集成**: 自动加载项目的 Dify 配置
- 📝 **结果输出**: 支持控制台输出和文件保存
- 🚀 **易于使用**: 简单的命令行接口
- 🔍 **错误诊断**: 详细的错误信息和调试日志

## 快速开始

### 基本用法

```bash
# 测试本地图片文件
go run main.go /path/to/your/image.jpg

# 使用详细输出模式
go run main.go -verbose /path/to/your/image.jpg

# 保存结果到JSON文件
go run main.go -output result.json /path/to/your/image.jpg

# 使用项目配置文件（默认）
go run main.go -config ../../config/server.yaml /path/to/your/image.jpg
```

### 测试示例图片

程序已包含一张示例测试图片：

```bash
# 使用内置示例图片测试
go run main.go test-images/test_screenshot.jpg

# 详细模式测试示例图片
go run main.go -verbose test-images/test_screenshot.jpg
```

### 编译后使用

```bash
# 编译程序
go build -o dify-test main.go

# 运行编译后的程序
./dify-test /path/to/image.jpg
```

## 命令行选项

| 选项 | 默认值 | 说明 |
|------|--------|------|
| `-verbose` | false | 启用详细输出模式，显示更多处理信息 |
| `-output` | "" | 将测试结果保存到指定的JSON文件 |
| `-url` | "" | 从指定URL下载测试图片（暂未实现） |
| `-config` | `../../config/server.yaml` | 指定配置文件路径 |
| `-version` | false | 显示程序版本信息 |
| `-help` | false | 显示帮助信息 |

## 输出信息说明

### 基本输出

- **配置信息**: Dify API 端点和设置
- **图片信息**: 文件路径、格式、大小
- **处理状态**: 成功/失败状态
- **处理时间**: 总耗时统计

### 详细输出（-verbose 模式）

- **处理器信息**: 版本和支持格式
- **验证步骤**: 每个验证过程的详细信息
- **API 调用**: 请求和响应详情
- **性能统计**: 各阶段处理时间

### 识别结果

成功识别后会显示：

```
=== 识别结果 ===
标题: 会议提醒
描述: 参加产品评审会议
日期: 2024-12-25
时间: 14:30
提前提醒: 15m
优先级: medium
任务列表: Work
```

### JSON 格式预览

```json
{
  "title": "会议提醒",
  "description": "参加产品评审会议",
  "date": "2024-12-25",
  "time": "14:30",
  "remind_before": "15m",
  "priority": "medium",
  "list": "Work"
}
```

## 配置要求

程序需要有效的 Dify API 配置。配置文件应包含：

```yaml
dify:
  api_endpoint: "http://your-dify-endpoint/v1"
  api_key: "your-dify-api-key"
  timeout: 30
```

### 配置验证

程序会自动验证：
- ✅ API 端点格式
- ✅ API 密钥有效性
- ✅ 超时设置合理性
- ✅ 其他配置参数

## 支持的图片格式

| 格式 | 扩展名 | 最大大小 |
|------|--------|----------|
| PNG | .png | 10MB |
| JPEG | .jpg, .jpeg | 10MB |
| BMP | .bmp | 10MB |
| GIF | .gif | 10MB |

## 错误处理

### 常见错误类型

1. **配置错误**
   ```
   ❌ 配置加载失败: Dify API key is required
   ```
   解决：检查配置文件中的 API 密钥

2. **文件错误**
   ```
   ❌ 图片准备失败: 文件不存在: /path/to/image.jpg
   ```
   解决：确认图片文件路径正确

3. **格式错误**
   ```
   ❌ 图片准备失败: 不支持的文件格式: .txt
   ```
   解决：使用支持的图片格式

4. **API 错误**
   ```
   ❌ 测试执行失败: dify processing failed: API timeout
   ```
   解决：检查网络连接和 API 服务状态

### 调试技巧

1. **使用详细模式**
   ```bash
   go run main.go -verbose image.jpg
   ```

2. **检查配置文件**
   ```bash
   go run main.go -config config/server.yaml image.jpg
   ```

3. **保存结果用于分析**
   ```bash
   go run main.go -output debug.json image.jpg
   ```

## 示例用法

### 1. 基本测试
```bash
# 测试单张图片
go run main.go screenshot.png
```

### 2. 详细分析
```bash
# 显示所有处理步骤
go run main.go -verbose screenshot.jpg
```

### 3. 批量测试脚本
```bash
#!/bin/bash
# 批量测试多张图片
for image in test-images/*.jpg; do
    echo "测试: $image"
    go run main.go -verbose -output "results/$(basename "$image" .json)" "$image"
done
```

### 4. 性能测试
```bash
# 测试大文件处理
go run main.go -verbose large_image.jpg

# 保存性能数据
go run main.go -output performance.json large_image.jpg
```

## 项目结构

```
cmd/test_dify/
├── main.go                 # 主程序文件
├── README.md              # 使用说明（本文件）
└── test-images/           # 测试图片目录
    └── test_screenshot.jpg # 示例测试图片
```

## 技术细节

### 处理流程

1. **配置加载**: 读取并验证配置文件
2. **图片验证**: 检查格式、大小、文件头
3. **API 调用**: 发送图片到 Dify API
4. **响应解析**: 提取结构化任务信息
5. **结果输出**: 格式化显示和保存结果

### 依赖项

- `github.com/allanpk716/to_icalendar/internal/dify` - Dify 客户端
- `github.com/allanpk716/to_icalendar/internal/config` - 配置管理
- `github.com/allanpk716/to_icalendar/internal/models` - 数据模型

### 性能特性

- **内存优化**: 流式处理大文件
- **并发安全**: 支持多实例运行
- **超时控制**: 防止长时间等待
- **错误恢复**: 优雅的错误处理

## 故障排除

### 连接问题

如果遇到 Dify API 连接问题：

1. 检查网络连接
2. 验证 API 端点地址
3. 确认 API 密钥有效
4. 检查防火墙设置

### 性能问题

如果处理速度较慢：

1. 使用 `-verbose` 查看详细耗时
2. 检查图片文件大小
3. 优化网络环境
4. 调整超时设置

### 内存问题

如果遇到内存不足：

1. 减小图片文件大小
2. 使用较低分辨率图片
3. 关闭详细日志输出

## 开发说明

### 扩展功能

程序设计为易于扩展：

- 添加新的图片格式支持
- 集成其他 AI 服务
- 自定义输出格式
- 批量处理功能

### 调试模式

使用详细模式进行开发调试：

```bash
go run main.go -verbose -debug-output debug.json test.jpg
```

## 许可证

本程序遵循项目整体许可证。

## 贡献

欢迎提交问题报告和功能请求。