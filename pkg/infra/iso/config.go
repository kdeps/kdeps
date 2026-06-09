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
	"errors"
	"fmt"
	"runtime"

	kdeps_debug "github.com/kdeps/kdeps/v2/pkg/debug"

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
	kdeps_debug.Log("enter: KernelCmdline")
	if arch == "arm64" {
		return "console=ttyAMA0 console=tty0"
	}

	return "console=ttyS0 console=tty0"
}

// GenerateConfig creates a LinuxKit configuration from a workflow and image name.
func GenerateConfig(
	imageName, hostname, arch string,
	workflow *domain.Workflow,
) (*LinuxKitConfig, error) {
	kdeps_debug.Log("enter: GenerateConfig")
	return GenerateConfigExtended(imageName, hostname, arch, workflow, false)
}

// GenerateConfigExtended creates a LinuxKit configuration with support for thin builds.
func GenerateConfigExtended(
	imageName, hostname, arch string,
	workflow *domain.Workflow,
	thin bool,
) (*LinuxKitConfig, error) {
	kdeps_debug.Log("enter: GenerateConfigExtended")
	if err := validateGenerateConfigArgs(imageName, workflow); err != nil {
		return nil, err
	}

	if hostname == "" {
		hostname = "kdeps"
	}

	if arch == "" {
		arch = runtime.GOARCH
	}

	config := buildBaseConfig(hostname, arch)

	// For thin builds, we don't bundle the image in the initrd.
	if thin {
		addThinBuildSteps(config, imageName)
	} else {
		// Bundle the app image directly in the initrd (fat build).
		addFatBuildService(config, imageName, workflow)
	}

	return config, nil
}

func validateGenerateConfigArgs(imageName string, workflow *domain.Workflow) error {
	kdeps_debug.Log("enter: validateGenerateConfigArgs")
	if workflow == nil {
		return errors.New("workflow cannot be nil")
	}

	if imageName == "" {
		return errors.New("image name cannot be empty")
	}
	return nil
}

func buildBaseConfig(hostname, arch string) *LinuxKitConfig {
	kdeps_debug.Log("enter: buildBaseConfig")
	return &LinuxKitConfig{
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
					"sh",
					"-c",
					"apk add --no-cache ncurses curl busybox-extras && cp /usr/bin/curl /usr/bin/telnet /usr/local/bin/",
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
}

//nolint:gochecknoglobals // test-replaceable global
var yamlMarshal = yaml.Marshal

// MarshalConfig marshals a LinuxKitConfig to YAML.
func MarshalConfig(config *LinuxKitConfig) ([]byte, error) {
	kdeps_debug.Log("enter: MarshalConfig")
	data, err := yamlMarshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal LinuxKit config: %w", err)
	}

	return data, nil
}
