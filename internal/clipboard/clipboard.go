package clipboard

import (
	"bytes"
	"fmt"
	stdimage "image"
	"image/png"
	_ "image/jpeg" // Support JPEG decoding
	"os"
	"path/filepath"
	"strings"
	"time"
	"unsafe"

	"github.com/allanpk716/to_icalendar/internal/image"
	"github.com/allanpk716/to_icalendar/internal/models"
	"github.com/atotto/clipboard"
	"github.com/disintegration/imaging"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/windows"
)

// Windows API constants
const (
	CF_TEXT            = 1
	CF_BITMAP          = 2
	CF_METAFILEPICT    = 3
	CF_SYLK            = 4
	CF_DIF             = 5
	CF_TIFF            = 6
	CF_OEMTEXT         = 7
	CF_DIB             = 8
	CF_PALETTE         = 9
	CF_PENDATA         = 10
	CF_RIFF            = 11
	CF_WAVE            = 12
	CF_UNICODETEXT     = 13
	CF_ENHMETAFILE     = 14
	CF_HDROP           = 15
	CF_LOCALE          = 16
	CF_DIBV5           = 17  // Remote Desktop format
	CF_MAX             = 18
	CF_OWNERDISPLAY    = 0x0080
	CF_DSPTEXT         = 0x0081
	CF_DSPBITMAP       = 0x0082
	CF_DSPMETAFILEPICT = 0x0083
	CF_DSPENHMETAFILE  = 0x008E
	CF_PRIVATEFIRST    = 0x0200
	CF_PRIVATELAST     = 0x02FF
	CF_GDIOBJFIRST     = 0x0300
	CF_GDIOBJLAST      = 0x03FF
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

// BITMAPV5HEADER structure for Windows DIBV5 format (Remote Desktop)
type BITMAPV5HEADER struct {
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
	RedMask       uint32
	GreenMask     uint32
	BlueMask      uint32
	AlphaMask     uint32
	CSType        uint32
	Endpoints     [3]uint32 // 3 * 4 uint32 values for RGB
	GammaRed      uint32
	GammaGreen    uint32
	GammaBlue     uint32
	Intent        uint32
	ProfileData   uint32
	ProfileSize   uint32
	Reserved      uint32
}

// WindowsClipboardReader implements Reader interface for Windows platform
type WindowsClipboardReader struct {
	normalizer    *image.ImageNormalizer
	logger        *logrus.Logger
	configManager *image.ConfigManager
}

// NewClipboardReader creates a new clipboard reader based on the platform
func NewClipboardReader() (Reader, error) {
	logger := logrus.New()
	configManager := image.NewConfigManager(".", logger)
	if err := configManager.LoadConfig(); err != nil {
		logger.Warnf("加载图片处理配置失败: %v", err)
	}

	return &WindowsClipboardReader{
		logger:        logger,
		configManager: configManager,
	}, nil
}

// NewClipboardReaderWithNormalizer creates a new clipboard reader with image normalizer
func NewClipboardReaderWithNormalizer(normalizer *image.ImageNormalizer, logger *logrus.Logger) (Reader, error) {
	if logger == nil {
		logger = logrus.New()
	}
	configManager := image.NewConfigManager(".", logger)
	if err := configManager.LoadConfig(); err != nil {
		logger.Warnf("加载图片处理配置失败: %v", err)
	}

	return &WindowsClipboardReader{
		normalizer:    normalizer,
		logger:        logger,
		configManager: configManager,
	}, nil
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

	// Check various image formats in order of preference
	// Remote Desktop formats have higher priority
	imageFormats := []struct {
		format   uintptr
		name     string
		priority int
	}{
		{CF_DIBV5, "CF_DIBV5", 1},      // Remote Desktop format (highest priority)
		{CF_DIB, "CF_DIB", 2},           // Standard DIB format
		{CF_BITMAP, "CF_BITMAP", 3},     // Bitmap format
		{CF_ENHMETAFILE, "CF_ENHMETAFILE", 4}, // Enhanced metafile
	}

	// Try each format
	for _, fmt := range imageFormats {
		ret, _, _ := procIsClipboardFormatAvailable.Call(fmt.format)
		if ret != 0 {
			r.logger.Debugf("检测到图片格式: %s", fmt.name)
			return r.readImageByFormat(fmt.format, fmt.name)
		}
	}

	// If standard formats fail, try CF_HDROP format (file-based images)
	r.logger.Info("标准图片格式检测失败，尝试文件格式 (CF_HDROP)...")
	if imageData, err := r.tryHDropFormat(); err == nil {
		return imageData, nil
	}

	// If standard formats fail, try MSTSC-specific formats
	r.logger.Info("文件格式检测失败，尝试 MSTSC 特定格式...")
	if imageData, err := r.tryMSTSCFormats(); err == nil {
		return imageData, nil
	}

	// List all available formats for debugging
	r.enumClipboardFormats()

	return nil, fmt.Errorf("no supported image data in clipboard")
}

// readImageByFormat reads image data based on specific format
func (r *WindowsClipboardReader) readImageByFormat(format uintptr, formatName string) ([]byte, error) {
	// Get clipboard data handle
	handle, _, err := procGetClipboardData.Call(format)
	if handle == 0 {
		return nil, fmt.Errorf("failed to get clipboard data for %s: %v", formatName, err)
	}

	// Lock the global memory to get a pointer
	pointer, _, err := procGlobalLock.Call(handle)
	if pointer == 0 {
		return nil, fmt.Errorf("failed to lock global memory for %s: %v", formatName, err)
	}
	defer procGlobalUnlock.Call(handle)

	// Get the size of the data
	size, _, err := procGlobalSize.Call(handle)
	if size == 0 {
		return nil, fmt.Errorf("failed to get global memory size for %s: %v", formatName, err)
	}

	r.logger.Debugf("开始读取 %s 格式图片，数据大小: %d bytes", formatName, size)

	switch format {
	case CF_DIBV5:
		return r.processDIBV5Data(pointer, size)
	case CF_DIB:
		return r.processDIBData(pointer, size)
	case CF_BITMAP:
		return r.processBitmapData(handle, pointer, size)
	case CF_ENHMETAFILE:
		return r.processEnhMetafileData(handle, pointer, size)
	default:
		return nil, fmt.Errorf("unsupported format: %s", formatName)
	}
}

// processDIBData processes CF_DIB format data
func (r *WindowsClipboardReader) processDIBData(pointer, size uintptr) ([]byte, error) {
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
	var img stdimage.Image
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

	r.logger.Debugf("DIB图片处理完成 - 尺寸: %dx%d, 位深度: %d", width, height, header.BitCount)

	return r.encodeAndNormalizeImage(img)
}

// processDIBV5Data processes CF_DIBV5 format data (Remote Desktop)
func (r *WindowsClipboardReader) processDIBV5Data(pointer, size uintptr) ([]byte, error) {
	r.logger.Debug("开始处理 DIBV5 格式数据 (远程桌面)")

	// Read the DIBV5 data
	data := make([]byte, size)
	copy(data, (*[1 << 30]byte)(unsafe.Pointer(pointer))[:size:size])

	// Parse BITMAPV5HEADER
	if len(data) < int(unsafe.Sizeof(BITMAPV5HEADER{})) {
		return nil, fmt.Errorf("insufficient data for BITMAPV5HEADER")
	}

	header := (*BITMAPV5HEADER)(unsafe.Pointer(&data[0]))

	// Validate header - accept common BITMAPV5HEADER sizes
	// The actual size might be 124 bytes (BITMAPV5HEADER) or other variations
	expectedSize := unsafe.Sizeof(BITMAPV5HEADER{})
	if header.Size != uint32(expectedSize) && header.Size != 124 {
		return nil, fmt.Errorf("invalid BITMAPV5HEADER size: %d (expected %d or 124)", header.Size, expectedSize)
	}

	// Calculate image properties
	width := int(header.Width)
	height := int(header.Height)
	if height < 0 {
		height = -height // Top-down bitmap
	}

	r.logger.Debugf("DIBV5 图片信息 - 尺寸: %dx%d, 位深度: %d, 压缩: %d",
		width, height, header.BitCount, header.Compression)

	// Calculate stride (bytes per row)
	var stride int
	switch header.BitCount {
	case 32:
		stride = width * 4
	case 24:
		stride = ((width * 3 + 3) / 4) * 4 // Align to 4 bytes
	case 16:
		stride = ((width * 2 + 3) / 4) * 4 // Align to 4 bytes
	case 8:
		stride = ((width + 3) / 4) * 4 // Align to 4 bytes
	default:
		return nil, fmt.Errorf("unsupported bit count for DIBV5: %d", header.BitCount)
	}

	// Find the start of pixel data (after BITMAPV5HEADER and palette)
	offset := int(unsafe.Sizeof(BITMAPV5HEADER{}))
	if header.BitCount == 8 {
		offset += 256 * 4 // Palette for 8-bit images
	} else if header.BitCount == 16 {
		offset += 256 * 4 // Palette for 16-bit images
	}

	if offset >= len(data) {
		return nil, fmt.Errorf("invalid DIBV5 data structure")
	}

	// Extract pixel data
	pixelData := data[offset:]
	if len(pixelData) < stride*height {
		return nil, fmt.Errorf("insufficient pixel data for DIBV5")
	}

	// Convert to Go image format based on bit depth and masks
	var img stdimage.Image
	switch header.BitCount {
	case 32:
		if header.RedMask != 0 || header.GreenMask != 0 || header.BlueMask != 0 {
			// Use color masks if provided
			img = r.convertDIBV5Masked(pixelData, width, height, stride,
				header.RedMask, header.GreenMask, header.BlueMask, header.AlphaMask)
		} else {
			// Default BGRA format
			img = r.convertBGRAtoRGBA(pixelData, width, height, stride)
		}
	case 24:
		img = r.convertBGRtoRGB(pixelData, width, height, stride)
	case 16:
		if header.RedMask != 0 || header.GreenMask != 0 || header.BlueMask != 0 {
			img = r.convertDIBV5Masked(pixelData, width, height, stride,
				header.RedMask, header.GreenMask, header.BlueMask, header.AlphaMask)
		} else {
			img = r.convert16BitToRGBA(pixelData, width, height, stride)
		}
	case 8:
		paletteOffset := offset - 256*4
		if paletteOffset < 0 {
			return nil, fmt.Errorf("invalid palette offset for 8-bit DIBV5")
		}
		img = r.convert8BitToRGBA(pixelData, data[paletteOffset:offset], width, height, stride)
	default:
		return nil, fmt.Errorf("unsupported bit count for DIBV5: %d", header.BitCount)
	}

	r.logger.Debugf("DIBV5图片处理完成 - 尺寸: %dx%d, 位深度: %d", width, height, header.BitCount)

	return r.encodeAndNormalizeImage(img)
}

// tryMSTSCFormats 尝试 MSTSC 特定的剪贴板格式
func (r *WindowsClipboardReader) tryMSTSCFormats() ([]byte, error) {
	r.logger.Debug("尝试 MSTSC 特定格式...")

	// MSTSC 常用的格式名称
	rdpFormatNames := []string{
		"Remote Desktop Bitmap",
		"RDP Bitmap",
		"Terminal Services Bitmap",
		"MS Remote Desktop Bitmap",
		"RemoteDesktop_Protocol_Bitmap",
	}

	// 尝试每个注册的格式名称
	for _, formatName := range rdpFormatNames {
		if formatID, err := r.registerClipboardFormat(formatName); err == nil {
			ret, _, _ := procIsClipboardFormatAvailable.Call(uintptr(formatID))
			if ret != 0 {
				r.logger.Debugf("检测到 MSTSC 格式: %s (%d)", formatName, formatID)
				if imageData, err := r.readMSTSCFormatData(uintptr(formatID), formatName); err == nil {
					return imageData, nil
				}
			}
		}
	}

	// 尝试可能的 RDP 格式 ID 范围 (0xC00-0xCFF)
	rdpFormatIDs := []uint32{
		0xC01, // CF_RDP_BITMAP
		0xC02, // CF_RDP_DIB
		0xC03, // CF_RDP_DISPLAY
		0xC04, // CF_RDP_BITMAPSTREAM
		0xC05, // CF_RDP_PALETTE
	}

	for _, formatID := range rdpFormatIDs {
		ret, _, _ := procIsClipboardFormatAvailable.Call(uintptr(formatID))
		if ret != 0 {
			r.logger.Debugf("检测到可能的 RDP 格式: %d", formatID)
			if imageData, err := r.readMSTSCFormatData(uintptr(formatID), fmt.Sprintf("RDP_Format_%d", formatID)); err == nil {
				return imageData, nil
			}
		}
	}

	// 最后尝试分析所有可用格式
	return r.analyzeAllFormatsForImages()
}

// registerClipboardFormat 注册或获取剪贴板格式 ID
func (r *WindowsClipboardReader) registerClipboardFormat(formatName string) (uint32, error) {
	// 使用 RegisterClipboardFormatA
	shell32 := windows.NewLazySystemDLL("shell32.dll")
	procRegisterClipboardFormat := shell32.NewProc("RegisterClipboardFormatA")

	formatNamePtr, err := windows.BytePtrFromString(formatName)
	if err != nil {
		return 0, err
	}

	ret, _, _ := procRegisterClipboardFormat.Call(uintptr(unsafe.Pointer(formatNamePtr)))
	if ret == 0 {
		return 0, fmt.Errorf("failed to register format: %s", formatName)
	}

	return uint32(ret), nil
}

// readMSTSCFormatData 读取 MSTSC 格式数据
func (r *WindowsClipboardReader) readMSTSCFormatData(format uintptr, formatName string) ([]byte, error) {
	// 获取剪贴板数据句柄
	handle, _, err := procGetClipboardData.Call(format)
	if handle == 0 {
		return nil, fmt.Errorf("failed to get clipboard data for %s: %v", formatName, err)
	}

	// 锁定全局内存
	pointer, _, err := procGlobalLock.Call(handle)
	if pointer == 0 {
		return nil, fmt.Errorf("failed to lock global memory for %s: %v", formatName, err)
	}
	defer procGlobalUnlock.Call(handle)

	// 获取数据大小
	size, _, err := procGlobalSize.Call(handle)
	if size == 0 {
		return nil, fmt.Errorf("failed to get global memory size for %s: %v", formatName, err)
	}

	r.logger.Debugf("开始读取 MSTSC 格式 %s，数据大小: %d bytes", formatName, size)

	// 读取原始数据
	data := make([]byte, size)
	copy(data, (*[1 << 30]byte)(unsafe.Pointer(pointer))[:size:size])

	// 尝试作为 DIB 或 DIBV5 处理
	return r.tryProcessAsDIB(data)
}

// tryProcessAsDIB 尝试将数据作为 DIB 或 DIBV5 处理
func (r *WindowsClipboardReader) tryProcessAsDIB(data []byte) ([]byte, error) {
	if len(data) < 40 {
		return nil, fmt.Errorf("数据不足以解析为 DIB")
	}

	// 检查是否是 DIBV5 (124 bytes) 或 DIB (40 bytes)
	if len(data) >= 124 && data[0] == 124 {
		r.logger.Debug("MSTSC 数据识别为 DIBV5 格式")
		pointer := uintptr(unsafe.Pointer(&data[0]))
		size := uintptr(len(data))
		return r.processDIBV5Data(pointer, size)
	} else if data[0] == 40 {
		r.logger.Debug("MSTSC 数据识别为 DIB 格式")
		pointer := uintptr(unsafe.Pointer(&data[0]))
		size := uintptr(len(data))
		return r.processDIBData(pointer, size)
	}

	// 如果头部不明显，尝试搜索可能的 DIB 头部
	for offset := 0; offset < min(100, len(data)-40); offset++ {
		if data[offset] == 40 || (len(data)-offset >= 124 && data[offset] == 124) {
			r.logger.Debugf("在偏移 %d 发现可能的 DIB 头部", offset)
			trimmedData := data[offset:]
			pointer := uintptr(unsafe.Pointer(&trimmedData[0]))
			size := uintptr(len(trimmedData))

			if data[offset] == 40 {
				return r.processDIBData(pointer, size)
			} else {
				return r.processDIBV5Data(pointer, size)
			}
		}
	}

	return nil, fmt.Errorf("无法在 MSTSC 数据中找到有效的图片格式")
}

// analyzeAllFormatsForImages 分析所有可用格式寻找图片数据
func (r *WindowsClipboardReader) analyzeAllFormatsForImages() ([]byte, error) {
	r.logger.Debug("分析所有剪贴板格式以寻找图片数据...")

	format := uintptr(0)
	count := 0

	for {
		nextFormat, _, _ := procEnumClipboardFormats.Call(format)
		if nextFormat == 0 {
			break
		}
		count++

		// 跳过已知的非图片格式
		if r.isLikelyImageFormat(nextFormat) {
			r.logger.Debugf("检查可能的图片格式 %d: 0x%X", count, nextFormat)

			// 尝试读取此格式的数据
			if imageData, err := r.tryReadFormatAsImage(nextFormat); err == nil {
				r.logger.Infof("成功从格式 0x%X 读取到图片数据", nextFormat)
				return imageData, nil
			}
		}

		format = nextFormat

		// 避免检查过多格式
		if count >= 30 {
			r.logger.Debug("已检查足够多格式，停止继续检查")
			break
		}
	}

	return nil, fmt.Errorf("未在任何格式中找到有效的图片数据")
}

// isLikelyImageFormat 判断是否可能是图片格式
func (r *WindowsClipboardReader) isLikelyImageFormat(format uintptr) bool {
	// 已知的图片格式
	imageFormats := []uintptr{
		CF_DIB, CF_DIBV5, CF_BITMAP, CF_ENHMETAFILE,
		CF_TIFF, CF_RIFF, // TIFF 和其他图片容器
	}

	for _, imgFormat := range imageFormats {
		if format == imgFormat {
			return true
		}
	}

	// 检查是否在 RDP 格式范围内
	if format >= 0xC00 && format <= 0xCFF {
		return true
	}

	// 检查是否是注册格式（通常 >= 0xC000）
	if format >= 0xC000 {
		return true
	}

	return false
}

// tryReadFormatAsImage 尝试将某个格式作为图片读取
func (r *WindowsClipboardReader) tryReadFormatAsImage(format uintptr) ([]byte, error) {
	handle, _, _ := procGetClipboardData.Call(format)
	if handle == 0 {
		return nil, fmt.Errorf("无法获取格式数据")
	}

	pointer, _, _ := procGlobalLock.Call(handle)
	if pointer == 0 {
		return nil, fmt.Errorf("无法锁定内存")
	}
	defer procGlobalUnlock.Call(handle)

	size, _, _ := procGlobalSize.Call(handle)
	if size == 0 {
		return nil, fmt.Errorf("数据大小为 0")
	}

	// 只尝试读取可能的大数据量格式（至少 1KB）
	if size < 1024 {
		return nil, fmt.Errorf("数据量太小，不太可能是图片")
	}

	data := make([]byte, size)
	copy(data, (*[1 << 30]byte)(unsafe.Pointer(pointer))[:size:size])

	// 尝试作为 DIB 处理
	return r.tryProcessAsDIB(data)
}

// convertDIBV5Masked converts pixel data using color masks (for 16/32-bit DIBV5)
func (r *WindowsClipboardReader) convertDIBV5Masked(pixelData []byte, width, height, stride int,
	redMask, greenMask, blueMask, alphaMask uint32) *stdimage.RGBA {

	img := stdimage.NewRGBA(stdimage.Rect(0, 0, width, height))

	// Default masks if not provided
	if redMask == 0 && greenMask == 0 && blueMask == 0 {
		redMask = 0xFF0000
		greenMask = 0x00FF00
		blueMask = 0x0000FF
	}

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Windows DIB is stored bottom-to-top, so we need to flip the Y coordinate
			srcY := height - 1 - y

			var pixel uint32
			if redMask != 0 || blueMask > 0xFF { // 32-bit
				srcOffset := srcY*stride + x*4
				if srcOffset+3 >= len(pixelData) {
					continue
				}
				pixel = uint32(pixelData[srcOffset]) |
					uint32(pixelData[srcOffset+1])<<8 |
					uint32(pixelData[srcOffset+2])<<16 |
					uint32(pixelData[srcOffset+3])<<24
			} else { // 16-bit
				srcOffset := srcY*stride + x*2
				if srcOffset+1 >= len(pixelData) {
					continue
				}
				pixel = uint32(pixelData[srcOffset]) |
					uint32(pixelData[srcOffset+1])<<8
			}

			// Extract color components using masks
			red := uint8((pixel & redMask) >> r.countTrailingZeros(redMask))
			green := uint8((pixel & greenMask) >> r.countTrailingZeros(greenMask))
			blue := uint8((pixel & blueMask) >> r.countTrailingZeros(blueMask))
			alpha := uint8(255)

			if alphaMask != 0 {
				alpha = uint8((pixel & alphaMask) >> r.countTrailingZeros(alphaMask))
			}

			dstOffset := y*img.Stride + x*4
			if dstOffset+3 < len(img.Pix) {
				img.Pix[dstOffset+0] = red
				img.Pix[dstOffset+1] = green
				img.Pix[dstOffset+2] = blue
				img.Pix[dstOffset+3] = alpha
			}
		}
	}

	return img
}

