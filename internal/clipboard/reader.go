package clipboard

import (
	"github.com/allanpk716/to_icalendar/internal/cache"
	"github.com/allanpk716/to_icalendar/internal/models"
)

// Reader defines the interface for reading clipboard content
type Reader interface {
	// ReadText reads text content from clipboard
	// Returns the text content and error if any
	ReadText() (string, error)

	// ReadImage reads image data from clipboard
	// Returns the image data as bytes and error if any
	ReadImage() ([]byte, error)

	// HasContent checks if clipboard has any readable content
	// Returns true if content is available, false otherwise
	HasContent() (bool, error)

	// GetContentType determines the type of content in clipboard
	// Returns the content type and error if any
	GetContentType() (models.ContentType, error)

	// Read reads any available content from clipboard
	// Returns ClipboardContent with either text or image data
	Read() (*models.ClipboardContent, error)
}

// Manager manages clipboard operations
type Manager struct {
	reader Reader
}

// NewManager creates a new clipboard manager
func NewManager() (*Manager, error) {
	reader, err := NewClipboardReader()
	if err != nil {
		return nil, err
	}

	return &Manager{
		reader: reader,
	}, nil
}

// NewManagerWithUnifiedCache creates a new clipboard manager with unified cache manager
func NewManagerWithUnifiedCache(unifiedCacheMgr *cache.UnifiedCacheManager) (*Manager, error) {
	reader, err := NewClipboardReaderWithUnifiedCache(unifiedCacheMgr)
	if err != nil {
		return nil, err
	}

	return &Manager{
		reader: reader,
	}, nil
}

// ReadText reads text content from clipboard
func (m *Manager) ReadText() (string, error) {
	return m.reader.ReadText()
}

// ReadImage reads image data from clipboard
func (m *Manager) ReadImage() ([]byte, error) {
	return m.reader.ReadImage()
}

// HasContent checks if clipboard has any readable content
func (m *Manager) HasContent() (bool, error) {
	return m.reader.HasContent()
}

// GetContentType determines the type of content in clipboard
func (m *Manager) GetContentType() (models.ContentType, error) {
	return m.reader.GetContentType()
}

// Read reads any available content from clipboard
func (m *Manager) Read() (*models.ClipboardContent, error) {
	return m.reader.Read()
}