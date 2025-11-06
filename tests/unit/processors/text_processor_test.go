package processors_test

import (
	"testing"

	"github.com/allanpk716/to_icalendar/internal/processors"
)

func TestTextProcessor_NewTextProcessor(t *testing.T) {
	// Test without Dify processor
	processor, err := processors.NewTextProcessor(nil)
	if err != nil {
		t.Fatalf("Failed to create text processor: %v", err)
	}

	if processor == nil {
		t.Error("Expected processor but got nil")
	}
}

func TestTextProcessor_QuickAnalyze(t *testing.T) {
	processor, err := processors.NewTextProcessor(nil)
	if err != nil {
		t.Fatalf("Failed to create text processor: %v", err)
	}

	tests := []struct {
		name           string
		input          string
		expectedHasDate bool
		expectedHasTime bool
		expectedUrgent  bool
		expectedMeeting bool
	}{
		{
			name:            "meeting with date and time",
			input:           "明天下午2点开会讨论项目进展",
			expectedHasDate: true,
			expectedHasTime: true,
			expectedUrgent:  false,
			expectedMeeting: true,
		},
		{
			name:            "urgent task with time",
			input:           "今天下午必须完成重要报告，非常紧急",
			expectedHasDate: true,
			expectedHasTime: true,
			expectedUrgent:  true,
			expectedMeeting: false,
		},
		{
			name:            "simple task without time",
			input:           "购买生活用品",
			expectedHasDate: false,
			expectedHasTime: false,
			expectedUrgent:  false,
			expectedMeeting: false,
		},
		{
			name:            "empty string",
			input:           "",
			expectedHasDate: false,
			expectedHasTime: false,
			expectedUrgent:  false,
			expectedMeeting: false,
		},
		{
			name:            "date only",
			input:           "2025年11月6日提交报告",
			expectedHasDate: true,
			expectedHasTime: false,
			expectedUrgent:  false,
			expectedMeeting: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analysis := processor.QuickAnalyze(tt.input)

			if analysis.HasDate != tt.expectedHasDate {
				t.Errorf("Expected HasDate=%v, got %v", tt.expectedHasDate, analysis.HasDate)
			}

			if analysis.HasTime != tt.expectedHasTime {
				t.Errorf("Expected HasTime=%v, got %v", tt.expectedHasTime, analysis.HasTime)
			}

			if analysis.IsUrgent != tt.expectedUrgent {
				t.Errorf("Expected IsUrgent=%v, got %v", tt.expectedUrgent, analysis.IsUrgent)
			}

			if analysis.IsMeeting != tt.expectedMeeting {
				t.Errorf("Expected IsMeeting=%v, got %v", tt.expectedMeeting, analysis.IsMeeting)
			}

			// Confidence should be between 0 and 1
			if analysis.Confidence < 0 || analysis.Confidence > 1 {
				t.Errorf("Confidence should be between 0 and 1, got %f", analysis.Confidence)
			}
		})
	}
}

func BenchmarkTextProcessor_QuickAnalyze(b *testing.B) {
	processor, err := processors.NewTextProcessor(nil)
	if err != nil {
		b.Fatalf("Failed to create text processor: %v", err)
	}

	testText := "明天下午2点开会讨论项目进展，非常重要，请准时参加"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = processor.QuickAnalyze(testText)
	}
}