// countTrailingZeros counts trailing zero bits in a uint32
func (r *WindowsClipboardReader) countTrailingZeros(value uint32) uint32 {
	if value == 0 {
		return 0
	}
	count := uint32(0)
	for (value & 1) == 0 {
		count++
		value >>= 1
	}
	return count
}

// convert16BitToRGBA converts 16-bit pixel data to RGBA image
func (r *WindowsClipboardReader) convert16BitToRGBA(pixelData []byte, width, height, stride int) *stdimage.RGBA {
	img := stdimage.NewRGBA(stdimage.Rect(0, 0, width, height))

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Windows DIB is stored bottom-to-top, so we need to flip the Y coordinate
			srcY := height - 1 - y
			srcOffset := srcY*stride + x*2
			dstOffset := y*img.Stride + x*4

			if srcOffset+1 < len(pixelData) && dstOffset+3 < len(img.Pix) {
				pixel := uint16(pixelData[srcOffset]) | uint16(pixelData[srcOffset+1])<<8

				// Extract RGB components (5-6-5 format)
				red := uint8((pixel & 0xF800) >> 11) << 3
				green := uint8((pixel & 0x07E0) >> 5) << 2
				blue := uint8(pixel & 0x001F) << 3

				img.Pix[dstOffset+0] = red
				img.Pix[dstOffset+1] = green
				img.Pix[dstOffset+2] = blue
				img.Pix[dstOffset+3] = 0xFF // Alpha
			}
		}
	}

	return img
}

