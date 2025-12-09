package clipboard

import (
	"fmt"
	"unsafe"

	"github.com/WQGroup/logger"
	"golang.org/x/sys/windows"
)

// MSTSC 特定的剪贴板格式
const (
	// 远程桌面协议可能使用的格式
	CF_RDP_BITMAP        = 0x0C01 // 远程桌面位图
	CF_RDP_DIB           = 0x0C02 // 远程桌面 DIB
	CF_RDP_DISPLAY       = 0x0C03 // 远程桌面显示
	CF_RDP_BITMAPSTREAM  = 0x0C04 // 远程桌面位图流
	CF_RDP_PALETTE       = 0x0C05 // 远程桌面调色板

	// 可能的注册格式名称
	RDP_FORMAT_NAME_1    = "Remote Desktop Bitmap"
	RDP_FORMAT_NAME_2    = "RDP Bitmap"
	RDP_FORMAT_NAME_3    = "Terminal Services Bitmap"
	RDP_FORMAT_NAME_4    = "MS Remote Desktop Bitmap"
)

// MSTSCEnhancedReader 增强的剪贴板读取器，专门处理 MSTSC
type MSTSCEnhancedReader struct {
	*WindowsClipboardReader
}

// NewMSTSCEnhancedReader 创建 MSTSC 增强读取器
func NewMSTSCEnhancedReader(baseReader *WindowsClipboardReader) *MSTSCEnhancedReader {
	return &MSTSCEnhancedReader{
		WindowsClipboardReader: baseReader,
	}
}

// ReadImageWithMSTSCSupport 支持 MSTSC 的图片读取
func (r *MSTSCEnhancedReader) ReadImageWithMSTSCSupport() ([]byte, error) {
	// 打开剪贴板
	ret, _, err := procOpenClipboard.Call(0)
	if ret == 0 {
		return nil, fmt.Errorf("failed to open clipboard: %v", err)
	}
	defer procCloseClipboard.Call()

	// 首先尝试标准的 DIBV5 和 DIB 格式
	standardFormats := []struct {
		format   uintptr
		name     string
		priority int
	}{
		{CF_DIBV5, "CF_DIBV5", 1},
		{CF_DIB, "CF_DIB", 2},
		{CF_BITMAP, "CF_BITMAP", 3},
	}

	for _, fmt := range standardFormats {
		ret, _, _ := procIsClipboardFormatAvailable.Call(fmt.format)
		if ret != 0 {
			logger.Debugf("检测到标准格式: %s", fmt.name)
			return r.readImageByFormat(fmt.format, fmt.name)
		}
	}

	// 如果标准格式失败，尝试 MSTSC 特定格式
	logger.Info("标准格式检测失败，尝试 MSTSC 特定格式...")
	return r.tryMSTSCFormats()
}

// tryMSTSCFormats 尝试 MSTSC 特定的剪贴板格式
func (r *MSTSCEnhancedReader) tryMSTSCFormats() ([]byte, error) {
	// 1. 尝试已知的 RDP 格式 ID
	rdpFormats := []struct {
		format   uintptr
		name     string
	}{
		{CF_RDP_BITMAP, "CF_RDP_BITMAP"},
		{CF_RDP_DIB, "CF_RDP_DIB"},
		{CF_RDP_DISPLAY, "CF_RDP_DISPLAY"},
		{CF_RDP_BITMAPSTREAM, "CF_RDP_BITMAPSTREAM"},
		{CF_RDP_PALETTE, "CF_RDP_PALETTE"},
	}

	for _, fmt := range rdpFormats {
		ret, _, _ := procIsClipboardFormatAvailable.Call(fmt.format)
		if ret != 0 {
			logger.Debugf("检测到 RDP 格式: %s", fmt.name)
			if data, err := r.readMSTSCFormat(fmt.format, fmt.name); err == nil {
				return data, nil
			}
		}
	}

	// 2. 尝试查找注册的 RDP 格式名称
	rdpNames := []string{
		RDP_FORMAT_NAME_1,
		RDP_FORMAT_NAME_2,
		RDP_FORMAT_NAME_3,
		RDP_FORMAT_NAME_4,
	}

	for _, name := range rdpNames {
		if formatID, err := r.getRegisteredFormatID(name); err == nil {
			ret, _, _ := procIsClipboardFormatAvailable.Call(uintptr(formatID))
			if ret != 0 {
				logger.Debugf("检测到注册的 RDP 格式: %s (%d)", name, formatID)
				if data, err := r.readMSTSCFormat(uintptr(formatID), name); err == nil {
					return data, nil
				}
			}
		}
	}

	// 3. 尝试分析所有可用格式，寻找可能的图片数据
	logger.Info("尝试分析所有可用格式...")
	return r.analyzeAllFormatsForImages()
}

