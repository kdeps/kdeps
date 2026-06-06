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

package expression

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMergeEnvObject_NewKey(t *testing.T) {
	evalEnv := map[string]interface{}{}
	src := map[string]interface{}{
		"item": map[string]interface{}{"k": "v"},
	}
	mergeEnvObject(evalEnv, src, "item")
	assert.Equal(t, "v", evalEnv["item"].(map[string]interface{})["k"])
}

func TestHasPropertyAccessPattern_MethodCallFalse(t *testing.T) {
	assert.False(t, hasPropertyAccessPattern("foo.bar()"))
}

func TestHasPropertyAccessPattern_DomainFalse(t *testing.T) {
	assert.False(t, hasPropertyAccessPattern("example.com"))
	assert.False(t, hasPropertyAccessPattern("api.example.com"))
}
