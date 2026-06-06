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

package python

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMaxOutputBytesFromEnv_Invalid(t *testing.T) {
	t.Setenv("KDEPS_PYTHON_MAX_OUTPUT_BYTES", "not-a-number")
	assert.Equal(t, int64(0), maxOutputBytesFromEnv())
}

func TestMaxOutputBytesFromEnv_NonPositive(t *testing.T) {
	t.Setenv("KDEPS_PYTHON_MAX_OUTPUT_BYTES", "-5")
	assert.Equal(t, int64(0), maxOutputBytesFromEnv())
}
