// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package http

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKillProcessIfRunning_NotRunning(t *testing.T) {
	assert.NoError(t, killProcessIfRunning(nil))
	assert.NoError(t, killProcessIfRunning(&exec.Cmd{}))
}
