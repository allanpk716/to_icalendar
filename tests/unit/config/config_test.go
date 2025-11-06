package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/allanpk716/to_icalendar/internal/config"
	"github.com/allanpk716/to_icalendar/internal/models"
)

func TestConfigManager_LoadServerConfig(t *testing.T) {
	tests := []struct {
		name          string
		configPath    string
		expectError   bool
		expectedError string
	}{
		{
			name:        "valid config file",
			configPath:  "../../../testdata/config_valid.yaml",
			expectError: false,
		},
		{
			name:          "nonexistent config file",
			configPath:    "../../../testdata/nonexistent.yaml",
			expectError:   true,
			expectedError: "no such file or directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configManager := config.NewConfigManager()

			serverConfig, err := configManager.LoadServerConfig(tt.configPath)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if tt.expectedError != "" && err.Error() != tt.expectedError {
					t.Errorf("Expected error containing %q, got %q", tt.expectedError, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if serverConfig == nil {
				t.Error("Expected server config but got nil")
				return
			}

			// Validate that config structure is loaded
			if serverConfig.Dify.APIEndpoint == "" {
				t.Error("Expected API endpoint to be loaded")
			}
		})
	}
}

func TestConfigManager_SaveServerConfig(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_config.yaml")

	configManager := config.NewConfigManager()

	// Create test config
	testConfig := &models.ServerConfig{
		Dify: models.DifyConfig{
			APIEndpoint: "https://api.dify.ai/v1",
			APIKey:      "test-api-key",
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
	err := configManager.SaveServerConfig(configPath, testConfig)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}

	// Load and verify
	loadedConfig, err := configManager.LoadServerConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load saved config: %v", err)
	}

	if loadedConfig.Dify.APIEndpoint != testConfig.Dify.APIEndpoint {
		t.Errorf("Expected API endpoint %q, got %q", testConfig.Dify.APIEndpoint, loadedConfig.Dify.APIEndpoint)
	}

	if loadedConfig.MicrosoftTodo.TenantID != testConfig.MicrosoftTodo.TenantID {
		t.Errorf("Expected tenant ID %q, got %q", testConfig.MicrosoftTodo.TenantID, loadedConfig.MicrosoftTodo.TenantID)
	}
}

func TestDifyConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      models.DifyConfig
		expectError bool
	}{
		{
			name: "valid config",
			config: models.DifyConfig{
				APIEndpoint: "https://api.dify.ai/v1",
				APIKey:      "valid-api-key",
				Model:       "gpt-3.5-turbo",
			},
			expectError: false,
		},
		{
			name: "empty API endpoint",
			config: models.DifyConfig{
				APIEndpoint: "",
				APIKey:      "valid-api-key",
				Model:       "gpt-3.5-turbo",
			},
			expectError: true,
		},
		{
			name: "empty API key",
			config: models.DifyConfig{
				APIEndpoint: "https://api.dify.ai/v1",
				APIKey:      "",
				Model:       "gpt-3.5-turbo",
			},
			expectError: true,
		},
		{
			name: "invalid API endpoint format",
			config: models.DifyConfig{
				APIEndpoint: "invalid-url",
				APIKey:      "valid-api-key",
				Model:       "gpt-3.5-turbo",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError && err == nil {
				t.Error("Expected validation error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected validation error: %v", err)
			}
		})
	}
}