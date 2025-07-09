package pkg

import (
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/stretchr/testify/assert"
)

func TestNewConfigurationManager(t *testing.T) {
	logger := logging.NewTestLogger()
	manager := NewConfigurationManager(logger)
	
	assert.NotNil(t, manager)
	assert.Equal(t, logger, manager.logger)
}

func TestGetBoolWithPKLPriority(t *testing.T) {
	logger := logging.NewTestLogger()
	manager := NewConfigurationManager(logger)

	tests := []struct {
		name         string
		pklValue     *bool
		defaultValue bool
		expected     ConfigurationValue[bool]
	}{
		{
			name:         "PKL value provided",
			pklValue:     BoolPtr(true),
			defaultValue: false,
			expected:     ConfigurationValue[bool]{Value: true, Source: SourcePKL},
		},
		{
			name:         "PKL value nil, use default",
			pklValue:     nil,
			defaultValue: false,
			expected:     ConfigurationValue[bool]{Value: false, Source: SourceDefault},
		},
		{
			name:         "PKL value false, use PKL",
			pklValue:     BoolPtr(false),
			defaultValue: true,
			expected:     ConfigurationValue[bool]{Value: false, Source: SourcePKL},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := manager.GetBoolWithPKLPriority(tt.pklValue, tt.defaultValue, "test")
			assert.Equal(t, tt.expected.Value, result.Value)
			assert.Equal(t, tt.expected.Source, result.Source)
		})
	}
}

func TestGetStringWithPKLPriority(t *testing.T) {
	logger := logging.NewTestLogger()
	manager := NewConfigurationManager(logger)

	tests := []struct {
		name         string
		pklValue     *string
		defaultValue string
		expected     ConfigurationValue[string]
	}{
		{
			name:         "PKL value provided",
			pklValue:     StringPtr("pkl-value"),
			defaultValue: "default-value",
			expected:     ConfigurationValue[string]{Value: "pkl-value", Source: SourcePKL},
		},
		{
			name:         "PKL value nil, use default",
			pklValue:     nil,
			defaultValue: "default-value",
			expected:     ConfigurationValue[string]{Value: "default-value", Source: SourceDefault},
		},
		{
			name:         "PKL value empty string, use PKL",
			pklValue:     StringPtr(""),
			defaultValue: "default-value",
			expected:     ConfigurationValue[string]{Value: "", Source: SourcePKL},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := manager.GetStringWithPKLPriority(tt.pklValue, tt.defaultValue, "test")
			assert.Equal(t, tt.expected.Value, result.Value)
			assert.Equal(t, tt.expected.Source, result.Source)
		})
	}
}

func TestGetUint16WithPKLPriority(t *testing.T) {
	logger := logging.NewTestLogger()
	manager := NewConfigurationManager(logger)

	tests := []struct {
		name         string
		pklValue     *uint16
		defaultValue uint16
		expected     ConfigurationValue[uint16]
	}{
		{
			name:         "PKL value provided",
			pklValue:     Uint16Ptr(3000),
			defaultValue: 8080,
			expected:     ConfigurationValue[uint16]{Value: 3000, Source: SourcePKL},
		},
		{
			name:         "PKL value nil, use default",
			pklValue:     nil,
			defaultValue: 8080,
			expected:     ConfigurationValue[uint16]{Value: 8080, Source: SourceDefault},
		},
		{
			name:         "PKL value zero, use PKL",
			pklValue:     Uint16Ptr(0),
			defaultValue: 8080,
			expected:     ConfigurationValue[uint16]{Value: 0, Source: SourcePKL},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := manager.GetUint16WithPKLPriority(tt.pklValue, tt.defaultValue, "test")
			assert.Equal(t, tt.expected.Value, result.Value)
			assert.Equal(t, tt.expected.Source, result.Source)
		})
	}
}

func TestGetIntWithPKLPriority(t *testing.T) {
	logger := logging.NewTestLogger()
	manager := NewConfigurationManager(logger)

	tests := []struct {
		name         string
		pklValue     *int
		defaultValue int
		expected     ConfigurationValue[int]
	}{
		{
			name:         "PKL value provided",
			pklValue:     IntPtr(200),
			defaultValue: 100,
			expected:     ConfigurationValue[int]{Value: 200, Source: SourcePKL},
		},
		{
			name:         "PKL value nil, use default",
			pklValue:     nil,
			defaultValue: 100,
			expected:     ConfigurationValue[int]{Value: 100, Source: SourceDefault},
		},
		{
			name:         "PKL value zero, use PKL",
			pklValue:     IntPtr(0),
			defaultValue: 100,
			expected:     ConfigurationValue[int]{Value: 0, Source: SourcePKL},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := manager.GetIntWithPKLPriority(tt.pklValue, tt.defaultValue, "test")
			assert.Equal(t, tt.expected.Value, result.Value)
			assert.Equal(t, tt.expected.Source, result.Source)
		})
	}
}

func TestLogConfigurationSummary(t *testing.T) {
	logger := logging.NewTestLogger()
	manager := NewConfigurationManager(logger)

	configs := map[string]ConfigurationValue[any]{
		"setting1": {Value: "pkl-value", Source: SourcePKL},
		"setting2": {Value: "default-value", Source: SourceDefault},
		"setting3": {Value: true, Source: SourcePKL},
		"setting4": {Value: 100, Source: SourceDefault},
	}

	manager.LogConfigurationSummary(configs)
	
	// The function should not panic and should log the summary
	// We can't easily test the log output, but we can verify the function runs
	assert.NotNil(t, manager)
}

func TestConfigurationSourceConstants(t *testing.T) {
	assert.Equal(t, ConfigurationSource("PKL"), SourcePKL)
	assert.Equal(t, ConfigurationSource("DEFAULT"), SourceDefault)
}

func TestConfigurationValue(t *testing.T) {
	// Test that ConfigurationValue can hold different types
	boolValue := ConfigurationValue[bool]{Value: true, Source: SourcePKL}
	stringValue := ConfigurationValue[string]{Value: "test", Source: SourceDefault}
	intValue := ConfigurationValue[int]{Value: 42, Source: SourcePKL}

	assert.Equal(t, true, boolValue.Value)
	assert.Equal(t, SourcePKL, boolValue.Source)
	
	assert.Equal(t, "test", stringValue.Value)
	assert.Equal(t, SourceDefault, stringValue.Source)
	
	assert.Equal(t, 42, intValue.Value)
	assert.Equal(t, SourcePKL, intValue.Source)
} 