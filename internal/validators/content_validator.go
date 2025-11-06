package validators

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	MaxTextLength   = 10000   // 最大文字长度
	MaxImageSize    = 10 * 1024 * 1024 // 最大图片大小 (10MB)
	MinImageSize    = 100     // 最小图片大小
	MaxFileNameLength = 255   // 最大文件名长度
)

// ValidationResult represents the result of content validation
type ValidationResult struct {
	IsValid  bool   `json:"is_valid"`
	ErrorType string `json:"error_type"`
	Message  string `json:"message"`
}

// ContentValidator handles validation of various content types
type ContentValidator struct {
	textPattern *regexp.Regexp
	allowedImageTypes []string
}

// NewContentValidator creates a new content validator
func NewContentValidator() *ContentValidator {
	return &ContentValidator{
		textPattern: regexp.MustCompile(`^[\p{L}\p{N}\s\p{P}\p{S}]*$`),
		allowedImageTypes: []string{
			"image/png",
			"image/jpeg",
			"image/jpg",
			"image/bmp",
			"image/gif",
			"image/webp",
		},
	}
}

// ValidateText validates text content
func (v *ContentValidator) ValidateText(text string) *ValidationResult {
	// 检查空内容
	if strings.TrimSpace(text) == "" {
		return &ValidationResult{
			IsValid:   false,
			ErrorType: "empty_content",
			Message:   "Text content cannot be empty",
		}
	}

	// 检查长度
	if len(text) > MaxTextLength {
		return &ValidationResult{
			IsValid:   false,
			ErrorType: "content_too_long",
			Message:   fmt.Sprintf("Text content too long: %d characters (max: %d)", len(text), MaxTextLength),
		}
	}

	// 检查是否包含危险字符
	if !v.textPattern.MatchString(text) {
		return &ValidationResult{
			IsValid:   false,
			ErrorType: "invalid_characters",
			Message:   "Text contains invalid characters",
		}
	}

	return &ValidationResult{
		IsValid: true,
	}
}

// ValidateImage validates image data
func (v *ContentValidator) ValidateImage(data []byte, fileName string) *ValidationResult {
	// 检查数据大小
	if len(data) == 0 {
		return &ValidationResult{
			IsValid:   false,
			ErrorType: "empty_data",
			Message:   "Image data cannot be empty",
		}
	}

	if len(data) < MinImageSize {
		return &ValidationResult{
			IsValid:   false,
			ErrorType: "data_too_small",
			Message:   fmt.Sprintf("Image data too small: %d bytes (min: %d)", len(data), MinImageSize),
		}
	}

	if len(data) > MaxImageSize {
		return &ValidationResult{
			IsValid:   false,
			ErrorType: "data_too_large",
			Message:   fmt.Sprintf("Image data too large: %d bytes (max: %d)", len(data), MaxImageSize),
		}
	}

	// 检查文件名
	if err := v.validateFileName(fileName); err != nil {
		return &ValidationResult{
			IsValid:   false,
			ErrorType: "invalid_filename",
			Message:   err.Error(),
		}
	}

	// 检查图片格式（通过头部字节）
	if err := v.validateImageFormat(data); err != nil {
		return &ValidationResult{
			IsValid:   false,
			ErrorType: "invalid_format",
			Message:   err.Error(),
		}
	}

	return &ValidationResult{
		IsValid: true,
	}
}

// validateFileName validates file name for security
func (v *ContentValidator) validateFileName(fileName string) error {
	if fileName == "" {
		return fmt.Errorf("file name cannot be empty")
	}

	if len(fileName) > MaxFileNameLength {
		return fmt.Errorf("file name too long: %d characters (max: %d)", len(fileName), MaxFileNameLength)
	}

	// 检查危险字符
	dangerousChars := []string{"..", "/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	for _, char := range dangerousChars {
		if strings.Contains(fileName, char) {
			return fmt.Errorf("file name contains dangerous character: %s", char)
		}
	}

	// 检查文件扩展名
	allowedExtensions := []string{".png", ".jpg", ".jpeg", ".bmp", ".gif", ".webp"}
	hasValidExtension := false
	for _, ext := range allowedExtensions {
		if strings.HasSuffix(strings.ToLower(fileName), ext) {
			hasValidExtension = true
			break
		}
	}

	if !hasValidExtension {
		return fmt.Errorf("invalid file extension, allowed: %v", allowedExtensions)
	}

	return nil
}

// validateImageFormat validates image format by checking header bytes
func (v *ContentValidator) validateImageFormat(data []byte) error {
	if len(data) < 8 {
		return fmt.Errorf("insufficient data to determine format")
	}

	// PNG 格式检查 (89 50 4E 47 0D 0A 1A 0A)
	if len(data) >= 8 &&
	   data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 &&
	   data[4] == 0x0D && data[5] == 0x0A && data[6] == 0x1A && data[7] == 0x0A {
		return nil
	}

	// JPEG 格式检查 (FF D8 FF)
	if len(data) >= 3 && data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		return nil
	}

	// BMP 格式检查 (42 4D)
	if len(data) >= 2 && data[0] == 0x42 && data[1] == 0x4D {
		return nil
	}

	// GIF 格式检查 (47 49 46 38)
	if len(data) >= 4 &&
	   data[0] == 0x47 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x38 {
		return nil
	}

	// WebP 格式检查 (52 49 46 46 ... 57 45 42 50)
	if len(data) >= 12 &&
	   data[0] == 0x52 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x46 &&
	   data[8] == 0x57 && data[9] == 0x45 && data[10] == 0x42 && data[11] == 0x50 {
		return nil
	}

	return fmt.Errorf("unsupported image format, supported formats: PNG, JPEG, BMP, GIF, WebP")
}

// ValidateAPIEndpoint validates API endpoint URL
func (v *ContentValidator) ValidateAPIEndpoint(endpoint string) *ValidationResult {
	if endpoint == "" {
		return &ValidationResult{
			IsValid:   false,
			ErrorType: "empty_endpoint",
			Message:   "API endpoint cannot be empty",
		}
	}

	// 简单的URL格式检查
	if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
		return &ValidationResult{
			IsValid:   false,
			ErrorType: "invalid_endpoint",
			Message:   "API endpoint must start with http:// or https://",
		}
	}

	return &ValidationResult{
		IsValid: true,
	}
}

// ValidateAPIKey validates API key format
func (v *ContentValidator) ValidateAPIKey(apiKey string) *ValidationResult {
	if apiKey == "" {
		return &ValidationResult{
			IsValid:   false,
			ErrorType: "empty_api_key",
			Message:   "API key cannot be empty",
		}
	}

	if len(apiKey) < 16 {
		return &ValidationResult{
			IsValid:   false,
			ErrorType: "invalid_api_key",
			Message:   "API key too short, must be at least 16 characters",
		}
	}

	// 检查是否为示例密钥
	if apiKey == "YOUR_DIFY_API_KEY" || apiKey == "sk-" || apiKey == "your_api_key_here" {
		return &ValidationResult{
			IsValid:   false,
			ErrorType: "placeholder_api_key",
			Message:   "Please configure a valid API key",
		}
	}

	return &ValidationResult{
		IsValid: true,
	}
}

// GetValidationRules returns information about validation rules
func (v *ContentValidator) GetValidationRules() map[string]interface{} {
	return map[string]interface{}{
		"max_text_length":    MaxTextLength,
		"max_image_size":     MaxImageSize,
		"min_image_size":     MinImageSize,
		"max_filename_length": MaxFileNameLength,
		"allowed_image_types": v.allowedImageTypes,
	}
}