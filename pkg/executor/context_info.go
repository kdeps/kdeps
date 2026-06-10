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
	"fmt"

	kdeps_debug "github.com/kdeps/kdeps/v2/pkg/debug"
)

type infoFieldHandler func(*ExecutionContext) (interface{}, error)

// infoFieldHandlers is the single source of truth for metadata field names:
// Info dispatches through it and IsMetadataField checks key membership.
//
//nolint:gochecknoglobals // dispatch table
var infoFieldHandlers = map[string]infoFieldHandler{
	// Shorthand "method"/"path" return empty string when there is no request
	// (for Get() compatibility); "request.method"/"request.path" error instead.
	"method":               infoShorthandMethod,
	"path":                 infoShorthandPath,
	"request.method":       (*ExecutionContext).getRequestMethod,
	"request.path":         (*ExecutionContext).getRequestPath,
	"filecount":            (*ExecutionContext).getFileCount,
	"files":                (*ExecutionContext).getFiles,
	"filenames":            infoFileNames,
	"filetypes":            infoFileTypes,
	"request.IP":           (*ExecutionContext).GetRequestIP,
	"IP":                   (*ExecutionContext).GetRequestIP,
	"request.ID":           (*ExecutionContext).GetRequestID,
	"ID":                   (*ExecutionContext).GetRequestID,
	"request_id":           (*ExecutionContext).GetRequestID,
	itemKeyIndex:           infoItemField(itemKeyIndex),
	itemKeyCount:           infoItemField(itemKeyCount),
	itemKeyCurrent:         infoItemField(itemKeyCurrent),
	itemKeyPrev:            infoItemField(itemKeyPrev),
	itemKeyNext:            infoItemField(itemKeyNext),
	"workflow.name":        infoWorkflowName,
	"name":                 infoWorkflowName,
	"workflow.version":     infoWorkflowVersion,
	"version":              infoWorkflowVersion,
	"workflow.description": infoWorkflowDescription,
	"description":          infoWorkflowDescription,
	"current_time":         (*ExecutionContext).getCurrentTime,
	"timestamp":            (*ExecutionContext).getCurrentTime,
	"session_id":           (*ExecutionContext).GetSessionID,
	"sessionId":            (*ExecutionContext).GetSessionID,
}

func infoShorthandMethod(ctx *ExecutionContext) (interface{}, error) {
	return infoRequestField(ctx, func(r *RequestContext) string { return r.Method }), nil
}

func infoShorthandPath(ctx *ExecutionContext) (interface{}, error) {
	return infoRequestField(ctx, func(r *RequestContext) string { return r.Path }), nil
}

func infoRequestField(ctx *ExecutionContext, field func(*RequestContext) string) string {
	if ctx.Request != nil {
		return field(ctx.Request)
	}
	return ""
}

func infoFileNames(ctx *ExecutionContext) (interface{}, error) { return ctx.GetAllFileNames() }

func infoFileTypes(ctx *ExecutionContext) (interface{}, error) { return ctx.GetAllFileTypes() }

func infoItemField(key string) infoFieldHandler {
	return func(ctx *ExecutionContext) (interface{}, error) {
		return ctx.getItemFromContext(key)
	}
}

func infoWorkflowName(ctx *ExecutionContext) (interface{}, error) {
	return ctx.Workflow.Metadata.Name, nil
}

func infoWorkflowVersion(ctx *ExecutionContext) (interface{}, error) {
	return ctx.Workflow.Metadata.Version, nil
}

func infoWorkflowDescription(ctx *ExecutionContext) (interface{}, error) {
	return ctx.Workflow.Metadata.Description, nil
}

func (ctx *ExecutionContext) Info(field string) (interface{}, error) {
	kdeps_debug.Log("enter: Info")
	if handler, ok := infoFieldHandlers[field]; ok {
		return handler(ctx)
	}
	return nil, fmt.Errorf("unknown info field: %s", field)
}

// IsMetadataField checks if a name is a metadata field (exported for testing).
func (ctx *ExecutionContext) IsMetadataField(name string) bool {
	kdeps_debug.Log("enter: IsMetadataField")
	_, ok := infoFieldHandlers[name]
	return ok
}
