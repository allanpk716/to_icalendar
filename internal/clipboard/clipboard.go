package clipboard

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"time"
	"unsafe"

	"github.com/allanpk716/to_icalendar/internal/models"
	"github.com/atotto/clipboard"
	"github.com/disintegration/imaging"
	"golang.org/x/sys/windows"
)

// Windows API constants
const (
	CF_DIB     = 8
	CF_BITMAP = 2
)

var (
	user32           = windows.NewLazySystemDLL("user32.dll")
	kernel32         = windows.NewLazySystemDLL("kernel32.dll")

	procOpenClipboard    = user32.NewProc("OpenClipboard")
	procCloseClipboard   = user32.NewProc("CloseClipboard")
	procGetClipboardData = user32.NewProc("GetClipboardData")
	procEnumClipboardFormats = user32.NewProc("EnumClipboardFormats")
	procIsClipboardFormatAvailable = user32.NewProc("IsClipboardFormatAvailable")
	procGlobalLock   = kernel32.NewProc("GlobalLock")
	procGlobalUnlock = kernel32.NewProc("GlobalUnlock")
	procGlobalSize   = kernel32.NewProc("GlobalSize")
)

// BITMAPINFO structure for Windows DIB format
type BITMAPINFOHEADER struct {
	Size          uint32
	Width         int32
	Height        int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter int32
	YPelsPerMeter int32
	ClrUsed       uint32
	ClrImportant  uint32
}

type BITMAPINFO struct {
	Header BITMAPINFOHEADER
	Colors [256]uint32 // Maximum palette size for 8-bit images
}

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

// ReadImage reads image data from clipboard using Windows API
func (r *WindowsClipboardReader) ReadImage() ([]byte, error) {
	// Open clipboard
	ret, _, err := procOpenClipboard.Call(0)
	if ret == 0 {
		return nil, fmt.Errorf("failed to open clipboard: %v", err)
	}
	defer procCloseClipboard.Call()

	// Check if DIB format is available
	ret, _, err = procIsClipboardFormatAvailable.Call(uintptr(CF_DIB))
	if ret == 0 {
		return nil, fmt.Errorf("no image data in clipboard")
	}

	// Get clipboard data handle
	handle, _, err := procGetClipboardData.Call(uintptr(CF_DIB))
	if handle == 0 {
		return nil, fmt.Errorf("failed to get clipboard data: %v", err)
	}

	// Lock the global memory to get a pointer
	pointer, _, err := procGlobalLock.Call(handle)
	if pointer == 0 {
		return nil, fmt.Errorf("failed to lock global memory: %v", err)
	}
	defer procGlobalUnlock.Call(handle)

	// Get the size of the data
	size, _, err := procGlobalSize.Call(handle)
	if size == 0 {
		return nil, fmt.Errorf("failed to get global memory size: %v", err)
	}

	// Read the DIB data
	data := make([]byte, size)
	copy(data, (*[1 << 30]byte)(unsafe.Pointer(pointer))[:size:size])

	// Parse BITMAPINFOHEADER
	if len(data) < int(unsafe.Sizeof(BITMAPINFOHEADER{})) {
		return nil, fmt.Errorf("insufficient data for BITMAPINFOHEADER")
	}

	header := (*BITMAPINFOHEADER)(unsafe.Pointer(&data[0]))

	// Calculate image properties
	width := int(header.Width)
	height := int(header.Height)
	if height < 0 {
		height = -height // Top-down bitmap
	}

	// Calculate stride (bytes per row)
	var stride int
	switch header.BitCount {
	case 32:
		stride = width * 4
	case 24:
		stride = ((width * 3 + 3) / 4) * 4 // Align to 4 bytes
	case 8:
		stride = ((width + 3) / 4) * 4 // Align to 4 bytes
	default:
		return nil, fmt.Errorf("unsupported bit count: %d", header.BitCount)
	}

	// Find the start of pixel data (after BITMAPINFOHEADER and palette)
	offset := int(unsafe.Sizeof(BITMAPINFOHEADER{}))
	if header.BitCount == 8 {
		offset += 256 * 4 // Palette for 8-bit images
	}

	if offset >= len(data) {
		return nil, fmt.Errorf("invalid DIB data structure")
	}

	// Extract pixel data
	pixelData := data[offset:]
	if len(pixelData) < stride*height {
		return nil, fmt.Errorf("insufficient pixel data")
	}

	// Convert to Go image format
	var img image.Image
	switch header.BitCount {
	case 32:
		img = r.convertBGRAtoRGBA(pixelData, width, height, stride)
	case 24:
		img = r.convertBGRtoRGB(pixelData, width, height, stride)
	case 8:
		img = r.convert8BitToRGBA(pixelData, data[offset-256*4:offset], width, height, stride)
	default:
		return nil, fmt.Errorf("unsupported bit count: %d", header.BitCount)
	}

	// Encode as PNG
	var buf bytes.Buffer
	err = png.Encode(&buf, img)
	if err != nil {
		return nil, fmt.Errorf("failed to encode image as PNG: %w", err)
	}

	return buf.Bytes(), nil
}

