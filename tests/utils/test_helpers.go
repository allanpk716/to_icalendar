package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/allanpk716/to_icalendar/internal/config"
	"github.com/allanpk716/to_icalendar/internal/models"
)

// TestEnvironment provides a managed test environment with temporary files and directories
type TestEnvironment struct {
	T          *testing.T
	TempDir    string
	ConfigPath string
	OutputDir  string
	Config     *models.ServerConfig
	Cleanup    func()
}

// SetupTestEnvironment creates a temporary test environment with default configuration
func SetupTestEnvironment(t *testing.T) *TestEnvironment {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "server.yaml")
	outputDir := filepath.Join(tempDir, "drafts")

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatalf("Failed to create output directory: %v", err)
	}

	// Create default test configuration
	testConfig := &models.ServerConfig{
		Dify: models.DifyConfig{
			APIEndpoint: "https://api.dify.ai/v1",
			APIKey:      "test-api-key-for-testing",
			Model:       "gpt-3.5-turbo",
		},
		MicrosoftTodo: models.MicrosoftTodoConfig{
			TenantID:     "test-tenant-id",
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
			Timezone:     "Asia/Shanghai",
		},
	}

	// Save configuration
	configManager := config.NewConfigManager()
	if err := configManager.SaveServerConfig(configPath, testConfig); err != nil {
		t.Fatalf("Failed to save test configuration: %v", err)
	}

	return &TestEnvironment{
		T:          t,
		TempDir:    tempDir,
		ConfigPath: configPath,
		OutputDir:  outputDir,
		Config:     testConfig,
		Cleanup:    func() { /* temp dir will be cleaned up automatically */ },
	}
}

// SetupEmptyTestEnvironment creates a test environment without any default files
func SetupEmptyTestEnvironment(t *testing.T) *TestEnvironment {
	tempDir := t.TempDir()
	outputDir := filepath.Join(tempDir, "drafts")

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatalf("Failed to create output directory: %v", err)
	}

	return &TestEnvironment{
		T:          t,
		TempDir:    tempDir,
		OutputDir:  outputDir,
		ConfigPath: filepath.Join(tempDir, "server.yaml"),
		Cleanup:    func() { /* temp dir will be cleaned up automatically */ },
	}
}

// LoadConfig loads the configuration from the test environment
func (te *TestEnvironment) LoadConfig() *models.ServerConfig {
	if te.Config == nil {
		configManager := config.NewConfigManager()
		config, err := configManager.LoadServerConfig(te.ConfigPath)
		if err != nil {
			te.T.Fatalf("Failed to load configuration: %v", err)
		}
		te.Config = config
	}
	return te.Config
}

// UpdateConfig updates the configuration in the test environment
func (te *TestEnvironment) UpdateConfig(newConfig *models.ServerConfig) {
	te.Config = newConfig
	configManager := config.NewConfigManager()
	if err := configManager.SaveServerConfig(te.ConfigPath, newConfig); err != nil {
		te.T.Fatalf("Failed to update configuration: %v", err)
	}
}

// CreateTempFile creates a temporary file with the given content
func (te *TestEnvironment) CreateTempFile(name, content string) string {
	filePath := filepath.Join(te.TempDir, name)
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		te.T.Fatalf("Failed to create temporary file: %v", err)
	}
	return filePath
}

// AssertFileExists checks if a file exists and reports an error if it doesn't
func (te *TestEnvironment) AssertFileExists(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		te.T.Errorf("Expected file to exist: %s", path)
	}
}

// AssertFileNotExists checks if a file doesn't exist and reports an error if it does
func (te *TestEnvironment) AssertFileNotExists(path string) {
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		te.T.Errorf("Expected file not to exist: %s", path)
	}
}

// GetTempPath returns a temporary path for the given filename
func (te *TestEnvironment) GetTempPath(filename string) string {
	return filepath.Join(te.TempDir, filename)
}

// GetOutputPath returns an output path for the given filename
func (te *TestEnvironment) GetOutputPath(filename string) string {
	return filepath.Join(te.OutputDir, filename)
}