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

package chat

import (
	"context"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppendUniqueEnvVars_SkipsDuplicate(t *testing.T) {
	t.Parallel()
	seen := map[string]bool{"FOO": true}
	var result []EnvVar
	appendUniqueEnvVars(&result, seen, []EnvVar{{Name: "FOO"}, {Name: "BAR"}})
	require.Len(t, result, 1)
	assert.Equal(t, "BAR", result[0].Name)
}

func TestGenerator_Generate_RetryExhausted(t *testing.T) {
	client := &mockLLMClient{reply: "no file blocks here"}
	origRetries := maxValidationRetries
	t.Cleanup(func() { maxValidationRetries = origRetries })
	maxValidationRetries = 0

	gen := NewGenerator(client, "llama3", "", "", nil)

	_, err := gen.Generate(context.Background(), []Turn{{Role: "user", Content: "test"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "retry loop exhausted")
}

func TestHTTPLLMClient_DoRequest_MarshalError(t *testing.T) {
	orig := jsonMarshal
	t.Cleanup(func() { jsonMarshal = orig })
	jsonMarshal = func(_ any) ([]byte, error) {
		return nil, errors.New("marshal failed")
	}

	c := &HTTPLLMClient{httpClient: http.DefaultClient}
	_, err := c.doRequest(context.Background(), "http://x", "", map[string]interface{}{"a": 1})
	require.Error(t, err)
}

func TestHTTPLLMClient_DoRequest_ReadBodyError(t *testing.T) {
	t.Parallel()
	c := &HTTPLLMClient{httpClient: &http.Client{
		Transport: roundTripperFunc(func(_ *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(&failReader{}),
			}, nil
		}),
	}}

	_, err := c.doRequest(context.Background(), "http://x", "", map[string]interface{}{"a": 1})
	require.Error(t, err)
}

func TestLoadSession_HomeDirError(t *testing.T) {
	orig := osUserHomeDir
	t.Cleanup(func() { osUserHomeDir = orig })
	osUserHomeDir = func() (string, error) {
		return "", errors.New("no home")
	}

	_, err := LoadSession("id")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "could not determine home directory")
}

func TestValidateResourceFile_UnmarshalError(t *testing.T) {
	orig := yamlUnmarshalToMap
	t.Cleanup(func() { yamlUnmarshalToMap = orig })
	yamlUnmarshalToMap = func(_ []byte, _ *map[string]interface{}) error {
		return errors.New("map unmarshal failed")
	}

	ids := map[string]bool{}
	var errs []string
	validateResourceFile("res.yaml", "actionId: main\n", ids, &errs)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0], "invalid YAML")
}

type failReader struct{}

func (failReader) Read(_ []byte) (int, error) { return 0, errors.New("read failed") }

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
