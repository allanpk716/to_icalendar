package clipboard

import (
	"context"
	"testing"

	"github.com/allanpk716/to_icalendar/internal/models"
	"github.com/allanpk716/to_icalendar/internal/services"
)

// TestClipboardService_EmptyClipboard 测试空剪贴板的错误处理
func TestClipboardService_EmptyClipboard(t *testing.T) {
	// 创建剪贴板服务实例
	logger := &mockLogger{}
	service := services.NewClipboardService(logger)

	// 测试读取空剪贴板
	ctx := context.Background()
	content, err := service.ReadContent(ctx)

	// 应该返回错误
	if err == nil {
		t.Error("期望返回错误，但返回了 nil")
	}

	// 验证错误信息包含关键信息
	expectedError := "no readable content found in clipboard"
	if err.Error() != expectedError {
		t.Errorf("错误信息不匹配，期望: %s, 实际: %s", expectedError, err.Error())
	}

	// 验证内容为 nil
	if content != nil {
		t.Error("期望内容为 nil，但返回了有效内容")
	}
}

// TestClipboardService_ProcessEmptyContent 测试处理空内容的错误处理
func TestClipboardService_ProcessEmptyContent(t *testing.T) {
	// 创建剪贴板服务实例
	logger := &mockLogger{}
	service := services.NewClipboardService(logger)

	// 测试处理空内容
	ctx := context.Background()
	_, err := service.ProcessContent(ctx, nil)

	// 应该返回错误
	if err == nil {
		t.Error("期望返回错误，但返回了 nil")
	}

	// 验证错误信息
	expectedError := "剪贴板内容为空"
	if err.Error() != expectedError {
		t.Errorf("错误信息不匹配，期望: %s, 实际: %s", expectedError, err.Error())
	}
}

// TestClipboardService_ValidContentProcessing 测试处理有效内容
func TestClipboardService_ValidContentProcessing(t *testing.T) {
	// 创建剪贴板服务实例
	logger := &mockLogger{}
	service := services.NewClipboardService(logger)

	// 创建测试内容
	testContent := &models.ClipboardContent{
		Type:     models.ContentTypeText,
		Text:     "测试文本内容",
		Metadata: map[string]interface{}{
			"length": 6,
			"format": "text",
		},
	}

	// 测试处理有效内容
	ctx := context.Background()
	result, err := service.ProcessContent(ctx, testContent)

	// 应该成功
	if err != nil {
		t.Errorf("期望成功，但返回了错误: %v", err)
	}

	// 验证结果
	if result == nil {
		t.Error("期望返回结果，但返回了 nil")
	}

	if result != nil && !result.Success {
		t.Error("期望结果为成功状态")
	}
}

// TestClipboardService_HasContent 测试检查剪贴板内容
func TestClipboardService_HasContent(t *testing.T) {
	// 创建剪贴板服务实例
	logger := &mockLogger{}
	service := services.NewClipboardService(logger)

	// 测试检查剪贴板内容
	hasContent, err := service.HasContent()

	// 在测试环境中，剪贴板可能为空，这是正常的
	// 我们主要测试调用是否成功，不会导致 panic
	if err != nil {
		t.Errorf("检查剪贴板内容时返回错误: %v", err)
	}

	// hasContent 的值取决于实际的剪贴板状态
	// 这里我们只验证返回值是布尔类型
	t.Logf("剪贴板有内容: %v", hasContent)
}

// TestClipboardService_GetContentType 测试获取内容类型
func TestClipboardService_GetContentType(t *testing.T) {
	// 创建剪贴板服务实例
	logger := &mockLogger{}
	service := services.NewClipboardService(logger)

	// 测试获取内容类型
	contentType, err := service.GetContentType()

	// 在测试环境中，可能会返回错误（如果剪贴板为空）
	if err != nil {
		t.Logf("获取内容类型时返回错误（可能是正常的）: %v", err)
		return
	}

	// 验证返回的内容类型是有效的字符串
	if contentType == "" {
		t.Error("期望返回有效的内容类型，但返回了空字符串")
	}

	t.Logf("内容类型: %s", contentType)
}

// mockLogger 模拟日志记录器
type mockLogger struct{}

func (m *mockLogger) Debug(format string, args ...interface{}) {}
func (m *mockLogger) Info(format string, args ...interface{})  {}
func (m *mockLogger) Warn(format string, args ...interface{})  {}
func (m *mockLogger) Error(format string, args ...interface{}) {}
func (m *mockLogger) Fatal(format string, args ...interface{}) {}

// TestClipboardContent_Metadata 测试剪贴板内容的元数据
func TestClipboardContent_Metadata(t *testing.T) {
	// 测试创建带有元数据的剪贴板内容
	content := &models.ClipboardContent{
		Type:     models.ContentTypeImage,
		Image:    []byte{1, 2, 3, 4, 5},
		FileName: "test.png",
		Metadata: map[string]interface{}{
			"size":   5,
			"format": "png",
			"width":  100,
			"height": 100,
		},
	}

	// 验证元数据
	if content.Metadata == nil {
		t.Error("期望有元数据，但 Metadata 为 nil")
	}

	size, ok := content.Metadata["size"].(int)
	if !ok {
		t.Error("期望 size 是整数类型")
	}

	if size != 5 {
		t.Errorf("期望 size 为 5，实际为 %d", size)
	}

	format, ok := content.Metadata["format"].(string)
	if !ok {
		t.Error("期望 format 是字符串类型")
	}

	if format != "png" {
		t.Errorf("期望 format 为 'png'，实际为 '%s'", format)
	}
}