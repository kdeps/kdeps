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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeAsset(t *testing.T, dir, set, name, body string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, set), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(dir, set, name+".yaml"), []byte(body), 0o600))
}

// useProject layers dir for the test and restores the registries after.
func useProject(t *testing.T, dir string) error {
	t.Helper()
	t.Cleanup(func() {
		projectAssetsDir.Store(nil)
		ReloadAssets()
	})
	return UseProjectAssets(dir)
}

// The project layer wins over ~/.kdeps, which wins over the shipped events.
func TestProjectAssets_LayerOrder(t *testing.T) {
	isolateEventsAndPresetsHome(t)
	home := os.Getenv("HOME")
	writeAsset(t, filepath.Join(home, ".kdeps"), "events", "chat-max-rounds",
		"name: chat-max-rounds\non:\n  rounds: 9\nrun: force_answer\n")
	writeAsset(t, filepath.Join(home, ".kdeps"), "events", "judge-max-rounds",
		"name: judge-max-rounds\non:\n  rounds: 7\nrun: force_answer\n")
	initEvents()
	assert.Equal(t, 9, effectiveRounds(eventChatMaxRounds), "~/.kdeps over the shipped value")

	project := t.TempDir()
	writeAsset(t, project, "events", "chat-max-rounds",
		"name: chat-max-rounds\non:\n  rounds: 2\nrun: force_answer\n")
	writeAsset(t, project, "harness", "accuracy",
		"kind: preamble-section\norder: 10\nsummary: project summary\nbody: PROJECT ACCURACY\n")
	require.NoError(t, useProject(t, project))

	assert.Equal(t, 2, effectiveRounds(eventChatMaxRounds), "the project wins")
	assert.Equal(t, 7, effectiveRounds(eventJudgeMaxRounds), "~/.kdeps still fills what the project leaves out")
	body, ok := chatHarnessSection("accuracy")
	assert.True(t, ok)
	assert.Contains(t, body, "PROJECT ACCURACY")
}

func TestProjectAssets_BadFileFailsTheRun(t *testing.T) {
	isolateEventsAndPresetsHome(t)
	project := t.TempDir()
	writeAsset(t, project, "events", "broken", "name: [\n")
	writeAsset(t, project, "presets", "broken", "name: [\n")
	errs := ProjectAssetErrors(project)
	assert.Len(t, errs, 2)
	err := useProject(t, project)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "project harness/, events/, actions/, presets/")
	assert.Nil(t, projectAssetsDir.Load(), "a bad project is not layered")
	assert.Empty(t, ProjectAssetErrors(t.TempDir()), "no folders, no errors")
}

func TestLayered_NoHome(t *testing.T) {
	projectAssetsDir.Store(nil)
	out, errs := layered("", os.ErrNotExist, "events", loadEventsDir)
	assert.Nil(t, out)
	assert.Len(t, errs, 1)
}
