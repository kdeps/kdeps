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

package config_test

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/config"
)

// A missing connection prompted at run time must be written to config.yaml and
// then be loadable via the normal config path.
func TestPromptAndSaveConnection_SQLRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	t.Setenv("KDEPS_CONFIG_PATH", path)

	in := bufio.NewReader(strings.NewReader("postgres://user:pass@host/db\n"))
	var out bytes.Buffer
	require.NoError(t, config.PromptAndSaveConnection(config.ConnKindSQL, "analytics", &out, in))

	cfg, err := config.LoadStruct()
	require.NoError(t, err)
	require.True(t, config.HasConnection(cfg, config.ConnKindSQL, "analytics"))
	assert.Equal(t, "postgres://user:pass@host/db", cfg.SQLConnections["analytics"].Connection)
}

// Saving a second connection must not drop the first.
func TestPromptAndSaveConnection_AppendsWithoutClobbering(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	t.Setenv("KDEPS_CONFIG_PATH", path)

	var out bytes.Buffer
	require.NoError(t, config.PromptAndSaveConnection(config.ConnKindSQL, "first", &out,
		bufio.NewReader(strings.NewReader("postgres://h/first\n"))))
	require.NoError(t, config.PromptAndSaveConnection(config.ConnKindSQL, "second", &out,
		bufio.NewReader(strings.NewReader("postgres://h/second\n"))))

	cfg, err := config.LoadStruct()
	require.NoError(t, err)
	assert.True(t, config.HasConnection(cfg, config.ConnKindSQL, "first"))
	assert.True(t, config.HasConnection(cfg, config.ConnKindSQL, "second"))
}

// A named connection supplied entirely via environment variables must load
// into the Config struct so executors and the pre-flight see it.
func TestConnectionEnvOverride_RoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	t.Setenv("KDEPS_CONFIG_PATH", path)
	t.Setenv("KDEPS_SQL_CONNECTIONS_ALERTS_CONNECTION", "postgres://u:p@h/db")

	cfg, err := config.LoadStruct()
	require.NoError(t, err)
	require.True(t, config.HasConnection(cfg, config.ConnKindSQL, "alerts"))
	assert.Equal(t, "postgres://u:p@h/db", cfg.SQLConnections["alerts"].Connection)
	assert.True(t, config.ConnectionInEnv(config.ConnKindSQL, "alerts"))
}

// A missing cloud LLM key prompted at run time must be written to config.yaml
// (and exported) so the chat resource can authenticate.
func TestPromptAndSaveLLMKey_RoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	t.Setenv("KDEPS_CONFIG_PATH", path)
	t.Setenv("DEEPSEEK_API_KEY", "")

	var out bytes.Buffer
	saved, err := config.PromptAndSaveLLMKey("deepseek", &out,
		bufio.NewReader(strings.NewReader("sk-deepseek-rt\n")))
	require.NoError(t, err)
	require.True(t, saved)

	cfg, err := config.LoadStruct()
	require.NoError(t, err)
	assert.True(t, config.HasLLMKey(cfg, "deepseek"))
	assert.Equal(t, "sk-deepseek-rt", os.Getenv("DEEPSEEK_API_KEY"))
}

// A missing apiServer token prompted at run time must be written to config.yaml.
func TestPromptAndSaveAPIToken_RoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	t.Setenv("KDEPS_CONFIG_PATH", path)
	t.Setenv("KDEPS_API_AUTH_TOKEN", "")

	var out bytes.Buffer
	token, err := config.PromptAndSaveAPIToken(&out,
		bufio.NewReader(strings.NewReader("integration-token\n")))
	require.NoError(t, err)
	assert.Equal(t, "integration-token", token)

	cfg, err := config.LoadStruct()
	require.NoError(t, err)
	assert.True(t, config.HasAPIToken(cfg))
}
