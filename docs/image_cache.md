# 图片缓存功能使用指南

## 功能概述

图片缓存功能已成功集成到 to_icalendar 项目中，用于诊断和调试剪贴板图片识别问题。现在系统会自动保存：

1. **原始图片**：从剪贴板直接读取的原始图片数据
2. **标准化后图片**：经过标准化处理后的图片数据

## 缓存机制

### 文件命名规则

- **原始图片**：`clipboard_original_{timestamp}.png`
- **标准化后图片**：`clipboard_normalized_{timestamp}.png`

其中 `timestamp` 格式为：`20060102_150405_000000`（精确到微秒）

### 缓存位置

默认缓存目录：`./cache/images`（相对于程序运行目录）

配置文件：`./image_processing.json`

## 配置选项

### 启用/禁用缓存

```json
{
  "enable_cache": true,
  "cache_dir": "cache/images",
  "max_cache_files": 50
}
```

**参数说明：**
- `enable_cache`: 是否启用图片缓存
- `cache_dir`: 缓存目录路径
- `max_cache_files`: 最大缓存文件数量，超出后自动删除最旧的文件

## 使用方法

### 1. 基本使用

```bash
# 复制图片到剪贴板（比如公司OA软件的会议通知截图）
# 然后运行：
./to_icalendar clip-upload
```

程序会自动：
- 读取剪贴板图片
- 保存原始图片到缓存目录
- 应用标准化处理
- 保存标准化后图片到缓存目录
- 发送到 Dify 进行识别

### 2. 查看缓存文件

```bash
# 查看缓存目录
ls cache/images

# 输出示例：
# clipboard_original_20251114_092458_123456.png
# clipboard_normalized_20251114_092458_123456.png
```

### 3. 对比分析

现在您可以对比原始图片和标准化后图片：

1. **原始图片** (`clipboard_original_*.png`):
   - 直接从剪贴板读取的数据
   - 保持原始分辨率和质量
   - 用于分析原始输入是否正确

2. **标准化后图片** (`clipboard_normalized_*.png`):
   - 经过尺寸调整、压缩优化的图片
   - 发送给 Dify API 的实际数据
   - 用于分析标准化过程是否合适

## 问题诊断流程

### 1. 复现问题

```bash
# 复制您的OA软件会议通知截图到剪贴板
./to_icalendar clip-upload
```

### 2. 检查缓存

```bash
# 查看最新生成的缓存文件
ls -la cache/images | tail -2
```

### 3. 分析图片质量

使用图片查看器打开：
- `clipboard_original_*.png` - 检查原始图片是否清晰、完整
- `clipboard_normalized_*.png` - 检查标准化后图片是否过度压缩或失真

### 4. 对比识别结果

根据图片内容分析可能的问题：

**原始图片问题：**
- 图片模糊或不清晰
- 文字太小难以识别
- 颜色对比度不足
- 图片格式异常

**标准化问题：**
- 尺寸调整导致文字过小
- 压缩过度导致细节丢失
- 色彩空间转换问题

## 高级配置

### 调整标准化参数

编辑 `image_processing.json` 文件：

```json
{
  "normalization": {
    "max_width": 1920,
    "max_height": 1080,
    "png_compression_level": 6,
    "output_format": "png",
    "keep_aspect_ratio": true
  }
}
```

### 针对OA软件的优化建议

```json
{
  "normalization": {
    "max_width": 1280,
    "max_height": 960,
    "png_compression_level": 3,
    "output_format": "png",
    "keep_aspect_ratio": true
  }
}
```

**说明：**
- 较小的压缩级别（3 vs 6）保持更好的图片质量
- 适中的尺寸平衡质量和文件大小

### 文档类图片优化

```json
{
  "normalization": {
    "max_width": 800,
    "max_height": 600,
    "png_compression_level": 6,
    "output_format": "png",
    "keep_aspect_ratio": true
  }
}
```

## 缓存管理

### 自动清理

系统会在保存新缓存文件时检查文件数量，如果超过 `max_cache_files` 限制，会自动删除最旧的文件。

### 手动清理

```bash
# 清空所有缓存文件
rm -rf cache/images/*

# 或者只删除特定类型的文件
rm cache/images/clipboard_original_*
rm cache/images/clipboard_normalized_*
```

## 故障排除

### 1. 缓存文件未生成

**检查配置：**
```bash
cat image_processing.json | grep enable_cache
```

**检查权限：**
```bash
ls -la cache/
```

### 2. 图片文件损坏

检查日志输出：
```bash
./to_icalendar clip-upload 2>&1 | grep -i cache
```

### 3. 缓存目录问题

确保目录存在且有写权限：
```bash
mkdir -p cache/images
chmod 755 cache/images
```

## 调试技巧

### 1. 启用详细日志

编辑配置启用调试模式：
```json
{
  "debug_mode": true,
  "debug_output_dir": "debug/images"
}
```

### 2. 手动测试标准化

使用测试程序验证功能：
```bash
go build -o test_cache.exe ./cmd/test_cache/main.go
./test_cache.exe
```

### 3. 分析文件大小

对比文件大小有助于识别问题：
```bash
ls -lh cache/images/
```

- 如果标准化后文件大小明显小于原始文件，可能过度压缩
- 如果标准化后文件大小接近或大于原始文件，可能标准化失败

## 示例场景

### 场景1：OA软件会议通知识别错误

**问题：** 识别结果与实际内容完全不符

**诊断步骤：**
1. 复制OA软件截图
2. 运行 `./to_icalendar clip-upload`
3. 检查 `clipboard_original_*.png` - 确认原始图片清晰
4. 检查 `clipboard_normalized_*.png` - 确认标准化后图片质量
5. 对比两个文件，判断是否是标准化过程导致的问题

### 场景2：识别结果包含乱码

**问题：** 识别结果包含不正确的文字

**可能原因：**
- 图片分辨率过低导致文字模糊
- 压缩过度丢失细节
- 色彩对比度不足

**解决方案：**
调整标准化参数，降低压缩级别，提高分辨率限制。

## 总结

图片缓存功能为您提供了强大的诊断工具，通过对比原始图片和标准化后图片，您可以准确定位识别问题的根源，并通过调整配置参数来优化识别效果。

现在您可以轻松地：
- 🔍 **诊断问题**：查看实际发送给 Dify 的图片
- 🎛️ **调整参数**：优化标准化设置
- 📊 **对比效果**：验证配置变更的效果
- 🗂️ **管理缓存**：控制存储空间使用