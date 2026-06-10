// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package http

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestBrowserRenderedContentType_Empty exercises the empty content-type
// branch at line 100 of middleware.go.
func TestBrowserRenderedContentType_Empty(t *testing.T) {
	assert.True(t, browserRenderedContentType(""))
}
