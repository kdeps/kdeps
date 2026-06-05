// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package llm

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsHealthy_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	assert.True(t, isHealthy(srv.URL))
}

func TestIsHealthy_NotOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	assert.False(t, isHealthy(srv.URL))
}

func TestIsHealthy_ConnectionRefused(t *testing.T) {
	assert.False(t, isHealthy("http://127.0.0.1:1"))
}

func TestIsHealthy_InvalidURL(t *testing.T) {
	assert.False(t, isHealthy("://invalid"))
}
