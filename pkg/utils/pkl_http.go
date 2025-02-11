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
			encodedValue := EncodeBase64String(strings.TrimSpace(value))
			headersLines = append(headersLines, fmt.Sprintf(`["%s"] = "%s"`, name, encodedValue))
		}
	}

	return "headers {\n" + strings.Join(headersLines, "\n") + "\n}"
}

// FormatRequestParams formats the query parameters into a string representation for inclusion in the .pkl file.
func FormatRequestParams(params map[string][]string) string {
	var paramsLines []string
	for param, values := range params {
		for _, value := range values {
			encodedValue := EncodeBase64String(strings.TrimSpace(value))
			paramsLines = append(paramsLines, fmt.Sprintf(`["%s"] = "%s"`, param, encodedValue))
		}
	}
	return "params {\n" + strings.Join(paramsLines, "\n") + "\n}"
}

// FormatResponseHeaders formats the HTTP headers into a string representation for inclusion in the .pkl file.
func FormatResponseHeaders(headers map[string]string) string {
	headersLines := make([]string, 0, len(headers))

	for name, value := range headers {
		headersLines = append(headersLines, fmt.Sprintf(`["%s"] = "%s"`, name, strings.TrimSpace(value)))
	}

	return "headers {\n" + strings.Join(headersLines, "\n") + "\n}"
}

// FormatResponseProperties formats the HTTP properties into a string representation for inclusion in the .pkl file.
func FormatResponseProperties(properties map[string]string) string {
	propertiesLines := make([]string, 0, len(properties))

	for name, value := range properties {
		propertiesLines = append(propertiesLines, fmt.Sprintf(`["%s"] = "%s"`, name, strings.TrimSpace(value)))
	}

	return "properties {\n" + strings.Join(propertiesLines, "\n") + "\n}"
}
