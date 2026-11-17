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

package cmd_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v3/cmd"
	"github.com/kdeps/kdeps/v3/pkg/agent"
)

func TestTheme_FlagAdvertisedInHelp(t *testing.T) {
	rootCmd := cmd.NewRootCmd()
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--help"})
	require.NoError(t, rootCmd.Execute())

	help := out.String()
	assert.Contains(t, help, "--theme")
	assert.Contains(t, help, "normal")
}

func TestTheme_OnlyNormalApplies(t *testing.T) {
	t.Cleanup(func() { agent.SetTheme("normal") })
	require.False(t, agent.SetTheme("black"))
	require.False(t, agent.SetTheme("vim"))
	assert.False(t, agent.StealthActive())
	assert.Equal(t, "normal", agent.CurrentThemeName())
	require.True(t, agent.SetTheme("normal"))
	assert.Equal(t, "normal", agent.CurrentThemeName())
}
