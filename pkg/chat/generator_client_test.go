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

	"github.com/stretchr/testify/require"
)

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
