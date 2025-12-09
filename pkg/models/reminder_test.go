package models

import (
	"testing"
	"time"
)

// TestRemindBeforePriority 测试用户设置与智能提醒功能的优先级
func TestRemindBeforePriority(t *testing.T) {
	timezone, _ := time.LoadLocation("Asia/Shanghai")

	testCases := []struct {
		name           string
		reminder       Reminder
		config         *ReminderConfig
		expectedBefore string
		desc           string
	}{
		{
			name: "用户设置15m，高优先级，智能提醒禁用",
			reminder: Reminder{
				Title:        "测试",
				Date:         "2025-11-06",
				Time:         "15:00",
				RemindBefore: "15m", // 用户明确设置
				Priority:     PriorityHigh,
			},
			config: &ReminderConfig{
				DefaultRemindBefore: "15m",
				EnableSmartReminder: false, // 智能提醒禁用
			},
			expectedBefore: "15m",
			desc:           "应该使用用户设置的15m，而不是智能提醒的30m",
		},
		{
			name: "用户设置15m，高优先级，智能提醒启用",
			reminder: Reminder{
				Title:        "测试",
				Date:         "2025-11-06",
				Time:         "15:00",
				RemindBefore: "15m", // 用户明确设置
				Priority:     PriorityHigh,
			},
			config: &ReminderConfig{
				DefaultRemindBefore: "15m",
				EnableSmartReminder: true, // 智能提醒启用
			},
			expectedBefore: "15m",
			desc:           "即使智能提醒启用，仍应使用用户设置的15m",
		},
		{
			name: "用户未设置，高优先级，智能提醒启用",
			reminder: Reminder{
				Title:        "测试",
				Date:         "2025-11-06",
				Time:         "15:00",
				RemindBefore: "", // 用户未设置
				Priority:     PriorityHigh,
			},
			config: &ReminderConfig{
				DefaultRemindBefore: "15m",
				EnableSmartReminder: true, // 智能提醒启用
			},
			expectedBefore: "30m",
			desc:           "应该使用智能提醒的30m",
		},
		{
			name: "用户未设置，高优先级，智能提醒禁用",
			reminder: Reminder{
				Title:        "测试",
				Date:         "2025-11-06",
				Time:         "15:00",
				RemindBefore: "", // 用户未设置
				Priority:     PriorityHigh,
			},
			config: &ReminderConfig{
				DefaultRemindBefore: "20m",
				EnableSmartReminder: false, // 智能提醒禁用
			},
			expectedBefore: "20m",
			desc:           "应该使用默认提醒时间20m",
		},
		{
			name: "用户未设置，中优先级，智能提醒启用",
			reminder: Reminder{
				Title:        "测试",
				Date:         "2025-11-06",
				Time:         "15:00",
				RemindBefore: "", // 用户未设置
				Priority:     PriorityMedium,
			},
			config: &ReminderConfig{
				DefaultRemindBefore: "15m",
				EnableSmartReminder: true, // 智能提醒启用
			},
			expectedBefore: "15m",
			desc:           "应该使用智能提醒的15m",
		},
		{
			name: "用户未设置，低优先级，智能提醒启用",
			reminder: Reminder{
				Title:        "测试",
				Date:         "2025-11-06",
				Time:         "15:00",
				RemindBefore: "", // 用户未设置
				Priority:     PriorityLow,
			},
			config: &ReminderConfig{
				DefaultRemindBefore: "15m",
				EnableSmartReminder: true, // 智能提醒启用
			},
			expectedBefore: "5m",
			desc:           "应该使用智能提醒的5m",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parsed, err := ParseReminderTimeWithConfig(tc.reminder, timezone, tc.config)
			if err != nil {
				t.Fatalf("解析失败: %v", err)
			}

			// 计算实际的提醒时间差
			expectedDueTime, _ := time.ParseInLocation("2006-01-02 15:04",
				tc.reminder.Date+" "+tc.reminder.Time, timezone)
			diff := expectedDueTime.Sub(parsed.AlarmTime)
			expectedDuration, _ := time.ParseDuration(tc.expectedBefore)

			if diff != expectedDuration {
				t.Errorf("%s: 期望提醒时间 %s，实际 %s",
					tc.desc, tc.expectedBefore, diff.String())
			}

			// 验证截止时间解析正确
			if parsed.DueTime != expectedDueTime {
				t.Errorf("截止时间解析错误: 期望 %s，实际 %s",
					expectedDueTime.Format("15:04"), parsed.DueTime.Format("15:04"))
			}
		})
	}
}

// TestGetSmartRemindTime 测试智能提醒时间获取
func TestGetSmartRemindTime(t *testing.T) {
	testCases := []struct {
		name           string
		config         ReminderConfig
		priority       Priority
		expectedResult string
	}{
		{
			name: "智能提醒禁用，高优先级",
			config: ReminderConfig{
				DefaultRemindBefore: "20m",
				EnableSmartReminder: false,
			},
			priority:       PriorityHigh,
			expectedResult: "20m", // 使用默认值
		},
		{
			name: "智能提醒启用，高优先级",
			config: ReminderConfig{
				DefaultRemindBefore: "20m",
				EnableSmartReminder: true,
			},
			priority:       PriorityHigh,
			expectedResult: "30m", // 使用智能提醒值
		},
		{
			name: "智能提醒启用，中优先级",
			config: ReminderConfig{
				DefaultRemindBefore: "20m",
				EnableSmartReminder: true,
			},
			priority:       PriorityMedium,
			expectedResult: "15m", // 使用智能提醒值
		},
		{
			name: "智能提醒启用，低优先级",
			config: ReminderConfig{
				DefaultRemindBefore: "20m",
				EnableSmartReminder: true,
			},
			priority:       PriorityLow,
			expectedResult: "5m", // 使用智能提醒值
		},
		{
			name: "智能提醒启用，未知优先级",
			config: ReminderConfig{
				DefaultRemindBefore: "25m",
				EnableSmartReminder: true,
			},
			priority:       Priority("unknown"),
			expectedResult: "25m", // 使用默认值
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.config.GetSmartRemindTime(tc.priority)
			if result != tc.expectedResult {
				t.Errorf("期望 %s，实际 %s", tc.expectedResult, result)
			}
		})
	}
}

// TestParseDuration 测试时间解析
func TestParseDuration(t *testing.T) {
	baseTime, _ := time.Parse("2006-01-02 15:04:05", "2025-11-06 15:00:00")

	testCases := []struct {
		name         string
		duration     string
		expectedDiff time.Duration
	}{
		{
			name:         "15分钟",
			duration:     "15m",
			expectedDiff: -15 * time.Minute,
		},
		{
			name:         "1小时",
			duration:     "1h",
			expectedDiff: -1 * time.Hour,
		},
		{
			name:         "2天",
			duration:     "2d",
			expectedDiff: -2 * 24 * time.Hour,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := parseDuration(baseTime, tc.duration)
			if err != nil {
				t.Fatalf("解析失败: %v", err)
			}

			diff := result.Sub(baseTime)
			if diff != tc.expectedDiff {
				t.Errorf("期望时间差 %v，实际 %v", tc.expectedDiff, diff)
			}
		})
	}
}