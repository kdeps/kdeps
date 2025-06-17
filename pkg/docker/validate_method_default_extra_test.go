package docker

import (
	"net/http"
	"testing"
)

// TestValidateMethodDefaultGET verifies that when the incoming request has an
// empty Method field validateMethod substitutes "GET" and returns the correct
// formatted string without error.
func TestValidateMethodDefaultGET(t *testing.T) {
	req := &http.Request{}

	got, err := validateMethod(req, []string{"GET"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := `method = "GET"`
	if got != want {
		t.Fatalf("unexpected result: got %q want %q", got, want)
	}
}
