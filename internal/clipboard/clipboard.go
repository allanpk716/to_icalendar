package clipboard

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"time"

	"github.com/allanpk716/to_icalendar/internal/models"
	"github.com/atotto/clipboard"
	"github.com/disintegration/imaging"
)

// WindowsClipboardReader implements Reader interface for Windows platform
type WindowsClipboardReader struct{}

// NewClipboardReader creates a new clipboard reader based on the platform
func NewClipboardReader() (Reader, error) {
	return &WindowsClipboardReader{}, nil
}

// ReadText reads text content from clipboard
func (r *WindowsClipboardReader) ReadText() (string, error) {
	text, err := clipboard.ReadAll()
	if err != nil {
		return "", fmt.Errorf("failed to read text from clipboard: %w", err)
	}

	if text == "" {
		return "", fmt.Errorf("clipboard is empty or no text content")
	}

	return text, nil
}

// ReadImage reads image data from clipboard
func (r *WindowsClipboardReader) ReadImage() ([]byte, error) {
	// Note: The atotto/clipboard library doesn't support image reading directly
	// For Windows, we would need to use Win32 API calls
	// For now, we'll implement a basic approach and enhance later

	// This is a placeholder implementation
	// In a real implementation, you would use platform-specific APIs
	// like golang.org/x/sys/windows for Windows clipboard image access

	return nil, fmt.Errorf("image clipboard reading not yet implemented - requires platform-specific API integration")
}

// HasContent checks if clipboard has any readable content
func (r *WindowsClipboardReader) HasContent() (bool, error) {
	// Try to read text first
	text, err := clipboard.ReadAll()
	if err == nil && text != "" {
		return true, nil
	}

	// TODO: Add image content detection when image reading is implemented

	return false, nil
}

// GetContentType determines the type of content in clipboard
func (r *WindowsClipboardReader) GetContentType() (models.ContentType, error) {
	// Try to read text first
	text, err := clipboard.ReadAll()
	if err == nil && text != "" {
		return models.ContentTypeText, nil
	}

	// TODO: Add image content detection when image reading is implemented

	// If no content found, return empty
	return models.ContentTypeEmpty, nil
}

// Read reads any available content from clipboard
func (r *WindowsClipboardReader) Read() (*models.ClipboardContent, error) {
	content := &models.ClipboardContent{}

	// Try to read text first
	text, err := r.ReadText()
	if err == nil && text != "" {
		content.Type = models.ContentTypeText
		content.Text = text
		return content, nil
	}

	// Try to read image (placeholder implementation)
	imageData, err := r.ReadImage()
	if err == nil && imageData != nil {
		content.Type = models.ContentTypeImage
		content.Image = imageData
		content.FileName = fmt.Sprintf("clipboard_%s.png", time.Now().Format("20060102_150405"))
		return content, nil
	}

	return nil, fmt.Errorf("no readable content found in clipboard")
}

// EnhancedClipboardReader implements advanced clipboard functionality
// This is a more advanced implementation that would handle image reading
type EnhancedClipboardReader struct {
	*WindowsClipboardReader
}

// NewEnhancedClipboardReader creates an enhanced clipboard reader
func NewEnhancedClipboardReader() (*EnhancedClipboardReader, error) {
	return &EnhancedClipboardReader{
		WindowsClipboardReader: &WindowsClipboardReader{},
	}, nil
}

// ProcessImage processes image data for better compatibility
func ProcessImage(imageData []byte, format string) ([]byte, error) {
	// Decode the image
	img, _, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// Resize if too large (optional)
	maxSize := 1024
	if img.Bounds().Dx() > maxSize || img.Bounds().Dy() > maxSize {
		img = imaging.Resize(img, maxSize, 0, imaging.Lanczos)
	}

	// Encode as PNG for consistency
	var buf bytes.Buffer
	err = png.Encode(&buf, img)
	if err != nil {
		return nil, fmt.Errorf("failed to encode image as PNG: %w", err)
	}

	return buf.Bytes(), nil
}

// ValidateClipboardContent validates the clipboard content
func ValidateClipboardContent(content *models.ClipboardContent) error {
	if content == nil {
		return fmt.Errorf("clipboard content is nil")
	}

	switch content.Type {
	case models.ContentTypeText:
		if content.Text == "" {
			return fmt.Errorf("text content is empty")
		}
	case models.ContentTypeImage:
		if len(content.Image) == 0 {
			return fmt.Errorf("image content is empty")
		}
		if content.FileName == "" {
			content.FileName = fmt.Sprintf("clipboard_%s.png", time.Now().Format("20060102_150405"))
		}
	case models.ContentTypeEmpty:
		return fmt.Errorf("clipboard content is empty")
	default:
		return fmt.Errorf("unknown clipboard content type: %s", content.Type)
	}

	return nil
}