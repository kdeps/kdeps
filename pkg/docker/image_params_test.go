package docker

import (
	"strings"
	"testing"
)

func TestGenerateParamsSectionVariants(t *testing.T) {
	// Test case 1: Empty map
	result := generateParamsSection("ARG", map[string]string{})
	if result != "" {
		t.Errorf("Expected empty string for empty map, got: %s", result)
	}

	t.Log("generateParamsSection empty map test passed")

	// Test case 2: Map with single entry without value
	items := map[string]string{
		"DEBUG": "",
	}
	result = generateParamsSection("ENV", items)
	if result != "ENV DEBUG" {
		t.Errorf("Expected 'ENV DEBUG', got: %s", result)
	}

	t.Log("generateParamsSection single entry without value test passed")

	// Test case 3: Map with single entry with value
	items = map[string]string{
		"PATH": "/usr/local/bin",
	}
	result = generateParamsSection("ARG", items)
	if result != "ARG PATH=\"/usr/local/bin\"" {
		t.Errorf("Expected 'ARG PATH=\"/usr/local/bin\"', got: %s", result)
	}

	t.Log("generateParamsSection single entry with value test passed")

	// Test case 4: Map with multiple entries
	items = map[string]string{
		"VAR1": "value1",
		"VAR2": "",
		"VAR3": "value3",
	}
	result = generateParamsSection("ENV", items)
	// The order of map iteration is not guaranteed, so check individual lines
	lines := strings.Split(result, "\n")
	lineSet := make(map[string]struct{})
	for _, l := range lines {
		lineSet[l] = struct{}{}
	}
	expectedLines := []string{"ENV VAR1=\"value1\"", "ENV VAR2", "ENV VAR3=\"value3\""}
	for _, el := range expectedLines {
		if _, ok := lineSet[el]; !ok {
			t.Errorf("Expected line '%s' not found in output: %s", el, result)
		}
	}

	t.Log("generateParamsSection multiple entries test passed")
}
