package processors

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/allanpk716/to_icalendar/internal/models"
)

// JSONGenerator handles generation of reminder JSON files
type JSONGenerator struct {
	outputDir    string
	filePrefix   string
	autoSave     bool
	indentOutput bool
}

// NewJSONGenerator creates a new JSON generator
func NewJSONGenerator(outputDir string) (*JSONGenerator, error) {
	// 确保输出目录存在
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	return &JSONGenerator{
		outputDir:    outputDir,
		filePrefix:   "reminder",
		autoSave:     true,
		indentOutput: true,
	}, nil
}

// GenerateFromResult generates JSON file from processing result
func (jg *JSONGenerator) GenerateFromResult(result *models.ProcessingResult) (string, error) {
	if result == nil {
		return "", fmt.Errorf("processing result is nil")
	}

	if !result.Success {
		return "", fmt.Errorf("processing failed: %s", result.ErrorMessage)
	}

	if result.Reminder == nil {
		return "", fmt.Errorf("no reminder data in result")
	}

	return jg.GenerateFromReminder(result.Reminder)
}

// GenerateFromReminder generates JSON file from reminder data
func (jg *JSONGenerator) GenerateFromReminder(reminder *models.Reminder) (string, error) {
	if reminder == nil {
		return "", fmt.Errorf("reminder is nil")
	}

	log.Printf("生成JSON文件，标题: %s", reminder.Title)

	// 生成文件名
	fileName := jg.generateFileName(reminder)
	filePath := filepath.Join(jg.outputDir, fileName)

	// 生成JSON数据
	jsonData, err := jg.generateJSON(reminder)
	if err != nil {
		return "", fmt.Errorf("failed to generate JSON: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(filePath, jsonData, 0644); err != nil {
		return "", fmt.Errorf("failed to write JSON file: %w", err)
	}

	log.Printf("JSON文件已生成: %s", filePath)

	return filePath, nil
}

// GenerateBatch generates multiple JSON files from reminders
func (jg *JSONGenerator) GenerateBatch(reminders []*models.Reminder) ([]string, error) {
	if len(reminders) == 0 {
		return []string{}, nil
	}

	log.Printf("批量生成 %d 个JSON文件", len(reminders))

	filePaths := make([]string, 0, len(reminders))
	errors := make([]error, 0)

	for i, reminder := range reminders {
		filePath, err := jg.GenerateFromReminder(reminder)
		if err != nil {
			log.Printf("生成第 %d 个JSON文件失败: %v", i+1, err)
			errors = append(errors, err)
			continue
		}
		filePaths = append(filePaths, filePath)
	}

	if len(errors) > 0 {
		log.Printf("批量生成完成，成功: %d, 失败: %d", len(filePaths), len(errors))
		return filePaths, fmt.Errorf("some files failed to generate: %v", errors)
	}

	log.Printf("批量生成完成，全部成功: %d", len(filePaths))
	return filePaths, nil
}

// GenerateFromParsedInfo generates JSON from parsed task info
func (jg *JSONGenerator) GenerateFromParsedInfo(parsedInfo *models.ParsedTaskInfo) (string, error) {
	if parsedInfo == nil {
		return "", fmt.Errorf("parsed info is nil")
	}

	// 创建提醒事项
	reminder := &models.Reminder{
		Title:        parsedInfo.Title,
		Description:  parsedInfo.Description,
		Date:         parsedInfo.Date,
		Time:         parsedInfo.Time,
		RemindBefore: parsedInfo.RemindBefore,
		List:         parsedInfo.List,
	}

	// 设置优先级
	switch parsedInfo.Priority {
	case "high":
		reminder.Priority = models.PriorityHigh
	case "low":
		reminder.Priority = models.PriorityLow
	default:
		reminder.Priority = models.PriorityMedium
	}

	return jg.GenerateFromReminder(reminder)
}

// generateFileName generates a unique filename for the reminder
func (jg *JSONGenerator) generateFileName(reminder *models.Reminder) string {
	timestamp := time.Now().Format("20060102_150405")

	// 创建安全的文件名（移除特殊字符）
	safeTitle := sanitizeFileName(reminder.Title)
	if safeTitle == "" {
		safeTitle = "untitled"
	}

	// 限制文件名长度
	if len(safeTitle) > 20 {
		safeTitle = safeTitle[:20]
	}

	fileName := fmt.Sprintf("%s_%s_%s.json", jg.filePrefix, safeTitle, timestamp)
	return fileName
}

// generateJSON generates JSON data from reminder
func (jg *JSONGenerator) generateJSON(reminder *models.Reminder) ([]byte, error) {
	var jsonData []byte
	var err error

	if jg.indentOutput {
		jsonData, err = json.MarshalIndent(reminder, "", "  ")
	} else {
		jsonData, err = json.Marshal(reminder)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to marshal reminder to JSON: %w", err)
	}

	return jsonData, nil
}

// ValidateReminder validates reminder data before generation
func (jg *JSONGenerator) ValidateReminder(reminder *models.Reminder) error {
	if reminder == nil {
		return fmt.Errorf("reminder is nil")
	}

	if reminder.Title == "" {
		return fmt.Errorf("reminder title is required")
	}

	if reminder.Date == "" {
		return fmt.Errorf("reminder date is required")
	}

	if reminder.Time == "" {
		return fmt.Errorf("reminder time is required")
	}

	// 验证日期格式
	if !isValidDateFormat(reminder.Date) {
		return fmt.Errorf("invalid date format: %s (expected YYYY-MM-DD)", reminder.Date)
	}

	// 验证时间格式
	if !isValidTimeFormat(reminder.Time) {
		return fmt.Errorf("invalid time format: %s (expected HH:MM)", reminder.Time)
	}

	return nil
}

// GenerateWithMetadata generates JSON with additional metadata
func (jg *JSONGenerator) GenerateWithMetadata(reminder *models.Reminder, metadata map[string]interface{}) (string, error) {
	// 创建包含元数据的结构
	type ReminderWithMetadata struct {
		*models.Reminder
		Metadata map[string]interface{} `json:"metadata,omitempty"`
		GeneratedAt string `json:"generated_at"`
		Version string `json:"version"`
	}

	reminderWithMeta := &ReminderWithMetadata{
		Reminder:    reminder,
		Metadata:    metadata,
		GeneratedAt: time.Now().Format(time.RFC3339),
		Version:     "1.0",
	}

	// 生成文件名
	fileName := fmt.Sprintf("%s_with_metadata_%s.json", jg.filePrefix, time.Now().Format("20060102_150405"))
	filePath := filepath.Join(jg.outputDir, fileName)

	// 生成JSON数据
	var jsonData []byte
	var err error

	if jg.indentOutput {
		jsonData, err = json.MarshalIndent(reminderWithMeta, "", "  ")
	} else {
		jsonData, err = json.Marshal(reminderWithMeta)
	}

	if err != nil {
		return "", fmt.Errorf("failed to generate JSON with metadata: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(filePath, jsonData, 0644); err != nil {
		return "", fmt.Errorf("failed to write JSON file with metadata: %w", err)
	}

	log.Printf("带元数据的JSON文件已生成: %s", filePath)

	return filePath, nil
}

// GetOutputDir returns the current output directory
func (jg *JSONGenerator) GetOutputDir() string {
	return jg.outputDir
}

// SetOutputDir sets a new output directory
func (jg *JSONGenerator) SetOutputDir(outputDir string) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	jg.outputDir = outputDir
	return nil
}

// SetFilePrefix sets the file prefix for generated files
func (jg *JSONGenerator) SetFilePrefix(prefix string) {
	jg.filePrefix = prefix
}

// SetAutoSave enables or disables auto-save
func (jg *JSONGenerator) SetAutoSave(autoSave bool) {
	jg.autoSave = autoSave
}

// SetIndentOutput enables or disables indented JSON output
func (jg *JSONGenerator) SetIndentOutput(indentOutput bool) {
	jg.indentOutput = indentOutput
}

// ListGeneratedFiles lists all generated JSON files
func (jg *JSONGenerator) ListGeneratedFiles() ([]string, error) {
	files, err := os.ReadDir(jg.outputDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read output directory: %w", err)
	}

	jsonFiles := make([]string, 0)
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".json" {
			jsonFiles = append(jsonFiles, filepath.Join(jg.outputDir, file.Name()))
		}
	}

	return jsonFiles, nil
}