// getRegisteredFormatID 获取注册格式的 ID
func (r *MSTSCEnhancedReader) getRegisteredFormatID(formatName string) (uint32, error) {
	// 使用 RegisterClipboardFormatA 检查格式是否已注册 (在 user32.dll 中)
	user32 := windows.NewLazySystemDLL("user32.dll")
	procRegisterClipboardFormat := user32.NewProc("RegisterClipboardFormatA")

	formatNamePtr, err := windows.BytePtrFromString(formatName)
	if err != nil {
		return 0, err
	}

	ret, _, _ := procRegisterClipboardFormat.Call(uintptr(unsafe.Pointer(formatNamePtr)))
	if ret == 0 {
		return 0, fmt.Errorf("format not registered: %s", formatName)
	}

	return uint32(ret), nil
}

// readMSTSCFormat 读取 MSTSC 特定格式
func (r *MSTSCEnhancedReader) readMSTSCFormat(format uintptr, formatName string) ([]byte, error) {
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

	logger.Debugf("开始读取 MSTSC 格式 %s，数据大小: %d bytes", formatName, size)

	// 读取原始数据
	data := make([]byte, size)
	copy(data, (*[1 << 30]byte)(unsafe.Pointer(pointer))[:size:size])

	// 根据格式类型进行处理
	switch formatName {
	case "CF_RDP_BITMAP", "CF_RDP_BITMAPSTREAM":
		return r.processRDPBitmap(data)
	case "CF_RDP_DIB":
		return r.processRDPDIB(data)
	case "CF_RDP_DISPLAY":
		return r.processRDPDisplay(data)
	default:
		// 尝试将其作为 DIB 或 DIBV5 处理
		return r.tryProcessAsDIB(data)
	}
}

// processRDPBitmap 处理 RDP 位图格式
func (r *MSTSCEnhancedReader) processRDPBitmap(data []byte) ([]byte, error) {
	logger.Debug("尝试处理 RDP 位图格式")

	// RDP 位图可能有特殊的头部，尝试跳过头部找到 DIB 数据
	for offset := 0; offset < len(data)-4; offset++ {
		// 寻找可能的 BITMAPINFOHEADER 标识 (通常是 40)
		if data[offset] == 40 && data[offset+1] == 0 && data[offset+2] == 0 && data[offset+3] == 0 {
			logger.Debugf("在偏移 %d 发现可能的 DIB 头部", offset)
			return r.processRDPDIB(data[offset:])
		}
	}

	// 如果没有找到 DIB 头部，尝试直接作为 DIB 处理
	return r.tryProcessAsDIB(data)
}

// processRDPDIB 处理 RDP DIB 格式
func (r *MSTSCEnhancedReader) processRDPDIB(data []byte) ([]byte, error) {
	logger.Debug("处理 RDP DIB 格式")

	// RDP DIB 可能有特殊的前缀，检查是否需要调整
	if len(data) < 40 {
		return nil, fmt.Errorf("RDP DIB 数据不足")
	}

	// 检查前几个字节是否看起来像 DIB 头部
	if data[0] == 40 || (data[0] == 124 && len(data) >= 124) {
		// 直接处理为标准 DIB 或 DIBV5
		return r.tryProcessAsDIB(data)
	}

	// 尝试寻找 DIB 头部
	for offset := 0; offset < min(100, len(data)-40); offset++ {
		if data[offset] == 40 || data[offset] == 124 {
			logger.Debugf("RDP DIB: 在偏移 %d 发现可能的头部", offset)
			return r.tryProcessAsDIB(data[offset:])
		}
	}

	return nil, fmt.Errorf("RDP DIB: 无法找到有效的图片头部")
}

