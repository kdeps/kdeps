package pkg_test

import (
	"testing"

	"github.com/kdeps/kdeps/pkg"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/stretchr/testify/assert"
)

func TestNewConfigurationManager(t *testing.T) {
	// Test creating a new configuration manager
	manager := pkg.NewConfigurationManager(nil)
	assert.NotNil(t, manager)
}

func TestGetBoolWithPKLPriority(t *testing.T) {
	logger := logging.NewTestLogger()
	manager := pkg.NewConfigurationManager(logger)

	tests := []struct {
		name         string
		pklValue     *bool
		defaultValue bool
		expected     pkg.ConfigurationValue[bool]
	}{
		{
			name:         "PKL value provided",
			pklValue:     pkg.BoolPtr(true),
			defaultValue: false,
			expected:     pkg.ConfigurationValue[bool]{Value: true, Source: pkg.SourcePKL},
		},
		{
			name:         "PKL value nil, use default",
			pklValue:     nil,
			defaultValue: false,
			expected:     pkg.ConfigurationValue[bool]{Value: false, Source: pkg.SourceDefault},
		},
		{
			name:         "PKL value false, use PKL",
			pklValue:     pkg.BoolPtr(false),
			defaultValue: true,
			expected:     pkg.ConfigurationValue[bool]{Value: false, Source: pkg.SourcePKL},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := manager.GetBoolWithPKLPriority(tt.pklValue, tt.defaultValue, tt.name)
			assert.Equal(t, tt.expected.Value, result.Value)
			assert.Equal(t, tt.expected.Source, result.Source)
		})
	}
}

func TestGetStringWithPKLPriority(t *testing.T) {
	logger := logging.NewTestLogger()
	manager := pkg.NewConfigurationManager(logger)

	tests := []struct {
		name         string
		pklValue     *string
		defaultValue string
		expected     pkg.ConfigurationValue[string]
	}{
		{
			name:         "PKL value provided",
			pklValue:     pkg.StringPtr("pkl-value"),
			defaultValue: "default-value",
			expected:     pkg.ConfigurationValue[string]{Value: "pkl-value", Source: pkg.SourcePKL},
		},
		{
			name:         "PKL value nil, use default",
			pklValue:     nil,
			defaultValue: "default-value",
			expected:     pkg.ConfigurationValue[string]{Value: "default-value", Source: pkg.SourceDefault},
		},
		{
			name:         "PKL value empty string, use PKL",
			pklValue:     pkg.StringPtr(""),
			defaultValue: "default-value",
			expected:     pkg.ConfigurationValue[string]{Value: "", Source: pkg.SourcePKL},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := manager.GetStringWithPKLPriority(tt.pklValue, tt.defaultValue, tt.name)
			assert.Equal(t, tt.expected.Value, result.Value)
			assert.Equal(t, tt.expected.Source, result.Source)
		})
	}
}

func TestGetUint16WithPKLPriority(t *testing.T) {
	logger := logging.NewTestLogger()
	manager := pkg.NewConfigurationManager(logger)

	tests := []struct {
		name         string
		pklValue     *uint16
		defaultValue uint16
		expected     pkg.ConfigurationValue[uint16]
	}{
		{
			name:         "PKL value provided",
			pklValue:     pkg.Uint16Ptr(3000),
			defaultValue: 8080,
			expected:     pkg.ConfigurationValue[uint16]{Value: 3000, Source: pkg.SourcePKL},
		},
		{
			name:         "PKL value nil, use default",
			pklValue:     nil,
			defaultValue: 8080,
			expected:     pkg.ConfigurationValue[uint16]{Value: 8080, Source: pkg.SourceDefault},
		},
		{
			name:         "PKL value zero, use PKL",
			pklValue:     pkg.Uint16Ptr(0),
			defaultValue: 8080,
			expected:     pkg.ConfigurationValue[uint16]{Value: 0, Source: pkg.SourcePKL},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := manager.GetUint16WithPKLPriority(tt.pklValue, tt.defaultValue, tt.name)
			assert.Equal(t, tt.expected.Value, result.Value)
			assert.Equal(t, tt.expected.Source, result.Source)
		})
	}
}

func TestGetIntWithPKLPriority(t *testing.T) {
	logger := logging.NewTestLogger()
	manager := pkg.NewConfigurationManager(logger)

	tests := []struct {
		name         string
		pklValue     *int
		defaultValue int
		expected     pkg.ConfigurationValue[int]
	}{
		{
			name:         "PKL value provided",
			pklValue:     pkg.IntPtr(200),
			defaultValue: 100,
			expected:     pkg.ConfigurationValue[int]{Value: 200, Source: pkg.SourcePKL},
		},
		{
			name:         "PKL value nil, use default",
			pklValue:     nil,
			defaultValue: 100,
			expected:     pkg.ConfigurationValue[int]{Value: 100, Source: pkg.SourceDefault},
		},
		{
			name:         "PKL value zero, use PKL",
			pklValue:     pkg.IntPtr(0),
			defaultValue: 100,
			expected:     pkg.ConfigurationValue[int]{Value: 0, Source: pkg.SourcePKL},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := manager.GetIntWithPKLPriority(tt.pklValue, tt.defaultValue, tt.name)
			assert.Equal(t, tt.expected.Value, result.Value)
			assert.Equal(t, tt.expected.Source, result.Source)
		})
	}
}

func TestLogConfigurationSummary(_ *testing.T) {
	logger := logging.NewTestLogger()
	manager := pkg.NewConfigurationManager(logger)

	configs := map[string]pkg.ConfigurationValue[any]{
		"setting1": {Value: "pkl-value", Source: pkg.SourcePKL},
		"setting2": {Value: "default-value", Source: pkg.SourceDefault},
		"setting3": {Value: true, Source: pkg.SourcePKL},
		"setting4": {Value: 100, Source: pkg.SourceDefault},
	}

	// This should not panic
	manager.LogConfigurationSummary(configs)
}

func TestConfigurationSourceConstants(t *testing.T) {
	assert.Equal(t, pkg.SourcePKL, pkg.ConfigurationSource("PKL"))
	assert.Equal(t, pkg.SourceDefault, pkg.ConfigurationSource("DEFAULT"))
}

func TestConfigurationValue(t *testing.T) {
	// Test ConfigurationValue struct
	configValue := pkg.ConfigurationValue[string]{
		Value:  "test",
		Source: pkg.SourcePKL,
	}

	assert.Equal(t, "test", configValue.Value)
	assert.Equal(t, pkg.SourcePKL, configValue.Source)
}

func TestBoolPtr(t *testing.T) {
	// Test BoolPtr function
	boolPtr := pkg.BoolPtr(true)
	assert.NotNil(t, boolPtr)
	assert.True(t, *boolPtr)
}

func TestConfigurationValueWithSource(t *testing.T) {
	// Test ConfigurationValue with different sources
	configValue := pkg.ConfigurationValue[string]{
		Value:  "default_value",
		Source: pkg.SourceDefault,
	}

	assert.Equal(t, "default_value", configValue.Value)
	assert.Equal(t, pkg.SourceDefault, configValue.Source)
}
