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
	"fmt"
	"sort"

	kdeps_debug "github.com/kdeps/kdeps/v2/pkg/debug"
	"github.com/kdeps/kdeps/v2/pkg/domain"
)

func addThinBuildSteps(config *LinuxKitConfig, imageName string) {
	kdeps_debug.Log("enter: addThinBuildSteps")
	config.Onboot = append(config.Onboot, LinuxKitImage{
		Name:  "mount-data",
		Image: "alpine:3.19",
		Command: []string{
			"sh", "-c",
			"mkdir -p /mnt/data && mount /dev/vda2 /mnt/data || mount /dev/sda2 /mnt/data || true",
		},
		Capabilities: []string{"all"},
		Binds:        []string{"/dev:/dev", "/mnt:/mnt:shared"},
	})

	config.Onboot = append(config.Onboot, LinuxKitImage{
		Name:  "import-image",
		Image: "linuxkit/containerd:" + linuxkitComponentTag,
		Command: []string{
			"sh", "-c",
			"ctr -n services images import /mnt/data/image.tar",
		},
		Binds: []string{"/run/containerd:/run/containerd", "/mnt:/mnt"},
	})

	// Use a detached container to run kdeps in thin mode
	config.Onboot = append(config.Onboot, LinuxKitImage{
		Name:  "start-kdeps",
		Image: "linuxkit/containerd:" + linuxkitComponentTag,
		Command: []string{
			"sh", "-c",
			fmt.Sprintf(
				"ctr -n services containers create --net-host %s kdeps && ctr -n services tasks start -d kdeps",
				imageName,
			),
		},
		Binds: []string{"/run/containerd:/run/containerd"},
	})
}

func buildKdepsEnvList(workflow *domain.Workflow) []string {
	envList := []string{
		"KDEPS_BIND_HOST=0.0.0.0",
		"KDEPS_PLATFORM=iso",
	}
	if !domain.ResolveInstallOllama(workflow) {
		if domain.HasChatResources(workflow) {
			envList = append(envList, "KDEPS_MODELS_DIR=/app/.kdeps/models")
		}
		return appendWorkflowEnv(envList, workflow)
	}

	envList = append(envList,
		"OLLAMA_HOST=127.0.0.1",
		"OLLAMA_MODELS=/root/.ollama/models",
	)
	return appendWorkflowEnv(envList, workflow)
}

func appendWorkflowEnv(envList []string, workflow *domain.Workflow) []string {
	if workflow.Settings.AgentSettings.Env == nil {
		return envList
	}

	keys := make([]string, 0, len(workflow.Settings.AgentSettings.Env))
	for k := range workflow.Settings.AgentSettings.Env {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		envList = append(
			envList,
			fmt.Sprintf("%s=%s", k, workflow.Settings.AgentSettings.Env[k]),
		)
	}
	return envList
}

func buildKdepsBinds(workflow *domain.Workflow) []string {
	binds := []string{"/var/run:/var/run"}
	if domain.ResolveInstallOllama(workflow) {
		binds = append(binds, "/dev:/dev")
	}
	// file backend does not need /dev bind
	return binds
}

func addFatBuildService(config *LinuxKitConfig, imageName string, workflow *domain.Workflow) {
	kdeps_debug.Log("enter: addFatBuildService")
	config.Services = append(config.Services, LinuxKitImage{
		Name:         "kdeps",
		Image:        imageName,
		Net:          "host",
		Capabilities: []string{"all"},
		Binds:        buildKdepsBinds(workflow),
		Env:          buildKdepsEnvList(workflow),
		Command: []string{
			"/entrypoint.sh",
			"/usr/bin/supervisord",
			"-c",
			"/etc/supervisord.conf",
		},
	})
}
