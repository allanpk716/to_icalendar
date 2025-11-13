package integration_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/allanpk716/to_icalendar/internal/clipboard"
	"github.com/allanpk716/to_icalendar/internal/config"
	"github.com/allanpk716/to_icalendar/internal/models"
	"github.com/allanpk716/to_icalendar/internal/processors"
	"github.com/allanpk716/to_icalendar/internal/validators"
)

func TestProcessingPipeline_FullTextWorkflow(t *testing.T) {
	// Setup test environment
	tempDir := t.TempDir()
	outputDir := filepath.Join(tempDir, "drafts")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatalf("Failed to create output directory: %v", err)
	}

	// Test text
	testText := "明天下午2点开会讨论项目进展，非常重要，请准时参加"

	// Step 1: Validate input
	validator := validators.NewContentValidator()
	textValidation := validator.ValidateText(testText)
	if !textValidation.IsValid {
		t.Fatalf("Text validation failed: %s", textValidation.Message)
	}

	// Step 2: Process text
	processor, err := processors.NewTextProcessor(nil)
	if err != nil {
		t.Fatalf("Failed to create text processor: %v", err)
	}

	analysis := processor.QuickAnalyze(testText)
	if !analysis.HasDate || !analysis.HasTime {
		t.Error("Expected text to contain date and time")
	}

	// Step 3: Parse task information
	parser := processors.NewTaskParser()
	taskInfo, err := parser.ParseFromText(testText)
	if err != nil {
		t.Fatalf("Failed to parse task: %v", err)
	}

	if taskInfo.Confidence < 0.5 {
		t.Errorf("Task confidence too low: %.2f", taskInfo.Confidence)
	}

	// Step 4: Generate JSON
	generator, err := processors.NewJSONGenerator(outputDir)
	if err != nil {
		t.Fatalf("Failed to create JSON generator: %v", err)
	}

	jsonFilePath, err := generator.GenerateFromParsedInfo(taskInfo)
	if err != nil {
		t.Fatalf("Failed to generate JSON: %v", err)
	}

	// Step 5: Verify output
	if _, err := os.Stat(jsonFilePath); os.IsNotExist(err) {
		t.Error("JSON file was not created")
	}

	t.Logf("Successfully processed text and generated JSON: %s", jsonFilePath)
}

func TestProcessingPipeline_ConfigLoadingToJSONGeneration(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "server.yaml")
	outputDir := filepath.Join(tempDir, "drafts")

	// Create test config
	testConfig := &models.ServerConfig{
		Dify: models.DifyConfig{
			APIEndpoint: "https://api.dify.ai/v1",
			APIKey:      "test-api-key-for-integration",
			Timeout:     30, // 添加超时时间
			Model:       "gpt-3.5-turbo",
		},
		MicrosoftTodo: models.MicrosoftTodoConfig{
			TenantID:     "test-tenant-id",
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
			Timezone:     "Asia/Shanghai",
		},
	}

	// Save config
	configManager := config.NewConfigManager()
	err := configManager.SaveServerConfig(configPath, testConfig)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Load config
	loadedConfig, err := configManager.LoadServerConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Validate loaded config
	if err := loadedConfig.Dify.Validate(); err != nil {
		t.Errorf("Loaded config validation failed: %v", err)
	}

	// Create test reminder
	reminder := &models.Reminder{
		Title:        "集成测试会议",
		Description:  "测试完整的处理流程",
		Date:         "2025-11-06",
		Time:         "15:00",
		RemindBefore: "30m",
		Priority:     models.PriorityHigh,
		List:         "测试",
	}

	// Generate JSON
	generator, err := processors.NewJSONGenerator(outputDir)
	if err != nil {
		t.Fatalf("Failed to create JSON generator: %v", err)
	}

	jsonFilePath, err := generator.GenerateFromReminder(reminder)
	if err != nil {
		t.Fatalf("Failed to generate JSON: %v", err)
	}

	// Verify file exists and content
	if _, err := os.Stat(jsonFilePath); os.IsNotExist(err) {
		t.Error("JSON file was not created")
	}

	t.Logf("Successfully completed config loading to JSON generation pipeline: %s", jsonFilePath)
}

