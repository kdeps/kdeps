// Copyright 2026 Kdeps, KvK 94834768
// Licensed under the Apache License, Version 2.0

package http_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kdeps/kdeps/v3/pkg/infra/http"
)

func TestBindPortFromEnv(t *testing.T) {
	t.Setenv("KDEPS_BIND_PORT", "")
	assert.Equal(t, 3000, http.BindPortFromEnv(3000))
	for _, bad := range []string{"x", "0", "-1", "70000"} {
		t.Setenv("KDEPS_BIND_PORT", bad)
		assert.Equal(t, 3000, http.BindPortFromEnv(3000), bad)
	}
	t.Setenv("KDEPS_BIND_PORT", "4123")
	assert.Equal(t, 4123, http.BindPortFromEnv(3000))
}
