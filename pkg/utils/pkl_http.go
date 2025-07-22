package utils

import (
	"fmt"
	"strings"
)

// FormatRequestHeaders formats the HTTP headers into a string representation for inclusion in the .pkl file.
func FormatRequestHeaders(headers map[string][]string) string {
	var headersLines []string
	for name, values := range headers {
		for _, value := range values {
			headersLines = append(headersLines, fmt.Sprintf(`["%s"] = "%s"`, name, strings.TrimSpace(value)))
		}
	}

	return "Headers {\n" + strings.Join(headersLines, "\n") + "\n}"
}

// FormatRequestParams formats the query parameters into a string representation for inclusion in the .pkl file.
func FormatRequestParams(params map[string][]string) string {
	var paramsLines []string
	for param, values := range params {
		for _, value := range values {
			trimmedValue := strings.TrimSpace(value)
			paramsLines = append(paramsLines, fmt.Sprintf(`["%s"] = "%s"`, param, trimmedValue))
		}
	}
	return "Params {\n" + strings.Join(paramsLines, "\n") + "\n}"
}

// isSimpleString checks if a string is simple enough to not need Base64 encoding
func isSimpleString(s string) bool {
	// Check if the string contains only alphanumeric characters, spaces, and common punctuation
	for _, r := range s {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') ||
			r == ' ' || r == '-' || r == '_' || r == '.' || r == ',' || r == '!' || r == '?' ||
			r == '(' || r == ')' || r == '[' || r == ']' || r == '{' || r == '}' ||
			r == '@' || r == '#' || r == '$' || r == '%' || r == '^' || r == '&' || r == '*' ||
			r == '+' || r == '=' || r == '|' || r == '\\' || r == '/' || r == ':' || r == ';' ||
			r == '<' || r == '>' || r == '"' || r == '\'' || r == '`' || r == '~') {
			return false
		}
	}
	return true
}

// FormatResponseHeaders formats the HTTP headers into a string representation for inclusion in the .pkl file.
func FormatResponseHeaders(headers map[string]string) string {
	headersLines := make([]string, 0, len(headers))

	for name, value := range headers {
		headersLines = append(headersLines, fmt.Sprintf(`["%s"] = "%s"`, name, strings.TrimSpace(value)))
	}

	return "Headers {\n" + strings.Join(headersLines, "\n") + "\n}"
}

// FormatResponseProperties formats the HTTP properties into a string representation for inclusion in the .pkl file.
func FormatResponseProperties(properties map[string]string) string {
	propertiesLines := make([]string, 0, len(properties))

	for name, value := range properties {
		propertiesLines = append(propertiesLines, fmt.Sprintf(`["%s"] = "%s"`, name, strings.TrimSpace(value)))
	}

	return "Properties {\n" + strings.Join(propertiesLines, "\n") + "\n}"
}
