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

package domain

import "strings"

const (
	ResourceTypeChat       = "chat"
	ResourceTypeHTTPClient = "httpClient"
)

// PrimaryResourceType identifies a mutually-exclusive primary execution block on a resource.
// Order matches executor dispatch precedence.
type PrimaryResourceType struct {
	Name    string
	Present func(*Resource) bool
}

// PrimaryResourceTypes returns the canonical registry of primary execution types.
func PrimaryResourceTypes() []PrimaryResourceType {
	return buildPrimaryResourceTypes()
}

// CountPrimaryResourceTypes returns how many mutually-exclusive primary execution blocks
// are set on a resource. Response blocks (apiResponse) are excluded because
// they combine with any other primary type.
func CountPrimaryResourceTypes(resource *Resource) int {
	n := 0
	for _, entry := range resourceExecCatalog {
		if entry.ResponseBlock {
			continue
		}
		if entry.PresentResource(resource) {
			n++
		}
	}
	return n
}

// HasPrimaryResourceType reports whether the resource defines a primary execution block.
func HasPrimaryResourceType(resource *Resource) bool {
	return CountPrimaryResourceTypes(resource) > 0
}

// PrimaryResourceTypeNames returns the canonical primary type names in dispatch order.
func PrimaryResourceTypeNames() []string {
	types := PrimaryResourceTypes()
	names := make([]string, len(types))
	for i, entry := range types {
		names[i] = entry.Name
	}
	return names
}

// PrimaryResourceTypesList returns a comma-separated list of primary type names.
func PrimaryResourceTypesList() string {
	return strings.Join(PrimaryResourceTypeNames(), ", ")
}

// IsPrimaryResourceTypeName reports whether name is a canonical primary execution YAML key.
func IsPrimaryResourceTypeName(name string) bool {
	for _, entry := range PrimaryResourceTypes() {
		if entry.Name == name {
			return true
		}
	}
	return false
}

// IsRecognizedResourceActionKey reports whether name is a primary execution key or response block.
func IsRecognizedResourceActionKey(name string) bool {
	return name == resourceActionAPIResponse || IsPrimaryResourceTypeName(name)
}

// componentBackedAction is a native workflow action that a shipped component
// replaces. Workflow resources call the component; the executor stays behind
// the kdeps API tool the component POSTs to.
type componentBackedAction struct {
	Name      string
	Component string
	Resource  func(*Resource) bool
	Inline    func(*ActionConfig) bool
}

//nolint:gochecknoglobals // registry table
var componentBackedActions = []componentBackedAction{
	{
		Name:      "searchLocal",
		Component: "search_local",
		Resource:  func(r *Resource) bool { return r.SearchLocal != nil },
		Inline:    func(a *ActionConfig) bool { return a.SearchLocal != nil },
	},
	{
		Name:      "exec",
		Component: "exec",
		Resource:  func(r *Resource) bool { return r.Exec != nil },
		Inline:    func(a *ActionConfig) bool { return a.Exec != nil },
	},
	{
		Name:      "python",
		Component: "python",
		Resource:  func(r *Resource) bool { return r.Python != nil },
		Inline:    func(a *ActionConfig) bool { return a.Python != nil },
	},
}

// ComponentBackedAction reports the native action a resource uses that must be
// invoked through a component instead. Component resources are not checked:
// only workflow resources reach this function.
func ComponentBackedAction(r *Resource) (string, string, bool) {
	if r == nil {
		return "", "", false
	}
	for _, entry := range componentBackedActions {
		if entry.Resource(r) {
			return entry.Name, entry.Component, true
		}
	}
	return "", "", false
}

// ComponentBackedInlineAction is ComponentBackedAction for a before:/after: step.
func ComponentBackedInlineAction(a *ActionConfig) (string, string, bool) {
	if a == nil {
		return "", "", false
	}
	for _, entry := range componentBackedActions {
		if entry.Inline(a) {
			return entry.Name, entry.Component, true
		}
	}
	return "", "", false
}

// PrimaryResourceEventName returns the event/telemetry label for the resource's execution type.
func PrimaryResourceEventName(r *Resource) string {
	for _, entry := range PrimaryResourceTypes() {
		if entry.Present(r) {
			return primaryResourceEventLabel(entry.Name)
		}
	}
	return "unknown"
}

func primaryResourceEventLabel(canonicalName string) string {
	switch canonicalName {
	case ResourceTypeChat:
		return "llm"
	case ResourceTypeHTTPClient:
		return "http"
	default:
		return canonicalName
	}
}
