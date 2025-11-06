package config_test

import (
	"strings"
	"testing"

	"github.com/allanpk716/to_icalendar/internal/config"
	"github.com/allanpk716/to_icalendar/internal/models"
)

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func TestConfigManager_LoadServerConfig(t *testing.T) {
	tests := []struct {
		name          string
		configPath    string
		expectError   bool
		expectedError string
	}{
		{
			name:        "valid config file",
			configPath:  "../../../tests/testdata/config_valid.yaml",
			expectError: false,
		},
		{
			name:          "nonexistent config file",
			configPath:    "../../../tests/testdata/nonexistent.yaml",
			expectError:   true,
			expectedError: "server config file not found",
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
				if tt.expectedError != "" && !contains(err.Error(), tt.expectedError) {
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
	t.Skip("SaveServerConfig method not implemented yet")
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
				MaxTokens:   1000,
				Timeout:     30,
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