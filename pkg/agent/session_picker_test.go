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

package agent

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPickStartupSession_EmptyStoreReturnsNew(t *testing.T) {
	store := NewSessionStore(t.TempDir())
	t.Cleanup(func() { _ = store.Close() })
	assert.Equal(t, "", PickStartupSession(store))
	assert.Equal(t, "", PickStartupSession(nil))
}

// Non-interactive terminal: RunListPicker returns "" so the picker degrades to
// "start a new session" without hanging.
func TestPickStartupSession_NonInteractive(t *testing.T) {
	store := NewSessionStore(t.TempDir())
	t.Cleanup(func() { _ = store.Close() })
	s := NewSession(0)
	s.Append("build the thing", "ok")
	if _, err := store.Save(s); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "", PickStartupSession(store))
}

func TestHumanizeSince(t *testing.T) {
	now := time.Now().UnixMilli()
	assert.Equal(t, "just now", humanizeSince(now))
	assert.Equal(t, "5m ago", humanizeSince(now-5*60_000))
	assert.Equal(t, "3h ago", humanizeSince(now-3*3600_000))
	assert.Equal(t, "2d ago", humanizeSince(now-2*24*3600_000))
	assert.Equal(t, "unknown", humanizeSince(0))
}

func TestSessionPickerOnDelete_RefusesSentinel(t *testing.T) {
	store := NewSessionStore(t.TempDir())
	t.Cleanup(func() { _ = store.Close() })

	removed, err := sessionPickerOnDelete(store)(newSessionSentinel)
	assert.False(t, removed, "the 'start new session' row must never be deletable")
	assert.NoError(t, err)
}

func TestSessionPickerOnDelete_DeletesRealSession(t *testing.T) {
	store := NewSessionStore(t.TempDir())
	t.Cleanup(func() { _ = store.Close() })
	s := NewSession(0)
	s.Append("hi", "hello")
	id, err := store.Save(s)
	assert.NoError(t, err)

	removed, err := sessionPickerOnDelete(store)(id)
	assert.True(t, removed)
	assert.NoError(t, err)

	_, loadErr := store.Load(id)
	assert.Error(t, loadErr, "the session must actually be gone from the store")
}

func TestSessionPickerOnDelete_PropagatesStoreError(t *testing.T) {
	store := NewSessionStore(t.TempDir())
	t.Cleanup(func() { _ = store.Close() })

	removed, err := sessionPickerOnDelete(store)("does-not-exist")
	assert.False(t, removed)
	assert.Error(t, err)
}

func TestPlural(t *testing.T) {
	assert.Equal(t, "", plural(1))
	assert.Equal(t, "s", plural(0))
	assert.Equal(t, "s", plural(2))
}