// processBitmapData processes CF_BITMAP format data
func (r *WindowsClipboardReader) processBitmapData(handle, pointer, size uintptr) ([]byte, error) {
	// For CF_BITMAP, we need to convert it to DIB first
	// This is a simplified implementation - in production, you'd use
	// additional Windows API calls to get bitmap information
	r.logger.Debug("处理 CF_BITMAP 格式数据")

	// For now, we'll return an error since CF_BITMAP to DIB conversion
	// requires more complex Windows API interactions
	return nil, fmt.Errorf("CF_BITMAP format requires additional Windows API implementation")
}

// processEnhMetafileData processes CF_ENHMETAFILE format data
func (r *WindowsClipboardReader) processEnhMetafileData(handle, pointer, size uintptr) ([]byte, error) {
	r.logger.Debug("处理 CF_ENHMETAFILE 格式数据")
	// Enhanced metafile processing is complex and requires GDI+ or Windows API calls
	return nil, fmt.Errorf("CF_ENHMETAFILE format not yet implemented")
}


// encodeAndNormalizeImage encodes image to PNG and applies normalization
func (r *WindowsClipboardReader) encodeAndNormalizeImage(img stdimage.Image) ([]byte, error) {
	// Apply image normalization if available
	if r.normalizer != nil {
		r.logger.Debug("应用图片标准化处理")
		normalizedImg, err := r.normalizer.NormalizeImage(img)
		if err != nil {
			r.logger.Warnf("图片标准化失败，使用原始图片: %v", err)
		} else {
			img = normalizedImg
			bounds := img.Bounds()
			r.logger.Debugf("标准化后图片尺寸: %dx%d", bounds.Dx(), bounds.Dy())
		}
	}

	// Encode as PNG
	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	if err != nil {
		return nil, fmt.Errorf("failed to encode image as PNG: %w", err)
	}

	r.logger.Debugf("PNG编码完成，最终大小: %d bytes", buf.Len())

	// Cache original image if enabled
	finalImageData := buf.Bytes()
	if r.configManager != nil && r.configManager.IsCacheEnabled() {
		timestamp := time.Now().Format("20060102_150405_000000")
		originalFilename := fmt.Sprintf("clipboard_original_%s.png", timestamp)

		// Cache the normalized image
		cachePath, err := r.configManager.SaveCacheImage(finalImageData, originalFilename)
		if err != nil {
			r.logger.Warnf("缓存图片失败: %v", err)
		} else {
			r.logger.Infof("图片已缓存: %s", cachePath)
		}
	}

	return finalImageData, nil
}

