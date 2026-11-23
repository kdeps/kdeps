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

package agent

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/kdeps/kdeps/v3/pkg/assetsetup"
	"github.com/kdeps/kdeps/v3/pkg/codegraph"
	kdepsconfig "github.com/kdeps/kdeps/v3/pkg/config"
	executorLLM "github.com/kdeps/kdeps/v3/pkg/executor/llm"
)

//nolint:gochecknoinits // wires asset installs to this package's reload
func init() { assetsetup.SetOnChange(ReloadAssets) }

// ReloadAssets re-reads every asset set (harness, events, actions, presets,
// tool definitions, slash commands) after an update.
// Already-registered tools keep their descriptions until the next start.
func ReloadAssets() {
	initHarness()
	initEvents()
	initPermissionPolicy()
	initToolSchemaPolicy()
	initMemoryRules()
	if err := codegraph.Configure(); err != nil && !errors.Is(err, fs.ErrNotExist) {
		fmt.Fprintf(os.Stderr, "codegraph: rules: %v\n", err)
	}
	initPresets()
	toolDefinitions = loadToolDefinitions()
	toolAliases = aliasesOf(toolDefinitions)
	registerDeclaredAPIRoutes()
	executorLLM.ReloadRegistry()
	executorLLM.ReloadGGUFRegistry()
	executorLLM.ReloadCloudModels()
	kdepsconfig.ReloadProviders()
}
