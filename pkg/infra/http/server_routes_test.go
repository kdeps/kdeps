// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package http_test

import (
	"encoding/json"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

// TestSetupRoutes_IncludesManagementRoutes verifies that SetupRoutes registers management routes.
func TestSetupRoutes_IncludesManagementRoutes(t *testing.T) {
	workflow := &domain.Workflow{}
	workflow.Metadata.Name = "test"
	workflow.Metadata.Version = "1.0.0"

	server := makeTestServer(t, workflow)
	server.SetupRoutes()

	t.Setenv("KDEPS_MANAGEMENT_TOKEN", "mgmt-test-token")

	req := httptest.NewRequest(stdhttp.MethodGet, "/_kdeps/status", nil)
	req.Header.Set("Authorization", "Bearer mgmt-test-token")
	rec := httptest.NewRecorder()
	server.Router.ServeHTTP(rec, req)
	assert.Equal(t, stdhttp.StatusOK, rec.Code)

	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.Equal(t, "ok", body["status"])
}
