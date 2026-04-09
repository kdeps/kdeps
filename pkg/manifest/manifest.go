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

//go:build !js

// Package manifest provides loading and validation of kdeps package manifests.
package manifest

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	goyaml "gopkg.in/yaml.v3"

	kdeps_debug "github.com/kdeps/kdeps/v2/pkg/debug"
)

// ManifestFile is the default name for a kdeps package manifest.
const ManifestFile = "kdeps.pkg.yaml"

// validTypes lists the allowed package type values.
var validTypes = map[string]struct{}{ //nolint:gochecknoglobals // package-level const map
	"component": {},
	"workflow":  {},
	"agency":    {},
}

// Manifest describes a kdeps registry package.
type Manifest struct {
	Name        string   `yaml:"name"`
	Version     string   `yaml:"version"`
	Type        string   `yaml:"type"`
	Description string   `yaml:"description"`
	License     string   `yaml:"license,omitempty"`
	Tags        []string `yaml:"tags,omitempty"`
}

// Load reads and parses the kdeps.pkg.yaml from dir.
func Load(dir string) (*Manifest, error) {
	kdeps_debug.Log("enter: Load")
	path := filepath.Join(dir, ManifestFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read manifest %s: %w", path, err)
	}
	var m Manifest
	parseErr := goyaml.Unmarshal(data, &m)
	if parseErr != nil {
		return nil, fmt.Errorf("parse manifest %s: %w", path, parseErr)
	}
	return &m, nil
}

// Validate checks that the manifest has all required fields and valid values.
func Validate(m *Manifest) error {
	kdeps_debug.Log("enter: Validate")
	if m.Name == "" {
		return errors.New("manifest: name is required")
	}
	if m.Version == "" {
		return errors.New("manifest: version is required")
	}
	if m.Type == "" {
		return errors.New("manifest: type is required")
	}
	if _, ok := validTypes[m.Type]; !ok {
		return fmt.Errorf("manifest: type %q must be one of: component, workflow, agency", m.Type)
	}
	return nil
}
