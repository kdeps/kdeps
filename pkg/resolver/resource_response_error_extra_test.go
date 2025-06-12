package resolver

import (
	"testing"
)

func TestHandleAPIErrorResponse_Extra(t *testing.T) {
	// Case 1: APIServerMode disabled – function should just relay fatal and return nil error
	dr := &DependencyResolver{APIServerMode: false}
	fatalRet, err := dr.HandleAPIErrorResponse(400, "bad", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !fatalRet {
		t.Errorf("expected fatal=true to passthrough when APIServerMode off")
	}

	// NOTE: paths where APIServerMode==true are exercised in resource_response_test.go; we only
	// verify the non-API path here to avoid external PKL dependencies.
}
