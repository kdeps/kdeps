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

package llm

import (
	"sync/atomic"
)

// freeModelHop reports whether a failed OpenRouter free-model call should
// move to another free id: a provider 429 or a provider refusal of that id.
// Agent mode installs it from its openrouter-* events.
//
//nolint:gochecknoglobals // installed once at startup, read on every chat
var freeModelHop atomic.Pointer[func(error) bool]

// SetFreeModelHop installs which errors move a free-model call to the next id.
func SetFreeModelHop(hop func(error) bool) { freeModelHop.Store(&hop) }

func hopsFreeModel(err error) bool {
	f := freeModelHop.Load()
	return err != nil && f != nil && (*f)(err)
}

// IsOpenRouterFreeID reports whether model is a catalog free id.
func IsOpenRouterFreeID(model string) bool {
	for _, id := range OpenRouterFreeModelIDs() {
		if id == model {
			return true
		}
	}
	return false
}

// NextOpenRouterFreeID returns the next catalog free id after current,
// skipping ids in skip. It wraps once; current itself is never returned.
func NextOpenRouterFreeID(current string, skip map[string]struct{}) (string, bool) {
	ids := OpenRouterFreeModelIDs()
	if len(ids) < 2 || !IsOpenRouterFreeID(current) {
		return "", false
	}
	start := 0
	for i, id := range ids {
		if id == current {
			start = i
			break
		}
	}
	for step := 1; step < len(ids); step++ {
		id := ids[(start+step)%len(ids)]
		if _, blocked := skip[id]; blocked {
			continue
		}
		return id, true
	}
	return "", false
}

// freeModelCall is a workflow chat's call on an OpenRouter free id, as agent
// mode makes it: a provider 429 or refusal moves the call to the next free
// id, and later rounds stay on the id that answered. newCall builds the call
// for one model. Any other backend or model calls newCall(model) unchanged.
func freeModelCall(backend, model string, newCall func(model string) chatCall) chatCall {
	if backend != openrouterRegistryItem || !IsOpenRouterFreeID(model) {
		return newCall(model)
	}
	current := model
	return func(messages []map[string]any, requestConfig ChatRequestConfig) (map[string]any, error) {
		tried := map[string]struct{}{}
		for {
			resp, err := newCall(current)(messages, requestConfig)
			if !hopsFreeModel(err) {
				return resp, err
			}
			tried[current] = struct{}{}
			next, ok := NextOpenRouterFreeID(current, tried)
			if !ok {
				return resp, err
			}
			current = next
		}
	}
}
