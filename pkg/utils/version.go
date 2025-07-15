package utils

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// CompareVersions compares two semantic version strings.
// Returns:
//
//	-1 if v1 < v2
//	 0 if v1 == v2
//	 1 if v1 > v2
//	error if either version is invalid
func CompareVersions(v1, v2 string) (int, error) {
	v1Parts, err := parseVersion(v1)
	if err != nil {
		return 0, fmt.Errorf("invalid version v1 '%s': %w", v1, err)
	}

	v2Parts, err := parseVersion(v2)
	if err != nil {
		return 0, fmt.Errorf("invalid version v2 '%s': %w", v2, err)
	}

	for i := 0; i < 3; i++ {
		if v1Parts[i] < v2Parts[i] {
			return -1, nil
		}
		if v1Parts[i] > v2Parts[i] {
			return 1, nil
		}
	}

	return 0, nil
}

// parseVersion parses a semantic version string (e.g., "1.2.3") into parts
func parseVersion(version string) ([3]int, error) {
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return [3]int{}, fmt.Errorf("version must have exactly 3 parts (major.minor.patch), got %d parts", len(parts))
	}

	var result [3]int
	for i, part := range parts {
		num, err := strconv.Atoi(part)
		if err != nil {
			return [3]int{}, fmt.Errorf("invalid version part '%s': must be a number", part)
		}
		if num < 0 {
			return [3]int{}, fmt.Errorf("version part cannot be negative: %d", num)
		}
		result[i] = num
	}

	return result, nil
}

// ValidateSchemaVersion validates that a schema version meets minimum requirements
func ValidateSchemaVersion(version string, minimumVersion string) error {
	if version == "" {
		return errors.New("schema version cannot be empty")
	}

	cmp, err := CompareVersions(version, minimumVersion)
	if err != nil {
		return fmt.Errorf("invalid schema version format: %w", err)
	}

	if cmp < 0 {
		return fmt.Errorf("schema version %s is below minimum supported version %s. Run 'kdeps upgrade' to update your schema versions", version, minimumVersion)
	}

	return nil
}

// IsSchemaVersionSupported checks if a schema version is supported (>= minimum)
func IsSchemaVersionSupported(version string, minimumVersion string) bool {
	return ValidateSchemaVersion(version, minimumVersion) == nil
}
