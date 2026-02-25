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

//go:build js

// Package python provides Python virtual environment management using uv.
package python

import (
	"errors"
	"time"
)

// Constants mirrored from uv.go for WASM compatibility.
const (
	MaxPackagesInVenvName = 3
	UVTimeout             = 5 * time.Minute
	IOToolsPythonVersion  = "3.12"
)

// Manager is a no-op stub for WASM builds (no subprocess execution).
type Manager struct {
	BaseDir string
}

// IOToolsBaseDir returns an empty path — no local venvs in WASM.
func IOToolsBaseDir() string { return "" }

// IOToolVenvPath returns an empty path in WASM.
func IOToolVenvPath(_ string) string { return "" }

// IOToolPythonBin returns an empty string in WASM (no Python available).
func IOToolPythonBin(_ string) string { return "" }

// IOToolBin returns an empty string in WASM (no local binaries available).
func IOToolBin(_, _ string) string { return "" }

// NewManager returns a no-op Manager in WASM.
func NewManager(_ string) *Manager { return &Manager{} }

// EnsureVenv is unsupported in WASM.
func (m *Manager) EnsureVenv(_ string, _ []string, _ string, _ string) (string, error) {
	return "", errors.New("python: EnsureVenv not supported in WASM")
}

// GetVenvName returns an empty string in WASM.
func (m *Manager) GetVenvName(_ string, _ []string, _ string) string { return "" }

// InstallPackages is unsupported in WASM.
func (m *Manager) InstallPackages(_ string, _ []string, _ ...string) error {
	return errors.New("python: InstallPackages not supported in WASM")
}

// InstallRequirements is unsupported in WASM.
func (m *Manager) InstallRequirements(_, _ string) error {
	return errors.New("python: InstallRequirements not supported in WASM")
}

// InstallTool is unsupported in WASM.
func (m *Manager) InstallTool(_, _ string, _ ...string) error {
	return errors.New("python: InstallTool not supported in WASM")
}

// GetPythonPath is unsupported in WASM.
func (m *Manager) GetPythonPath(_ string) (string, error) {
	return "", errors.New("python: GetPythonPath not supported in WASM")
}
