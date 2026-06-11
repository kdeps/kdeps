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

package executor

import (
	kdeps_debug "github.com/kdeps/kdeps/v2/pkg/debug"
	"github.com/kdeps/kdeps/v2/pkg/namespace"
)

func (ctx *ExecutionContext) Get(name string, typeHint ...string) (interface{}, error) {
	kdeps_debug.Log("enter: Get")
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()

	// If type hint is provided, use it directly.
	if len(typeHint) > 0 {
		return ctx.getByType(name, typeHint[0])
	}

	// Namespace-prefixed paths are resolved via config structs first.
	// Fall back to auto-detection if resolution fails (preserves backward compat
	// for names like "workflow.name" that are also metadata fields).
	if namespace.IsNamespacedPath(name) {
		if val, err := ctx.GetConfigField(name); err == nil {
			return val, nil
		}
	}

	// Auto-detection priority chain
	return ctx.getWithAutoDetection(name)
}
