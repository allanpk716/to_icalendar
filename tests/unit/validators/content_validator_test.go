package validators_test

import (
	"testing"

	"github.com/allanpk716/to_icalendar/internal/validators"
)

func TestContentValidator_NewContentValidator(t *testing.T) {
	validator := validators.NewContentValidator()
	if validator == nil {
		t.Error("Expected validator but got nil")
	}
}

func TestContentValidator_ValidateText(t *testing.T) {
	validator := validators.NewContentValidator()

	tests := []struct {
		name        string
		text        string
		expectValid bool
		expectError bool
	}{
		{
			name:        "valid short text",
			text:        "明天下午2点开会",
			expectValid: true,
			expectError: false,
		},
		{
			name:        "valid medium text",
			text:        "明天下午2点在会议室A开会讨论项目进展，请准时参加，需要准备相关材料",
			expectValid: true,
			expectError: false,
		},
		{
			name:        "empty text",
			text:        "",
			expectValid: false,
			expectError: false,
		},
		{
			name:        "whitespace only text",
			text:        "   \n\t   ",
			expectValid: false,
			expectError: false,
		},
		{
			name:        "text that's too long",
			text:        string(make([]byte, 11000)), // 11KB text
			expectValid: false,
			expectError: false,
		},
		{
			name:        "maximum valid text",
			text:        string(make([]byte, 9000)), // 9KB text
			expectValid: true,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validation := validator.ValidateText(tt.text)

			if validation.IsValid != tt.expectValid {
				t.Errorf("Expected valid=%v, got %v", tt.expectValid, validation.IsValid)
			}

			if (validation.ErrorMessage != "") != tt.expectError {
				if tt.expectError && validation.ErrorMessage == "" {
					t.Error("Expected error message but got none")
				} else if !tt.expectError && validation.ErrorMessage != "" {
					t.Errorf("Unexpected error message: %s", validation.ErrorMessage)
				}
			}
		})
	}
}

func TestContentValidator_ValidateImage(t *testing.T) {
	validator := validators.NewContentValidator()

	tests := []struct {
		name        string
		imageData   []byte
		fileName    string
		expectValid bool
		expectError bool
	}{
		{
			name:        "valid small image",
			imageData:   make([]byte, 1000), // 1KB
			fileName:    "test.png",
			expectValid: true,
			expectError: false,
		},
		{
			name:        "empty image data",
			imageData:   []byte{},
			fileName:    "test.png",
			expectValid: false,
			expectError: false,
		},
		{
			name:        "nil image data",
			imageData:   nil,
			fileName:    "test.png",
			expectValid: false,
			expectError: false,
		},
		{
			name:        "image that's too large",
			imageData:   make([]byte, 11*1024*1024), // 11MB
			fileName:    "large.png",
			expectValid: false,
			expectError: false,
		},
		{
			name:        "valid large image",
			imageData:   make([]byte, 5*1024*1024), // 5MB
			fileName:    "medium.jpg",
			expectValid: true,
			expectError: false,
		},
		{
			name:        "unsupported file format",
			imageData:   make([]byte, 1000),
			fileName:    "test.exe",
			expectValid: false,
			expectError: false,
		},
		{
			name:        "supported PNG format",
			imageData:   make([]byte, 1000),
			fileName:    "screenshot.png",
			expectValid: true,
			expectError: false,
		},
		{
			name:        "supported JPG format",
			imageData:   make([]byte, 1000),
			fileName:    "photo.jpg",
			expectValid: true,
			expectError: false,
		},
		{
			name:        "supported JPEG format",
			imageData:   make([]byte, 1000),
			fileName:    "image.jpeg",
			expectValid: true,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validation := validator.ValidateImage(tt.imageData, tt.fileName)

			if validation.IsValid != tt.expectValid {
				t.Errorf("Expected valid=%v, got %v", tt.expectValid, validation.IsValid)
			}

			if (validation.ErrorMessage != "") != tt.expectError {
				if tt.expectError && validation.ErrorMessage == "" {
					t.Error("Expected error message but got none")
				} else if !tt.expectError && validation.ErrorMessage != "" {
					t.Errorf("Unexpected error message: %s", validation.ErrorMessage)
				}
			}
		})
	}
}

func TestContentValidator_ValidateAPIEndpoint(t *testing.T) {
	validator := validators.NewContentValidator()

	tests := []struct {
		name        string
		endpoint    string
		expectValid bool
		expectError bool
	}{
		{
			name:        "valid HTTPS endpoint",
			endpoint:    "https://api.dify.ai/v1",
			expectValid: true,
			expectError: false,
		},
		{
			name:        "valid HTTP endpoint",
			endpoint:    "http://localhost:8000/v1",
			expectValid: true,
			expectError: false,
		},
		{
			name:        "empty endpoint",
			endpoint:    "",
			expectValid: false,
			expectError: false,
		},
		{
			name:        "invalid URL format",
			endpoint:    "not-a-url",
			expectValid: false,
			expectError: false,
		},
		{
			name:        "missing protocol",
			endpoint:    "api.dify.ai/v1",
			expectValid: false,
			expectError: false,
		},
		{
			name:        "invalid protocol",
			endpoint:    "ftp://api.dify.ai/v1",
			expectValid: false,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validation := validator.ValidateAPIEndpoint(tt.endpoint)

			if validation.IsValid != tt.expectValid {
				t.Errorf("Expected valid=%v, got %v", tt.expectValid, validation.IsValid)
			}

			if (validation.ErrorMessage != "") != tt.expectError {
				if tt.expectError && validation.ErrorMessage == "" {
					t.Error("Expected error message but got none")
				} else if !tt.expectError && validation.ErrorMessage != "" {
					t.Errorf("Unexpected error message: %s", validation.ErrorMessage)
				}
			}
		})
	}
}

func TestContentValidator_ValidateAPIKey(t *testing.T) {
	validator := validators.NewContentValidator()

	tests := []struct {
		name        string
		apiKey      string
		expectValid bool
		expectError bool
	}{
		{
			name:        "valid API key",
			apiKey:      "sk-1234567890abcdef",
			expectValid: true,
			expectError: false,
		},
		{
			name:        "empty API key",
			apiKey:      "",
			expectValid: false,
			expectError: false,
		},
		{
			name:        "placeholder API key",
			apiKey:      "YOUR_DIFY_API_KEY",
			expectValid: false,
			expectError: false,
		},
		{
			name:        "short API key",
			apiKey:      "123",
			expectValid: false,
			expectError: false,
		},
		{
			name:        "whitespace only API key",
			apiKey:      "   \t   ",
			expectValid: false,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validation := validator.ValidateAPIKey(tt.apiKey)

			if validation.IsValid != tt.expectValid {
				t.Errorf("Expected valid=%v, got %v", tt.expectValid, validation.IsValid)
			}

			if (validation.ErrorMessage != "") != tt.expectError {
				if tt.expectError && validation.ErrorMessage == "" {
					t.Error("Expected error message but got none")
				} else if !tt.expectError && validation.ErrorMessage != "" {
					t.Errorf("Unexpected error message: %s", validation.ErrorMessage)
				}
			}
		})
	}
}

// Benchmark tests
func BenchmarkContentValidator_ValidateText(b *testing.B) {
	validator := validators.NewContentValidator()
	testText := "明天下午2点开会讨论项目进展，请准时参加"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validator.ValidateText(testText)
	}
}

func BenchmarkContentValidator_ValidateImage(b *testing.B) {
	validator := validators.NewContentValidator()
	imageData := make([]byte, 1000)
	fileName := "test.png"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validator.ValidateImage(imageData, fileName)
	}
}