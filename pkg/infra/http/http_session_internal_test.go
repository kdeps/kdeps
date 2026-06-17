// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package http

import (
	"crypto/tls"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsSecureRequest_TLS(t *testing.T) {
	req := httptest.NewRequest(stdhttp.MethodGet, "https://example.com/", nil)
	req.TLS = &tls.ConnectionState{}
	assert.True(t, isSecureRequest(req))
}

func TestPanicToError_Error(t *testing.T) {
	t.Parallel()
	msg, err := panicToError(assert.AnError)
	assert.Equal(t, assert.AnError.Error(), msg)
	assert.ErrorIs(t, err, assert.AnError)
}

func TestPanicToError_String(t *testing.T) {
	t.Parallel()
	msg, err := panicToError("oops")
	assert.Equal(t, "oops", msg)
	assert.EqualError(t, err, "oops")
}

func TestPanicToError_Other(t *testing.T) {
	t.Parallel()
	msg, err := panicToError(42)
	assert.Equal(t, "42", msg)
	assert.EqualError(t, err, "42")
}

func TestHeadersAlreadyWritten_NotChecker(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	assert.False(t, headersAlreadyWritten(rec))
}

func TestHeadersAlreadyWritten_Wrapper(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	wrapper := &ResponseWriterWrapper{ResponseWriter: rec}
	assert.False(t, headersAlreadyWritten(wrapper))
	wrapper.WriteHeader(stdhttp.StatusOK)
	assert.True(t, headersAlreadyWritten(wrapper))
}

func TestNewSessionCookie_Fields(t *testing.T) {
	t.Parallel()
	c := newSessionCookie("abc123", true)
	assert.Equal(t, SessionCookieName, c.Name)
	assert.Equal(t, "abc123", c.Value)
	assert.Equal(t, "/", c.Path)
	assert.True(t, c.HttpOnly)
	assert.True(t, c.Secure)
	assert.Equal(t, sessionCookieMaxAge, c.MaxAge)
}

func TestAppErrorFromPanic_NonDebug(t *testing.T) {
	t.Parallel()
	appErr := appErrorFromPanic(assert.AnError, "boom", false)
	assert.NotNil(t, appErr)
}

func TestAppErrorFromPanic_Debug(t *testing.T) {
	t.Parallel()
	appErr := appErrorFromPanic(assert.AnError, "boom", true)
	assert.NotNil(t, appErr)
}

func TestRecoverPanic_HeadersAlreadyWritten(t *testing.T) {
	rec := httptest.NewRecorder()
	wrapper := &ResponseWriterWrapper{ResponseWriter: rec}
	wrapper.WriteHeader(stdhttp.StatusOK)

	req := httptest.NewRequest(stdhttp.MethodGet, "/", nil)
	func() {
		defer RecoverPanic(wrapper, req, false)
		panic("after headers")
	}()

	assert.Equal(t, stdhttp.StatusOK, rec.Code)
}
