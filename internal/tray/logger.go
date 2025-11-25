package tray

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// TrayLogger 托盘组件专用日志记录器
type TrayLogger struct {
	infoLogger  *log.Logger
	errorLogger *log.Logger
	debugLogger *log.Logger
	logFile     *os.File
	debugMode   bool
}

// NewTrayLogger 创建新的托盘日志记录器
func NewTrayLogger(debugMode bool) *TrayLogger {
	// 创建日志目录
	logDir := "logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Printf("Failed to create log directory: %v", err)
	}

	// 创建日志文件
	timestamp := time.Now().Format("2006-01-02")
	logFileName := filepath.Join(logDir, fmt.Sprintf("tray_%s.log", timestamp))
	logFile, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Printf("Failed to open log file: %v", err)
		return &TrayLogger{
			infoLogger:  log.New(os.Stdout, "[TRAY-INFO] ", log.LstdFlags),
			errorLogger: log.New(os.Stderr, "[TRAY-ERROR] ", log.LstdFlags),
			debugLogger: log.New(os.Stdout, "[TRAY-DEBUG] ", log.LstdFlags),
			debugMode:   debugMode,
		}
	}

	return &TrayLogger{
		infoLogger:  log.New(logFile, "[TRAY-INFO] ", log.LstdFlags|log.Lshortfile),
		errorLogger: log.New(logFile, "[TRAY-ERROR] ", log.LstdFlags|log.Lshortfile),
		debugLogger: log.New(logFile, "[TRAY-DEBUG] ", log.LstdFlags|log.Lshortfile),
		logFile:     logFile,
		debugMode:   debugMode,
	}
}

// Info 记录信息日志
func (tl *TrayLogger) Info(msg string) {
	tl.infoLogger.Println(msg)
	if tl.logFile != nil {
		tl.infoLogger.SetOutput(os.Stdout)
		tl.infoLogger.Println(msg)
		tl.infoLogger.SetOutput(tl.logFile)
	}
}

// Infof 记录格式化信息日志
func (tl *TrayLogger) Infof(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	tl.Info(msg)
}

// Error 记录错误日志
func (tl *TrayLogger) Error(msg string) {
	tl.errorLogger.Println(msg)
	if tl.logFile != nil {
		tl.errorLogger.SetOutput(os.Stderr)
		tl.errorLogger.Println(msg)
		tl.errorLogger.SetOutput(tl.logFile)
	}
}

// Errorf 记录格式化错误日志
func (tl *TrayLogger) Errorf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	tl.Error(msg)
}

// Debug 记录调试日志
func (tl *TrayLogger) Debug(msg string) {
	if tl.debugMode {
		tl.debugLogger.Println(msg)
		if tl.logFile != nil {
			tl.debugLogger.SetOutput(os.Stdout)
			tl.debugLogger.Println(msg)
			tl.debugLogger.SetOutput(tl.logFile)
		}
	}
}

// Debugf 记录格式化调试日志
func (tl *TrayLogger) Debugf(format string, args ...interface{}) {
	if tl.debugMode {
		msg := fmt.Sprintf(format, args...)
		tl.Debug(msg)
	}
}

// Close 关闭日志文件
func (tl *TrayLogger) Close() error {
	if tl.logFile != nil {
		return tl.logFile.Close()
	}
	return nil
}

// 全局日志记录器实例
var GlobalLogger *TrayLogger

// InitLogger 初始化全局日志记录器
func InitLogger(debugMode bool) {
	GlobalLogger = NewTrayLogger(debugMode)
}

// LogInfo 记录信息日志（全局函数）
func LogInfo(msg string) {
	if GlobalLogger != nil {
		GlobalLogger.Info(msg)
	} else {
		log.Printf("[TRAY-INFO] %s", msg)
	}
}

// LogInfof 记录格式化信息日志（全局函数）
func LogInfof(format string, args ...interface{}) {
	if GlobalLogger != nil {
		GlobalLogger.Infof(format, args)
	} else {
		log.Printf("[TRAY-INFO] "+format, args...)
	}
}

// LogError 记录错误日志（全局函数）
func LogError(msg string) {
	if GlobalLogger != nil {
		GlobalLogger.Error(msg)
	} else {
		log.Printf("[TRAY-ERROR] %s", msg)
	}
}

// LogErrorf 记录格式化错误日志（全局函数）
func LogErrorf(format string, args ...interface{}) {
	if GlobalLogger != nil {
		GlobalLogger.Errorf(format, args)
	} else {
		log.Printf("[TRAY-ERROR] "+format, args...)
	}
}

// LogDebug 记录调试日志（全局函数）
func LogDebug(msg string) {
	if GlobalLogger != nil {
		GlobalLogger.Debug(msg)
	} else {
		log.Printf("[TRAY-DEBUG] %s", msg)
	}
}

// LogDebugf 记录格式化调试日志（全局函数）
func LogDebugf(format string, args ...interface{}) {
	if GlobalLogger != nil {
		GlobalLogger.Debugf(format, args)
	} else {
		log.Printf("[TRAY-DEBUG] "+format, args...)
	}
}