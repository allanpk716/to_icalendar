package dify

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/allanpk716/to_icalendar/internal/models"
)

// MockDifyClient 模拟Dify客户端
type MockDifyClient struct {
	mock.Mock
}

func (m *MockDifyClient) ProcessImage(ctx context.Context, imageData []byte, fileName string, userID string) (*models.DifyResponse, error) {
	args := m.Called(ctx, imageData, fileName, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.DifyResponse), args.Error(1)
}

// MockResponseParser 模拟响应解析器
type MockResponseParser struct {
	mock.Mock
}

func (m *MockResponseParser) ParseReminderResponse(response string) (*models.ParsedTaskInfo, error) {
	args := m.Called(response)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ParsedTaskInfo), args.Error(1)
}

func TestScreenshotProcessor_ProcessScreenshot_Success(t *testing.T) {
	// 准备测试数据
	config := &models.DifyConfig{
		APIEndpoint: "https://api.dify.ai/v1",
		APIKey:      "test-key",
		Timeout:     30,
	}

	// 创建模拟对象
	mockClient := new(MockDifyClient)
	mockParser := new(MockResponseParser)

	// 设置模拟行为
	testImageData := []byte("fake-image-data")
	testFileName := "test.png"

	difyResponse := &models.DifyResponse{
		Answer:    `{"title":"测试任务","date":"2025-11-15","time":"14:00","priority":"medium"}`,
		MessageID: "msg-123",
	}

	parsedInfo := &models.ParsedTaskInfo{
		Title:      "测试任务",
		Date:       "2025-11-15",
		Time:       "14:00",
		Priority:   "medium",
		Confidence: 0.9,
	}

	mockClient.On("ProcessImage", mock.Anything, testImageData, testFileName, mock.AnythingOfType("string")).Return(difyResponse, nil)
	mockParser.On("ParseReminderResponse", difyResponse.Answer).Return(parsedInfo, nil)

	// 创建处理器
	processor := &ScreenshotProcessorImpl{
		client: mockClient,
		parser: mockParser,
		config: config,
	}

	// 执行测试
	screenshot := &ScreenshotInput{
		Data:     testImageData,
		FileName: testFileName,
		Format:   "png",
	}

	ctx := context.Background()
	result, err := processor.ProcessScreenshot(ctx, screenshot)

	// 验证结果
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "测试任务", result.Title)
	assert.Equal(t, "2025-11-15", result.Date)
	assert.Equal(t, "14:00", result.Time)
	assert.Equal(t, "medium", result.Priority)
	assert.Equal(t, "15m", result.RemindBefore) // 默认值
	assert.Equal(t, "Default", result.List)     // 默认值

	// 验证模拟调用
	mockClient.AssertExpectations(t)
	mockParser.AssertExpectations(t)
}

func TestScreenshotProcessor_ValidateInput_EmptyData(t *testing.T) {
	config := &models.DifyConfig{
		APIEndpoint: "https://api.dify.ai/v1",
		APIKey:      "test-key",
		Timeout:     30,
	}

	processor := &ScreenshotProcessorImpl{
		config: config,
	}

	screenshot := &ScreenshotInput{
		Data:     []byte{},
		FileName: "empty.png",
		Format:   "png",
	}

	err := processor.ValidateInput(screenshot)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "screenshot data is empty")
}

func TestScreenshotProcessor_ValidateInput_NilInput(t *testing.T) {
	config := &models.DifyConfig{
		APIEndpoint: "https://api.dify.ai/v1",
		APIKey:      "test-key",
		Timeout:     30,
	}

	processor := &ScreenshotProcessorImpl{
		config: config,
	}

	err := processor.ValidateInput(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "screenshot input is nil")
}

func TestScreenshotProcessor_ValidateInput_UnsupportedFormat(t *testing.T) {
	config := &models.DifyConfig{
		APIEndpoint: "https://api.dify.ai/v1",
		APIKey:      "test-key",
		Timeout:     30,
	}

	processor := &ScreenshotProcessorImpl{
		config: config,
	}

	screenshot := &ScreenshotInput{
		Data:     []byte("fake-data"),
		FileName: "test.xyz",
		Format:   "xyz",
	}

	err := processor.ValidateInput(screenshot)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported image format")
}

