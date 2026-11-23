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

package knowledge

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// vaultDirs reads the layout when Init runs, not at package load, so a
// first-run install in the same process still yields the folder names.
func TestVaultDirs_ReadAtCall(t *testing.T) {
	f := Layout().Folders
	assert.Equal(t, []string{f.Projects, f.Shared, f.Inbox, f.Archive}, vaultDirs())
	assert.NotEmpty(t, vaultDirs()[0])
}
