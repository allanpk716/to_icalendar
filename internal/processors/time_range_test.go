package processors

import (
	"testing"
)

func TestTimeRangeParser(t *testing.T) {
	// 创建任务解析器
	parser := NewTaskParser()

	// 测试用例：时间范围格式
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
			input:    "09:00-10:30",
			expected: "09:00",
			desc:     "上午时间范围（无空格）",
		},
		{
			input:    "14:30",
			expected: "14:30",
			desc:     "单个时间（非范围）",
		},
		{
			input:    "上午",
			expected: "09:00",
			desc:     "相对时间",
		},
		{
			input:    "9点",
			expected: "09:00",
			desc:     "中文时间格式",
		},
		{
			input:    "invalid time",
			expected: "invalid time",
			desc:     "无效时间格式",
		},
		{
			input:    "会议时间：14:30 - 16:30",
			expected: "14:30",
			desc:     "包含文本的时间范围",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			result := parser.normalizeTime(tc.input)
			if result != tc.expected {
				t.Errorf("测试失败: %s\n输入: %s\n期望: %s\n实际: %s",
					tc.desc, tc.input, tc.expected, result)
			} else {
				t.Logf("测试通过: %s -> %s", tc.input, result)
			}
		})
	}
}

func TestExtractTimeFromRange(t *testing.T) {
	parser := NewTaskParser()

	testCases := []struct {
		input    string
		expected string
		desc     string
	}{
		{"14:30 - 16:30", "14:30", "标准时间范围"},
		{"09:00-10:30", "09:00", "上午时间范围"},
		{"14:30~16:30", "14:30", "波浪号分隔"},
		{"14:30到16:30", "14:30", "中文字符分隔"},
		{"会议14:30-16:30进行", "14:30", "包含文本的时间范围"},
		{"14:30", "14:30", "单个时间"},
		{"invalid", "", "无效格式"},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			result := parser.extractTimeFromRange(tc.input)
			if result != tc.expected {
				t.Errorf("测试失败: %s\n输入: %s\n期望: %s\n实际: %s",
					tc.desc, tc.input, tc.expected, result)
			} else {
				t.Logf("测试通过: %s -> %s", tc.input, result)
			}
		})
	}
}