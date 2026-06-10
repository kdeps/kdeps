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

func TestErrorHandlerMiddleware(t *testing.T) {
	t.Run("adds debug mode to context", func(t *testing.T) {
		middleware := http.ErrorHandlerMiddleware(true)
		handler := middleware(func(_ stdhttp.ResponseWriter, r *stdhttp.Request) {
			debugMode := http.GetDebugMode(r.Context())
			assert.True(t, debugMode)
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest(stdhttp.MethodGet, "/test", nil)
		handler(w, req)
	})

	t.Run("recovers from panic", func(t *testing.T) {
		middleware := http.ErrorHandlerMiddleware(false)
		handler := middleware(func(_ stdhttp.ResponseWriter, _ *stdhttp.Request) {
			panic("test panic")
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest(stdhttp.MethodGet, "/test", nil)
		handler(w, req)

		// Should recover and return 500 error
		assert.Equal(t, stdhttp.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "Internal server error")
	})

	t.Run("recovers from string panic", func(t *testing.T) {
		middleware := http.ErrorHandlerMiddleware(false)
		handler := middleware(func(_ stdhttp.ResponseWriter, _ *stdhttp.Request) {
			panic("string panic message")
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest(stdhttp.MethodGet, "/test", nil)
		handler(w, req)

		assert.Equal(t, stdhttp.StatusInternalServerError, w.Code)
	})

	t.Run("recovers from error panic", func(t *testing.T) {
		middleware := http.ErrorHandlerMiddleware(false)
		testErr := assert.AnError
		handler := middleware(func(_ stdhttp.ResponseWriter, _ *stdhttp.Request) {
			panic(testErr)
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest(stdhttp.MethodGet, "/test", nil)
		handler(w, req)

		assert.Equal(t, stdhttp.StatusInternalServerError, w.Code)
	})
}

func TestDebugModeMiddleware(t *testing.T) {
	t.Run("debug mode from env true", func(t *testing.T) {
		t.Setenv("DEBUG", "true")
		middleware := http.DebugModeMiddleware()
		handler := middleware(func(_ stdhttp.ResponseWriter, r *stdhttp.Request) {
			debugMode := http.GetDebugMode(r.Context())
			assert.True(t, debugMode)
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest(stdhttp.MethodGet, "/test", nil)
		handler(w, req)
	})

	t.Run("debug mode from env 1", func(t *testing.T) {
		t.Setenv("DEBUG", "1")
		middleware := http.DebugModeMiddleware()
		handler := middleware(func(_ stdhttp.ResponseWriter, r *stdhttp.Request) {
			debugMode := http.GetDebugMode(r.Context())
			assert.True(t, debugMode)
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest(stdhttp.MethodGet, "/test", nil)
		handler(w, req)
	})

	t.Run("debug mode disabled", func(t *testing.T) {
		t.Setenv("DEBUG", "false")
		middleware := http.DebugModeMiddleware()
		handler := middleware(func(_ stdhttp.ResponseWriter, r *stdhttp.Request) {
			debugMode := http.GetDebugMode(r.Context())
			assert.False(t, debugMode)
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest(stdhttp.MethodGet, "/test", nil)
		handler(w, req)
	})
}
