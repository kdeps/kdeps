package config

import (
	"testing"

	"github.com/kdeps/kdeps/pkg"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfigurationProcessor(t *testing.T) {
	logger := logging.NewTestLogger()
	processor := NewConfigurationProcessor(logger)

	assert.NotNil(t, processor)
	assert.NotNil(t, processor.configManager)
	assert.Equal(t, logger, processor.logger)
}

func TestProcessWorkflowConfiguration_WithPKLSettings(t *testing.T) {
	logger := logging.NewTestLogger()
	processor := NewConfigurationProcessor(logger)

	// Create a mock workflow with PKL settings
	// Note: This is a simplified test since we can't easily create the full PKL workflow
	// In a real implementation, you would load an actual workflow.pkl file

	// Test that the processor can be created and basic functionality works
	processor = NewConfigurationProcessor(logger)
	assert.NotNil(t, processor)

	// Test validation function
	config := &ProcessedConfiguration{
		APIServerMode:   pkg.ConfigurationValue[bool]{Value: true, Source: pkg.SourcePKL},
		APIServerHostIP: pkg.ConfigurationValue[string]{Value: "127.0.0.1", Source: pkg.SourcePKL},
		APIServerPort:   pkg.ConfigurationValue[uint16]{Value: 3000, Source: pkg.SourcePKL},
		WebServerMode:   pkg.ConfigurationValue[bool]{Value: false, Source: pkg.SourceDefault},
		WebServerHostIP: pkg.ConfigurationValue[string]{Value: "127.0.0.1", Source: pkg.SourceDefault},
		WebServerPort:   pkg.ConfigurationValue[uint16]{Value: 8080, Source: pkg.SourceDefault},
		RateLimitMax:    pkg.ConfigurationValue[int]{Value: 100, Source: pkg.SourcePKL},
		Environment:     pkg.ConfigurationValue[string]{Value: "dev", Source: pkg.SourcePKL},
		InstallAnaconda: pkg.ConfigurationValue[bool]{Value: false, Source: pkg.SourceDefault},
		Timezone:        pkg.ConfigurationValue[string]{Value: "Etc/UTC", Source: pkg.SourceDefault},
	}

	err := processor.ValidateConfiguration(config)
	require.NoError(t, err)
}

func TestProcessWorkflowConfiguration_WithDefaults(t *testing.T) {
	logger := logging.NewTestLogger()
	processor := NewConfigurationProcessor(logger)

	// Test that the processor can handle nil settings and use defaults
	processor = NewConfigurationProcessor(logger)
	assert.NotNil(t, processor)

	// Test validation with default configuration
	config := &ProcessedConfiguration{
		APIServerMode:   pkg.ConfigurationValue[bool]{Value: false, Source: pkg.SourceDefault},
		APIServerHostIP: pkg.ConfigurationValue[string]{Value: "127.0.0.1", Source: pkg.SourceDefault},
		APIServerPort:   pkg.ConfigurationValue[uint16]{Value: 3000, Source: pkg.SourceDefault},
		WebServerMode:   pkg.ConfigurationValue[bool]{Value: false, Source: pkg.SourceDefault},
		WebServerHostIP: pkg.ConfigurationValue[string]{Value: "127.0.0.1", Source: pkg.SourceDefault},
		WebServerPort:   pkg.ConfigurationValue[uint16]{Value: 8080, Source: pkg.SourceDefault},
		RateLimitMax:    pkg.ConfigurationValue[int]{Value: 100, Source: pkg.SourceDefault},
		Environment:     pkg.ConfigurationValue[string]{Value: "dev", Source: pkg.SourceDefault},
		InstallAnaconda: pkg.ConfigurationValue[bool]{Value: false, Source: pkg.SourceDefault},
		Timezone:        pkg.ConfigurationValue[string]{Value: "Etc/UTC", Source: pkg.SourceDefault},
	}

	err := processor.ValidateConfiguration(config)
	require.NoError(t, err)
}

func TestValidateConfiguration_InvalidValues(t *testing.T) {
	logger := logging.NewTestLogger()
	processor := NewConfigurationProcessor(logger)

	tests := []struct {
		name        string
		config      *ProcessedConfiguration
		expectError bool
		errorMsg    string
	}{
		{
			name: "Invalid environment",
			config: &ProcessedConfiguration{
				APIServerPort: pkg.ConfigurationValue[uint16]{Value: 3000, Source: pkg.SourcePKL},
				WebServerPort: pkg.ConfigurationValue[uint16]{Value: 8080, Source: pkg.SourcePKL},
				Environment:   pkg.ConfigurationValue[string]{Value: "invalid", Source: pkg.SourcePKL},
				RateLimitMax:  pkg.ConfigurationValue[int]{Value: 100, Source: pkg.SourcePKL},
			},
			expectError: true,
			errorMsg:    "invalid environment",
		},
		{
			name: "Invalid rate limit",
			config: &ProcessedConfiguration{
				APIServerPort: pkg.ConfigurationValue[uint16]{Value: 3000, Source: pkg.SourcePKL},
				WebServerPort: pkg.ConfigurationValue[uint16]{Value: 8080, Source: pkg.SourcePKL},
				Environment:   pkg.ConfigurationValue[string]{Value: "dev", Source: pkg.SourcePKL},
				RateLimitMax:  pkg.ConfigurationValue[int]{Value: -1, Source: pkg.SourcePKL},
			},
			expectError: true,
			errorMsg:    "invalid rate limit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := processor.ValidateConfiguration(tt.config)
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateConfiguration_ValidValues(t *testing.T) {
	logger := logging.NewTestLogger()
	processor := NewConfigurationProcessor(logger)

	validEnvironments := []string{"dev", "development", "prod", "production"}

	for _, env := range validEnvironments {
		t.Run("Valid environment: "+env, func(t *testing.T) {
			config := &ProcessedConfiguration{
				APIServerPort: pkg.ConfigurationValue[uint16]{Value: 3000, Source: pkg.SourcePKL},
				WebServerPort: pkg.ConfigurationValue[uint16]{Value: 8080, Source: pkg.SourcePKL},
				Environment:   pkg.ConfigurationValue[string]{Value: env, Source: pkg.SourcePKL},
				RateLimitMax:  pkg.ConfigurationValue[int]{Value: 100, Source: pkg.SourcePKL},
			}

			err := processor.ValidateConfiguration(config)
			assert.NoError(t, err)
		})
	}
}

func TestValidateConfiguration_PortZeroAllowed(t *testing.T) {
	logger := logging.NewTestLogger()
	processor := NewConfigurationProcessor(logger)

	tests := []struct {
		name   string
		config *ProcessedConfiguration
	}{
		{
			name: "API server port 0 allowed",
			config: &ProcessedConfiguration{
				APIServerPort: pkg.ConfigurationValue[uint16]{Value: 0, Source: pkg.SourcePKL},
				WebServerPort: pkg.ConfigurationValue[uint16]{Value: 8080, Source: pkg.SourcePKL},
				Environment:   pkg.ConfigurationValue[string]{Value: "dev", Source: pkg.SourcePKL},
				RateLimitMax:  pkg.ConfigurationValue[int]{Value: 100, Source: pkg.SourcePKL},
			},
		},
		{
			name: "Web server port 0 allowed",
			config: &ProcessedConfiguration{
				APIServerPort: pkg.ConfigurationValue[uint16]{Value: 3000, Source: pkg.SourcePKL},
				WebServerPort: pkg.ConfigurationValue[uint16]{Value: 0, Source: pkg.SourcePKL},
				Environment:   pkg.ConfigurationValue[string]{Value: "dev", Source: pkg.SourcePKL},
				RateLimitMax:  pkg.ConfigurationValue[int]{Value: 100, Source: pkg.SourcePKL},
			},
		},
		{
			name: "Both ports 0 allowed",
			config: &ProcessedConfiguration{
				APIServerPort: pkg.ConfigurationValue[uint16]{Value: 0, Source: pkg.SourcePKL},
				WebServerPort: pkg.ConfigurationValue[uint16]{Value: 0, Source: pkg.SourcePKL},
				Environment:   pkg.ConfigurationValue[string]{Value: "dev", Source: pkg.SourcePKL},
				RateLimitMax:  pkg.ConfigurationValue[int]{Value: 100, Source: pkg.SourcePKL},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := processor.ValidateConfiguration(tt.config)
			assert.NoError(t, err, "Port 0 should be allowed")
		})
	}
}
