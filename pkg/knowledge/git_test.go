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

package knowledge

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetBranch(t *testing.T) {
	v, card, _ := searchFixture(t)
	require.NoError(t, card.SetBranch("feat/a"))
	assert.Equal(t, "feat/a", card.Branch())
	require.NoError(t, card.SetBranch("feat/a"), "same branch: unchanged")
	require.NoError(t, card.SetBranch("feat/b"))
	data, _ := os.ReadFile(card.File)
	assert.Equal(t, 1, strings.Count(string(data), "| branch | "), "one branch row")
	assert.Contains(t, string(data), "| branch | feat/b |")
	assert.NotNil(t, v.CardForDir(card.Places()[0]), "a branch row is not a place")
	require.NoError(t, card.SetBranch(""))
	assert.Empty(t, card.Branch())
}
