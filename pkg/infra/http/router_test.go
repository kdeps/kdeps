// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package http_test

import (
	"bytes"
	stdhttp "net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	httppkg "github.com/kdeps/kdeps/v2/pkg/infra/http"
)

// TestRequireManagementAuth_NoTokenConfigured verifies 503 when env var is unset.
func TestRequireManagementAuth_NoTokenConfigured(t *testing.T) {
	t.Setenv("KDEPS_MANAGEMENT_TOKEN", "")
	server := makeTestServer(t, nil)
	server.SetupManagementRoutes()

	// PUT /_kdeps/workflow without token should return 503
	req := httptest.NewRequest(stdhttp.MethodPut, "/_kdeps/workflow", bytes.NewBufferString("yaml"))
	rec := httptest.NewRecorder()
	server.Router.ServeHTTP(rec, req)
	assert.Equal(t, stdhttp.StatusServiceUnavailable, rec.Code)
}

// TestRequireManagementAuth_WrongToken verifies 401 for wrong bearer token.
func TestRequireManagementAuth_WrongToken(t *testing.T) {
	t.Setenv("KDEPS_MANAGEMENT_TOKEN", "secret-token")
	server := makeTestServer(t, nil)
	server.SetupManagementRoutes()

	req := httptest.NewRequest(stdhttp.MethodPut, "/_kdeps/workflow", bytes.NewBufferString("yaml"))
	req.Header.Set("Authorization", "Bearer wrong-token")
	rec := httptest.NewRecorder()
	server.Router.ServeHTTP(rec, req)
	assert.Equal(t, stdhttp.StatusUnauthorized, rec.Code)
}

// TestRequireManagementAuth_MissingHeader verifies 401 when Authorization header is absent.
func TestRequireManagementAuth_MissingHeader(t *testing.T) {
	t.Setenv("KDEPS_MANAGEMENT_TOKEN", "secret-token")
	server := makeTestServer(t, nil)
	server.SetupManagementRoutes()

	req := httptest.NewRequest(stdhttp.MethodPut, "/_kdeps/workflow", bytes.NewBufferString("yaml"))
	rec := httptest.NewRecorder()
	server.Router.ServeHTTP(rec, req)
	assert.Equal(t, stdhttp.StatusUnauthorized, rec.Code)
}

// TestRequireManagementAuth_ValidToken verifies that a correct token passes through.
func TestRequireManagementAuth_ValidToken(t *testing.T) {
	t.Setenv("KDEPS_MANAGEMENT_TOKEN", "valid-token")
	tmpDir := t.TempDir()
	workflowPath := filepath.Join(tmpDir, "workflow.yaml")
	validYAML := `apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: auth-test
  version: 1.0.0
  targetActionId: a
settings:
  agentSettings:
    timezone: UTC
`
	require.NoError(t, os.WriteFile(workflowPath, []byte(validYAML), 0600))

	server := makeTestServer(t, nil)
	server.SetWorkflowPath(workflowPath)
	server.SetupManagementRoutes()

	req := httptest.NewRequest(
		stdhttp.MethodPut,
		"/_kdeps/workflow",
		bytes.NewBufferString(validYAML),
	)
	req.Header.Set("Authorization", "Bearer valid-token")
	rec := httptest.NewRecorder()
	server.Router.ServeHTTP(rec, req)
	assert.Equal(t, stdhttp.StatusOK, rec.Code)
}

// TestRequireManagementAuth_ReloadRequiresToken verifies reload also requires auth.
func TestRequireManagementAuth_ReloadRequiresToken(t *testing.T) {
	t.Setenv("KDEPS_MANAGEMENT_TOKEN", "")
	server := makeTestServer(t, nil)
	server.SetupManagementRoutes()

	req := httptest.NewRequest(stdhttp.MethodPost, "/_kdeps/reload", nil)
	rec := httptest.NewRecorder()
	server.Router.ServeHTTP(rec, req)
	assert.Equal(t, stdhttp.StatusServiceUnavailable, rec.Code)
}

