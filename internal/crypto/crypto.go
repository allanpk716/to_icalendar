package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/shirou/gopsutil/v3/host"
)

const (
	encryptedPasswordFile = ".encrypted_password"
	keyLength             = 32 // AES-256
)

// PasswordManager 管理密码的加密存储和解密
type PasswordManager struct {
	dataDir string
	key     []byte
}

// NewPasswordManager 创建密码管理器
func NewPasswordManager(dataDir string) (*PasswordManager, error) {
	pm := &PasswordManager{dataDir: dataDir}

	// 生成或加载加密密钥
	key, err := pm.generateOrLoadKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate/load key: %w", err)
	}

	pm.key = key
	return pm, nil
}

// SavePassword 加密并保存密码
func (pm *PasswordManager) SavePassword(password string) error {
	// 确保数据目录存在
	if err := os.MkdirAll(pm.dataDir, 0700); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	// 加密密码
	encrypted, err := pm.encrypt(password)
	if err != nil {
		return fmt.Errorf("failed to encrypt password: %w", err)
	}

	// 保存到文件
	filePath := filepath.Join(pm.dataDir, encryptedPasswordFile)
	return os.WriteFile(filePath, []byte(encrypted), 0600)
}

// LoadPassword 解密并加载密码
func (pm *PasswordManager) LoadPassword() (string, error) {
	filePath := filepath.Join(pm.dataDir, encryptedPasswordFile)

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return "", fmt.Errorf("encrypted password file not found")
	}

	// 读取加密数据
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read encrypted password file: %w", err)
	}

	// 解密密码
	password, err := pm.decrypt(string(data))
	if err != nil {
		return "", fmt.Errorf("failed to decrypt password: %w", err)
	}

	return password, nil
}

// HasSavedPassword 检查是否有已保存的密码
func (pm *PasswordManager) HasSavedPassword() bool {
	filePath := filepath.Join(pm.dataDir, encryptedPasswordFile)
	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}

// generateOrLoadKey 生成或加载加密密钥
func (pm *PasswordManager) generateOrLoadKey() ([]byte, error) {
	// 基于机器特征生成密钥
	machineID, err := pm.getMachineID()
	if err != nil {
		return nil, fmt.Errorf("failed to get machine ID: %w", err)
	}

	// 使用SHA256生成32字节的密钥
	hash := sha256.Sum256([]byte(machineID))
	return hash[:], nil
}

// getMachineID 获取机器唯一标识
func (pm *PasswordManager) getMachineID() (string, error) {
	hostInfo, err := host.Info()
	if err != nil {
		return "", err
	}

	// 组合多个机器特征来生成唯一ID
	machineID := fmt.Sprintf("%s-%s-%s-%s",
		hostInfo.HostID,
		hostInfo.Platform,
		hostInfo.PlatformVersion,
		hostInfo.KernelVersion)

	// 如果无法获取HostID，使用其他信息
	if hostInfo.HostID == "" {
		machineID = fmt.Sprintf("%s-%s-%s-%s",
			hostInfo.Hostname,
			hostInfo.Platform,
			hostInfo.PlatformVersion,
			hostInfo.KernelVersion)
	}

	return machineID, nil
}

// encrypt 使用AES-256-GCM加密数据
func (pm *PasswordManager) encrypt(plaintext string) (string, error) {
	// 创建AES cipher
	block, err := aes.NewCipher(pm.key)
	if err != nil {
		return "", err
	}

	// 创建GCM模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// 生成随机nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// 加密数据
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// 使用base64编码
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decrypt 使用AES-256-GCM解密数据
func (pm *PasswordManager) decrypt(ciphertext string) (string, error) {
	// base64解码
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	// 创建AES cipher
	block, err := aes.NewCipher(pm.key)
	if err != nil {
		return "", err
	}

	// 创建GCM模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// 检查数据长度
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	// 分离nonce和密文
	nonce, ciphertext_bytes := data[:nonceSize], data[nonceSize:]

	// 解密数据
	plaintext, err := gcm.Open(nil, nonce, ciphertext_bytes, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// ClearSensitiveData 清理内存中的敏感数据
func (pm *PasswordManager) ClearSensitiveData() {
	// 清零密钥
	for i := range pm.key {
		pm.key[i] = 0
	}
	pm.key = nil
}

// GetPasswordFilePath 获取密码文件路径（用于调试）
func (pm *PasswordManager) GetPasswordFilePath() string {
	return filepath.Join(pm.dataDir, encryptedPasswordFile)
}