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

package config

import (
	"bufio"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeChatDefaults_MaxOutputBytes(t *testing.T) {
	dst := &ChatDefaults{}
	src := &ChatDefaults{MaxOutputBytes: 4096}
	mergeChatDefaults(dst, src)
	assert.Equal(t, int64(4096), dst.MaxOutputBytes)
}

func TestMergeHTTPDefaults_MaxResponseBytes(t *testing.T) {
	dst := &HTTPDefaults{}
	src := &HTTPDefaults{MaxResponseBytes: 8192}
	mergeHTTPDefaults(dst, src)
	assert.Equal(t, int64(8192), dst.MaxResponseBytes)
}

func TestMergePythonDefaults_MaxOutputBytes(t *testing.T) {
	dst := &PythonDefaults{}
	src := &PythonDefaults{MaxOutputBytes: 2048}
	mergePythonDefaults(dst, src)
	assert.Equal(t, int64(2048), dst.MaxOutputBytes)
}

func TestMergeExecDefaults_MaxOutputBytes(t *testing.T) {
	dst := &ExecDefaults{}
	src := &ExecDefaults{MaxOutputBytes: 1024}
	mergeExecDefaults(dst, src)
	assert.Equal(t, int64(1024), dst.MaxOutputBytes)
}

func TestLoad_PathError(t *testing.T) {
	origHome := osUserHomeDir
	t.Cleanup(func() { osUserHomeDir = origHome })
	osUserHomeDir = func() (string, error) {
		return "", errors.New("no home")
	}

	cfg, err := load()
	require.NoError(t, err)
	assert.NotNil(t, cfg)
}

func TestValidate_PathEmpty(t *testing.T) {
	origHome := osUserHomeDir
	t.Cleanup(func() { osUserHomeDir = origHome })
	osUserHomeDir = func() (string, error) {
		return "", errors.New("no home")
	}

	warnings := (&Config{}).Validate("")
	assert.Empty(t, warnings)
}

func TestRunConfigFileCheck_EmptyPath(t *testing.T) {
	origHome := osUserHomeDir
	t.Cleanup(func() { osUserHomeDir = origHome })
	osUserHomeDir = func() (string, error) {
		return "", errors.New("no home")
	}

	r := &doctorRunner{healthy: true}
	r.configFile()
	require.NotEmpty(t, r.checks)
	assert.Equal(t, HealthFail, r.checks[0].Status)
}

func TestRunAgentsCheck_AgentsDirError(t *testing.T) {
	origHome := osUserHomeDir
	t.Cleanup(func() { osUserHomeDir = origHome })
	osUserHomeDir = func() (string, error) {
		return "", errors.New("no home")
	}

	r := &doctorRunner{healthy: true}
	r.agents(&Config{})
	require.NotEmpty(t, r.checks)
	assert.Equal(t, HealthWarn, r.checks[0].Status)
}

func TestRunCriticalEnvCheck_WarnThreshold(t *testing.T) {
	// Leave exactly 3 of 6 critical vars unset (at envWarnThreshold).
	t.Setenv("OLLAMA_HOST", "http://localhost:11434")
	t.Setenv("KDEPS_DEFAULT_BACKEND", "ollama")
	t.Setenv("KDEPS_LLM_MODELS", "gpt-4")

	r := &doctorRunner{healthy: true}
	r.criticalEnv()
	require.NotEmpty(t, r.checks)
	assert.Equal(t, HealthWarn, r.checks[0].Status)
	assert.Contains(t, r.checks[0].Message, "missing:")
}

func TestBootstrapInteractive_ConfigureProviderError(t *testing.T) {
	origReadSecret := readSecretFunc
	t.Cleanup(func() { readSecretFunc = origReadSecret })
	readSecretFunc = func(_ *bufio.Reader) (string, error) {
		return "", errors.New("secret read failed")
	}

	reader := bufio.NewReader(strings.NewReader("2\n"))
	var out testWriter
	err := bootstrapInteractive(&out, reader, "/tmp/test-config.yaml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "secret read failed")
}
