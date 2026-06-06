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
	"errors"
	"fmt"

	kdeps_debug "github.com/kdeps/kdeps/v2/pkg/debug"
	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/parser/expression"
)

func (e *Engine) executeAPIResponse(
	resource *domain.Resource,
	ctx *ExecutionContext,
) (interface{}, error) {
	kdeps_debug.Log("enter: executeAPIResponse")
	if err := e.ensureResponseEvaluator(ctx); err != nil {
		return nil, err
	}

	env := e.buildEvaluationEnvironment(ctx)
	apiResponseConfig := resource.APIResponse

	evaluatedResponse, err := e.evaluateResponseValue(apiResponseConfig.Response, env)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate API response: %w", err)
	}

	successBool, err := e.resolveAPIResponseSuccess(apiResponseConfig, env)
	if err != nil {
		return nil, err
	}

	apiResponse := map[string]interface{}{
		"success": successBool,
		"data":    evaluatedResponse,
	}

	if metaMap := e.buildAPIResponseMeta(apiResponseConfig, env); len(metaMap) > 0 {
		apiResponse["_meta"] = metaMap
	}
	e.applyLLMMetadataToResponse(apiResponse, ctx)

	return apiResponse, nil
}

// ensureResponseEvaluator initializes the evaluator required for API response evaluation.
func (e *Engine) ensureResponseEvaluator(ctx *ExecutionContext) error {
	if e.evaluator != nil {
		return nil
	}
	if ctx == nil {
		return errors.New("execution context required for API response")
	}
	e.evaluator = expression.NewEvaluator(ctx.API)
	return nil
}

// resolveAPIResponseSuccess evaluates the success flag, defaulting to true when unset.
func (e *Engine) resolveAPIResponseSuccess(
	config *domain.APIResponseConfig,
	env map[string]interface{},
) (bool, error) {
	if config.Success == nil {
		return true, nil
	}
	evaluatedSuccess, successErr := e.evaluateResponseValue(config.Success, env)
	if successErr != nil {
		return false, fmt.Errorf("failed to evaluate API response success: %w", successErr)
	}
	successBool, validBool := domain.ParseBool(evaluatedSuccess)
	if !validBool {
		return false, nil
	}
	return successBool, nil
}

// buildAPIResponseMeta assembles optional _meta fields from the API response config.
func (e *Engine) buildAPIResponseMeta(
	config *domain.APIResponseConfig,
	env map[string]interface{},
) map[string]interface{} {
	metaMap := make(map[string]interface{})

	if config.Headers != nil {
		if headers := e.evaluateResponseHeaders(config.Headers, env); len(headers) > 0 {
			metaMap["headers"] = headers
		}
	}
	if config.Model != "" {
		if evaluatedModel, evalErr := e.evaluateResponseValue(config.Model, env); evalErr == nil {
			metaMap["model"] = fmt.Sprintf("%v", evaluatedModel)
		}
	}
	if config.Backend != "" {
		if evaluatedBackend, evalErr := e.evaluateResponseValue(config.Backend, env); evalErr == nil {
			metaMap["backend"] = fmt.Sprintf("%v", evaluatedBackend)
		}
	}
	return metaMap
}

// evaluateResponseHeaders evaluates configured response headers into string maps.
func (e *Engine) evaluateResponseHeaders(
	headersConfig interface{},
	env map[string]interface{},
) map[string]string {
	evaluatedHeaders, evalErr := e.evaluateResponseValue(headersConfig, env)
	if evalErr != nil {
		return nil
	}

	headers := make(map[string]string)
	if hMap, ok := evaluatedHeaders.(map[string]interface{}); ok {
		for k, v := range hMap {
			headers[k] = fmt.Sprintf("%v", v)
		}
		return headers
	}
	if sMap, okS := evaluatedHeaders.(map[string]string); okS {
		for k, v := range sMap {
			val, _ := e.evaluateResponseValue(v, env)
			headers[k] = fmt.Sprintf("%v", val)
		}
	}
	return headers
}

// applyLLMMetadataToResponse merges LLM metadata from context into the API response _meta block.
func (e *Engine) applyLLMMetadataToResponse(
	apiResponse map[string]interface{},
	ctx *ExecutionContext,
) {
	if ctx == nil || ctx.LLMMetadata == nil {
		return
	}
	if ctx.LLMMetadata.Model == "" && ctx.LLMMetadata.Backend == "" {
		return
	}

	metaMap, ok := apiResponse["_meta"].(map[string]interface{})
	if ok {
		if ctx.LLMMetadata.Model != "" && metaMap["model"] == nil {
			metaMap["model"] = ctx.LLMMetadata.Model
		}
		if ctx.LLMMetadata.Backend != "" && metaMap["backend"] == nil {
			metaMap["backend"] = ctx.LLMMetadata.Backend
		}
		return
	}

	newMetaMap := make(map[string]interface{})
	if ctx.LLMMetadata.Model != "" {
		newMetaMap["model"] = ctx.LLMMetadata.Model
	}
	if ctx.LLMMetadata.Backend != "" {
		newMetaMap["backend"] = ctx.LLMMetadata.Backend
	}
	if len(newMetaMap) > 0 {
		apiResponse["_meta"] = newMetaMap
	}
}

// evaluateResponseValue recursively evaluates expressions in response values.
// Handles maps, arrays, and strings with expressions.
func (e *Engine) evaluateResponseValue(
	value interface{},
	env map[string]interface{},
) (interface{}, error) {
	kdeps_debug.Log("enter: evaluateResponseValue")
	// Handle string values - check if they contain expressions
	if str, ok := value.(string); ok {
		parser := expression.NewParser()
		expr, err := parser.ParseValue(str)
		if err != nil {
			return nil, fmt.Errorf("failed to parse expression: %w", err)
		}

		evaluatedValue, err := e.evaluator.Evaluate(expr, env)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate expression: %w", err)
		}

		return evaluatedValue, nil
	}

	// Handle maps - recursively evaluate each value
	if dataMap, ok := value.(map[string]interface{}); ok {
		result := make(map[string]interface{})
		for key, val := range dataMap {
			evaluatedValue, err := e.evaluateResponseValue(val, env)
			if err != nil {
				return nil, fmt.Errorf("failed to evaluate response field '%s': %w", key, err)
			}
			result[key] = evaluatedValue
		}
		return result, nil
	}

	// Handle arrays/slices - recursively evaluate each element
	if dataSlice, ok := value.([]interface{}); ok {
		result := make([]interface{}, len(dataSlice))
		for i, val := range dataSlice {
			evaluatedValue, err := e.evaluateResponseValue(val, env)
			if err != nil {
				return nil, fmt.Errorf(
					"failed to evaluate response array element at index %d: %w",
					i,
					err,
				)
			}
			result[i] = evaluatedValue
		}
		return result, nil
	}

	// For other types (numbers, booleans, nil), return as-is
	return value, nil
}
