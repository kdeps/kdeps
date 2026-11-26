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

package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v3/pkg/secrets"
	kdepstools "github.com/kdeps/kdeps/v3/pkg/tools"
)

func notionTools() []string { return []string{"notion_get", "notion_post", "notion_patch"} }

// noNotionToken clears every source of the Notion token: env and vault.
func noNotionToken(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("NOTION_TOKEN", "")
	t.Setenv("NOTION_API_KEY", "")
	t.Setenv(secrets.EnvPassword, "")
	t.Setenv(secrets.EnvPasswordFile, "")
	secrets.UseProject("")
}

func TestNotionTools_AssetsDeclareThem(t *testing.T) {
	for _, id := range notionTools() {
		d, ok := toolDefinitions[id]
		require.True(t, ok, id)
		require.NotNil(t, d.HTTP, "%s has an http: block", id)
		require.NoError(t, d.HTTP.Validate(), id)
	}
}

func TestNotionTools_NotRegisteredWithoutToken(t *testing.T) {
	noNotionToken(t)
	reg := kdepstools.NewRegistry()
	RegisterBuiltinTools(context.Background(), reg)
	for _, id := range notionTools() {
		assert.Nil(t, reg.Get(id), id)
	}
}

func TestNotionGet_Declared(t *testing.T) {
	noNotionToken(t)
	api := newFakeAPI(t, http.StatusOK, `{"object":"page","id":"p1"}`)
	tool := declaredTool(t, "notion_get", map[string]string{"NOTION_API_KEY": "secret_fallback"})

	out, err := tool.Execute(map[string]any{"path": "/blocks/b1/children?page_size=50"})
	require.NoError(t, err)
	assert.JSONEq(t, `{"object":"page","id":"p1"}`, out)

	req := api.last(t)
	assert.Equal(t, http.MethodGet, req.Method)
	assert.Equal(t, "api.notion.com", req.Host)
	assert.Equal(t, "/v1/blocks/b1/children", req.Path)
	assert.Equal(t, "page_size=50", req.RawQuery)
	assert.Equal(t, "Bearer secret_fallback", req.Header.Get("Authorization"), "falls back to NOTION_API_KEY")
	assert.Equal(t, "2022-06-28", req.Header.Get("Notion-Version"))

	_, err = tool.Execute(map[string]any{})
	require.ErrorContains(t, err, "notion_get: path is required")
}

func TestNotionPostAndPatch_Declared(t *testing.T) {
	for _, tc := range []struct{ id, method, path, body string }{
		{"notion_post", http.MethodPost, "databases/d1/query", `{"page_size":10}`},
		{"notion_patch", http.MethodPatch, "pages/p1", `{"archived":true}`},
	} {
		t.Run(tc.id, func(t *testing.T) {
			noNotionToken(t)
			api := newFakeAPI(t, http.StatusOK, `{"object":"list"}`)
			tool := declaredTool(t, tc.id, map[string]string{"NOTION_TOKEN": "ntn_env"})

			var body map[string]any
			require.NoError(t, json.Unmarshal([]byte(tc.body), &body))
			out, err := tool.Execute(map[string]any{"path": tc.path, "body": body})
			require.NoError(t, err)
			assert.JSONEq(t, `{"object":"list"}`, out)

			req := api.last(t)
			assert.Equal(t, tc.method, req.Method)
			assert.Equal(t, "api.notion.com", req.Host)
			assert.Equal(t, "/v1/"+tc.path, req.Path)
			assert.Equal(t, "Bearer ntn_env", req.Header.Get("Authorization"))
			assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
			assert.JSONEq(t, tc.body, req.Body)

			_, err = tool.Execute(map[string]any{"path": tc.path})
			require.ErrorContains(t, err, tc.id+": body is required")
		})
	}
}

func TestNotionGet_TokenFromVault(t *testing.T) {
	noNotionToken(t)
	t.Cleanup(secrets.SetWorkFactorForTesting(10))
	proj := t.TempDir()
	require.NoError(t, secrets.Write(secrets.ProjectPath(proj), "pw", map[string]string{"notion_token": "ntn_vault"}))
	secrets.UseProject(proj)
	t.Cleanup(func() { secrets.UseProject("") })
	t.Setenv(secrets.EnvPassword, "pw")

	api := newFakeAPI(t, http.StatusOK, `{"object":"user"}`)
	tool := declaredTool(t, "notion_get", nil)
	_, err := tool.Execute(map[string]any{"path": "users/me"})
	require.NoError(t, err)
	assert.Equal(t, "Bearer ntn_vault", api.last(t).Header.Get("Authorization"))
}
