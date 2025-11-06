package utils

import (
	"github.com/allanpk716/to_icalendar/internal/models"
)

// GetTestReminders returns a collection of test reminder objects
func GetTestReminders() map[string]*models.Reminder {
	return map[string]*models.Reminder{
		"meeting": {
			Title:        "项目评审会议",
			Description:  "讨论Q4项目进度和下一步计划",
			Date:         "2025-11-06",
			Time:         "14:00",
			RemindBefore: "15m",
			Priority:     models.PriorityHigh,
			List:         "会议",
		},
		"task": {
			Title:        "完成项目报告",
			Description:  "需要在本周五之前完成季度项目总结报告",
			Date:         "2025-11-08",
			Time:         "17:00",
			RemindBefore: "1h",
			Priority:     models.PriorityMedium,
			List:         "工作",
		},
		"urgent": {
			Title:        "紧急bug修复",
			Description:  "生产环境发现严重bug，需要立即修复",
			Date:         "今天",
			Time:         "现在",
			RemindBefore: "5m",
			Priority:     models.PriorityHigh,
			List:         "紧急",
		},
		"personal": {
			Title:        "健身房锻炼",
			Description:  "每周三次的健身计划",
			Date:         "明天",
			Time:         "19:00",
			RemindBefore: "30m",
			Priority:     models.PriorityLow,
			List:         "个人",
		},
	}
}

// GetInvalidReminders returns a collection of invalid reminder objects for testing validation
func GetInvalidReminders() map[string]*models.Reminder {
	return map[string]*models.Reminder{
		"empty_title": {
			Title:        "",
			Description:  "测试描述",
			Date:         "2025-11-06",
			Time:         "14:00",
			RemindBefore: "15m",
			Priority:     models.PriorityMedium,
			List:         "测试",
		},
		"invalid_date": {
			Title:        "测试标题",
			Description:  "测试描述",
			Date:         "invalid-date",
			Time:         "14:00",
			RemindBefore: "15m",
			Priority:     models.PriorityMedium,
			List:         "测试",
		},
		"invalid_time": {
			Title:        "测试标题",
			Description:  "测试描述",
			Date:         "2025-11-06",
			Time:         "25:00", // Invalid time
			RemindBefore: "15m",
			Priority:     models.PriorityMedium,
			List:         "测试",
		},
	}
}

// GetTestTaskInfos returns a collection of test task info objects
func GetTestTaskInfos() map[string]*models.ParsedTaskInfo {
	return map[string]*models.ParsedTaskInfo{
		"meeting_task": {
			Title:         "开会讨论项目进展",
			Date:          "明天",
			Time:          "下午2点",
			Priority:      "高",
			List:          "工作",
			Confidence:    0.85,
			OriginalText:  "明天下午2点开会讨论项目进展，请准时参加",
		},
		"urgent_task": {
			Title:         "完成重要报告",
			Date:          "今天",
			Time:          "下午",
			Priority:      "高",
			List:          "工作",
			Confidence:    0.92,
			OriginalText:  "今天下午必须完成重要报告，非常紧急",
		},
		"simple_task": {
			Title:         "购买生活用品",
			Date:          "",
			Time:          "",
			Priority:      "中",
			List:          "个人",
			Confidence:    0.65,
			OriginalText:  "购买生活用品",
		},
		"low_confidence": {
			Title:         "一些事情",
			Date:          "",
			Time:          "",
			Priority:      "",
			List:          "",
			Confidence:    0.25,
			OriginalText:  "一些事情",
		},
	}
}

// GetTestTexts returns a collection of test text strings for processing
func GetTestTexts() map[string]string {
	return map[string]string{
		"meeting_with_datetime": "明天下午2点开会讨论项目进展",
		"urgent_task":          "今天下午必须完成重要报告，非常紧急",
		"simple_task":          "购买生活用品",
		"empty_text":           "",
		"whitespace_only":      "   \n\t   ",
		"long_text":            "这是一个很长的任务描述，包含了很多细节信息，需要在不同时间段完成多个步骤，并且涉及多个人员的协作，这是一个非常复杂的项目管理任务",
		"date_only":            "2025年11月6日提交报告",
		"time_only":            "下午3点有个会议",
		"invalid_format":       "asdfghjklqwertyuiopzxcvbnm",
		"mixed_content":        "明天下午2点在会议室A开会讨论Q4项目进展，需要准备相关材料，请准时参加，非常重要的是还要带上笔记本电脑",
	}
}

// GetTestConfigs returns a collection of test configuration objects
func GetTestConfigs() map[string]*models.ServerConfig {
	return map[string]*models.ServerConfig{
		"valid_config": {
			Dify: models.DifyConfig{
				APIEndpoint: "https://api.dify.ai/v1",
				APIKey:      "test-valid-api-key",
				Model:       "gpt-3.5-turbo",
			},
			MicrosoftTodo: models.MicrosoftTodoConfig{
				TenantID:     "test-tenant-id",
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				Timezone:     "Asia/Shanghai",
			},
		},
		"invalid_dify_config": {
			Dify: models.DifyConfig{
				APIEndpoint: "", // Empty endpoint
				APIKey:      "", // Empty API key
				Model:       "gpt-3.5-turbo",
			},
			MicrosoftTodo: models.MicrosoftTodoConfig{
				TenantID:     "test-tenant-id",
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				Timezone:     "Asia/Shanghai",
			},
		},
		"placeholder_config": {
			Dify: models.DifyConfig{
				APIEndpoint: "https://api.dify.ai/v1",
				APIKey:      "YOUR_DIFY_API_KEY", // Placeholder
				Model:       "gpt-3.5-turbo",
			},
			MicrosoftTodo: models.MicrosoftTodoConfig{
				TenantID:     "test-tenant-id",
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				Timezone:     "Asia/Shanghai",
			},
		},
	}
}

// GetTestImageData returns test image data for testing image processing
func GetTestImageData() map[string][]byte {
	return map[string][]byte{
		"small_png":  make([]byte, 1024),     // 1KB
		"medium_jpg": make([]byte, 50*1024),  // 50KB
		"large_png":  make([]byte, 1024*1024), // 1MB
		"empty":      []byte{},               // Empty
	}
}

// GetTestFileNames returns test file names for different formats
func GetTestFileNames() map[string]string {
	return map[string]string{
		"valid_png":    "test.png",
		"valid_jpg":    "test.jpg",
		"valid_jpeg":   "test.jpeg",
		"invalid_exe":  "test.exe",
		"no_extension": "test",
		"dotted_name":  "test.file.png",
	}
}