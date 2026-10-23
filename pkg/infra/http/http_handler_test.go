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
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

// TestServerHandler_ServesWithoutListener verifies Handler wires routes and
// serves requests in-process, with no port bound.
func TestServerHandler_ServesWithoutListener(t *testing.T) {
	workflow := &domain.Workflow{}
	workflow.Metadata.Name = "handler-test"
	workflow.Metadata.Version = "1.0.0"

	server := makeTestServer(t, workflow)
	handler, err := server.Handler(false)
	require.NoError(t, err)
	require.NotNil(t, handler)

	req := httptest.NewRequest(stdhttp.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, stdhttp.StatusOK, rec.Code)
}

// TestServerHandler_RequiresAuthToken verifies Handler applies the same
// security middleware as Start: an apiServer without a token fails to wire.
func TestServerHandler_RequiresAuthToken(t *testing.T) {
	t.Setenv("KDEPS_API_AUTH_TOKEN", "")

	workflow := &domain.Workflow{}
	workflow.Metadata.Name = "handler-auth"
	workflow.Settings.APIServer = &domain.APIServerConfig{}

	server := makeTestServer(t, workflow)
	_, err := server.Handler(false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "KDEPS_API_AUTH_TOKEN")
}
