package timezone

import (
	"testing"
	"time"
)

func TestGetTimezoneLocation(t *testing.T) {
	tests := []struct {
		name     string
		timezone string
		want     string
	}{
		{"UTC timezone", "UTC", "UTC"},
		{"Empty timezone", "", "UTC"},
		{"Shanghai timezone", "Asia/Shanghai", "Asia/Shanghai"},
		{"New York timezone", "America/New_York", "America/New_York"},
		{"Invalid timezone", "Invalid/Timezone", "UTC"}, // 应该回退到UTC
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetTimezoneLocation(tt.timezone)
			if got.String() != tt.want {
				t.Errorf("GetTimezoneLocation() = %v, want %v", got.String(), tt.want)
			}
		})
	}
}

func TestConvertToTargetTimezone(t *testing.T) {
	// 固定的UTC时间用于测试
	utcTime := time.Date(2025, 11, 27, 14, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		utcTime   time.Time
		targetTZ  string
		expected  string // 期望的显示时间
	}{
		{
			name:     "UTC to UTC",
			utcTime:  utcTime,
			targetTZ: "UTC",
			expected: "2025-11-27 14:00:00",
		},
		{
			name:     "UTC to Shanghai",
			utcTime:  utcTime,
			targetTZ: "Asia/Shanghai",
			expected: "2025-11-27 22:00:00", // UTC+8
		},
		{
			name:     "UTC to New York",
			utcTime:  utcTime,
			targetTZ: "America/New_York",
			expected: "2025-11-27 09:00:00", // UTC-5 (标准时间)
		},
		{
			name:     "Empty timezone",
			utcTime:  utcTime,
			targetTZ: "",
			expected: "2025-11-27 14:00:00", // 保持UTC
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConvertToTargetTimezone(tt.utcTime, tt.targetTZ)
			gotStr := got.Format("2006-01-02 15:04:05")
			if gotStr != tt.expected {
				t.Errorf("ConvertToTargetTimezone() = %v, want %v", gotStr, tt.expected)
			}
		})
	}
}

func TestIsValidTimezone(t *testing.T) {
	tests := []struct {
		name     string
		timezone string
		want     bool
	}{
		{"Valid UTC", "UTC", true},
		{"Valid Shanghai", "Asia/Shanghai", true},
		{"Valid New York", "America/New_York", true},
		{"Valid London", "Europe/London", true},
		{"Empty timezone", "", true},  // 空时区应该被处理为UTC
		{"Invalid timezone", "Invalid/Timezone", false},
		{"Null timezone", "Null", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidTimezone(tt.timezone)
			if got != tt.want {
				t.Errorf("IsValidTimezone() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetSupportedTimezones(t *testing.T) {
	timezones := GetSupportedTimezones()

	if len(timezones) == 0 {
		t.Error("GetSupportedTimezones() should return at least one timezone")
	}

	// 检查是否包含常见的时区
	expectedTimezones := []string{"UTC", "Asia/Shanghai", "America/New_York"}
	for _, expected := range expectedTimezones {
		found := false
		for _, tz := range timezones {
			if tz == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected timezone %s not found in supported timezones", expected)
		}
	}
}

func TestFormatTimeForGraphAPI(t *testing.T) {
	// 测试时间：2025-11-27 14:00:00 UTC
	testTime := time.Date(2025, 11, 27, 14, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		t         time.Time
		timezone  string
		expected  string
	}{
		{
			name:     "UTC timezone",
			t:        testTime,
			timezone: "UTC",
			expected: "2025-11-27T14:00:00", // UTC时间格式
		},
		{
			name:     "Empty timezone",
			t:        testTime,
			timezone: "",
			expected: "2025-11-27T14:00:00", // 默认为UTC
		},
		{
			name:     "Shanghai timezone",
			t:        testTime,
			timezone: "Asia/Shanghai",
			expected: "2025-11-27T22:00:00", // 转换为上海时间
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatTimeForGraphAPI(tt.t, tt.timezone)
			if got != tt.expected {
				t.Errorf("FormatTimeForGraphAPI() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// 基准测试：检查时区转换性能
func BenchmarkConvertToTargetTimezone(b *testing.B) {
	utcTime := time.Date(2025, 11, 27, 14, 0, 0, 0, time.UTC)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ConvertToTargetTimezone(utcTime, "Asia/Shanghai")
	}
}

// 测试14:00这个特定时间点的转换（对应原问题的测试用例）
func TestAfternoonTimeConversion(t *testing.T) {
	// 模拟原问题：下午14:00在上海时区
	localTime := time.Date(2025, 11, 27, 14, 0, 0, 0,
		time.FixedZone("CST", 8*3600)) // 上海时间 UTC+8

	// 转换为UTC（应该得到06:00 UTC）
	utcTime := localTime.UTC()
	expectedUTC := "2025-11-27 06:00:00"
	if utcTime.Format("2006-01-02 15:04:05") != expectedUTC {
		t.Errorf("Expected UTC time %s, got %s", expectedUTC, utcTime.Format("2006-01-02 15:04:05"))
	}

	// 从UTC转换回上海时区（应该得到14:00）
	shanghaiTime := ConvertToTargetTimezone(utcTime, "Asia/Shanghai")
	expectedShanghai := "2025-11-27 14:00:00"
	if shanghaiTime.Format("2006-01-02 15:04:05") != expectedShanghai {
		t.Errorf("Expected Shanghai time %s, got %s", expectedShanghai, shanghaiTime.Format("2006-01-02 15:04:05"))
	}

	t.Logf("✅ 下午14:00时间转换测试通过:")
	t.Logf("   上海本地时间: %s", localTime.Format("2006-01-02 15:04:05"))
	t.Logf("   UTC时间: %s", utcTime.Format("2006-01-02 15:04:05"))
	t.Logf("   转换回上海时间: %s", shanghaiTime.Format("2006-01-02 15:04:05"))
}