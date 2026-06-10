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
