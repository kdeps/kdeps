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

package yaml_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/parser/yaml"
)

// TestParseResource_Jinja2Preprocessing verifies that Jinja2 control tags in a
// resource YAML are rendered before the YAML is parsed.
// Kdeps runtime API calls like {{ get('url') }} are automatically preserved
// without requiring {% raw %} blocks.
func TestParseResource_Jinja2Preprocessing(t *testing.T) {
	t.Setenv("ENABLE_HTTP", "true")

	resourceYAML := `actionId: fetchData
name: Fetch Data
{% if env.ENABLE_HTTP == 'true' %}
httpClient:
  method: GET
  url: "{{ get('url') }}"
{% endif %}
`
	tmpDir := t.TempDir()
	resourcePath := filepath.Join(tmpDir, "resource.yaml")
	require.NoError(t, os.WriteFile(resourcePath, []byte(resourceYAML), 0600))

	parser := yaml.NewParser(&mockSchemaValidator{}, &mockExprParser{})
	res, err := parser.ParseResource(resourcePath)
	require.NoError(t, err)
	require.NotNil(t, res)

	assert.Equal(t, "fetchData", res.ActionID)
	// The runtime expression is auto-protected by PreprocessYAML and preserved as-is.
	require.NotNil(t, res.HTTPClient)
	assert.Equal(t, "{{ get('url') }}", res.HTTPClient.URL)
}

// TestParseResource_RuntimeExpressionsPreserved ensures that {{ }} runtime
// API expressions in a resource YAML file are preserved verbatim through Jinja2
// preprocessing (auto-protected) and available for runtime evaluation.
func TestParseResource_RuntimeExpressionsPreserved(t *testing.T) {
	resourceYAML := `actionId: response
name: API Response
httpClient:
  method: GET
  url: "{{ get('url') }}"
`
	tmpDir := t.TempDir()
	resourcePath := filepath.Join(tmpDir, "resource.yaml")
	require.NoError(t, os.WriteFile(resourcePath, []byte(resourceYAML), 0600))

	parser := yaml.NewParser(&mockSchemaValidator{}, &mockExprParser{})
	res, err := parser.ParseResource(resourcePath)
	require.NoError(t, err)
	require.NotNil(t, res)

	// The runtime expression is auto-protected and preserved verbatim after Jinja2 preprocessing.
	require.NotNil(t, res.HTTPClient)
	assert.Equal(t, "{{ get('url') }}", res.HTTPClient.URL)
}

// TestParseResource_MixedJinja2AndRuntimeExpressions verifies that a resource YAML
// file can mix Jinja2 control flow ({% if %}) with multiple kdeps runtime API calls
// without any manual {% raw %} blocks.
func TestParseResource_MixedJinja2AndRuntimeExpressions(t *testing.T) {
	t.Setenv("ENABLE_CALL", "yes")

	resourceYAML := `actionId: callAPI
name: Call API
{% if env.ENABLE_CALL == 'yes' %}
httpClient:
  method: GET
  url: "{{ get('url') }}"
  headers:
    X-Request-ID: "{{ info('request_id') }}"
{% endif %}
`
	tmpDir := t.TempDir()
	resourcePath := filepath.Join(tmpDir, "resource.yaml")
	require.NoError(t, os.WriteFile(resourcePath, []byte(resourceYAML), 0600))

	parser := yaml.NewParser(&mockSchemaValidator{}, &mockExprParser{})
	res, err := parser.ParseResource(resourcePath)
	require.NoError(t, err)
	require.NotNil(t, res)

	require.NotNil(t, res.HTTPClient)
	// Both kdeps runtime expressions are auto-protected and preserved verbatim.
	assert.Equal(t, "{{ get('url') }}", res.HTTPClient.URL)
	assert.Equal(t, "{{ info('request_id') }}", res.HTTPClient.Headers["X-Request-ID"])
}