// TestMatchPattern_TrailingWildcard verifies that MatchPattern handles trailing
// wildcard (*) patterns (lines 168-176).
func TestMatchPattern_TrailingWildcard(t *testing.T) {
	router := &httppkg.Router{}

	assert.True(t, router.MatchPattern("/files/*", "/files/foo/bar"),
		"should match multi-segment path with trailing wildcard")
	assert.True(t, router.MatchPattern("/files/*", "/files/foo"),
		"should match single-segment path with trailing wildcard")
	assert.True(t, router.MatchPattern("/files/*", "/files/"),
		"should match trailing slash with trailing wildcard")
	assert.False(t, router.MatchPattern("/files/*", "/other/file"),
		"should not match different prefix")
	assert.False(t, router.MatchPattern("/files/*", "/file"),
		"should not match shorter path that does not start with prefix")
	assert.False(t, router.MatchPattern("/files/*", "/"),
		"should not match root when prefix is /files")
	assert.False(t, router.MatchPattern("/files/*", ""),
		"should not match empty path when prefix is /files")
	assert.True(t, router.MatchPattern("/*", "/anything"),
		"should match any single path with wildcard-only prefix")
}

// TestMatchPattern_MiddleWildcard verifies MatchPattern with wildcard in the
// middle of a pattern, covering the second wildcard condition at line 186.
func TestMatchPattern_MiddleWildcard(t *testing.T) {
	router := &httppkg.Router{}

	// Middle wildcard matches any single segment in that position
	assert.True(t, router.MatchPattern("/api/*/files", "/api/v1/files"),
		"should match middle wildcard with one segment")
	assert.True(t, router.MatchPattern("/api/*/files", "/api/v2/files"),
		"should match middle wildcard with different segment")
	assert.False(t, router.MatchPattern("/api/*/files", "/api/v1/other/files"),
		"should not match with extra segments")
	assert.False(t, router.MatchPattern("/api/*/files", "/api/files"),
		"should not match with missing segment")
}

// TestRouter_AllowedMethods_PatternMatch verifies that a 405 response includes
// the Allow header with methods from pattern-matched routes when no exact match
// exists (lines 150-153).
func TestRouter_AllowedMethods_PatternMatch(t *testing.T) {
	router := httppkg.NewRouter()

	handler := func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
		w.WriteHeader(stdhttp.StatusOK)
	}

	// Register a parameterized route
	router.GET("/api/:id", handler)

	// Send POST to a matching path — no exact POST handler, but GET matches via pattern
	w := httptest.NewRecorder()
	req := httptest.NewRequest(stdhttp.MethodPost, "/api/123", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, stdhttp.StatusMethodNotAllowed, w.Code)
	allowHeader := w.Header().Get("Allow")
	assert.Contains(t, allowHeader, "GET",
		"405 response should include GET in Allow header for pattern-matched route")
}

func TestNewRouter(t *testing.T) {
	router := httppkg.NewRouter()
	assert.NotNil(t, router)
}

func TestRouter_VerbRegistration(t *testing.T) {
	t.Parallel()
	handler := func(w stdhttp.ResponseWriter, _ *stdhttp.Request) { w.WriteHeader(stdhttp.StatusOK) }
	router := httppkg.NewRouter()
	router.GET("/get", handler)
	router.POST("/post", handler)
	router.PUT("/put", handler)
	router.DELETE("/delete", handler)
	router.PATCH("/patch", handler)
	router.OPTIONS("/options", handler)

	for _, tc := range []struct{ method, path string }{
		{"GET", "/get"}, {"POST", "/post"}, {"PUT", "/put"},
		{"DELETE", "/delete"}, {"PATCH", "/patch"}, {"OPTIONS", "/options"},
	} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(tc.method, tc.path, nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, stdhttp.StatusOK, w.Code, "%s %s should return 200", tc.method, tc.path)
	}
}

func TestRouter_Use_MiddlewareApplied(t *testing.T) {
	t.Parallel()
	var called bool
	router := httppkg.NewRouter()
	router.Use(func(next stdhttp.HandlerFunc) stdhttp.HandlerFunc {
		return func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
			called = true
			next(w, r)
		}
	})
	router.GET("/hello", func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
		w.WriteHeader(stdhttp.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(stdhttp.MethodGet, "/hello", nil)
	router.ServeHTTP(w, req)
	assert.True(t, called, "middleware should be called")
	assert.Equal(t, stdhttp.StatusOK, w.Code)
}

func TestRouter_ApplyMiddleware_NoOp(t *testing.T) {
	t.Parallel()
	router := httppkg.NewRouter()
	var called bool
	h := func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
		called = true
		w.WriteHeader(stdhttp.StatusOK)
	}
	wrapped := router.ApplyMiddleware(h)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(stdhttp.MethodGet, "/", nil)
	wrapped(w, req)
	assert.True(t, called)
}