// enumClipboardFormats lists all available clipboard formats for debugging
func (r *WindowsClipboardReader) enumClipboardFormats() {
	r.logger.Debug("枚举剪贴板格式...")
	format := uintptr(0)
	count := 0

	for {
		nextFormat, _, _ := procEnumClipboardFormats.Call(format)
		if nextFormat == 0 {
			break
		}
		count++
		r.logger.Debugf("可用格式 %d: %d", count, nextFormat)
		format = nextFormat

		// Limit to first 10 formats to avoid spam
		if count >= 10 {
			r.logger.Debug("...")
			break
		}
	}

	if count == 0 {
		r.logger.Debug("剪贴板中没有可用格式")
	}
}

// HasContent checks if clipboard has any readable content
func (r *WindowsClipboardReader) HasContent() (bool, error) {
	// First check for image content using Windows API
	// This is more reliable than checking text first, as text checking may miss images
	ret, _, err := procOpenClipboard.Call(0)
	if ret != 0 {
		defer procCloseClipboard.Call()

		// Check all supported image formats
		imageFormats := []uintptr{CF_DIBV5, CF_DIB, CF_BITMAP, CF_ENHMETAFILE, CF_HDROP}
		for _, format := range imageFormats {
			ret, _, _ := procIsClipboardFormatAvailable.Call(format)
			if ret != 0 {
				r.logger.Debugf("检测到图片格式可用: %d", format)
				return true, nil
			}
		}
	}

	// Then try to read text
	text, err := clipboard.ReadAll()
	if err == nil && text != "" {
		return true, nil
	}

	return false, nil
}

