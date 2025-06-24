package utils

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseNonInteractive(t *testing.T) {
	tests := []struct {
		name                string
		envValue            string
		expectedInteractive bool
		expectedAnswer      string
	}{
		// Interactive mode cases
		{"Empty value", "", false, ""},
		{"Explicit false", "false", false, ""},
		{"Explicit no", "no", false, ""},
		{"Zero value", "0", false, ""},
		{"Case insensitive false", "FALSE", false, ""},
		{"Case insensitive no", "NO", false, ""},

		// Non-interactive mode with default behavior
		{"Traditional 1", "1", true, ""},
		{"Explicit true", "true", true, ""},
		{"Explicit yes", "yes", true, ""},
		{"Case insensitive true", "TRUE", true, ""},
		{"Case insensitive yes", "YES", true, ""},

		// Non-interactive mode with predefined answers
		{"Answer y", "y", true, "y"},
		{"Answer Y", "Y", true, "y"},
		{"Answer n", "n", true, "n"},
		{"Answer N", "N", true, "n"},

		// Non-interactive mode with custom answers
		{"Custom answer", "my-agent", true, "my-agent"},
		{"Custom answer with spaces", "custom agent", true, "custom agent"},
		{"Numeric answer", "42", true, "42"},

		// Edge cases
		{"Whitespace around value", "  y  ", true, "y"},
		{"Mixed case custom", "MyAgent", true, "myagent"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original value
			originalValue := os.Getenv("NON_INTERACTIVE")
			defer os.Setenv("NON_INTERACTIVE", originalValue)

			// Set test value
			os.Setenv("NON_INTERACTIVE", tt.envValue)

			config := ParseNonInteractive()
			assert.Equal(t, tt.expectedInteractive, config.IsNonInteractive,
				"IsNonInteractive mismatch for value: %s", tt.envValue)
			assert.Equal(t, tt.expectedAnswer, config.PredefinedAnswer,
				"PredefinedAnswer mismatch for value: %s", tt.envValue)
		})
	}
}

func TestIsNonInteractive(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected bool
	}{
		{"Interactive mode", "", false},
		{"Interactive mode false", "false", false},
		{"Non-interactive mode", "1", true},
		{"Non-interactive mode true", "true", true},
		{"Non-interactive mode y", "y", true},
		{"Non-interactive mode custom", "custom-answer", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original value
			originalValue := os.Getenv("NON_INTERACTIVE")
			defer os.Setenv("NON_INTERACTIVE", originalValue)

			// Set test value
			os.Setenv("NON_INTERACTIVE", tt.envValue)

			result := IsNonInteractive()
			assert.Equal(t, tt.expected, result,
				"IsNonInteractive mismatch for value: %s", tt.envValue)
		})
	}
}

func TestParseNonInteractive_UnsetEnvironment(t *testing.T) {
	// Save original value
	originalValue := os.Getenv("NON_INTERACTIVE")
	defer func() {
		if originalValue != "" {
			os.Setenv("NON_INTERACTIVE", originalValue)
		} else {
			os.Unsetenv("NON_INTERACTIVE")
		}
	}()

	// Unset the environment variable
	os.Unsetenv("NON_INTERACTIVE")

	config := ParseNonInteractive()
	assert.False(t, config.IsNonInteractive, "Should be interactive when env var is unset")
	assert.Empty(t, config.PredefinedAnswer, "Should have empty answer when env var is unset")
}

// TestNonInteractiveUsageExamples demonstrates comprehensive usage of the new NON_INTERACTIVE functionality
func TestNonInteractiveUsageExamples(t *testing.T) {
	// Save original value
	originalValue := os.Getenv("NON_INTERACTIVE")
	defer os.Setenv("NON_INTERACTIVE", originalValue)

	examples := []struct {
		name           string
		envValue       string
		description    string
		expectedMode   bool
		expectedAnswer string
	}{
		// Interactive modes
		{
			name:           "Default Interactive",
			envValue:       "",
			description:    "Default behavior - interactive mode",
			expectedMode:   false,
			expectedAnswer: "",
		},
		{
			name:           "Explicit Interactive",
			envValue:       "false",
			description:    "Explicitly set to interactive mode",
			expectedMode:   false,
			expectedAnswer: "",
		},

		// Traditional non-interactive modes
		{
			name:           "Traditional Non-Interactive",
			envValue:       "1",
			description:    "Traditional NON_INTERACTIVE=1 mode",
			expectedMode:   true,
			expectedAnswer: "",
		},
		{
			name:           "Boolean True",
			envValue:       "true",
			description:    "Boolean-style non-interactive mode",
			expectedMode:   true,
			expectedAnswer: "",
		},

		// Predefined answer modes
		{
			name:           "Yes Answer",
			envValue:       "y",
			description:    "Non-interactive with 'y' answer for prompts",
			expectedMode:   true,
			expectedAnswer: "y",
		},
		{
			name:           "No Answer",
			envValue:       "n",
			description:    "Non-interactive with 'n' answer for prompts",
			expectedMode:   true,
			expectedAnswer: "n",
		},
		{
			name:           "Custom Agent Name",
			envValue:       "my-awesome-agent",
			description:    "Non-interactive with custom agent name",
			expectedMode:   true,
			expectedAnswer: "my-awesome-agent",
		},
		{
			name:           "Numeric Value",
			envValue:       "42",
			description:    "Non-interactive with numeric answer",
			expectedMode:   true,
			expectedAnswer: "42",
		},
	}

	for _, example := range examples {
		t.Run(example.name, func(t *testing.T) {
			t.Logf("Example: %s", example.description)
			t.Logf("Setting NON_INTERACTIVE=%q", example.envValue)

			// Set the environment variable
			os.Setenv("NON_INTERACTIVE", example.envValue)

			// Test the parsing
			config := ParseNonInteractive()
			assert.Equal(t, example.expectedMode, config.IsNonInteractive,
				"IsNonInteractive mismatch for %s", example.description)
			assert.Equal(t, example.expectedAnswer, config.PredefinedAnswer,
				"PredefinedAnswer mismatch for %s", example.description)

			// Test the convenience function
			isNonInteractive := IsNonInteractive()
			assert.Equal(t, example.expectedMode, isNonInteractive,
				"IsNonInteractive() mismatch for %s", example.description)

			t.Logf("✓ Result: IsNonInteractive=%t, PredefinedAnswer=%q",
				config.IsNonInteractive, config.PredefinedAnswer)
		})
	}
}
