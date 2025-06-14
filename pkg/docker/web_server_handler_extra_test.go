package docker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/resolver"
	"github.com/kdeps/kdeps/pkg/schema"
	webserver "github.com/kdeps/schema/gen/web_server"
	"github.com/kdeps/schema/gen/web_server/webservertype"
	"github.com/spf13/afero"
)

// TestWebServerHandler_Static verifies that static file serving works via the
// returned gin.HandlerFunc.
func TestWebServerHandler_Static(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fs := afero.NewOsFs()
	dataDir := t.TempDir()
	publicPath := "public"
	if err := fs.MkdirAll(filepath.Join(dataDir, publicPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// create a simple file to be served.
	if err := afero.WriteFile(fs, filepath.Join(dataDir, publicPath, "hello.txt"), []byte("world"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	dr := &resolver.DependencyResolver{
		Fs:      fs,
		Logger:  logging.NewTestLogger(),
		DataDir: dataDir,
	}

	route := &webserver.WebServerRoutes{
		Path:       "/public",
		PublicPath: publicPath,
		ServerType: webservertype.Static,
	}

	handler := WebServerHandler(context.Background(), "", route, dr)

	req := httptest.NewRequest(http.MethodGet, "/public/hello.txt", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	handler(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if body := w.Body.String(); body != "world" {
		t.Fatalf("unexpected body: %s", body)
	}

	_ = schema.SchemaVersion(context.Background())
}

// TestWebServerHandler_AppError checks that missing host triggers HTTP 500.
func TestWebServerHandler_AppError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fs := afero.NewOsFs()
	dr := &resolver.DependencyResolver{
		Fs:      fs,
		Logger:  logging.NewTestLogger(),
		DataDir: "/data",
	}

	var port uint16 = 1234
	route := &webserver.WebServerRoutes{
		Path:       "/proxy",
		ServerType: webservertype.App,
		AppPort:    &port,
	}

	handler := WebServerHandler(context.Background(), "", route, dr)

	req := httptest.NewRequest(http.MethodGet, "/proxy/x", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	handler(c)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}

	_ = schema.SchemaVersion(context.Background())
}
