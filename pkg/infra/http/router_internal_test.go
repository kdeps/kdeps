// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package http

import (
	stdhttp "net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPathRegisteredForMethod_UnknownMethod(t *testing.T) {
	router := NewRouter()
	router.GET("/api/test", func(stdhttp.ResponseWriter, *stdhttp.Request) {})

	assert.False(t, routerPathRegisteredForMethod(router, stdhttp.MethodPost, "/api/test"))
}