func TestProcessingPipeline_ClipboardToJSONIntegration(t *testing.T) {
	// This test requires clipboard content, so we'll simulate it
	tempDir := t.TempDir()
	outputDir := filepath.Join(tempDir, "drafts")

	// Create clipboard manager
	manager, err := clipboard.NewManager()
	if err != nil {
		t.Fatalf("Failed to create clipboard manager: %v", err)
	}

	// Check if clipboard has content (optional test)
	hasContent, err := manager.HasContent()
	if err != nil {
		t.Fatalf("Failed to check clipboard content: %v", err)
	}

	if !hasContent {
		t.Skip("Clipboard is empty, skipping integration test")
	}

	// Read clipboard content
	content, err := manager.Read()
	if err != nil {
		t.Fatalf("Failed to read clipboard content: %v", err)
	}

	// Process based on content type
	if content.Type == models.ContentTypeText {
		// Validate content
		validator := validators.NewContentValidator()
		textValidation := validator.ValidateText(content.Text)
		if !textValidation.IsValid {
			t.Skipf("Clipboard text validation failed: %s", textValidation.Message)
		}

		// Process text
		processor, err := processors.NewTextProcessor(nil)
		if err != nil {
			t.Fatalf("Failed to create text processor: %v", err)
		}

		analysis := processor.QuickAnalyze(content.Text)
		parser := processors.NewTaskParser()
		taskInfo, err := parser.ParseFromText(content.Text)
		if err != nil {
			t.Fatalf("Failed to parse task from clipboard: %v", err)
		}

		if taskInfo.Confidence > 0.5 {
			// Generate JSON
			generator, err := processors.NewJSONGenerator(outputDir)
			if err != nil {
				t.Fatalf("Failed to create JSON generator: %v", err)
			}

			jsonFilePath, err := generator.GenerateFromParsedInfo(taskInfo)
			if err != nil {
				t.Fatalf("Failed to generate JSON from clipboard: %v", err)
			}

			// Verify output
			if _, err := os.Stat(jsonFilePath); os.IsNotExist(err) {
				t.Error("JSON file from clipboard was not created")
			}

			t.Logf("Successfully processed clipboard content: analysis=%v, confidence=%.2f, json=%s",
				analysis.HasDate && analysis.HasTime, taskInfo.Confidence, jsonFilePath)
		} else {
			t.Skipf("Clipboard content confidence too low: %.2f", taskInfo.Confidence)
		}
	} else {
		t.Skipf("Clipboard content type not supported for this test: %s", content.Type)
	}
}

func TestProcessingPipeline_ErrorHandlingWorkflow(t *testing.T) {
	// Test error handling throughout the pipeline
	tempDir := t.TempDir()

	// Test 1: Invalid text validation
	validator := validators.NewContentValidator()
	emptyValidation := validator.ValidateText("")
	if emptyValidation.IsValid {
		t.Error("Expected empty text validation to fail")
	}

	// Test 2: Invalid image validation
	invalidImageValidation := validator.ValidateImage([]byte{}, "test.png")
	if invalidImageValidation.IsValid {
		t.Error("Expected empty image validation to fail")
	}

	// Test 3: Invalid config handling
	configManager := config.NewConfigManager()
	_, err := configManager.LoadServerConfig("nonexistent.yaml")
	if err == nil {
		t.Error("Expected error when loading nonexistent config")
	}

	// Test 4: Invalid JSON generation
	outputDir := filepath.Join(tempDir, "drafts")
	generator, err := processors.NewJSONGenerator(outputDir)
	if err != nil {
		t.Fatalf("Failed to create JSON generator: %v", err)
	}

	invalidReminder := &models.Reminder{
		Title:        "", // Empty title should fail validation
		Description:  "Test description",
		Date:         "2025-11-06",
		Time:         "15:00",
		RemindBefore: "30m",
		Priority:     models.PriorityMedium,
		List:         "测试",
	}

	err = generator.ValidateReminder(invalidReminder)
	if err == nil {
		t.Error("Expected validation error for reminder with empty title")
	}

	t.Log("Error handling workflow tests completed successfully")
}

func TestProcessingPipeline_PerformanceIntegration(t *testing.T) {
	// Test performance of the complete pipeline
	tempDir := t.TempDir()
	outputDir := filepath.Join(tempDir, "drafts")

	testTexts := []string{
		"明天下午2点开会讨论项目进展",
		"今天下午必须完成重要报告，非常紧急",
		"下周三上午10点参加产品评审会议",
		"周五下午5点前提交周报",
		"明天早上9点团队晨会",
	}

	processor, err := processors.NewTextProcessor(nil)
	if err != nil {
		t.Fatalf("Failed to create text processor: %v", err)
	}

	parser := processors.NewTaskParser()
	generator, err := processors.NewJSONGenerator(outputDir)
	if err != nil {
		t.Fatalf("Failed to create JSON generator: %v", err)
	}

	startTime := time.Now()
	successCount := 0

	for _, text := range testTexts {
		// Process text
		analysis := processor.QuickAnalyze(text)
		taskInfo, err := parser.ParseFromText(text)
		if err != nil {
			t.Logf("Failed to parse text: %s, error: %v", text, err)
			continue
		}

		if taskInfo.Confidence > 0.5 {
			_, err := generator.GenerateFromParsedInfo(taskInfo)
			if err != nil {
				t.Logf("Failed to generate JSON for: %s, error: %v", text, err)
				continue
			}
			successCount++
		}

		t.Logf("Processed: %s -> hasDate=%v, hasTime=%v, confidence=%.2f",
			text[:min(len(text), 20)], analysis.HasDate, analysis.HasTime, taskInfo.Confidence)
	}

	totalTime := time.Since(startTime)
	avgTime := totalTime / time.Duration(len(testTexts))

	t.Logf("Performance integration test completed:")
	t.Logf("  Total texts processed: %d", len(testTexts))
	t.Logf("  Successful JSON generations: %d", successCount)
	t.Logf("  Total time: %v", totalTime)
	t.Logf("  Average time per text: %v", avgTime)

	if successCount == 0 {
		t.Error("No texts were successfully processed to JSON")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}