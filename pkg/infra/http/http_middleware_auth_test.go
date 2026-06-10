// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package http_test

import (
	stdhttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kdeps/kdeps/v2/pkg/infra/http"
)

func TestAuthMiddleware(t *testing.T) {
	t.Run("passes through when no token configured", func(t *testing.T) {
		middleware := http.AuthMiddleware("")
		called := false
		handler := middleware(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
			called = true
			w.WriteHeader(stdhttp.StatusOK)
		})
		w := httptest.NewRecorder()
		req := httptest.NewRequest(stdhttp.MethodGet, "/api", nil)
		handler(w, req)
		assert.True(t, called)
		assert.Equal(t, stdhttp.StatusOK, w.Code)
	})

	t.Run("health endpoint always exempt", func(t *testing.T) {
		middleware := http.AuthMiddleware("secret")
		called := false
		handler := middleware(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
			called = true
			w.WriteHeader(stdhttp.StatusOK)
		})
		w := httptest.NewRecorder()
		req := httptest.NewRequest(stdhttp.MethodGet, "/health", nil)
		handler(w, req)
		assert.True(t, called)
		assert.Equal(t, stdhttp.StatusOK, w.Code)
	})

	t.Run("management endpoints exempt from API auth", func(t *testing.T) {
		middleware := http.AuthMiddleware("api-secret")
		called := false
		handler := middleware(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
			called = true
			w.WriteHeader(stdhttp.StatusOK)
		})
		w := httptest.NewRecorder()
		req := httptest.NewRequest(stdhttp.MethodGet, "/_kdeps/status", nil)
		req.Header.Set("Authorization", "Bearer management-secret")
		handler(w, req)
		assert.True(t, called)
		assert.Equal(t, stdhttp.StatusOK, w.Code)
	})

	t.Run("bearer token accepted", func(t *testing.T) {
		middleware := http.AuthMiddleware("mytoken")
		called := false
		handler := middleware(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
			called = true
			w.WriteHeader(stdhttp.StatusOK)
		})
		w := httptest.NewRecorder()
		req := httptest.NewRequest(stdhttp.MethodGet, "/api", nil)
		req.Header.Set("Authorization", "Bearer mytoken")
		handler(w, req)
		assert.True(t, called)
		assert.Equal(t, stdhttp.StatusOK, w.Code)
	})

	t.Run("bearer token trims surrounding whitespace", func(t *testing.T) {
		middleware := http.AuthMiddleware("mytoken")
		called := false
		handler := middleware(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
			called = true
			w.WriteHeader(stdhttp.StatusOK)
		})
		w := httptest.NewRecorder()
		req := httptest.NewRequest(stdhttp.MethodGet, "/api", nil)
		req.Header.Set("Authorization", "Bearer  mytoken ")
		handler(w, req)
		assert.True(t, called)
		assert.Equal(t, stdhttp.StatusOK, w.Code)
	})

	t.Run("X-Api-Key accepted", func(t *testing.T) {
		middleware := http.AuthMiddleware("mytoken")
		called := false
		handler := middleware(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
			called = true
			w.WriteHeader(stdhttp.StatusOK)
		})
		w := httptest.NewRecorder()
		req := httptest.NewRequest(stdhttp.MethodGet, "/api", nil)
		req.Header.Set("X-Api-Key", "mytoken")
		handler(w, req)
		assert.True(t, called)
		assert.Equal(t, stdhttp.StatusOK, w.Code)
	})

	t.Run("wrong token rejected with 401", func(t *testing.T) {
		middleware := http.AuthMiddleware("mytoken")
		called := false
		handler := middleware(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
			called = true
			w.WriteHeader(stdhttp.StatusOK)
		})
		w := httptest.NewRecorder()
		req := httptest.NewRequest(stdhttp.MethodGet, "/api", nil)
		req.Header.Set("Authorization", "Bearer wrongtoken")
		handler(w, req)
		assert.False(t, called)
		assert.Equal(t, stdhttp.StatusUnauthorized, w.Code)
	})

	t.Run("missing auth rejected with 401", func(t *testing.T) {
		middleware := http.AuthMiddleware("mytoken")
		called := false
		handler := middleware(func(_ stdhttp.ResponseWriter, _ *stdhttp.Request) {
			called = true
		})
		w := httptest.NewRecorder()
		req := httptest.NewRequest(stdhttp.MethodGet, "/api", nil)
		handler(w, req)
		assert.False(t, called)
		assert.Equal(t, stdhttp.StatusUnauthorized, w.Code)
	})
}
