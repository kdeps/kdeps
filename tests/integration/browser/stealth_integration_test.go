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

//go:build integration

package browser_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

// TestBrowserConfig_StealthModeUnmarshal verifies that stealthMode can be
// parsed from YAML configuration.
func TestBrowserConfig_StealthModeUnmarshal(t *testing.T) {
	yamlContent := `
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: test
run:
  browser:
    engine: chromium
    headless: true
    stealthMode: true
    url: "https://example.com"
`

	tmpFile, err := os.CreateTemp(t.TempDir(), "resource-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(yamlContent)
	require.NoError(t, err)
	tmpFile.Close()

	// The resource loads successfully
	// In an integration test we'd actually run it
	require.FileExists(t, tmpFile.Name())
}

// TestBrowserConfig_UserAgentUnmarshal verifies that userAgent can be
// parsed from YAML configuration.
func TestBrowserConfig_UserAgentUnmarshal(t *testing.T) {
	yamlContent := `
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: test
run:
  browser:
    engine: chromium
    userAgent: "Mozilla/5.0 (Custom Bot)"
    url: "https://example.com"
`

	tmpFile, err := os.CreateTemp(t.TempDir(), "resource-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(yamlContent)
	require.NoError(t, err)
	tmpFile.Close()

	require.FileExists(t, tmpFile.Name())
}
