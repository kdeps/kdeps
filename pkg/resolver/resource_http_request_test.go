package resolver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/apple/pkl-go/pkl"
	"github.com/kdeps/kdeps/pkg/logging"
	pklHTTP "github.com/kdeps/schema/gen/http"
	"github.com/spf13/afero"
)

func TestDoRequest_GET(t *testing.T) {
	// Spin up a lightweight HTTP server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("q") != "test" {
			t.Errorf("missing query param")
		}
		w.Header().Set("X-Custom", "val")
		_, _ = w.Write([]byte("hello"))
	}))
	defer srv.Close()

	client := &pklHTTP.ResourceHTTPClient{
		Method: "GET",
		Url:    srv.URL,
		Params: &map[string]string{"q": "test"},
	}

	dr := &DependencyResolver{
		Fs:      afero.NewMemMapFs(),
		Context: context.Background(),
		Logger:  logging.GetLogger(),
	}

	if err := dr.DoRequest(client); err != nil {
		t.Fatalf("DoRequest returned error: %v", err)
	}
	if client.Response == nil || client.Response.Body == nil {
		t.Fatalf("response body not set")
	}
	if *client.Response.Body != "hello" {
		t.Errorf("unexpected response body: %s", *client.Response.Body)
	}
	if (*client.Response.Headers)["X-Custom"] != "val" {
		t.Errorf("header missing: %v", client.Response.Headers)
	}
	if client.Timestamp == nil || client.Timestamp.Unit != pkl.Nanosecond {
		t.Errorf("timestamp not set properly: %+v", client.Timestamp)
	}
}
