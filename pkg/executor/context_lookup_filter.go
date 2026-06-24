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
)

// getFromBody retrieves value from request body with filtering.
func (ctx *ExecutionContext) getFromBody(name string) (interface{}, error) {
	kdeps_debug.Log("enter: getFromBody")
	if ctx.Request == nil {
		return nil, errors.New("no request")
	}

	if ctx.Request.Body == nil {
		// If parameter filtering is enabled and body is nil, this is an error
		// because we expect the body to be available for filtered access
		if len(ctx.allowedParams) > 0 {
			return nil, fmt.Errorf(
				"parameter '%s' not found (request body not available for filtering)",
				name,
			)
		}
		return nil, errors.New("no body")
	}

	return ctx.GetFilteredValue(ctx.Request.Body, name, contextFieldBody)
}

// getFromQuery retrieves value from query parameters with filtering.
func (ctx *ExecutionContext) getFromQuery(name string) (interface{}, error) {
	kdeps_debug.Log("enter: getFromQuery")
	if ctx.Request == nil {
		return nil, errors.New("no request")
	}

	return ctx.getFilteredStringValue(ctx.Request.Query, name, "query")
}

// getFromHeaders retrieves value from headers with filtering.
func (ctx *ExecutionContext) getFromHeaders(name string) (interface{}, error) {
	kdeps_debug.Log("enter: getFromHeaders")
	if ctx.Request == nil {
		return nil, errors.New("no request")
	}

	if len(ctx.allowedHeaders) > 0 {
		if ctx.IsHeaderAllowed(name) {
			return ctx.findHeaderValue(name)
		}
	} else {
		return ctx.findHeaderValue(name)
	}

	return nil, errors.New("not found in headers")
}

// getFromUploadedFiles retrieves file content from uploaded files.
func (ctx *ExecutionContext) getFromUploadedFiles(name string) (interface{}, error) {
	kdeps_debug.Log("enter: getFromUploadedFiles")
	if ctx.Request == nil {
		return nil, errors.New("no request")
	}

	if file, err := ctx.GetUploadedFile(name); err == nil {
		return ReadFile(file.Path)
	}

	return nil, errors.New("not found in uploaded files")
}

// createNotFoundError creates a helpful error message for missing values.
func (ctx *ExecutionContext) createNotFoundError(name string) error {
	kdeps_debug.Log("enter: createNotFoundError")
	return fmt.Errorf(
		"value '%s' not found in any context. Try: get('%s', 'memory'), get('%s', 'session'), "+
			"get('%s', 'output'), get('%s', 'param'), get('%s', 'header'), or get('%s', 'file')",
		name, name, name, name, name, name, name)
}