// GetContentType determines the type of content in clipboard
func (r *WindowsClipboardReader) GetContentType() (models.ContentType, error) {
	// First check for image content using Windows API
	ret, _, err := procOpenClipboard.Call(0)
	if ret != 0 {
		defer procCloseClipboard.Call()

		// Check all supported image formats
		imageFormats := []uintptr{CF_DIBV5, CF_DIB, CF_BITMAP, CF_ENHMETAFILE, CF_HDROP}
		for _, format := range imageFormats {
			ret, _, _ := procIsClipboardFormatAvailable.Call(format)
			if ret != 0 {
				r.logger.Debugf("检测到图片格式: %d", format)
				return models.ContentTypeImage, nil
			}
		}
	}

	// Then try to read text
	text, err := clipboard.ReadAll()
	if err == nil && text != "" {
		return models.ContentTypeText, nil
	}

	// If no content found, return empty
	return models.ContentTypeEmpty, nil
}

// Read reads any available content from clipboard
func (r *WindowsClipboardReader) Read() (*models.ClipboardContent, error) {
	content := &models.ClipboardContent{}

	// First try to read image (consistent with HasContent and GetContentType)
	imageData, err := r.ReadImage()
	if err == nil && imageData != nil {
		content.Type = models.ContentTypeImage
		content.Image = imageData
		content.FileName = fmt.Sprintf("clipboard_%s.png", time.Now().Format("20060102_150405"))
		return content, nil
	}

	// Then try to read text
	text, err := r.ReadText()
	if err == nil && text != "" {
		content.Type = models.ContentTypeText
		content.Text = text
		return content, nil
	}

	return nil, fmt.Errorf("no readable content found in clipboard")
}

