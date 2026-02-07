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
	Name         string   `yaml:"name"`
	Image        string   `yaml:"image"`
	Net          string   `yaml:"net,omitempty"`
	Capabilities []string `yaml:"capabilities,omitempty"`
	Binds        []string `yaml:"binds,omitempty"`
	Env          []string `yaml:"env,omitempty"`
	Command      []string `yaml:"command,omitempty"`
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
// The arch parameter specifies the target architecture ("amd64" or "arm64").
// If arch is empty, it defaults to the host architecture.
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
	var envList []string
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
	if shouldInstallOllama(workflow) {
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
		},
		Onboot: []LinuxKitImage{
			{
				Name:  "dhcpcd",
				Image: fmt.Sprintf("linuxkit/dhcpcd:%s", linuxkitComponentTag),
			},
		},
		Services: []LinuxKitImage{
			{
				Name:  "getty",
				Image: fmt.Sprintf("linuxkit/getty:%s", linuxkitComponentTag),
				Env:   []string{"INSECURE=true"},
			},
			{
				Name:         "kdeps",
				Image:        imageName,
				Net:          "host",
				Capabilities: []string{"all"},
				Binds:        binds,
				Env:          envList,
			},
		},
		Files: []LinuxKitFile{
			{
				Path:     "etc/hostname",
				Contents: hostname,
			},
		},
	}

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

// shouldInstallOllama determines if Ollama is needed (mirrors docker builder logic).
func shouldInstallOllama(workflow *domain.Workflow) bool {
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
