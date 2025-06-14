package docker

import (
	"context"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kdeps/kdeps/pkg/resolver"
	"github.com/kdeps/kdeps/pkg/schema"
	webserver "github.com/kdeps/schema/gen/web_server"
	"github.com/kdeps/schema/gen/web_server/webservertype"
	"github.com/spf13/afero"
)

// TestHandleStaticRequest serves a real file via the unexported static handler and
// verifies we get a 200 and the expected payload. Uses OsFs + tmp dir per guidelines.
func TestHandleStaticRequest_Static(t *testing.T) {
	// Reference schema version (project rule)
	_ = schema.SchemaVersion(context.Background())

	gin.SetMode(gin.TestMode)

	fs := afero.NewOsFs()
	tempDir := t.TempDir()

	// Create data/public directory and file
	dataDir := filepath.Join(tempDir, "data")
	publicDir := filepath.Join(dataDir, "public")
	if err := fs.MkdirAll(publicDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	content := []byte("hello-static")
	filePath := filepath.Join(publicDir, "index.txt")
	if err := afero.WriteFile(fs, filePath, content, 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	// Build route definition
	route := &webserver.WebServerRoutes{
		Path:       "/static",
		PublicPath: "public",
		ServerType: webservertype.Static,
	}

	// Prepare gin context
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("GET", "/static/index.txt", nil)

	// Invoke static handler directly
	handleStaticRequest(ctx, filepath.Join(dataDir, route.PublicPath), route)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if body := w.Body.String(); body != string(content) {
		t.Fatalf("unexpected body: %s", body)
	}

	_ = resolver.DependencyResolver{}
}