// convertBGRAtoRGBA converts 32-bit BGRA pixel data to RGBA image
func (r *WindowsClipboardReader) convertBGRAtoRGBA(pixelData []byte, width, height, stride int) *stdimage.RGBA {
	img := stdimage.NewRGBA(stdimage.Rect(0, 0, width, height))

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Windows DIB is stored bottom-to-top, so we need to flip the Y coordinate
			srcY := height - 1 - y
			srcOffset := srcY*stride + x*4
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

	r.logger.Debugf("BGRA图片转换完成，已应用Y轴翻转，尺寸: %dx%d", width, height)
	return img
}

// convertBGRtoRGB converts 24-bit BGR pixel data to RGB image
func (r *WindowsClipboardReader) convertBGRtoRGB(pixelData []byte, width, height, stride int) *stdimage.RGBA {
	img := stdimage.NewRGBA(stdimage.Rect(0, 0, width, height))

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Windows DIB is stored bottom-to-top, so we need to flip the Y coordinate
			srcY := height - 1 - y
			srcOffset := srcY*stride + x*3
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

	r.logger.Debugf("BGR图片转换完成，已应用Y轴翻转，尺寸: %dx%d", width, height)
	return img
}

// convert8BitToRGBA converts 8-bit palette pixel data to RGBA image
func (r *WindowsClipboardReader) convert8BitToRGBA(pixelData, palette []byte, width, height, stride int) *stdimage.RGBA {
	img := stdimage.NewRGBA(stdimage.Rect(0, 0, width, height))

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Windows DIB is stored bottom-to-top, so we need to flip the Y coordinate
			srcY := height - 1 - y
			srcOffset := srcY*stride + x
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

	r.logger.Debugf("8位调色板图片转换完成，已应用Y轴翻转，尺寸: %dx%d", width, height)
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
	img, _, err := stdimage.Decode(bytes.NewBuffer(imageData))
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

// tryHDropFormat 尝试从 CF_HDROP 格式读取图片文件
func (r *WindowsClipboardReader) tryHDropFormat() ([]byte, error) {
	r.logger.Debug("尝试 CF_HDROP (文件) 格式...")

	// 检查 CF_HDROP 格式是否可用
	ret, _, _ := procIsClipboardFormatAvailable.Call(CF_HDROP)
	if ret == 0 {
		return nil, fmt.Errorf("CF_HDROP format not available")
	}

	// 获取剪贴板数据句柄
	handle, _, err := procGetClipboardData.Call(CF_HDROP)
	if handle == 0 {
		return nil, fmt.Errorf("failed to get clipboard data for CF_HDROP: %v", err)
	}

	// 锁定全局内存
	pointer, _, err := procGlobalLock.Call(handle)
	if pointer == 0 {
		return nil, fmt.Errorf("failed to lock global memory for CF_HDROP: %v", err)
	}
	defer procGlobalUnlock.Call(handle)

	// 获取数据大小
	size, _, err := procGlobalSize.Call(handle)
	if size == 0 {
		return nil, fmt.Errorf("failed to get global memory size for CF_HDROP: %v", err)
	}

	r.logger.Debugf("开始读取 CF_HDROP 数据，大小: %d bytes", size)

	// 读取原始数据
	data := make([]byte, size)
	copy(data, (*[1 << 30]byte)(unsafe.Pointer(pointer))[:size:size])

	// 解析 DROPFILES 结构
	if len(data) < 20 {
		return nil, fmt.Errorf("data insufficient for DROPFILES structure")
	}

	// 解析文件列表
	dropFiles := (*struct {
		pFiles uint32
		pt     struct{ X, Y int32 }
		fNC    uint32
		fWide  uint32
	})(unsafe.Pointer(&data[0]))

	filesOffset := dropFiles.pFiles
	if filesOffset == 0 || filesOffset >= uint32(len(data)) {
		return nil, fmt.Errorf("invalid file list offset in DROPFILES")
	}

	// 解析文件路径
	filesData := data[filesOffset:]
	fileList := r.parseFileList(filesData, dropFiles.fWide != 0)

	r.logger.Debugf("CF_HDROP 解析到 %d 个文件", len(fileList))

	// 查找第一个图片文件
	for _, filePath := range fileList {
		if r.isImageFile(filePath) {
			r.logger.Debugf("找到图片文件: %s", filePath)

			// 检查文件是否存在
			if _, err := os.Stat(filePath); err != nil {
				r.logger.Warnf("图片文件不存在: %s, 错误: %v", filePath, err)
				continue
			}

			// 读取图片文件
			imageData, err := os.ReadFile(filePath)
			if err != nil {
				r.logger.Warnf("读取图片文件失败: %s, 错误: %v", filePath, err)
				continue
			}

			r.logger.Infof("成功从文件读取图片: %s, 大小: %d bytes", filePath, len(imageData))

			// 应用图片标准化
			if r.normalizer != nil {
				r.logger.Debug("应用图片标准化处理到文件图片")

				// 解码图片
				img, _, err := stdimage.Decode(bytes.NewBuffer(imageData))
				if err != nil {
					r.logger.Warnf("解码文件图片失败，使用原始数据: %v", err)
				} else {
					normalizedImg, err := r.normalizer.NormalizeImage(img)
					if err != nil {
						r.logger.Warnf("文件图片标准化失败，使用原始图片: %v", err)
					} else {
						bounds := normalizedImg.Bounds()
						r.logger.Debugf("文件图片标准化后尺寸: %dx%d", bounds.Dx(), bounds.Dy())

						// 重新编码为 PNG
						var buf bytes.Buffer
						err = png.Encode(&buf, normalizedImg)
						if err != nil {
							r.logger.Warnf("编码标准化图片失败，使用原始数据: %v", err)
						} else {
							imageData = buf.Bytes()
							r.logger.Debugf("文件图片标准化完成，最终大小: %d bytes", len(imageData))
						}
					}
				}
			}

			// 缓存原始图片（在标准化之前）
			if r.configManager != nil && r.configManager.IsCacheEnabled() {
				timestamp := time.Now().Format("20060102_150405_000000")
				originalFilename := fmt.Sprintf("hdrop_file_%s.png", timestamp)
				cachePath, err := r.configManager.SaveCacheImage(imageData, originalFilename)
				if err != nil {
					r.logger.Warnf("缓存HDROP图片失败: %v", err)
				} else {
					r.logger.Infof("HDROP图片已缓存: %s", cachePath)
				}
			}

			return imageData, nil
		}
	}

	return nil, fmt.Errorf("no image files found in CF_HDROP data")
}

// parseFileList 解析 CF_HDROP 格式中的文件列表
func (r *WindowsClipboardReader) parseFileList(data []byte, isWide bool) []string {
	var files []string

	if isWide {
		// 宽字符 (UTF-16) 解析
		utf16Data := data
		var currentFile strings.Builder

		for i := 0; i < len(utf16Data); i += 2 {
			if i+1 >= len(utf16Data) {
				break
			}

			char := uint16(utf16Data[i]) | uint16(utf16Data[i+1])<<8
			if char == 0 {
				// 字符串结束
				if currentFile.Len() > 0 {
					files = append(files, currentFile.String())
					currentFile.Reset()
				}
				// 检查是否是双空格结束（文件列表结束）
				if i+2 < len(utf16Data) &&
				   uint16(utf16Data[i+2])|uint16(utf16Data[i+3])<<8 == 0 {
					break
				}
			} else {
				currentFile.WriteRune(rune(char))
			}
		}
	} else {
		// ANSI 解析
		var currentFile strings.Builder
		for _, b := range data {
			if b == 0 {
				// 字符串结束
				if currentFile.Len() > 0 {
					files = append(files, currentFile.String())
					currentFile.Reset()
				}
				// 检查是否是双空格结束（文件列表结束）
				if len(files) > 0 {
					break
				}
			} else {
				currentFile.WriteByte(b)
			}
		}
	}

	return files
}

// isImageFile 检查文件是否为图片文件
func (r *WindowsClipboardReader) isImageFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	imageExts := []string{".jpg", ".jpeg", ".png", ".bmp", ".gif", ".tiff", ".tif", ".webp", ".ico"}

	for _, imgExt := range imageExts {
		if ext == imgExt {
			return true
		}
	}
	return false
}