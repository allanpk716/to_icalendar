package dify

import (
	"testing"
)

func TestIsValidTimeWithRange(t *testing.T) {
	testCases := []struct {
		input    string
		expected bool
		desc     string
	}{
		// 标准时间格式
		{"14:30", true, "标准时间格式"},
		{"9:00", true, "单数字小时格式"},
		{"23:59", true, "有效时间边界"},
		{"00:00", true, "有效时间边界"},
		{"24:00", false, "无效小时"},
		{"14:60", false, "无效分钟"},

		// 时间范围格式
		{"14:30 - 16:30", true, "标准时间范围格式（带空格）"},
		{"14:30-16:30", true, "标准时间范围格式（无空格）"},
		{"14:30~16:30", true, "波浪号时间范围格式"},
		{"14:30到16:30", true, "中文字符时间范围格式"},
		{"14:30至16:30", true, "中文字符时间范围格式（至）"},
		{"09:00 - 10:30", true, "上午时间范围"},
		{"9:00-10:30", true, "单数字小时时间范围"},

		// 无效格式
		{"invalid", false, "无效格式"},
		{"", false, "空字符串"},
		{"14:30:15", false, "包含秒的格式"},
		{"25:00", false, "超出范围的小时"},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			result := isValidTime(tc.input)
			if result != tc.expected {
				t.Errorf("测试失败: %s\n输入: %s\n期望: %v\n实际: %v",
					tc.desc, tc.input, tc.expected, result)
			} else {
				t.Logf("测试通过: %s -> %v", tc.input, result)
			}
		})
	}
}

func TestParseTimeFromRangeInDify(t *testing.T) {
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
			input:    "14:30",
			expected: "14:30",
			desc:     "单个时间（非范围）",
		},
		{
			input:    "invalid",
			expected: "invalid",
			desc:     "无效时间格式",
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