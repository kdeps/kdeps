// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

//go:build !js

package iso

import (
	"os"
	"strings"

	kdeps_debug "github.com/kdeps/kdeps/v2/pkg/debug"
	"github.com/kdeps/kdeps/v2/pkg/domain"
)

// ShouldInstallOllama determines if Ollama is needed (mirrors docker builder logic).
func ShouldInstallOllama(workflow *domain.Workflow) bool {
	kdeps_debug.Log("enter: ShouldInstallOllama")
	if workflow.Settings.AgentSettings.InstallOllama != nil {
		return *workflow.Settings.AgentSettings.InstallOllama
	}

	if workflowHasChatResources(workflow) {
		backend := os.Getenv("KDEPS_DEFAULT_BACKEND")
		if backend == "" || backend == backendOllama {
			return true
		}
	}

	if routerJSON := os.Getenv("KDEPS_LLM_ROUTER"); routerJSON != "" &&
		strings.Contains(routerJSON, `"ollama"`) {
		return true
	}

	return os.Getenv("KDEPS_LLM_MODELS") != ""
}
