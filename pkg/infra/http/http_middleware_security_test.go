// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package http_test

import (
	"crypto/tls"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kdeps/kdeps/v2/pkg/infra/http"
)

func TestSecurityHeadersMiddleware(t *testing.T) {
	middleware := http.SecurityHeadersMiddleware(false)
	handler := middleware(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
		w.WriteHeader(stdhttp.StatusOK)
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(stdhttp.MethodGet, "/", nil)
	handler(w, req)
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
	assert.Equal(t, "strict-origin-when-cross-origin", w.Header().Get("Referrer-Policy"))
	assert.Equal(t, "camera=(), microphone=(), geolocation=()", w.Header().Get("Permissions-Policy"))
	assert.Empty(t, w.Header().Get("Content-Security-Policy"))
	assert.Empty(t, w.Header().Get("Strict-Transport-Security"))
}

func TestSecurityHeadersMiddleware_WithCSP(t *testing.T) {
	middleware := http.SecurityHeadersMiddleware(true)
	handler := middleware(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
		w.WriteHeader(stdhttp.StatusOK)
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(stdhttp.MethodGet, "/", nil)
	handler(w, req)
	assert.Contains(t, w.Header().Get("Content-Security-Policy"), "default-src 'none'")
}

func TestSecurityHeadersMiddleware_TLS(t *testing.T) {
	middleware := http.SecurityHeadersMiddleware(false)
	handler := middleware(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
		w.WriteHeader(stdhttp.StatusOK)
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(stdhttp.MethodGet, "/", nil)
	req.TLS = &tls.ConnectionState{}
	handler(w, req)
	assert.Equal(t, "max-age=31536000; includeSubDomains", w.Header().Get("Strict-Transport-Security"))
}