// processRDPDisplay 处理 RDP 显示格式
func (r *MSTSCEnhancedReader) processRDPDisplay(data []byte) ([]byte, error) {
	logger.Debug("处理 RDP 显示格式")

	// RDP 显示格式可能包含多种数据，尝试解析
	return r.tryProcessAsDIB(data)
}

// tryProcessAsDIB 尝试将数据作为 DIB 或 DIBV5 处理
func (r *MSTSCEnhancedReader) tryProcessAsDIB(data []byte) ([]byte, error) {
	if len(data) < 40 {
		return nil, fmt.Errorf("数据不足以解析为 DIB")
	}

	// 检查是否是 DIBV5 (124 bytes) 或 DIB (40 bytes)
	if len(data) >= 124 && data[0] == 124 {
		logger.Debug("尝试作为 DIBV5 处理")
		return r.processDIBV5DataDirect(data)
	} else if data[0] == 40 {
		logger.Debug("尝试作为 DIB 处理")
		return r.processDIBDataDirect(data)
	}

	return nil, fmt.Errorf("无法识别 DIB 格式")
}

// processDIBV5DataDirect 直接处理 DIBV5 数据
func (r *MSTSCEnhancedReader) processDIBV5DataDirect(data []byte) ([]byte, error) {
	// 模拟指针和大小
	pointer := uintptr(unsafe.Pointer(&data[0]))
	size := uintptr(len(data))

	// 调用现有的 DIBV5 处理函数
	return r.processDIBV5Data(pointer, size)
}

// processDIBDataDirect 直接处理 DIB 数据
func (r *MSTSCEnhancedReader) processDIBDataDirect(data []byte) ([]byte, error) {
	// 模拟指针和大小
	pointer := uintptr(unsafe.Pointer(&data[0]))
	size := uintptr(len(data))

	// 调用现有的 DIB 处理函数
	return r.processDIBData(pointer, size)
}

// analyzeAllFormatsForImages 分析所有可用格式寻找图片数据
func (r *MSTSCEnhancedReader) analyzeAllFormatsForImages() ([]byte, error) {
	logger.Debug("枚举所有剪贴板格式以寻找图片数据...")

	format := uintptr(0)
	count := 0

	for {
		nextFormat, _, _ := procEnumClipboardFormats.Call(format)
		if nextFormat == 0 {
			break
		}
		count++
		logger.Debugf("检查格式 %d: 0x%X", count, nextFormat)

		// 尝试读取此格式的数据
		if data, err := r.tryReadFormatAsImage(nextFormat); err == nil {
			logger.Infof("成功从格式 0x%X 读取到图片数据", nextFormat)
			return data, nil
		}

		format = nextFormat

		// 避免检查过多格式
		if count >= 50 {
			logger.Warn("已检查 50 个格式，停止继续检查")
			break
		}
	}

	return nil, fmt.Errorf("未在任何格式中找到有效的图片数据")
}

// tryReadFormatAsImage 尝试将某个格式作为图片读取
func (r *MSTSCEnhancedReader) tryReadFormatAsImage(format uintptr) ([]byte, error) {
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

	// 只尝试读取可能的大数据量格式
	if size < 1000 {
		return nil, fmt.Errorf("数据量太小，不太可能是图片")
	}

	data := make([]byte, size)
	copy(data, (*[1 << 30]byte)(unsafe.Pointer(pointer))[:size:size])

	// 尝试作为 DIB 处理
	return r.tryProcessAsDIB(data)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}