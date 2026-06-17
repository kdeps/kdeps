package http

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

func TestBuildProxiedPath_StripPrefix(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "/resource", buildProxiedPath("/api", "/api/resource"))
}

func TestBuildProxiedPath_RootRoute(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "/foo", buildProxiedPath("/", "foo"))
	assert.Equal(t, "/foo", buildProxiedPath("/", "/foo"))
}

func TestWildcardRoutePath_NoTrailingSlash(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "/api/*", wildcardRoutePath("/api"))
}

func TestWildcardRoutePath_WithTrailingSlash(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "/api/*", wildcardRoutePath("/api/"))
}

func TestListenAddrFromHostPort(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "127.0.0.1:8080", listenAddrFromHostPort("127.0.0.1", 8080))
}

func TestRequireAppRoutePort_Zero(t *testing.T) {
	t.Parallel()
	_, ok := requireAppRoutePort(&domain.WebRoute{AppPort: 0})
	assert.False(t, ok)
}

func TestRequireAppRoutePort_NonZero(t *testing.T) {
	t.Parallel()
	port, ok := requireAppRoutePort(&domain.WebRoute{AppPort: 3000})
	assert.True(t, ok)
	assert.Equal(t, 3000, port)
}

func TestIsWebSocketHandshakeOK_Switching(t *testing.T) {
	t.Parallel()
	assert.True(t, isWebSocketHandshakeOK(&http.Response{StatusCode: http.StatusSwitchingProtocols}))
}

func TestIsWebSocketHandshakeOK_NotSwitching(t *testing.T) {
	t.Parallel()
	assert.False(t, isWebSocketHandshakeOK(&http.Response{StatusCode: http.StatusOK}))
}

func TestCopyQueryString(t *testing.T) {
	t.Parallel()
	src := &url.URL{RawQuery: "foo=bar&x=1"}
	dst := &url.URL{}
	copyQueryString(dst, src)
	assert.Equal(t, "foo=bar&x=1", dst.RawQuery)
}

func TestSetProxyHost(t *testing.T) {
	t.Parallel()
	target := &url.URL{Host: "backend.example.com:9000"}
	dst := &url.URL{}
	setProxyHost(dst, target)
	assert.Equal(t, "backend.example.com:9000", dst.Host)
}

func TestHTTPURLFromHostPort(t *testing.T) {
	t.Parallel()
	u, err := httpURLFromHostPort("127.0.0.1", 9090)
	require.NoError(t, err)
	assert.Equal(t, "127.0.0.1:9090", u.Host)
}
