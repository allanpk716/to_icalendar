package processors_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/allanpk716/to_icalendar/internal/models"
	"github.com/allanpk716/to_icalendar/internal/processors"
)

func TestJSONGenerator_NewJSONGenerator(t *testing.T) {
	tempDir := t.TempDir()

	generator, err := processors.NewJSONGenerator(tempDir)
	if err != nil {
		t.Fatalf("Failed to create JSON generator: %v", err)
	}

	if generator == nil {
		t.Error("Expected generator but got nil")
	}
}

func TestJSONGenerator_GenerateFromReminder(t *testing.T) {
	tempDir := t.TempDir()
	generator, err := processors.NewJSONGenerator(tempDir)
	if err != nil {
		t.Fatalf("Failed to create JSON generator: %v", err)
	}

	reminder := &models.Reminder{
		Title:        "项目评审会议",
		Description:  "讨论Q4项目进度和下一步计划",
		Date:         "2025-11-06",
		Time:         "14:00",
		RemindBefore: "15m",
		Priority:     models.PriorityHigh,
		List:         "会议",
	}

	filePath, err := generator.GenerateFromReminder(reminder)
	if err != nil {
		t.Fatalf("Failed to generate JSON: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("JSON file was not created")
	}

	// Verify file is in correct directory
	if !filepath.HasPrefix(filePath, tempDir) {
		t.Error("Generated file is not in the correct directory")
	}

	// Verify file has .json extension
	if filepath.Ext(filePath) != ".json" {
		t.Errorf("Expected .json extension, got %s", filepath.Ext(filePath))
	}

	t.Logf("Generated JSON file: %s", filePath)
}

func TestJSONGenerator_GenerateFromParsedInfo(t *testing.T) {
	tempDir := t.TempDir()
	generator, err := processors.NewJSONGenerator(tempDir)
	if err != nil {
		t.Fatalf("Failed to create JSON generator: %v", err)
	}

	taskInfo := &models.ParsedTaskInfo{
		Title:      "开会讨论项目进展",
		Date:       "明天",
		Time:       "下午2点",
		Priority:   "中",
		List:       "工作",
		Confidence: 0.85,
		OriginalText: "明天下午2点开会讨论项目进展",
	}

	filePath, err := generator.GenerateFromParsedInfo(taskInfo)
	if err != nil {
		t.Fatalf("Failed to generate JSON: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("JSON file was not created")
	}

	t.Logf("Generated JSON file: %s", filePath)
}

func TestJSONGenerator_ValidateReminder(t *testing.T) {
	tempDir := t.TempDir()
	generator, err := processors.NewJSONGenerator(tempDir)
	if err != nil {
		t.Fatalf("Failed to create JSON generator: %v", err)
	}

	tests := []struct {
		name        string
		reminder    *models.Reminder
		expectError bool
	}{
		{
			name: "valid reminder",
			reminder: &models.Reminder{
				Title:        "会议提醒",
				Description:  "重要会议",
				Date:         "2025-11-06",
				Time:         "14:00",
				RemindBefore: "15m",
				Priority:     models.PriorityMedium,
				List:         "工作",
			},
			expectError: false,
		},
		{
			name: "empty title",
			reminder: &models.Reminder{
				Title:        "",
				Description:  "重要会议",
				Date:         "2025-11-06",
				Time:         "14:00",
				RemindBefore: "15m",
				Priority:     models.PriorityMedium,
				List:         "工作",
			},
			expectError: true,
		},
		{
			name: "invalid date format",
			reminder: &models.Reminder{
				Title:        "会议提醒",
				Description:  "重要会议",
				Date:         "invalid-date",
				Time:         "14:00",
				RemindBefore: "15m",
				Priority:     models.PriorityMedium,
				List:         "工作",
			},
			expectError: true,
		},
		{
			name: "invalid time format",
			reminder: &models.Reminder{
				Title:        "会议提醒",
				Description:  "重要会议",
				Date:         "2025-11-06",
				Time:         "25:00", // Invalid time
				RemindBefore: "15m",
				Priority:     models.PriorityMedium,
				List:         "工作",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := generator.ValidateReminder(tt.reminder)

			if tt.expectError && err == nil {
				t.Error("Expected validation error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected validation error: %v", err)
			}
		})
	}
}