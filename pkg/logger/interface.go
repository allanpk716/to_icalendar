package logger

import "github.com/allanpk716/to_icalendar/pkg/models"

// Logger 日志接口
type Logger interface {
	Debug(args ...interface{})
	Debugf(format string, args ...interface{})
	Info(args ...interface{})
	Infof(format string, args ...interface{})
	Warn(args ...interface{})
	Warnf(format string, args ...interface{})
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
	IsDebugEnabled() bool
	UpdateConfig(config *models.LoggingConfig) error
	GetLogFilePath() string
}