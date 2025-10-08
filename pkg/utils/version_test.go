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
		{"valid version above minimum", "0.4.0-dev", "0.4.0-dev", false},
		{"valid version equal to minimum", "0.4.0-dev", "0.4.0-dev", false},
		{"valid higher version", "1.0.0", "0.4.0-dev", false},
		{"below minimum", "0.1.9", "0.4.0-dev", true},
		{"empty version", "", "0.4.0-dev", true},
		{"invalid format", "1.2", "0.4.0-dev", true},
		{"non-numeric", "1.a.3", "0.4.0-dev", true},
		{"negative version", "-1.0.0", "0.4.0-dev", true},
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
		{"version above minimum", "0.4.0-dev", "0.4.0-dev", true},
		{"minimum version", "0.4.0-dev", "0.4.0-dev", true},
		{"higher version", "1.0.0", "0.4.0-dev", true},
		{"below minimum", "0.1.9", "0.4.0-dev", false},
		{"empty version", "", "0.4.0-dev", false},
		{"invalid format", "1.2", "0.4.0-dev", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsSchemaVersionSupported(tt.version, tt.minimumVersion)
			assert.Equal(t, tt.supported, result)
		})
	}
}

func TestParseVersionWithSuffixes(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected [3]int
		hasError bool
	}{
		{"basic version", "1.2.3", [3]int{1, 2, 3}, false},
		{"hyphen suffix", "1.2.3-dev", [3]int{1, 2, 3}, false},
		{"plus suffix", "1.2.3+build", [3]int{1, 2, 3}, false},
		{"complex suffix with hyphen and plus", "1.2.3-alpha+1000", [3]int{1, 2, 3}, false},
		{"complex suffix with plus and hyphen", "1.2.3+build-time", [3]int{1, 2, 3}, false},
		{"hyphen in patch only", "1.2.3-dev", [3]int{1, 2, 3}, false},
		{"plus in patch only", "1.2.3+123", [3]int{1, 2, 3}, false},
		{"complex suffix starting with plus", "1.2.3+build-123-final", [3]int{1, 2, 3}, false},
		{"complex suffix starting with hyphen", "1.2.3-alpha+build+123", [3]int{1, 2, 3}, false},
		{"multiple hyphens and plus", "1.2.3-rc1+build-final", [3]int{1, 2, 3}, false},
		{"zero version with suffix", "0.0.0-dev", [3]int{0, 0, 0}, false},
		{"large numbers with suffix", "10.20.30-beta+build", [3]int{10, 20, 30}, false},
		{"invalid - empty after suffix", "1.2.-dev", [3]int{}, true},
		{"invalid - non-numeric before suffix", "1.a.3-dev", [3]int{}, true},
		{"invalid - too few parts", "1.2", [3]int{}, true},
		{"invalid - too many parts", "1.2.3.4", [3]int{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseVersion(tt.version)

			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestCompareVersionsWithSuffixes(t *testing.T) {
	tests := []struct {
		name     string
		v1       string
		v2       string
		expected int
		hasError bool
	}{
		{"both with hyphens equal", "1.2.3-dev", "1.2.3-prod", 0, false},
		{"both with plus equal", "1.2.3+build", "1.2.3+final", 0, false},
		{"mixed suffixes equal", "1.2.3-dev", "1.2.3+build", 0, false},
		{"complex suffixes equal", "1.2.3-alpha+1000", "1.2.3+build-time", 0, false},
		{"suffix vs no suffix equal", "1.2.3-dev", "1.2.3", 0, false},
		{"v1 greater with suffix", "2.0.0-dev", "1.9.9+build", 1, false},
		{"v1 less with suffix", "1.0.0+build", "2.0.0-dev", -1, false},
		{"patch comparison with suffixes", "1.2.4-dev", "1.2.3+build", 1, false},
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
