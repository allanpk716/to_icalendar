package clipboard_test

import (
	"testing"

	"github.com/allanpk716/to_icalendar/internal/clipboard"
	"github.com/allanpk716/to_icalendar/internal/models"
)

func TestManager_NewManager(t *testing.T) {
	manager, err := clipboard.NewManager()
	if err != nil {
		t.Fatalf("Failed to create clipboard manager: %v", err)
	}

	if manager == nil {
		t.Error("Expected manager but got nil")
	}
}

func TestManager_HasContent(t *testing.T) {
	manager, err := clipboard.NewManager()
	if err != nil {
		t.Fatalf("Failed to create clipboard manager: %v", err)
	}

	// This test may pass or fail depending on system clipboard state
	// We're mainly testing that the method doesn't panic
	hasContent, err := manager.HasContent()
	if err != nil {
		t.Errorf("Failed to check clipboard content: %v", err)
	}

	// hasContent can be true or false, both are valid
	t.Logf("Clipboard has content: %v", hasContent)
}

func TestManager_GetContentType(t *testing.T) {
	manager, err := clipboard.NewManager()
	if err != nil {
		t.Fatalf("Failed to create clipboard manager: %v", err)
	}

	contentType, err := manager.GetContentType()
	if err != nil {
		t.Errorf("Failed to get content type: %v", err)
	}

	// Verify content type is valid
	switch contentType {
	case models.ContentTypeText, models.ContentTypeImage, models.ContentTypeEmpty, models.ContentTypeUnknown:
		// Valid types
		t.Logf("Content type: %s", contentType)
	default:
		t.Errorf("Invalid content type: %s", contentType)
	}
}

func TestManager_Read(t *testing.T) {
	manager, err := clipboard.NewManager()
	if err != nil {
		t.Fatalf("Failed to create clipboard manager: %v", err)
	}

	// Check if clipboard has content first
	hasContent, err := manager.HasContent()
	if err != nil {
		t.Fatalf("Failed to check clipboard content: %v", err)
	}

	if !hasContent {
		t.Skip("Clipboard is empty, skipping read test")
	}

	content, err := manager.Read()
	if err != nil {
		t.Errorf("Failed to read clipboard content: %v", err)
		return
	}

	if content == nil {
		t.Error("Expected content but got nil")
		return
	}

	// Validate content type
	switch content.Type {
	case models.ContentTypeText:
		if content.Text == "" {
			t.Error("Text content is empty")
		}
		t.Logf("Read text content: %q", content.Text)
	case models.ContentTypeImage:
		if len(content.Image) == 0 {
			t.Error("Image content is empty")
		}
		t.Logf("Read image content: %d bytes", len(content.Image))
	default:
		t.Errorf("Unsupported content type: %s", content.Type)
	}
}

// Benchmark tests for performance measurement
func BenchmarkManager_HasContent(b *testing.B) {
	manager, err := clipboard.NewManager()
	if err != nil {
		b.Fatalf("Failed to create clipboard manager: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = manager.HasContent()
	}
}

func BenchmarkManager_GetContentType(b *testing.B) {
	manager, err := clipboard.NewManager()
	if err != nil {
		b.Fatalf("Failed to create clipboard manager: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = manager.GetContentType()
	}
}