func TestScreenshotProcessor_ValidateInput_FileTooLarge(t *testing.T) {
	config := &models.DifyConfig{
		APIEndpoint: "https://api.dify.ai/v1",
		APIKey:      "test-key",
		Timeout:     30,
	}

	processor := &ScreenshotProcessorImpl{
		config: config,
	}

	// 创建超过10MB的数据
	largeData := make([]byte, 11*1024*1024) // 11MB
	screenshot := &ScreenshotInput{
		Data:     largeData,
		FileName: "large.png",
		Format:   "png",
	}

	err := processor.ValidateInput(screenshot)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file size")
	assert.Contains(t, err.Error(), "exceeds maximum allowed size")
}

func TestScreenshotProcessor_ValidateImageContent_InvalidFormat(t *testing.T) {
	config := &models.DifyConfig{
		APIEndpoint: "https://api.dify.ai/v1",
		APIKey:      "test-key",
		Timeout:     30,
	}

	processor := &ScreenshotProcessorImpl{
		config: config,
	}

	// 测试无效图片数据
	invalidData := []byte("this is not an image")
	err := processor.validateImageContent(invalidData)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid image format")
}

func TestScreenshotProcessor_ValidateImageContent_ValidFormats(t *testing.T) {
	config := &models.DifyConfig{
		APIEndpoint: "https://api.dify.ai/v1",
		APIKey:      "test-key",
		Timeout:     30,
	}

	processor := &ScreenshotProcessorImpl{
		config: config,
	}

	// 测试PNG文件头
	pngData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	err := processor.validateImageContent(pngData)
	assert.NoError(t, err)

	// 测试JPEG文件头
	jpegData := []byte{0xFF, 0xD8, 0xFF, 0xE0}
	err = processor.validateImageContent(jpegData)
	assert.NoError(t, err)
}

func TestScreenshotProcessor_GetProcessorInfo(t *testing.T) {
	config := &models.DifyConfig{
		APIEndpoint: "https://api.dify.ai/v1",
		APIKey:      "test-key",
		Timeout:     30,
	}

	processor := &ScreenshotProcessorImpl{
		config: config,
	}

	info := processor.GetProcessorInfo()
	assert.NotNil(t, info)
	assert.Equal(t, "DifyScreenshotProcessor", info.Name)
	assert.Equal(t, "1.0.0", info.Version)
	assert.Contains(t, info.SupportedFormats, "png")
	assert.Contains(t, info.SupportedFormats, "jpg")
	assert.Equal(t, int64(10*1024*1024), info.MaxFileSize)
}

func TestExtractImageFormat(t *testing.T) {
	tests := []struct {
		fileName string
		expected string
	}{
		{"test.png", "png"},
		{"image.JPG", "jpg"},
		{"photo.jpeg", "jpeg"},
		{"picture.bmp", "bmp"},
		{"animation.gif", "gif"},
		{"unknown.xyz", "xyz"},
		{"noextension", "unknown"},
	}

	for _, test := range tests {
		t.Run(test.fileName, func(t *testing.T) {
			result := ExtractImageFormat(test.fileName)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestNewScreenshotProcessor_InvalidConfig(t *testing.T) {
	// 测试空配置
	config := &models.DifyConfig{}
	processor, err := NewScreenshotProcessor(config)
	assert.Nil(t, processor)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid dify config")
}

func TestNewScreenshotProcessor_ValidConfig(t *testing.T) {
	config := &models.DifyConfig{
		APIEndpoint: "https://api.dify.ai/v1",
		APIKey:      "test-key",
		Timeout:     30,
	}

	processor, err := NewScreenshotProcessor(config)
	assert.NoError(t, err)
	assert.NotNil(t, processor)
	assert.NotNil(t, processor.client)
	assert.NotNil(t, processor.parser)
}

func TestGenerateRequestID(t *testing.T) {
	id1 := generateRequestID()
	time.Sleep(1 * time.Millisecond) // 确保时间差
	id2 := generateRequestID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2)
	assert.Contains(t, id1, "scr_")
	assert.Contains(t, id2, "scr_")
}