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
	stdhttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	httppkg "github.com/kdeps/kdeps/v2/pkg/infra/http"
)

// TestSetSessionCookie_XForwardedProto verifies that SetSessionCookie sets Secure=true
// when the X-Forwarded-Proto header is https from a trusted proxy peer.
func TestSetSessionCookie_XForwardedProto(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(stdhttp.MethodGet, "/test", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("X-Forwarded-Proto", "https")
	ctx := context.WithValue(req.Context(), httppkg.TrustedProxiesKey, []string{"10.0.0.0/8"})
	req = req.WithContext(ctx)

	httppkg.SetSessionCookie(w, req, "test-session-id")

	cookies := w.Result().Cookies()
	found := false
	for _, cookie := range cookies {
		if cookie.Name == httppkg.SessionCookieName {
			assert.True(t, cookie.Secure,
				"cookie should have Secure flag when X-Forwarded-Proto is https")
			found = true
			break
		}
	}
	assert.True(t, found, "session cookie should be set")
}

func TestSetSessionCookie_UntrustedForwardedProto(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(stdhttp.MethodGet, "/test", nil)
	req.Header.Set("X-Forwarded-Proto", "https")

	httppkg.SetSessionCookie(w, req, "test-session-id")

	cookies := w.Result().Cookies()
	for _, cookie := range cookies {
		if cookie.Name == httppkg.SessionCookieName {
			assert.False(t, cookie.Secure,
				"cookie must not be Secure when X-Forwarded-Proto is not from a trusted proxy")
		}
	}
}
