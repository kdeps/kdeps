package docker

import (
	"strings"
	"testing"
)

func TestGenerateParamsSection_Extra(t *testing.T) {
	input := map[string]string{"USER": "root", "DEBUG": ""}
	got := generateParamsSection("ENV", input)

	// The slice order is not guaranteed; ensure both expected lines exist.
	if !(containsLine(got, `ENV USER="root"`) && containsLine(got, `ENV DEBUG`)) {
		t.Fatalf("unexpected section: %s", got)
	}
}

// helper to search line in multi-line string.
func containsLine(s, line string) bool {
	for _, l := range strings.Split(s, "\n") {
		if l == line {
			return true
		}
	}
	return false
}
