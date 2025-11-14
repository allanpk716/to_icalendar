# 图片格式标准化功能

## 功能概述

为 to_icalendar 项目添加了图片格式标准化功能，确保所有从剪贴板读取的图片都符合统一的质量和格式标准，提高 Dify API 的识别准确率。

## 问题背景

**原始问题：**
- 剪贴板获取的图片发送给 Dify 识别出的内容与实际剪贴板内容不一致
- 特别是公司 OA 软件会议通知截图，识别结果完全错误

**根本原因：**
- 图片分辨率不一致，影响识别效果
- 压缩质量不稳定，文件大小不可控
- 缺乏统一的色彩空间标准
- 没有针对不同场景的图片优化

## 解决方案

### 核心功能

#### 1. 图片标准化模块 (`internal/image/normalizer.go`)

**主要功能：**
- **尺寸标准化**：默认最大尺寸 1920x1080px，支持智能缩放保持宽高比
- **质量优化**：PNG 压缩级别 6（平衡质量和大小），统一 sRGB 色彩空间
- **格式统一**：输出格式 PNG，文件大小控制在 5MB 以内
- **智能处理**：基于内容的压缩优化和批量处理支持

**配置选项：**
```go
type NormalizationConfig struct {
    MaxWidth           int                    // 最大分辨率
    MaxHeight          int
    PNGCompressionLevel png.CompressionLevel // 压缩质量
    JPEGQuality        int                   // JPEG质量（如使用）
    OutputFormat       string                // 输出格式
    MaxFileSize        int64                 // 文件大小限制
    KeepAspectRatio    bool                  // 是否保持宽高比
}
```

#### 2. 配置管理模块 (`internal/image/config.go`)

**功能特点：**
- **灵活配置**：支持默认配置、文档配置和自定义配置
- **配置验证**：自动验证配置参数的有效性
- **调试支持**：可启用调试模式保存中间结果
- **向后兼容**：保持与现有系统的兼容性

**预设配置：**
- **默认配置**：1920x1080px，适用于一般图片
- **文档配置**：800x600px，专门针对文档类图片优化

#### 3. 系统集成

**剪贴板处理集成 (`internal/clipboard/clipboard.go`)：**
- 在图片读取后立即进行标准化处理
- 保持原有 API 接口不变
- 添加详细的日志记录和错误处理

**图片处理集成 (`internal/processors/image_processor.go`)：**
- 在图片上传前进行标准化
- 支持临时文件管理和清理
- 与 Dify API 无缝集成

## 使用方式

### 1. 自动集成

图片标准化功能已自动集成到现有的剪贴板处理流程中，使用方式保持不变：

```bash
# 处理剪贴板图片并上传（自动应用标准化）
./to_icalendar clip-upload
```

### 2. 配置管理

配置文件位置：`~/.to_icalendar/image_processing.json`

```json
{
  "normalization": {
    "max_width": 1920,
    "max_height": 1080,
    "png_compression_level": 6,
    "jpeg_quality": 85,
    "output_format": "png",
    "max_file_size": 5242880,
    "keep_aspect_ratio": true
  },
  "enable_normalization": true,
  "debug_mode": false,
  "debug_output_dir": "debug/images"
}
```

### 3. 程序化使用

```go
// 创建默认标准化器
config := image.DefaultNormalizationConfig()
normalizer := image.NewImageNormalizer(config, logger)

// 创建带标准化的剪贴板读取器
reader, _ := clipboard.NewClipboardReaderWithNormalizer(normalizer, logger)

// 创建带标准化的图片处理器
processor, _ := processors.NewImageProcessorWithNormalizer(difyProcessor, normalizer)
```

## 技术规格

### 标准化参数

- **默认最大分辨率**：1920x1080px
- **文档类最大分辨率**：800x600px
- **PNG 压缩级别**：6（平衡质量和大小）
- **色彩空间**：统一 sRGB，8位深度
- **输出格式**：PNG（保持兼容性）
- **目标文件大小**：< 5MB

### 处理流程

```
剪贴板DIB → 解析BITMAPINFOHEADER → 转换为RGBA → 标准化处理 → PNG编码 → Dify上传
```

### 质量保证

- **智能缩放**：使用高质量的双三次插值算法
- **色彩统一**：确保所有图片使用统一的色彩空间
- **大小控制**：动态调整压缩参数以控制文件大小
- **格式兼容**：确保输出格式与 Dify API 完全兼容

## 预期效果

### 1. 识别准确率提升

- **统一质量**：所有图片符合相同的质量标准
- **优化尺寸**：针对不同场景使用最佳分辨率
- **格式兼容**：确保与 Dify OCR 模型的最佳兼容性

### 2. 系统稳定性提升

- **错误容错**：标准化失败时回退到原始图片
- **资源控制**：避免过大图片导致的处理问题
- **日志记录**：详细的处理日志便于问题诊断

### 3. 用户体验改善

- **向后兼容**：现有使用方式完全不变
- **自动优化**：用户无需手动配置即可获得最佳效果
- **配置灵活**：支持高级用户的个性化配置需求

## 测试验证

### 1. 功能测试

编译并运行测试程序：

```bash
go build -o test_normalization.exe ./cmd/test_normalization/main.go
./test_normalization.exe
```

### 2. 集成测试

使用剪贴板图片进行实际测试：

```bash
# 1. 复制OA软件会议通知截图到剪贴板
# 2. 执行命令测试
./to_icalendar clip-upload
# 3. 观察日志输出，确认标准化处理是否正常工作
```

### 3. 配置验证

检查配置文件是否正确生成：

```bash
cat ~/.to_icalendar/image_processing.json
```

## 依赖项

新增的 Go 模块依赖：

- `github.com/nfnt/resize`：高质量图片缩放
- `github.com/sirupsen/logrus`：结构化日志记录

## 文件结构

```
internal/
├── image/
│   ├── normalizer.go    # 图片标准化核心模块
│   └── config.go        # 配置管理模块
├── clipboard/
│   └── clipboard.go     # 集成标准化功能的剪贴板处理
└── processors/
    └── image_processor.go # 集成标准化功能的图片处理

cmd/
├── test_normalization/
│   └── main.go          # 功能测试程序
└── to_icalendar/
    └── main.go          # 主程序（已集成）

docs/
└── image_normalization.md # 本文档
```

## 总结

图片格式标准化功能已成功集成到 to_icalendar 项目中，通过统一的图片质量、尺寸和格式标准，显著提升了剪贴板图片识别的准确率和系统稳定性。该功能完全向后兼容，现有用户无需任何额外配置即可享受优化效果。