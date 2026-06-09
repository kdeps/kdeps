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

const (
	storageTypeMemory  = "memory"
	storageTypeSession = "session"
	storageTypeItem    = "item"
	storageTypeLoop    = "loop"

	// Item context keys.
	itemKeyCurrent = "current"
	itemKeyCount   = "count"
	itemKeyAll     = "all"
	itemKeyIndex   = "index"
	itemKeyPrev    = "prev"
	itemKeyNext    = "next"
	itemKeyItems   = "items"

	// Loop context keys (stored in Items map with "loop." prefix to avoid collision).
	loopKeyIndex   = "loop.index"
	loopKeyCount   = "loop.count"
	loopKeyResults = "loop.results"

	// Default TTL values.
	defaultSessionTTLMinutes = 30

	// String splitting constants.
	agentPathParts = 2
	agentSpecParts = 2

	// Context key names for input-processor outputs.
	keyInputTranscript  = "inputTranscript"
	keyInputMedia       = "inputMedia"
	keyInputFileContent = "inputFileContent"
	keyInputFilePath    = "inputFilePath"

	// Input type name used in switch statements and form field names.
	inputTypeFile = "file"

	// Config namespace names used in expression routing.
	nsConfig    = "config"
	nsWorkflow  = "workflow"
	nsResource  = "resource"
	nsComponent = "component"
	nsAgency    = "agency"
)
