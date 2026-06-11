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

package namespace_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/namespace"
)

func TestAll(t *testing.T) {
	assert.Equal(t, []string{
		"config", "workflow", "resource", "component", "agency",
	}, namespace.All())
}

func TestIsKnown(t *testing.T) {
	assert.True(t, namespace.IsKnown("config"))
	assert.False(t, namespace.IsKnown("bogus"))
}

func TestIsNamespacedPath(t *testing.T) {
	assert.True(t, namespace.IsNamespacedPath("config.llm.provider"))
	assert.True(t, namespace.IsNamespacedPath("workflow.settings"))
	assert.True(t, namespace.IsNamespacedPath("resource.myRes.field"))
	assert.True(t, namespace.IsNamespacedPath("component.myComp.key"))
	assert.True(t, namespace.IsNamespacedPath("agency.myAgency.key"))
	assert.False(t, namespace.IsNamespacedPath("plain"))
	assert.False(t, namespace.IsNamespacedPath(""))
}

func TestSplitPath(t *testing.T) {
	ns, rest, err := namespace.SplitPath("config.llm.provider")
	require.NoError(t, err)
	assert.Equal(t, "config", ns)
	assert.Equal(t, "llm.provider", rest)

	_, _, err = namespace.SplitPath("noprefix")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid config path")

	_, _, err = namespace.SplitPath("config.")
	require.Error(t, err)
}
