// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package iso

import (
	"errors"
	"fmt"
	"runtime"
	"sort"

	"gopkg.in/yaml.v3"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

const (
	// LinuxKit component versions.
	linuxkitKernelTag    = "6.6.71"
	linuxkitComponentTag = "v1.3.0"
	linuxkitMountTag     = "v1.1.0"
	linuxkitFormatTag    = "v1.1.0"

	backendOllama = "ollama"
)

// LinuxKitConfig represents a LinuxKit YAML configuration.
type LinuxKitConfig struct {
	Kernel   LinuxKitKernel  `yaml:"kernel"`
	Init     []string        `yaml:"init"`
	Onboot   []LinuxKitImage `yaml:"onboot,omitempty"`
	Services []LinuxKitImage `yaml:"services"`
	Files    []LinuxKitFile  `yaml:"files,omitempty"`
}

// LinuxKitKernel configures the kernel image and boot parameters.
type LinuxKitKernel struct {
	Image   string `yaml:"image"`
	Cmdline string `yaml:"cmdline,omitempty"`
}

// LinuxKitImage configures a service or onboot container.
type LinuxKitImage struct {
	Name              string   `yaml:"name"`
	Image             string   `yaml:"image"`
	Net               string   `yaml:"net,omitempty"`
	Capabilities      []string `yaml:"capabilities,omitempty"`
	Binds             []string `yaml:"binds,omitempty"`
	Env               []string `yaml:"env,omitempty"`
	Command           []string `yaml:"command,omitempty"`
	RootfsPropagation string   `yaml:"rootfsPropagation,omitempty"`
}

// LinuxKitFile adds a file to the root filesystem.
type LinuxKitFile struct {
	Path     string `yaml:"path"`
	Contents string `yaml:"contents,omitempty"`
}

// KernelCmdline returns the appropriate kernel console cmdline for the given architecture.
func KernelCmdline(arch string) string {
	if arch == "arm64" {
		return "console=ttyAMA0 console=tty0"
	}

	return "console=ttyS0 console=tty0"
}

// GenerateConfig creates a LinuxKit configuration from a workflow and image name.
func GenerateConfig(imageName, hostname, arch string, workflow *domain.Workflow) (*LinuxKitConfig, error) {
	if workflow == nil {
		return nil, errors.New("workflow cannot be nil")
	}

	if imageName == "" {
		return nil, errors.New("image name cannot be empty")
	}

	if hostname == "" {
		hostname = "kdeps"
	}

	if arch == "" {
		arch = runtime.GOARCH
	}

	// Build env vars for the service container
	envList := []string{
		"KDEPS_BIND_HOST=0.0.0.0",
		"KDEPS_PLATFORM=iso",
	}
	if ShouldInstallOllama(workflow) {
		envList = append(envList,
			"OLLAMA_HOST=127.0.0.1",
			"OLLAMA_MODELS=/root/.ollama/models",
		)
	}
	if workflow.Settings.AgentSettings.Env != nil {
		keys := make([]string, 0, len(workflow.Settings.AgentSettings.Env))
		for k := range workflow.Settings.AgentSettings.Env {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			envList = append(envList, fmt.Sprintf("%s=%s", k, workflow.Settings.AgentSettings.Env[k]))
		}
	}

	// Build service binds
	binds := []string{
		"/var/run:/var/run",
	}
	if ShouldInstallOllama(workflow) {
		binds = append(binds, "/dev:/dev")
	}

	config := &LinuxKitConfig{
		Kernel: LinuxKitKernel{
			Image:   fmt.Sprintf("linuxkit/kernel:%s", linuxkitKernelTag),
			Cmdline: KernelCmdline(arch),
		},
		Init: []string{
			fmt.Sprintf("linuxkit/init:%s", linuxkitComponentTag),
			fmt.Sprintf("linuxkit/runc:%s", linuxkitComponentTag),
			fmt.Sprintf("linuxkit/containerd:%s", linuxkitComponentTag),
			fmt.Sprintf("linuxkit/ca-certificates:%s", linuxkitComponentTag),
		},
		Services: []LinuxKitImage{
			{
				Name:  "dhcpcd",
				Image: fmt.Sprintf("linuxkit/dhcpcd:%s", linuxkitComponentTag),
			},
			{
				Name:  "getty",
				Image: fmt.Sprintf("linuxkit/getty:%s", linuxkitComponentTag),
				Env:   []string{"INSECURE=true"},
			},
		},
		Onboot: []LinuxKitImage{
			{
				Name:  "utils",
				Image: "alpine:3.19",
				Command: []string{
					"sh", "-c",
					"apk add --no-cache curl busybox-extras && cp /usr/bin/curl /usr/bin/telnet /usr/local/bin/",
				},
				Binds: []string{
					"/usr/local/bin:/usr/local/bin",
				},
			},
		},
		Files: []LinuxKitFile{
			{
				Path:     "etc/hostname",
				Contents: hostname,
			},
		},
	}

	// Bundle the app image directly in the initrd (fat build).
	// Explicitly invoke entrypoint.sh → supervisord so both kdeps and ollama
	// run properly. We can't rely on containerd merging ENTRYPOINT + CMD
	// from the image config, so we set the full command chain here.
	config.Services = append(config.Services, LinuxKitImage{
		Name:         "kdeps",
		Image:        imageName,
		Net:          "host",
		Capabilities: []string{"all"},
		Binds:        binds,
		Env:          envList,
		Command:      []string{"/entrypoint.sh", "/usr/bin/supervisord", "-c", "/etc/supervisord.conf"},
	})

	return config, nil
}

// MarshalConfig marshals a LinuxKitConfig to YAML.
func MarshalConfig(config *LinuxKitConfig) ([]byte, error) {
	data, err := yaml.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal LinuxKit config: %w", err)
	}

	return data, nil
}

// ShouldInstallOllama determines if Ollama is needed (mirrors docker builder logic).
func ShouldInstallOllama(workflow *domain.Workflow) bool {
	if workflow.Settings.AgentSettings.InstallOllama != nil {
		return *workflow.Settings.AgentSettings.InstallOllama
	}

	for _, resource := range workflow.Resources {
		if resource.Run.Chat != nil {
			backend := resource.Run.Chat.Backend
			if backend == backendOllama {
				return true
			}
			if backend == "" && resource.Run.Chat.APIKey == "" {
				return true
			}
		}
	}

	return len(workflow.Settings.AgentSettings.Models) > 0
}
