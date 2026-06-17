package http

import (
	stdhttp "net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsParamPattern(t *testing.T) {
	t.Parallel()
	assert.True(t, isParamPattern(":id"))
	assert.False(t, isParamPattern("id"))
	assert.False(t, isParamPattern("*"))
}

func TestIsWildcardPattern(t *testing.T) {
	t.Parallel()
	assert.True(t, isWildcardPattern("*"))
	assert.False(t, isWildcardPattern(":id"))
	assert.False(t, isWildcardPattern("foo"))
}

func TestPatternPartMatches(t *testing.T) {
	t.Parallel()
	assert.True(t, patternPartMatches(":id", "123"))
	assert.True(t, patternPartMatches("*", "anything"))
	assert.True(t, patternPartMatches("api", "api"))
	assert.False(t, patternPartMatches("api", "v1"))
}

func TestStripTrailingWildcard_WithWildcard(t *testing.T) {
	t.Parallel()
	parts, ok := stripTrailingWildcard([]string{"api", "v1", "*"})
	assert.True(t, ok)
	assert.Equal(t, []string{"api", "v1"}, parts)
}

func TestStripTrailingWildcard_WithoutWildcard(t *testing.T) {
	t.Parallel()
	parts, ok := stripTrailingWildcard([]string{"api", "v1"})
	assert.False(t, ok)
	assert.Equal(t, []string{"api", "v1"}, parts)
}

func TestStripTrailingWildcard_Empty(t *testing.T) {
	t.Parallel()
	parts, ok := stripTrailingWildcard(nil)
	assert.False(t, ok)
	assert.Nil(t, parts)
}

func TestMatchRouterPattern(t *testing.T) {
	t.Parallel()
	assert.True(t, matchRouterPattern("/api/:id", "/api/123"))
	assert.True(t, matchRouterPattern("/api/*", "/api/v1/resource"))
	assert.True(t, matchRouterPattern("/exact", "/exact"))
	assert.False(t, matchRouterPattern("/api/:id", "/api"))
	assert.False(t, matchRouterPattern("/api", "/other"))
}

func TestLongestMatchingPattern(t *testing.T) {
	t.Parallel()
	noop := func(stdhttp.ResponseWriter, *stdhttp.Request) {}
	routes := map[string]stdhttp.HandlerFunc{
		"/api/:id":        noop,
		"/api/:id/detail": noop,
	}
	handler := longestMatchingPattern(routes, "/api/123/detail", matchRouterPattern)
	assert.NotNil(t, handler, "should find the longest matching pattern")
}

func TestPathRegisteredInRoutes_ExactMatch(t *testing.T) {
	t.Parallel()
	noop := func(stdhttp.ResponseWriter, *stdhttp.Request) {}
	routes := map[string]stdhttp.HandlerFunc{"/exact": noop}
	assert.True(t, pathRegisteredInRoutes(routes, "/exact", matchRouterPattern))
}

func TestPathRegisteredInRoutes_PatternMatch(t *testing.T) {
	t.Parallel()
	noop := func(stdhttp.ResponseWriter, *stdhttp.Request) {}
	routes := map[string]stdhttp.HandlerFunc{"/api/:id": noop}
	assert.True(t, pathRegisteredInRoutes(routes, "/api/456", matchRouterPattern))
	assert.False(t, pathRegisteredInRoutes(routes, "/other", matchRouterPattern))
}
