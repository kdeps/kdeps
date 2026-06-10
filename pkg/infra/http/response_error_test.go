// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package http_test

import (
	"context"
	"encoding/json"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	httppkg "github.com/kdeps/kdeps/v2/pkg/infra/http"
)

// TestRespondWithError_DebugModeNonAppError exercises the debugMode path with a plain
// (non-AppError) error, covering the error message injection (line 90-91), WithStack
// (line 99-100), and WithDetails (line 104-105) branches.
func TestRespondWithError_DebugModeNonAppError(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(stdhttp.MethodGet, "/test", nil)
	ctx := context.WithValue(req.Context(), httppkg.RequestIDKey, "test-id")
	req = req.WithContext(ctx)

	httppkg.RespondWithError(w, req, assert.AnError, true)

	assert.Equal(t, stdhttp.StatusInternalServerError, w.Code)
	var response map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	assert.False(t, response["success"].(bool))

	errorData := response["error"].(map[string]interface{})
	// Debug mode should include the original error message
	assert.Contains(t, errorData["message"].(string), assert.AnError.Error())
	// Debug mode should include a stack trace
	stack, hasStack := errorData["stack"]
	assert.True(t, hasStack, "stack should be present in debug mode")
	assert.NotEmpty(t, stack, "stack should not be empty")
}
