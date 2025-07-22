package resolver_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/resolver"
)

// Helper to create a DependencyResolver with only a logger (FS not needed for these pure funcs).
func newValidationTestResolver() *resolver.DependencyResolver {
	return &resolver.DependencyResolver{
		Logger: logging.NewTestLogger(),
	}
}

func TestValidateRequestParams(t *testing.T) {
	dr := newValidationTestResolver()
	fileContent := `request.params("id")\nrequest.params("page")`

	// Allowed case
	if err := dr.ValidateRequestParams(fileContent, []string{"id", "page"}); err != nil {
		t.Errorf("unexpected error for allowed params: %v", err)
	}
	// Disallowed case
	if err := dr.ValidateRequestParams(fileContent, []string{"id"}); err == nil {
		t.Errorf("expected error for disallowed param, got nil")
	}
}

func TestValidateRequestHeaders(t *testing.T) {
	dr := newValidationTestResolver()
	fileContent := `request.header("Authorization")\nrequest.header("X-Custom")`

	if err := dr.ValidateRequestHeaders(fileContent, []string{"Authorization", "X-Custom"}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if err := dr.ValidateRequestHeaders(fileContent, []string{"Authorization"}); err == nil {
		t.Errorf("expected error for header not allowed")
	}
}

func TestValidateRequestPathAndMethod(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dr := newValidationTestResolver()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/resource", nil)

	// Path allowed
	if err := dr.ValidateRequestPath(c, []string{"/api/resource", "/foo"}); err != nil {
		t.Errorf("unexpected path error: %v", err)
	}
	// Path not allowed
	if err := dr.ValidateRequestPath(c, []string{"/foo"}); err == nil {
		t.Errorf("expected path validation error, got nil")
	}

	// Method allowed
	if err := dr.ValidateRequestMethod(c, []string{"GET", "POST"}); err != nil {
		t.Errorf("unexpected method error: %v", err)
	}
	// Method not allowed
	if err := dr.ValidateRequestMethod(c, []string{"POST"}); err == nil {
		t.Errorf("expected method validation error, got nil")
	}
}

func TestValidationFunctions_EmptyAllowedLists(t *testing.T) {
	dr := newValidationTestResolver()

	fileContent := `request.params("id")\nrequest.header("Auth")`

	// Empty allowed slices should permit everything (return nil)
	if err := dr.ValidateRequestParams(fileContent, nil); err != nil {
		t.Fatalf("validateRequestParams unexpected error: %v", err)
	}
	if err := dr.ValidateRequestHeaders(fileContent, nil); err != nil {
		t.Fatalf("validateRequestHeaders unexpected error: %v", err)
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPatch, "/any/path", nil)

	if err := dr.ValidateRequestPath(c, nil); err != nil {
		t.Fatalf("validateRequestPath unexpected error: %v", err)
	}
	if err := dr.ValidateRequestMethod(c, nil); err != nil {
		t.Fatalf("validateRequestMethod unexpected error: %v", err)
	}
}
