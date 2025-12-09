package timezone

import (
	"time"

	"github.com/WQGroup/logger"
)

// GetTimezoneLocation 安全获取时区位置
// 如果无法加载指定时区，则回退到UTC
func GetTimezoneLocation(tz string) *time.Location {
	if tz == "" || tz == "UTC" {
		logger.Debugf("使用UTC时区")
		return time.UTC
	}

	// 尝试加载时区
	location, err := time.LoadLocation(tz)
	if err != nil {
		logger.Errorf("无法加载时区 '%s': %v，回退到UTC", tz, err)
		logger.Warnf("Windows用户请注意：如果时区加载失败，建议在配置中使用 'UTC' 时区")
		return time.UTC
	}

	logger.Debugf("成功加载时区: %s", tz)
	return location
}

// ConvertToTargetTimezone 将UTC时间转换为目标时区
func ConvertToTargetTimezone(utcTime time.Time, targetTZ string) time.Time {
	if targetTZ == "" || targetTZ == "UTC" {
		logger.Debugf("目标时区为UTC或空，无需转换")
		return utcTime
	}

	location := GetTimezoneLocation(targetTZ)
	targetTime := utcTime.In(location)

	logger.Debugf("时间转换: UTC %s -> %s %s",
		utcTime.Format("2006-01-02 15:04:05"),
		targetTZ,
		targetTime.Format("2006-01-02 15:04:05"))

	return targetTime
}

// IsValidTimezone 验证时区名称是否有效
func IsValidTimezone(tz string) bool {
	if tz == "" || tz == "UTC" {
		return true
	}

	// 尝试加载时区来验证其有效性
	_, err := time.LoadLocation(tz)
	if err != nil {
		logger.Debugf("时区 '%s' 验证失败: %v", tz, err)
		return false
	}

	logger.Debugf("时区 '%s' 验证通过", tz)
	return true
}

// GetSupportedTimezones 获取支持的时区列表
func GetSupportedTimezones() []string {
	return []string{
		"UTC",
		"Asia/Shanghai",
		"Asia/Tokyo",
		"America/New_York",
		"America/Los_Angeles",
		"Europe/London",
		"Europe/Paris",
		"Australia/Sydney",
	}
}

// FormatTimeForGraphAPI 将时间格式化为Microsoft Graph API的标准格式
func FormatTimeForGraphAPI(t time.Time, timezone string) string {
	if timezone == "" || timezone == "UTC" {
		return t.UTC().Format("2006-01-02T15:04:05")
	}

	targetTime := ConvertToTargetTimezone(t, timezone)
	return targetTime.Format("2006-01-02T15:04:05")
}