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

package events

import (
	"encoding/json"
	"io"
	"sync"
)

// Emitter receives execution events from the workflow engine.
type Emitter interface {
	Emit(e Event)
	Close()
}

// NopEmitter discards all events. Used when --events is not set.
type NopEmitter struct{}

func (NopEmitter) Emit(Event) {}
func (NopEmitter) Close()     {}

// NDJSONEmitter writes one JSON object per line to w.
// It is safe for concurrent use.
type NDJSONEmitter struct {
	mu  sync.Mutex
	enc *json.Encoder
	w   io.Writer
}

// NewNDJSONEmitter returns an emitter that writes NDJSON to w.
// Typically w is os.Stderr so stdout stays clean for workflow output.
func NewNDJSONEmitter(w io.Writer) *NDJSONEmitter {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return &NDJSONEmitter{enc: enc, w: w}
}

func (e *NDJSONEmitter) Emit(ev Event) {
	e.mu.Lock()
	defer e.mu.Unlock()
	_ = e.enc.Encode(ev)
}

func (e *NDJSONEmitter) Close() {}

// MultiEmitter fans out every event to all child emitters.
type MultiEmitter struct {
	emitters []Emitter
}

// NewMultiEmitter returns an emitter that fans out to all provided emitters.
func NewMultiEmitter(emitters ...Emitter) *MultiEmitter {
	return &MultiEmitter{emitters: emitters}
}

func (m *MultiEmitter) Emit(e Event) {
	for _, em := range m.emitters {
		em.Emit(e)
	}
}

func (m *MultiEmitter) Close() {
	for _, em := range m.emitters {
		em.Close()
	}
}

// ChanEmitter sends events to a channel. Useful for SSE or testing.
type ChanEmitter struct {
	ch chan Event
}

// NewChanEmitter returns an emitter backed by a buffered channel.
func NewChanEmitter(bufSize int) *ChanEmitter {
	return &ChanEmitter{ch: make(chan Event, bufSize)}
}

func (c *ChanEmitter) Emit(e Event) {
	select {
	case c.ch <- e:
	default:
		// Drop if channel full — never block the engine.
	}
}

func (c *ChanEmitter) Close() { close(c.ch) }

// C returns the underlying channel for consumption.
func (c *ChanEmitter) C() <-chan Event { return c.ch }
