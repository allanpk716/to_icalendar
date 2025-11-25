package tray

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewIconLoader(t *testing.T) {
	loader := NewIconLoader()

	require.NotNil(t, loader)
	assert.Equal(t, 0, loader.GetCacheSize())
}

func TestIconLoader_LoadIcon(t *testing.T) {
	loader := NewIconLoader()

	// 创建临时测试文件
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test-icon-32.png")

	// 创建一个简单的PNG文件（实际上是空文件，仅用于测试）
	err := os.WriteFile(testFile, []byte("fake png content"), 0644)
	require.NoError(t, err)

	// 测试加载图标
	icon, err := loader.LoadIcon(testFile, 32)
	require.NoError(t, err)
	require.NotNil(t, icon)

	assert.Equal(t, testFile, icon.FilePath)
	assert.Equal(t, 32, icon.Size)
	assert.True(t, icon.IsActive)

	// 验证图标被缓存
	cachedIcon, exists := loader.GetIcon(testFile)
	assert.True(t, exists)
	assert.Equal(t, icon.ID, cachedIcon.ID)
}

func TestIconLoader_LoadIcon_FileNotExists(t *testing.T) {
	loader := NewIconLoader()

	// 尝试加载不存在的文件
	icon, err := loader.LoadIcon("non-existent-file.png", 32)

	require.Error(t, err)
	require.Nil(t, icon)

	// 验证错误类型
	trayErr, isTrayErr := IsTrayError(err)
	require.True(t, isTrayErr)
	assert.Equal(t, ErrCodeIconLoad, trayErr.Code)
}

func TestIconLoader_LoadIcon_UnsupportedFormat(t *testing.T) {
	loader := NewIconLoader()

	// 创建临时测试文件（不支持的格式）
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test-icon.txt")

	err := os.WriteFile(testFile, []byte("not an image"), 0644)
	require.NoError(t, err)

	// 尝试加载不支持的格式
	icon, err := loader.LoadIcon(testFile, 32)

	require.Error(t, err)
	require.Nil(t, icon)

	// 验证错误类型
	trayErr, isTrayErr := IsTrayError(err)
	require.True(t, isTrayErr)
	assert.Equal(t, ErrCodeIconLoad, trayErr.Code)
}

func TestIconLoader_LoadIconSet(t *testing.T) {
	loader := NewIconLoader()

	// 创建临时测试文件
	tempDir := t.TempDir()
	basePath := filepath.Join(tempDir, "test-icon")

	// 创建不同尺寸的图标文件
	sizes := []int{16, 32, 48}
	for _, size := range sizes {
		testFile := fmt.Sprintf("%s-%d.png", basePath, size)
		err := os.WriteFile(testFile, []byte("fake png content"), 0644)
		require.NoError(t, err)
	}

	// 测试加载图标集合
	iconSet, err := loader.LoadIconSet(basePath)
	require.NoError(t, err)
	require.NotEmpty(t, iconSet)

	// 验证不同尺寸的图标都已加载
	for _, size := range sizes {
		icon, exists := iconSet[size]
		assert.True(t, exists, "图标 %dx%d 应该存在", size, size)
		assert.Equal(t, size, icon.Size)
	}
}

func TestIconLoader_LoadIconSet_NoFiles(t *testing.T) {
	loader := NewIconLoader()

	// 尝试加载不存在的图标集合
	iconSet, err := loader.LoadIconSet("non-existent-icon")

	require.Error(t, err)
	require.Nil(t, iconSet)
}

func TestIconLoader_GetIconBySize(t *testing.T) {
	loader := NewIconLoader()

	// 创建临时测试文件
	tempDir := t.TempDir()
	testFile32 := filepath.Join(tempDir, "test-icon-32.png")
	testFile48 := filepath.Join(tempDir, "test-icon-48.png")

	// 创建不同尺寸的图标文件
	os.WriteFile(testFile32, []byte("fake png content"), 0644)
	os.WriteFile(testFile48, []byte("fake png content"), 0644)

	// 加载图标
	loader.LoadIcon(testFile32, 32)
	loader.LoadIcon(testFile48, 48)

	// 测试按尺寸获取图标
	icon32, exists32 := loader.GetIconBySize(32)
	assert.True(t, exists32)
	assert.Equal(t, 32, icon32.Size)

	icon48, exists48 := loader.GetIconBySize(48)
	assert.True(t, exists48)
	assert.Equal(t, 48, icon48.Size)

	// 测试获取不存在的尺寸
	icon64, exists64 := loader.GetIconBySize(64)
	assert.False(t, exists64)
	assert.Nil(t, icon64)
}

func TestIconLoader_RemoveIcon(t *testing.T) {
	loader := NewIconLoader()

	// 创建临时测试文件
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test-icon-32.png")

	err := os.WriteFile(testFile, []byte("fake png content"), 0644)
	require.NoError(t, err)

	// 加载图标
	loader.LoadIcon(testFile, 32)

	// 验证图标已缓存
	assert.Equal(t, 1, loader.GetCacheSize())

	// 移除图标
	removed := loader.RemoveIcon(testFile)
	assert.True(t, removed)

	// 验证图标已从缓存中移除
	assert.Equal(t, 0, loader.GetCacheSize())

	// 尝试移除不存在的图标
	removedAgain := loader.RemoveIcon("non-existent-file.png")
	assert.False(t, removedAgain)
}

func TestIconLoader_ClearCache(t *testing.T) {
	loader := NewIconLoader()

	// 创建临时测试文件
	tempDir := t.TempDir()
	testFiles := []string{
		filepath.Join(tempDir, "test-icon-16.png"),
		filepath.Join(tempDir, "test-icon-32.png"),
		filepath.Join(tempDir, "test-icon-48.png"),
	}

	// 加载多个图标
	for _, file := range testFiles {
		os.WriteFile(file, []byte("fake png content"), 0644)
		loader.LoadIcon(file, 32)
	}

	// 验证图标已缓存
	assert.Greater(t, loader.GetCacheSize(), 0)

	// 清空缓存
	loader.ClearCache()

	// 验证缓存已清空
	assert.Equal(t, 0, loader.GetCacheSize())
}

func TestIconLoader_ValidateIconFile(t *testing.T) {
	loader := NewIconLoader()

	// 创建临时测试文件
	tempDir := t.TempDir()

	// 测试有效文件
	validFile := filepath.Join(tempDir, "valid.png")
	err := os.WriteFile(validFile, []byte("fake png content"), 0644)
	require.NoError(t, err)

	err = loader.ValidateIconFile(validFile)
	assert.NoError(t, err)

	// 测试不存在的文件
	err = loader.ValidateIconFile("non-existent.png")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "文件不存在")

	// 测试不支持的格式
	invalidFile := filepath.Join(tempDir, "invalid.txt")
	err = os.WriteFile(invalidFile, []byte("text content"), 0644)
	require.NoError(t, err)

	err = loader.ValidateIconFile(invalidFile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "不支持的文件格式")

	// 测试目录
	err = loader.ValidateIconFile(tempDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "目录而非文件")
}

func TestGetSupportedFormats(t *testing.T) {
	formats := GetSupportedFormats()

	require.NotEmpty(t, formats)
	assert.Contains(t, formats, ".png")
	assert.Contains(t, formats, ".ico")
	assert.Contains(t, formats, ".jpg")
	assert.Contains(t, formats, ".jpeg")
	assert.Contains(t, formats, ".bmp")
}