// CleanupOldFiles removes JSON files older than specified duration
func (jg *JSONGenerator) CleanupOldFiles(maxAge time.Duration) error {
	files, err := jg.ListGeneratedFiles()
	if err != nil {
		return err
	}

	cutoff := time.Now().Add(-maxAge)
	removedCount := 0

	for _, filePath := range files {
		info, err := os.Stat(filePath)
		if err != nil {
			log.Printf("无法获取文件信息: %s, 错误: %v", filePath, err)
			continue
		}

		if info.ModTime().Before(cutoff) {
			if err := os.Remove(filePath); err != nil {
				log.Printf("删除旧文件失败: %s, 错误: %v", filePath, err)
			} else {
				log.Printf("已删除旧文件: %s", filePath)
				removedCount++
			}
		}
	}

	log.Printf("清理完成，删除了 %d 个旧文件", removedCount)
	return nil
}

// GetGeneratorInfo returns generator information
func (jg *JSONGenerator) GetGeneratorInfo() map[string]interface{} {
	return map[string]interface{}{
		"output_dir":    jg.outputDir,
		"file_prefix":   jg.filePrefix,
		"auto_save":     jg.autoSave,
		"indent_output": jg.indentOutput,
		"version":       "1.0.0",
	}
}

// sanitizeFileName removes or replaces unsafe characters from filename
func sanitizeFileName(name string) string {
	// 定义不安全的字符
	unsafeChars := []string{
		"<", ">", ":", `"`, "/", "\\", "|", "?", "*",
		" ", "\t", "\n", "\r", "\x00",
	}

	safeName := name
	for _, char := range unsafeChars {
		safeName = strings.ReplaceAll(safeName, char, "_")
	}

	// 移除连续的下划线
	underscorePattern := regexp.MustCompile(`_+`)
	safeName = underscorePattern.ReplaceAllString(safeName, "_")

	// 移除开头和结尾的下划线
	safeName = strings.Trim(safeName, "_")

	return safeName
}

