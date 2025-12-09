package models

import (
	"testing"
	"time"
)

func TestParseTimeFromRange(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
		desc     string
	}{
		{
			input:    "14:30 - 16:30",
			expected: "14:30",
			desc:     "标准时间范围格式（带空格）",
		},
		{
			input:    "14:30-16:30",
			expected: "14:30",
			desc:     "标准时间范围格式（无空格）",
		},
		{
			input:    "14:30~16:30",
			expected: "14:30",
			desc:     "波浪号时间范围格式",
		},
		{
			input:    "14:30到16:30",
			expected: "14:30",
			desc:     "中文字符时间范围格式",
		},
		{
			input:    "14:30至16:30",
			expected: "14:30",
			desc:     "中文字符时间范围格式（至）",
		},
		{
			input:    "09:00 - 10:30",
			expected: "09:00",
			desc:     "上午时间范围",
		},
		{
			input:    "9:00-10:30",
			expected: "9:00",
			desc:     "上午时间范围（单数字小时）",
		},
		{
			input:    "14:30",
			expected: "14:30",
			desc:     "单个时间（非范围）",
		},
		{
			input:    "9:00",
			expected: "9:00",
			desc:     "单个时间（单数字小时）",
		},
		{
			input:    "invalid time",
			expected: "invalid time",
			desc:     "无效时间格式",
		},
		{
			input:    "会议14:30-16:30进行",
			expected: "14:30",
			desc:     "包含文本的时间范围",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			result := parseTimeFromRange(tc.input)
			if result != tc.expected {
				t.Errorf("测试失败: %s\n输入: %s\n期望: %s\n实际: %s",
					tc.desc, tc.input, tc.expected, result)
			} else {
				t.Logf("测试通过: %s -> %s", tc.input, result)
			}
		})
	}
}

func TestParseReminderTimeWithRange(t *testing.T) {
	// 创建测试用的时区
	timezone, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatalf("加载时区失败: %v", err)
	}

	testCases := []struct {
		reminder Reminder
		expected string
		desc     string
	}{
		{
			reminder: Reminder{
				Title:        "会议",
				Description:  "测试会议",
				Date:         "2025-11-06",
				Time:         "14:30 - 16:30",
				RemindBefore: "30m",
				Priority:     PriorityHigh,
				List:         "会议",
			},
			expected: "2025-11-06 14:30",
			desc:     "时间范围提醒",
		},
		{
			reminder: Reminder{
				Title:        "会议",
				Description:  "测试会议",
				Date:         "2025-11-06",
				Time:         "09:00-10:30",
				RemindBefore: "15m",
				Priority:     PriorityMedium,
				List:         "工作",
			},
			expected: "2025-11-06 09:00",
			desc:     "上午时间范围提醒",
		},
		{
			reminder: Reminder{
				Title:        "会议",
				Description:  "测试会议",
				Date:         "2025-11-06",
				Time:         "14:30",
				RemindBefore: "30m",
				Priority:     PriorityLow,
				List:         "个人",
			},
			expected: "2025-11-06 14:30",
			desc:     "单个时间提醒",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			parsed, err := ParseReminderTime(tc.reminder, timezone)
			if err != nil {
				t.Errorf("解析失败: %s\n错误: %v", tc.desc, err)
				return
			}

			actual := parsed.DueTime.Format("2006-01-02 15:04")
			if actual != tc.expected {
				t.Errorf("测试失败: %s\n期望: %s\n实际: %s",
					tc.desc, tc.expected, actual)
			} else {
				t.Logf("测试通过: %s -> %s", tc.desc, actual)
			}
		})
	}
}

func TestIsValidTimeFormat(t *testing.T) {
	testCases := []struct {
		input    string
		expected bool
		desc     string
	}{
		{"14:30", true, "标准时间格式"},
		{"9:00", true, "单数字小时格式"},
		{"23:59", true, "有效时间边界"},
		{"00:00", true, "有效时间边界"},
		{"24:00", false, "无效小时"},
		{"14:60", false, "无效分钟"},
		{"14:30:15", false, "包含秒的格式"},
		{"invalid", false, "无效格式"},
		{"", false, "空字符串"},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			result := isValidTimeFormat(tc.input)
			if result != tc.expected {
				t.Errorf("测试失败: %s\n输入: %s\n期望: %v\n实际: %v",
					tc.desc, tc.input, tc.expected, result)
			} else {
				t.Logf("测试通过: %s -> %v", tc.input, result)
			}
		})
	}
}