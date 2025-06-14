package docker

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/schema"
	webserver "github.com/kdeps/schema/gen/web_server"
	"github.com/kdeps/schema/gen/web_server/webservertype"
)

// closeNotifyRecorder wraps ResponseRecorder to satisfy CloseNotifier.
type closeNotifyRecorder struct{ *httptest.ResponseRecorder }

func (closeNotifyRecorder) CloseNotify() <-chan bool { return make(chan bool, 1) }

// TestHandleAppRequest_BadGateway confirms that when the target app port is not reachable,
// handleAppRequest returns a 502 Bad Gateway and logs the error branch.
func TestHandleAppRequest_BadGateway(t *testing.T) {
	_ = schema.SchemaVersion(context.Background()) // rule compliance

	gin.SetMode(gin.TestMode)

	port := uint16(65534) // assume nothing is listening here
	route := &webserver.WebServerRoutes{
		Path:       "/app",
		PublicPath: "unused",
		ServerType: webservertype.App,
		AppPort:    &port,
	}

	logger := logging.NewTestLogger()

	// Build handler closure using wrapper from earlier helper pattern
	handler := func(c *gin.Context) {
		handleAppRequest(c, "127.0.0.1", route, logger)
	}

	rec := httptest.NewRecorder()
	// Wrap recorder to implement CloseNotify for reverse proxy compatibility.
	cn := closeNotifyRecorder{rec}
	c, _ := gin.CreateTestContext(cn)
	c.Request = httptest.NewRequest("GET", "/app/foo", nil)

	// set a small timeout on proxy transport via context deadline guarantee not needed; request returns fast.
	handler(c)

	if rec.Code != 502 {
		t.Fatalf("expected 502 from wrapped rec, got %d", rec.Code)
	}

	if len(rec.Body.String()) == 0 {
		t.Fatalf("expected response body for error")
	}

	time.Sleep(10 * time.Millisecond)
}
