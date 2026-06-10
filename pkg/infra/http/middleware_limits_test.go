// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package http_test

import (
	"io"
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kdeps/kdeps/v2/pkg/infra/http"
)

func TestUploadMiddleware(t *testing.T) {
	t.Run("passes through non-multipart requests", func(t *testing.T) {
		middleware := http.UploadMiddleware(10 * 1024 * 1024) // 10MB
		handlerCalled := false
		handler := middleware(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
			handlerCalled = true
			w.WriteHeader(stdhttp.StatusOK)
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest(stdhttp.MethodGet, "/test", nil)
		req.Header.Set("Content-Type", "application/json")
		handler(w, req)

		assert.True(t, handlerCalled)
		assert.Equal(t, stdhttp.StatusOK, w.Code)
	})

	t.Run("rejects large multipart requests", func(t *testing.T) {
		middleware := http.UploadMiddleware(100) // Very small limit
		handlerCalled := false
		handler := middleware(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
			handlerCalled = true
			w.WriteHeader(stdhttp.StatusOK)
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest(stdhttp.MethodPost, "/test", nil)
		req.Header.Set("Content-Type", "multipart/form-data")
		req.ContentLength = 200 // Larger than limit
		handler(w, req)

		assert.False(t, handlerCalled)
		assert.Equal(t, stdhttp.StatusRequestEntityTooLarge, w.Code)
		assert.Contains(t, w.Body.String(), "Request body too large")
	})

	t.Run("allows small multipart requests", func(t *testing.T) {
		middleware := http.UploadMiddleware(10 * 1024 * 1024) // 10MB
		handlerCalled := false
		handler := middleware(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
			handlerCalled = true
			w.WriteHeader(stdhttp.StatusOK)
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest(stdhttp.MethodPost, "/test", nil)
		req.Header.Set("Content-Type", "multipart/form-data")
		req.ContentLength = 100 // Within limit
		handler(w, req)

		assert.True(t, handlerCalled)
		assert.Equal(t, stdhttp.StatusOK, w.Code)
	})
}

func TestBodyLimitMiddleware_ExceedsLimit(t *testing.T) {
	// MaxBytesReader caps at 5 bytes; handler drains the body triggering the error path.
	middleware := http.BodyLimitMiddleware(5)
	handler := middleware(func(_ stdhttp.ResponseWriter, r *stdhttp.Request) {
		// Drain body inside handler so the post-handler drain also hits the limit.
		_, _ = io.ReadAll(r.Body)
	})

	body := strings.NewReader("this is way more than five bytes")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(stdhttp.MethodPost, "/api", body)
	req.Header.Set("Content-Type", "application/json")
	handler(w, req)
	// MaxBytesReader error triggers 413 from RespondWithError.
	assert.Equal(t, stdhttp.StatusRequestEntityTooLarge, w.Code)
}

func TestConcurrentLimitMiddleware(t *testing.T) {
	t.Run("allows requests under the limit", func(t *testing.T) {
		middleware := http.ConcurrentLimitMiddleware(5)
		called := 0
		handler := middleware(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
			called++
			w.WriteHeader(stdhttp.StatusOK)
		})
		for range 3 {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(stdhttp.MethodGet, "/api", nil)
			handler(w, req)
			assert.Equal(t, stdhttp.StatusOK, w.Code)
		}
		assert.Equal(t, 3, called)
	})

	t.Run("returns 503 when limit reached", func(t *testing.T) {
		// limit=1, block the slot then immediately try a second request
		middleware := http.ConcurrentLimitMiddleware(1)

		blocked := make(chan struct{})
		unblock := make(chan struct{})
		handler := middleware(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
			close(blocked) // signal that we're inside the handler
			<-unblock      // wait until test lets us proceed
			w.WriteHeader(stdhttp.StatusOK)
		})

		// Occupy the one slot in a goroutine
		done := make(chan struct{})
		go func() {
			defer close(done)
			w := httptest.NewRecorder()
			req := httptest.NewRequest(stdhttp.MethodGet, "/slow", nil)
			handler(w, req)
		}()

		<-blocked // first request is inside the handler; semaphore full

		// Second request should be rejected
		w2 := httptest.NewRecorder()
		req2 := httptest.NewRequest(stdhttp.MethodGet, "/api", nil)
		handler(w2, req2)
		assert.Equal(t, stdhttp.StatusServiceUnavailable, w2.Code)

		close(unblock)
		<-done
	})
}

func TestRateLimitMiddleware(t *testing.T) {
	t.Run("allows requests within limit", func(t *testing.T) {
		middleware := http.RateLimitMiddleware(600, 10, nil) // 10 req/s sustained, burst 10
		called := 0
		handler := middleware(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
			called++
			w.WriteHeader(stdhttp.StatusOK)
		})
		for i := range 5 {
			_ = i
			w := httptest.NewRecorder()
			req := httptest.NewRequest(stdhttp.MethodGet, "/api", nil)
			req.RemoteAddr = "127.0.0.1:1234"
			handler(w, req)
			assert.Equal(t, stdhttp.StatusOK, w.Code)
		}
		assert.Equal(t, 5, called)
	})

	t.Run("rate limits by forwarded IP from trusted proxy", func(t *testing.T) {
		middleware := http.RateLimitMiddleware(1, 1, []string{"10.0.0.1"})
		handler := middleware(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
			w.WriteHeader(stdhttp.StatusOK)
		})

		makeReq := func() *stdhttp.Request {
			req := httptest.NewRequest(stdhttp.MethodGet, "/api", nil)
			req.RemoteAddr = "10.0.0.1:443"
			req.Header.Set("X-Forwarded-For", "198.51.100.5")
			return req
		}

		w1 := httptest.NewRecorder()
		handler(w1, makeReq())
		assert.Equal(t, stdhttp.StatusOK, w1.Code)

		w2 := httptest.NewRecorder()
		handler(w2, makeReq())
		assert.Equal(t, stdhttp.StatusTooManyRequests, w2.Code)
	})

	t.Run("rate limits after burst exhausted", func(t *testing.T) {
		// 1 req/min, burst 1 - second request from same IP should be limited
		middleware := http.RateLimitMiddleware(1, 1, nil)
		handler := middleware(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
			w.WriteHeader(stdhttp.StatusOK)
		})

		// First request - should pass
		w1 := httptest.NewRecorder()
		req1 := httptest.NewRequest(stdhttp.MethodGet, "/api", nil)
		req1.RemoteAddr = "10.0.0.1:9999"
		handler(w1, req1)
		assert.Equal(t, stdhttp.StatusOK, w1.Code)

		// Second request immediately - should be rate limited
		w2 := httptest.NewRecorder()
		req2 := httptest.NewRequest(stdhttp.MethodGet, "/api", nil)
		req2.RemoteAddr = "10.0.0.1:9999"
		handler(w2, req2)
		assert.Equal(t, stdhttp.StatusTooManyRequests, w2.Code)
		assert.Equal(t, "60", w2.Header().Get("Retry-After"))
	})
}

func TestBodyLimitMiddleware(t *testing.T) {
	t.Run("skips multipart requests", func(t *testing.T) {
		middleware := http.BodyLimitMiddleware(10)
		called := false
		handler := middleware(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
			called = true
			w.WriteHeader(stdhttp.StatusOK)
		})
		w := httptest.NewRecorder()
		req := httptest.NewRequest(stdhttp.MethodPost, "/api", nil)
		req.Header.Set("Content-Type", "multipart/form-data; boundary=xxx")
		handler(w, req)
		assert.True(t, called)
	})

	t.Run("passes small body", func(t *testing.T) {
		middleware := http.BodyLimitMiddleware(1024)
		called := false
		handler := middleware(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
			called = true
			w.WriteHeader(stdhttp.StatusOK)
		})
		w := httptest.NewRecorder()
		req := httptest.NewRequest(stdhttp.MethodPost, "/api", stdhttp.NoBody)
		req.Header.Set("Content-Type", "application/json")
		handler(w, req)
		assert.True(t, called)
	})
}