// isValidDateFormat checks if date string is in YYYY-MM-DD format
func isValidDateFormat(dateStr string) bool {
	if len(dateStr) != 10 {
		return false
	}

	if dateStr[4] != '-' || dateStr[7] != '-' {
		return false
	}

	// 验证数字格式
	for i, char := range dateStr {
		if i == 4 || i == 7 {
			continue
		}
		if char < '0' || char > '9' {
			return false
		}
	}

	return true
}

// isValidTimeFormat checks if time string is in HH:MM format and has valid values
func isValidTimeFormat(timeStr string) bool {
	if len(timeStr) != 5 {
		return false
	}

	if timeStr[2] != ':' {
		return false
	}

	// 验证小时
	hours := timeStr[:2]
	for _, char := range hours {
		if char < '0' || char > '9' {
			return false
		}
	}

	// 验证分钟
	minutes := timeStr[3:]
	for _, char := range minutes {
		if char < '0' || char > '9' {
			return false
		}
	}

	// 验证时间值的有效性
	hourValue := (timeStr[0]-'0')*10 + (timeStr[1]-'0')
	minuteValue := (timeStr[3]-'0')*10 + (timeStr[4]-'0')

	// 检查小时范围（0-23）
	if hourValue < 0 || hourValue > 23 {
		return false
	}

	// 检查分钟范围（0-59）
	if minuteValue < 0 || minuteValue > 59 {
		return false
	}

	return true
}