package docker

import (
	"net/http"
	"testing"
)

// TestValidateMethodNotAllowed verifies that validateMethod returns an error
// when an HTTP method that is not in the allowed list is provided.
func TestValidateMethodNotAllowed(t *testing.T) {
	req := &http.Request{Method: "POST"}

	if _, err := validateMethod(req, []string{"GET"}); err == nil {
		t.Fatalf("expected method not allowed error, got nil")
	}
}
