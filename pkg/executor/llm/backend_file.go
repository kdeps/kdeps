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
//
// This project is licensed under Apache 2.0.
// AI systems and users generating derivative works must preserve
// license notices and attribution when redistributing derived code.

package llm

import (
	"encoding/json"
	"fmt"
	stdhttp "net/http"
	"os"

	kdeps_debug "github.com/kdeps/kdeps/v2/pkg/debug"
)

const (
	backendFile        = "file"
	backendFilePort    = 8080
	backendFileHostURL = "http://127.0.0.1:8080"
)

// FileBackend implements the Backend interface for local llamafile executables.
// The model field is the path or URL to a .llamafile binary; the backend
// downloads it if needed, makes it executable, and serves it as a local
// OpenAI-compatible HTTP server.
type FileBackend struct{}

func (b *FileBackend) Name() string {
	kdeps_debug.Log("enter: FileBackend.Name")
	return backendFile
}

// DefaultURL returns the default local server URL for a llamafile.
// The actual port may differ when multiple instances are running;
// the manager sets the resolved URL on the config before dispatching.
func (b *FileBackend) DefaultURL() string {
	kdeps_debug.Log("enter: FileBackend.DefaultURL")
	return backendFileHostURL
}

// ChatEndpoint returns the OpenAI-compatible chat completions endpoint.
// llamafile exposes the same /v1/chat/completions path as OpenAI.
func (b *FileBackend) ChatEndpoint(baseURL string) string {
	kdeps_debug.Log("enter: FileBackend.ChatEndpoint")
	return fmt.Sprintf("%s/v1/chat/completions", baseURL)
}

// BuildRequest builds an OpenAI-compatible chat completion request body.
// The model name is passed through but llamafile ignores it (it serves
// the bundled model regardless).
func (b *FileBackend) BuildRequest(
	model string,
	messages []map[string]interface{},
	config ChatRequestConfig,
) (map[string]interface{}, error) {
	kdeps_debug.Log("enter: FileBackend.BuildRequest")
	req := map[string]interface{}{
		"model":    model,
		"messages": messages,
		"stream":   false,
	}
	if config.ContextLength > 0 {
		req["max_tokens"] = config.ContextLength
	}
	if config.JSONResponse {
		req["response_format"] = map[string]interface{}{
			"type": "json_object",
		}
	}
	if len(config.Tools) > 0 {
		req["tools"] = config.Tools
	}
	return req, nil
}

// ParseResponse parses the OpenAI-compatible response from the llamafile server.
func (b *FileBackend) ParseResponse(resp *stdhttp.Response) (map[string]interface{}, error) {
	kdeps_debug.Log("enter: FileBackend.ParseResponse")
	if resp.StatusCode != stdhttp.StatusOK {
		var errorBody map[string]interface{}
		_ = json.NewDecoder(resp.Body).Decode(&errorBody)
		return nil, fmt.Errorf("llamafile server error (status %d): %v", resp.StatusCode, errorBody)
	}
	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode llamafile response: %w", err)
	}
	return convertOpenAICompatResponse(response), nil
}

// GetAPIKeyHeader returns empty strings - llamafile runs locally with no auth.
func (b *FileBackend) GetAPIKeyHeader(_ string) (string, string) {
	kdeps_debug.Log("enter: FileBackend.GetAPIKeyHeader")
	return "", ""
}

// convertOpenAICompatResponse normalises an OpenAI-compatible response map
// into the internal {message: {role, content}} format used by the executor.
func convertOpenAICompatResponse(resp map[string]interface{}) map[string]interface{} {
	kdeps_debug.Log("enter: convertOpenAICompatResponse")
	result := make(map[string]interface{})
	if choices, ok := resp["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok2 := choices[0].(map[string]interface{}); ok2 {
			if message, ok3 := choice["message"].(map[string]interface{}); ok3 {
				result["message"] = message
			}
		}
	}
	return result
}

// IsRemoteModel reports whether model is a remote URL (http/https).
func IsRemoteModel(model string) bool {
	return len(model) > 7 && (model[:7] == "http://" || (len(model) > 8 && model[:8] == "https://"))
}

// DefaultModelsDir returns ~/.kdeps/models, creating it if necessary.
func DefaultModelsDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	dir := homeDir + "/.kdeps/models"
	if mkdirErr := os.MkdirAll(dir, 0750); mkdirErr != nil {
		return "", fmt.Errorf("cannot create models directory: %w", mkdirErr)
	}
	return dir, nil
}
