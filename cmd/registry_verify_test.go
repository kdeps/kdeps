// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build !js

package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewRegistryVerifyCmd_Structure(t *testing.T) {
	cmd := newRegistryVerifyCmd()
	assert.Equal(t, "verify <path>", cmd.Use)
	assert.Contains(t, cmd.Short, "Verify")
	assert.NoError(t, cmd.Args(cmd, []string{"."}))
	assert.Error(t, cmd.Args(cmd, []string{}))
	assert.Error(t, cmd.Args(cmd, []string{"a", "b"}))
}

func TestNewRegistryVerifyCmd_RunE(t *testing.T) {
	cmd := newRegistryVerifyCmd()
	cmd.SetArgs([]string{"/nonexistent/path"})
	err := cmd.Execute()
	// Should fail because the path doesn't exist or doRegistryVerify rejects it
	assert.Error(t, err)
}
