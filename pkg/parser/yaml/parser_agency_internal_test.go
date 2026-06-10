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

package yaml

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAutoDiscoverAgents_SkipNonKdeps(t *testing.T) {
	dir := t.TempDir()
	agentsDir := filepath.Join(dir, "agents")
	require.NoError(t, os.MkdirAll(agentsDir, 0o755))

	// Place a non-.kdeps file — it should be silently skipped.
	require.NoError(t, os.WriteFile(
		filepath.Join(agentsDir, "readme.txt"),
		[]byte("hello"), 0o644,
	))

	p := newWhiteboxParser()
	paths, err := p.autoDiscoverAgents(dir)
	require.NoError(t, err)
	assert.Empty(t, paths, "no .kdeps files should be discovered")
}
