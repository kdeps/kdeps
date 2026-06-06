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

package browser

import (
	"fmt"
	"strings"

	kdeps_debug "github.com/kdeps/kdeps/v2/pkg/debug"
	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/executor"
	"github.com/kdeps/kdeps/v2/pkg/parser/expression"
)

// ─── helpers ──────────────────────────────────────────────────────────────────

func failAction(base map[string]interface{}, msg string) map[string]interface{} {
	kdeps_debug.Log("enter: failAction")
	base["success"] = false
	base["error"] = msg
	return base
}

func errorResult(err error, sessionID string, actionResults []interface{}) map[string]interface{} {
	kdeps_debug.Log("enter: errorResult")
	res := map[string]interface{}{
		"success":   false,
		"error":     err.Error(),
		"sessionId": sessionID,
	}
	if actionResults != nil {
		res["actionResults"] = actionResults
	}
	return res
}

func resolveAction(a domain.BrowserAction, ctx *executor.ExecutionContext) domain.BrowserAction {
	kdeps_debug.Log("enter: resolveAction")
	a.Selector = evaluateText(a.Selector, ctx)
	a.Value = evaluateText(a.Value, ctx)
	a.Script = evaluateText(a.Script, ctx)
	a.URL = evaluateText(a.URL, ctx)
	a.Wait = evaluateText(a.Wait, ctx)
	a.OutputFile = evaluateText(a.OutputFile, ctx)
	a.Key = evaluateText(a.Key, ctx)
	for i, f := range a.Files {
		a.Files[i] = evaluateText(f, ctx)
	}
	return a
}

func evaluateText(text string, ctx *executor.ExecutionContext) string {
	kdeps_debug.Log("enter: evaluateText")
	if !strings.Contains(text, "{{") {
		return text
	}
	if ctx == nil || ctx.API == nil {
		return text
	}
	eval := expression.NewEvaluator(ctx.API)
	env := ctx.BuildEvaluatorEnv()
	expr := &domain.Expression{Raw: text, Type: domain.ExprTypeInterpolated}
	result, err := eval.Evaluate(expr, env)
	if err != nil {
		return text
	}
	if s, ok := result.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", result)
}
