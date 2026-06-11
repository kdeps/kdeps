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

package targz_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/archive/targz"
)

func TestSafeJoin(t *testing.T) {
	dir := t.TempDir()
	path, err := targz.SafeJoin(dir, "workflow.yaml")
	require.NoError(t, err)
	assert.Contains(t, path, "workflow.yaml")

	_, err = targz.SafeJoin(dir, "../escape")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid archive path")
}

func TestResolveTarget_AbsSkip(t *testing.T) {
	dir := t.TempDir()
	opts := targz.RegistryOptions()

	_, skip, err := targz.ResolveTarget(dir, ".", opts)
	require.NoError(t, err)
	assert.True(t, skip)

	target, skip, err := targz.ResolveTarget(dir, "agent/workflow.yaml", opts)
	require.NoError(t, err)
	assert.False(t, skip)
	assert.Contains(t, target, "workflow.yaml")
}
