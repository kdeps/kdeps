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

func TestRequestIDMiddleware(t *testing.T) {
	middleware := http.RequestIDMiddleware()

	t.Run("generates new request ID when missing", func(t *testing.T) {
		handler := middleware(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
			requestID := http.GetRequestID(r.Context())
			assert.NotEmpty(t, requestID)
			// Should be in response header
			assert.Equal(t, requestID, w.Header().Get("X-Request-ID"))
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest(stdhttp.MethodGet, "/test", nil)
		handler(w, req)
	})

	t.Run("uses existing request ID from header", func(t *testing.T) {
		existingID := "existing-request-id-123"
		handler := middleware(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
			requestID := http.GetRequestID(r.Context())
			assert.Equal(t, existingID, requestID)
			assert.Equal(t, existingID, w.Header().Get("X-Request-ID"))
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest(stdhttp.MethodGet, "/test", nil)
		req.Header.Set("X-Request-ID", existingID)
		handler(w, req)
	})
}

func TestSessionMiddleware(t *testing.T) {
	t.Run("reads session cookie and stores in context", func(t *testing.T) {
		middleware := http.SessionMiddleware()
		sessionID := "test-session-id-123"
		handler := middleware(func(_ stdhttp.ResponseWriter, r *stdhttp.Request) {
			retrievedSessionID := http.GetSessionID(r.Context())
			assert.Equal(t, sessionID, retrievedSessionID)
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest(stdhttp.MethodGet, "/test", nil)
		cookie := &stdhttp.Cookie{
			Name:  http.SessionCookieName,
			Value: sessionID,
		}
		req.AddCookie(cookie)
		handler(w, req)
	})

	t.Run("generates new session ID when no cookie present", func(t *testing.T) {
		middleware := http.SessionMiddleware()
		handler := middleware(func(_ stdhttp.ResponseWriter, r *stdhttp.Request) {
			sessionID := http.GetSessionID(r.Context())
			assert.NotEmpty(t, sessionID)
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest(stdhttp.MethodGet, "/test", nil)
		handler(w, req)
	})

	t.Run("generates new session ID when cookie value is empty", func(t *testing.T) {
		middleware := http.SessionMiddleware()
		handler := middleware(func(_ stdhttp.ResponseWriter, r *stdhttp.Request) {
			sessionID := http.GetSessionID(r.Context())
			assert.NotEmpty(t, sessionID)
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest(stdhttp.MethodGet, "/test", nil)
		cookie := &stdhttp.Cookie{
			Name:  http.SessionCookieName,
			Value: "",
		}
		req.AddCookie(cookie)
		handler(w, req)
	})
}
