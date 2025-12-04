package logger

import (
	"log"
	"os"

	"github.com/WQGroup/logger"
	"github.com/sirupsen/logrus"
	"to_icalendar_tray/internal/models"
)

var (
	instance *Manager
)

// Manager 统一日志管理器
type Manager struct {
	config *models.LoggingConfig
}

// NewManager 创建新的日志管理器实例
func NewManager(config *models.LoggingConfig) *Manager {
	return &Manager{
		config: config,
	}
}

// GetInstance 获取单例日志管理器
func GetInstance() *Manager {
	if instance == nil {
		// 如果没有初始化，使用默认配置
		defaultConfig := &models.LoggingConfig{
			Level:         "info",
			ConsoleOutput: true,
			FileOutput:    true,
			LogDir:        "./Logs/",
		}
		instance = NewManager(defaultConfig)
		instance.Initialize()
	}
	return instance
}

// Initialize 初始化日志系统
func (m *Manager) Initialize() error {
	// 配置高级设置
	settings := logger.NewSettings()
	settings.LogNameBase = "to_icalendar"
	settings.Level = logger.GetLogger().Level // 保持当前级别

	// 设置日志级别
	settings.Level = convertLogLevel(m.config.Level)

	// 设置日志目录
	if m.config.LogDir != "" {
		settings.LogRootFPath = m.config.LogDir
	}

	// 应用设置
	logger.SetLoggerSettings(settings)

	return nil
}

// Debug 调试级别日志
func (m *Manager) Debug(args ...interface{}) {
	if m.config.FileOutput || m.config.ConsoleOutput {
		logger.Debug(args...)
	}
}

// Debugf 格式化调试日志
func (m *Manager) Debugf(format string, args ...interface{}) {
	if m.config.FileOutput || m.config.ConsoleOutput {
		logger.Debugf(format, args...)
	}
}

// Info 信息级别日志
func (m *Manager) Info(args ...interface{}) {
	if m.config.FileOutput || m.config.ConsoleOutput {
		logger.Info(args...)
	}
}

// Infof 格式化信息日志
func (m *Manager) Infof(format string, args ...interface{}) {
	if m.config.FileOutput || m.config.ConsoleOutput {
		logger.Infof(format, args...)
	}
}

// Warn 警告级别日志
func (m *Manager) Warn(args ...interface{}) {
	if m.config.FileOutput || m.config.ConsoleOutput {
		logger.Warn(args...)
	}
}

// Warnf 格式化警告日志
func (m *Manager) Warnf(format string, args ...interface{}) {
	if m.config.FileOutput || m.config.ConsoleOutput {
		logger.Warnf(format, args...)
	}
}

// Error 错误级别日志
func (m *Manager) Error(args ...interface{}) {
	if m.config.FileOutput || m.config.ConsoleOutput {
		logger.Error(args...)
	}
}

// Errorf 格式化错误日志
func (m *Manager) Errorf(format string, args ...interface{}) {
	if m.config.FileOutput || m.config.ConsoleOutput {
		logger.Errorf(format, args...)
	}
}

// Fatal 致命错误日志
func (m *Manager) Fatal(args ...interface{}) {
	if m.config.FileOutput || m.config.ConsoleOutput {
		logger.Fatal(args...)
	}
}

// Fatalf 格式化致命错误日志
func (m *Manager) Fatalf(format string, args ...interface{}) {
	if m.config.FileOutput || m.config.ConsoleOutput {
		logger.Fatalf(format, args...)
	}
}

// IsDebugEnabled 检查是否启用了调试级别
func (m *Manager) IsDebugEnabled() bool {
	return m.config.Level == "debug"
}

// UpdateConfig 更新日志配置
func (m *Manager) UpdateConfig(config *models.LoggingConfig) error {
	m.config = config
	return m.Initialize()
}

// GetLogFilePath 获取当前日志文件路径
func (m *Manager) GetLogFilePath() string {
	return logger.CurrentFileName()
}

// convertLogLevel 将字符串级别转换为 logrus.Level
func convertLogLevel(levelStr string) logrus.Level {
	switch levelStr {
	case "debug":
		return logrus.DebugLevel
	case "info":
		return logrus.InfoLevel
	case "warn":
		return logrus.WarnLevel
	case "error":
		return logrus.ErrorLevel
	case "fatal":
		return logrus.ErrorLevel // 使用error级别
	default:
		return logrus.InfoLevel
	}
}

// 全局便捷函数
func Debug(args ...interface{}) {
	GetInstance().Debug(args...)
}

func Debugf(format string, args ...interface{}) {
	GetInstance().Debugf(format, args...)
}

func Info(args ...interface{}) {
	GetInstance().Info(args...)
}

func Infof(format string, args ...interface{}) {
	GetInstance().Infof(format, args...)
}

func Warn(args ...interface{}) {
	GetInstance().Warn(args...)
}

func Warnf(format string, args ...interface{}) {
	GetInstance().Warnf(format, args...)
}

func Error(args ...interface{}) {
	GetInstance().Error(args...)
}

func Errorf(format string, args ...interface{}) {
	GetInstance().Errorf(format, args...)
}

func Fatal(args ...interface{}) {
	GetInstance().Fatal(args...)
}

func Fatalf(format string, args ...interface{}) {
	GetInstance().Fatalf(format, args...)
}

func Initialize(config *models.LoggingConfig) error {
	instance = NewManager(config)
	return instance.Initialize()
}

// GetLogger 获取日志管理器实例（为了兼容性）
func GetLogger() *Manager {
	return GetInstance()
}

// GetStdLogger 获取标准库 logger（为了兼容性）
func (m *Manager) GetStdLogger() *log.Logger {
	// 返回一个简单的标准 logger，实际日志会通过我们的管理器处理
	return log.New(os.Stdout, "[to_icalendar] ", log.LstdFlags)
}