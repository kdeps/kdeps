// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package http

import (
	"fmt"
	stdhttp "net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

func TestErrorDetailString_Nil(t *testing.T) {
	assert.Empty(t, errorDetailString(nil))
}

// fakePreflightError mirrors executor.PreflightError without importing it.
type fakePreflightError struct {
	code    int
	message string
}

func (e *fakePreflightError) Error() string {
	return fmt.Sprintf("preflight error (code %d): %s", e.code, e.message)
}

func (e *fakePreflightError) PreflightStatus() (int, string) {
	return e.code, e.message
}

func TestNormalizeToAppError_PreflightUsesConfiguredCode(t *testing.T) {
	preflight := &fakePreflightError{code: 400, message: "'q' is required"}
	wrapped := fmt.Errorf("preflight check failed for llm: %w", preflight)

	appErr := normalizeToAppError(wrapped, false)

	assert.Equal(t, domain.ErrCodePreflightFailed, appErr.Code)
	assert.Equal(t, stdhttp.StatusBadRequest, appErr.StatusCode)
	assert.Equal(t, "'q' is required", appErr.Message)
}

func TestNormalizeToAppError_PreflightInvalidCodeFallsBack(t *testing.T) {
	for _, code := range []int{0, 99, 600, 9999} {
		appErr := normalizeToAppError(&fakePreflightError{code: code, message: "bad input"}, false)

		assert.Equal(t, domain.ErrCodePreflightFailed, appErr.Code)
		assert.Equal(t, domain.GetHTTPStatus(domain.ErrCodePreflightFailed), appErr.StatusCode, "code %d", code)
		assert.Equal(t, "bad input", appErr.Message)
	}
}

func TestNormalizeToAppError_AppErrorTakesPrecedence(t *testing.T) {
	appErr := normalizeToAppError(domain.NewAppError(domain.ErrCodeNotFound, "missing"), false)
	assert.Equal(t, domain.ErrCodeNotFound, appErr.Code)
}
