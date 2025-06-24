package utils

import (
	"os"
	"strings"
)

// NonInteractiveConfig represents the configuration for non-interactive mode
type NonInteractiveConfig struct {
	IsNonInteractive bool
	PredefinedAnswer string // "y" or "n" for yes/no prompts, or custom value for other prompts
}

// ParseNonInteractive parses the NON_INTERACTIVE environment variable
//
// Supported values:
// - Interactive mode: "", "0", "false", "no" (case insensitive)
// - Non-interactive with default behavior: "1", "true", "yes" (case insensitive)
// - Non-interactive with predefined answers:
//   - "y" = answer "y" to prompts
//   - "n" = answer "n" to prompts
//   - Any other value = use that value as the answer to prompts
//
// Examples:
//
//	NON_INTERACTIVE=""           -> Interactive mode
//	NON_INTERACTIVE="false"      -> Interactive mode
//	NON_INTERACTIVE="1"          -> Non-interactive, default behavior
//	NON_INTERACTIVE="true"       -> Non-interactive, default behavior
//	NON_INTERACTIVE="y"          -> Non-interactive, answer "y" to prompts
//	NON_INTERACTIVE="n"          -> Non-interactive, answer "n" to prompts
//	NON_INTERACTIVE="my-agent"   -> Non-interactive, use "my-agent" as answer
func ParseNonInteractive() NonInteractiveConfig {
	value := strings.ToLower(strings.TrimSpace(os.Getenv("NON_INTERACTIVE")))

	switch value {
	case "", "0", "false", "no":
		return NonInteractiveConfig{IsNonInteractive: false}
	case "1", "true", "yes":
		return NonInteractiveConfig{IsNonInteractive: true}
	case "y":
		return NonInteractiveConfig{IsNonInteractive: true, PredefinedAnswer: "y"}
	case "n":
		return NonInteractiveConfig{IsNonInteractive: true, PredefinedAnswer: "n"}
	default:
		// Any other value is treated as non-interactive with that value as the answer
		return NonInteractiveConfig{IsNonInteractive: true, PredefinedAnswer: value}
	}
}

// IsNonInteractive returns true if we're in non-interactive mode
// This is a convenience function for when you only need to know the mode,
// not the predefined answer.
func IsNonInteractive() bool {
	return ParseNonInteractive().IsNonInteractive
}