// HasContent checks if clipboard has any readable content
func (r *WindowsClipboardReader) HasContent() (bool, error) {
	// Try to read text first
	text, err := clipboard.ReadAll()
	if err == nil && text != "" {
		return true, nil
	}

	// Check for image content
	ret, _, err := procOpenClipboard.Call(0)
	if ret == 0 {
		// Can't open clipboard, likely empty or access denied
		return false, nil
	}
	defer procCloseClipboard.Call()

	// Check if image format is available
	ret, _, _ = procIsClipboardFormatAvailable.Call(uintptr(CF_DIB))
	if ret != 0 {
		return true, nil
	}

	return false, nil
}

// GetContentType determines the type of content in clipboard
func (r *WindowsClipboardReader) GetContentType() (models.ContentType, error) {
	// Try to read text first
	text, err := clipboard.ReadAll()
	if err == nil && text != "" {
		return models.ContentTypeText, nil
	}

	// Check for image content
	ret, _, err := procOpenClipboard.Call(0)
	if ret == 0 {
		// Can't open clipboard, likely empty or access denied
		return models.ContentTypeEmpty, nil
	}
	defer procCloseClipboard.Call()

	// Check if image format is available
	ret, _, _ = procIsClipboardFormatAvailable.Call(uintptr(CF_DIB))
	if ret != 0 {
		return models.ContentTypeImage, nil
	}

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

// convertBGRAtoRGBA converts 32-bit BGRA pixel data to RGBA image
func (r *WindowsClipboardReader) convertBGRAtoRGBA(pixelData []byte, width, height, stride int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			srcOffset := y*stride + x*4
			dstOffset := y*img.Stride + x*4

			if srcOffset+3 < len(pixelData) && dstOffset+3 < len(img.Pix) {
				// Convert BGRA to RGBA
				img.Pix[dstOffset+0] = pixelData[srcOffset+2] // R
				img.Pix[dstOffset+1] = pixelData[srcOffset+1] // G
				img.Pix[dstOffset+2] = pixelData[srcOffset+0] // B
				img.Pix[dstOffset+3] = pixelData[srcOffset+3] // A
			}
		}
	}

	return img
}

// convertBGRtoRGB converts 24-bit BGR pixel data to RGB image
func (r *WindowsClipboardReader) convertBGRtoRGB(pixelData []byte, width, height, stride int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			srcOffset := y*stride + x*3
			dstOffset := y*img.Stride + x*4

			if srcOffset+2 < len(pixelData) && dstOffset+3 < len(img.Pix) {
				// Convert BGR to RGBA
				img.Pix[dstOffset+0] = pixelData[srcOffset+2] // R
				img.Pix[dstOffset+1] = pixelData[srcOffset+1] // G
				img.Pix[dstOffset+2] = pixelData[srcOffset+0] // B
				img.Pix[dstOffset+3] = 0xFF                   // A (fully opaque)
			}
		}
	}

	return img
}

// convert8BitToRGBA converts 8-bit palette pixel data to RGBA image
func (r *WindowsClipboardReader) convert8BitToRGBA(pixelData, palette []byte, width, height, stride int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			srcOffset := y*stride + x
			dstOffset := y*img.Stride + x*4

			if srcOffset < len(pixelData) && dstOffset+3 < len(img.Pix) {
				palIndex := int(pixelData[srcOffset]) * 4
				if palIndex+3 < len(palette) {
					// Convert BGR palette to RGBA
					img.Pix[dstOffset+0] = palette[palIndex+2] // R
					img.Pix[dstOffset+1] = palette[palIndex+1] // G
					img.Pix[dstOffset+2] = palette[palIndex+0] // B
					img.Pix[dstOffset+3] = 0xFF                  // A (fully opaque)
				}
			}
		}
	}

	return img
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