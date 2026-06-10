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
)

func TestHandleManagementOpenAPI_NoWorkflow(t *testing.T) {
	server := makeTestServer(t, nil)
	server.SetupManagementRoutes()

	req := httptest.NewRequest(stdhttp.MethodGet, "/_kdeps/openapi", nil)
	rec := httptest.NewRecorder()
	server.HandleManagementOpenAPI(rec, req)

	assert.Equal(t, stdhttp.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))

	assert.Equal(t, "3.0.3", body["openapi"])
	info, ok := body["info"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "kdeps agent", info["title"])
}

func TestHandleManagementOpenAPI_WithWorkflow(t *testing.T) {
	server := makeTestServer(t, chatbotTestWorkflow())
	server.SetupManagementRoutes()

	req := httptest.NewRequest(stdhttp.MethodGet, "/_kdeps/openapi", nil)
	rec := httptest.NewRecorder()
	server.HandleManagementOpenAPI(rec, req)

	assert.Equal(t, stdhttp.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))

	// Top-level OpenAPI version
	assert.Equal(t, "3.0.3", body["openapi"])

	// Info block
	info, ok := body["info"].(map[string]interface{})
	require.True(t, ok, "info block missing")
	assert.Equal(t, "chatbot", info["title"])
	assert.Equal(t, "A simple chatbot agent", info["description"])
	assert.Equal(t, "1.0.0", info["version"])

	// Paths block
	paths, ok := body["paths"].(map[string]interface{})
	require.True(t, ok, "paths block missing")
	assert.Contains(t, paths, "/api/v1/chat")

	// POST operation
	chatPath, ok := paths["/api/v1/chat"].(map[string]interface{})
	require.True(t, ok)
	postOp, ok := chatPath["post"].(map[string]interface{})
	require.True(t, ok, "POST operation missing")

	assert.Equal(t, "llmResource", postOp["operationId"])

	// Request body
	reqBody, ok := postOp["requestBody"].(map[string]interface{})
	require.True(t, ok, "requestBody missing")
	assert.Equal(t, true, reqBody["required"])
}

func TestHandleManagementOpenAPI_RouteRegistered(t *testing.T) {
	t.Setenv("KDEPS_MANAGEMENT_TOKEN", "mgmt-test-token")
	server := makeTestServer(t, chatbotTestWorkflow())
	server.SetupManagementRoutes()

	req := httptest.NewRequest(stdhttp.MethodGet, "/_kdeps/openapi", nil)
	req.Header.Set("Authorization", "Bearer mgmt-test-token")
	rec := httptest.NewRecorder()

	// Test via router to confirm the route is registered.
	server.Router.ServeHTTP(rec, req)

	assert.Equal(t, stdhttp.StatusOK, rec.Code)
}

func TestHandleManagementSchema_NoWorkflow(t *testing.T) {
	server := makeTestServer(t, nil)
	server.SetupManagementRoutes()

	req := httptest.NewRequest(stdhttp.MethodGet, "/_kdeps/schema", nil)
	rec := httptest.NewRecorder()
	server.HandleManagementSchema(rec, req)

	assert.Equal(t, stdhttp.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))

	assert.Equal(t, "https://json-schema.org/draft/2020-12/schema", body["$schema"])
	assert.Equal(t, "kdeps agent", body["title"])
	assert.Equal(t, "object", body["type"])
}

func TestHandleManagementSchema_WithWorkflow(t *testing.T) {
	server := makeTestServer(t, chatbotTestWorkflow())
	server.SetupManagementRoutes()

	req := httptest.NewRequest(stdhttp.MethodGet, "/_kdeps/schema", nil)
	rec := httptest.NewRecorder()
	server.HandleManagementSchema(rec, req)

	assert.Equal(t, stdhttp.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))

	assert.Equal(t, "https://json-schema.org/draft/2020-12/schema", body["$schema"])
	assert.Equal(t, "chatbot", body["title"])
	assert.Equal(t, "A simple chatbot agent", body["description"])
	assert.Equal(t, "object", body["type"])

	// Properties block
	props, ok := body["properties"].(map[string]interface{})
	require.True(t, ok, "properties block missing")
	msgProp, ok := props["message"].(map[string]interface{})
	require.True(t, ok, "message property missing")
	assert.Equal(t, "string", msgProp["type"])

	// Required
	required, ok := body["required"].([]interface{})
	require.True(t, ok, "required block missing")
	assert.Contains(t, required, "message")
}

func TestHandleManagementSchema_RouteRegistered(t *testing.T) {
	t.Setenv("KDEPS_MANAGEMENT_TOKEN", "mgmt-test-token")
	server := makeTestServer(t, chatbotTestWorkflow())
	server.SetupManagementRoutes()

	req := httptest.NewRequest(stdhttp.MethodGet, "/_kdeps/schema", nil)
	req.Header.Set("Authorization", "Bearer mgmt-test-token")
	rec := httptest.NewRecorder()

	// Test via router to confirm the route is registered.
	server.Router.ServeHTTP(rec, req)

	assert.Equal(t, stdhttp.StatusOK, rec.Code)
}
