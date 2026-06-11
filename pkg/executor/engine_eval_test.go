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

package executor

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAtTime_RFC3339(t *testing.T) {
	t.Parallel()
	got, err := parseAtTime("2026-03-15T10:00:00Z")
	require.NoError(t, err)
	assert.Equal(t, 2026, got.Year())
	assert.Equal(t, time.March, got.Month())
	assert.Equal(t, 15, got.Day())
}

func TestParseAtTime_TimeOfDay(t *testing.T) {
	t.Parallel()
	got, err := parseAtTime("23:59:59")
	require.NoError(t, err)
	assert.Equal(t, 23, got.Hour())
	assert.Equal(t, 59, got.Minute())
}

func TestParseAtTime_DateOnly(t *testing.T) {
	t.Parallel()
	got, err := parseAtTime("2026-12-25")
	require.NoError(t, err)
	assert.Equal(t, 2026, got.Year())
	assert.Equal(t, time.December, got.Month())
	assert.Equal(t, 25, got.Day())
	assert.Equal(t, 0, got.Hour())
}

func TestParseAtTime_Invalid(t *testing.T) {
	t.Parallel()
	_, err := parseAtTime("not-a-time")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unrecognised at time format")
}
