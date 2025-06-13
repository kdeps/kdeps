package resolver

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestValidationFunctions_EmptyAllowedLists(t *testing.T) {
	dr := newValidationTestResolver()

	fileContent := `request.params("id")\nrequest.header("Auth")`

	// Empty allowed slices should permit everything (return nil)
	if err := dr.validateRequestParams(fileContent, nil); err != nil {
		t.Fatalf("validateRequestParams unexpected error: %v", err)
	}
	if err := dr.validateRequestHeaders(fileContent, nil); err != nil {
		t.Fatalf("validateRequestHeaders unexpected error: %v", err)
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("PATCH", "/any/path", nil)

	if err := dr.validateRequestPath(c, nil); err != nil {
		t.Fatalf("validateRequestPath unexpected error: %v", err)
	}
	if err := dr.validateRequestMethod(c, nil); err != nil {
		t.Fatalf("validateRequestMethod unexpected error: %v", err)
	}
}
