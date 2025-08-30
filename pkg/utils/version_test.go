package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name     string
		v1       string
		v2       string
		expected int
		hasError bool
	}{
		{"equal versions", "1.2.3", "1.2.3", 0, false},
		{"v1 greater major", "2.0.0", "1.9.9", 1, false},
		{"v1 less major", "1.0.0", "2.0.0", -1, false},
		{"v1 greater minor", "1.2.0", "1.1.9", 1, false},
		{"v1 less minor", "1.1.0", "1.2.0", -1, false},
		{"v1 greater patch", "1.2.3", "1.2.2", 1, false},
		{"v1 less patch", "1.2.2", "1.2.3", -1, false},
		{"invalid v1", "1.2", "1.2.3", 0, true},
		{"invalid v2", "1.2.3", "1.2", 0, true},
		{"non-numeric v1", "1.a.3", "1.2.3", 0, true},
		{"non-numeric v2", "1.2.3", "1.b.3", 0, true},
		{"negative version", "-1.2.3", "1.2.3", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CompareVersions(tt.v1, tt.v2)

			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestValidateSchemaVersion(t *testing.T) {
	tests := []struct {
		name           string
		version        string
		minimumVersion string
		hasError       bool
	}{
		{"valid version above minimum", "0.3.0", "0.3.0", false},
		{"valid version equal to minimum", "0.3.0", "0.3.0", false},
		{"valid higher version", "1.0.0", "0.3.0", false},
		{"below minimum", "0.1.9", "0.3.0", true},
		{"empty version", "", "0.3.0", true},
		{"invalid format", "1.2", "0.3.0", true},
		{"non-numeric", "1.a.3", "0.3.0", true},
		{"negative version", "-1.0.0", "0.3.0", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSchemaVersion(tt.version, tt.minimumVersion)

			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIsSchemaVersionSupported(t *testing.T) {
	tests := []struct {
		name           string
		version        string
		minimumVersion string
		supported      bool
	}{
		{"version above minimum", "0.3.0", "0.3.0", true},
		{"minimum version", "0.3.0", "0.3.0", true},
		{"higher version", "1.0.0", "0.3.0", true},
		{"below minimum", "0.1.9", "0.3.0", false},
		{"empty version", "", "0.3.0", false},
		{"invalid format", "1.2", "0.3.0", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsSchemaVersionSupported(tt.version, tt.minimumVersion)
			assert.Equal(t, tt.supported, result)
		})
	}
}
