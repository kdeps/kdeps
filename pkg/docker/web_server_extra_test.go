package docker

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/resolver"
	webserver "github.com/kdeps/schema/gen/web_server"
	"github.com/kdeps/schema/gen/web_server/webservertype"
	"github.com/spf13/afero"
)

func TestHandleAppRequest_Misconfiguration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	route := &webserver.WebServerRoutes{
		Path:       "/app",
		PublicPath: "app",
		ServerType: webservertype.App,
		AppPort:    func() *uint16 { v := uint16(3000); return &v }(),
	}

	dr := &resolver.DependencyResolver{
		Logger:  logging.NewTestLogger(),
		Fs:      afero.NewMemMapFs(),
		DataDir: "/tmp",
	}

	// hostIP is empty -> should trigger error branch and return 500
	handler := handleAppRequestWrapper("", route, dr.Logger)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/app", nil)

	handler(c)

	if w.Code != 500 {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

// helper to expose handleAppRequest (unexported) via closure
func handleAppRequestWrapper(hostIP string, route *webserver.WebServerRoutes, logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		handleAppRequest(c, hostIP, route, logger)
	}
}
