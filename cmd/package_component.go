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

package cmd

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	kdeps_debug "github.com/kdeps/kdeps/v2/pkg/debug"
)

// printComponentPackageSuccess prints the post-package summary for components.
func printComponentPackageSuccess(archivePath string) {
	fmt.Fprintln(os.Stdout, "✓ Component manifest validated")
	fmt.Fprintln(os.Stdout, "✓ Resources collected")
	fmt.Fprintln(os.Stdout, "✓ Package created")
	fmt.Fprintln(os.Stdout)
	fmt.Fprintf(os.Stdout, "Created: %s\n", archivePath)
}

// PackageComponentWithFlags packages a component directory into a .komponent archive.
func PackageComponentWithFlags(_ *cobra.Command, args []string, flags *PackageFlags) error {
	kdeps_debug.Log("enter: PackageComponentWithFlags")
	componentDir := args[0]
	fmt.Fprintf(os.Stdout, "Packaging component: %s\n\n", componentDir)

	componentFile := FindComponentFile(componentDir)
	if componentFile == "" {
		return fmt.Errorf("no component.yaml / component.yml found in %s", componentDir)
	}

	parser, err := newPackageYAMLParserFunc()
	if err != nil {
		return err
	}

	component, err := parser.ParseComponent(componentFile)
	if err != nil {
		return fmt.Errorf("failed to parse component: %w", err)
	}

	outputDir, pkgName := resolvePackageOutputDir(
		flags,
		fmt.Sprintf("%s-%s", component.Metadata.Name, component.Metadata.Version),
	)
	archivePath := filepath.Join(outputDir, pkgName+komponentExtension)
	if archiveErr := CreateComponentPackageArchive(componentDir, archivePath); archiveErr != nil {
		return fmt.Errorf("failed to create component archive: %w", archiveErr)
	}

	printComponentPackageSuccess(archivePath)
	return nil
}

// CreateComponentPackageArchive creates a .komponent tar.gz archive from componentDir.
// It includes:
//   - component.yaml (and .j2 variants)
//   - resources/
//   - HTML/CSS/JS/template files
//   - Respects .kdepsignore; excludes hidden entries.
func CreateComponentPackageArchive(componentDir, archivePath string) error {
	kdeps_debug.Log("enter: CreateComponentPackageArchive")
	if err := os.MkdirAll(filepath.Dir(archivePath), 0o750); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	file, err := os.Create(archivePath)
	if err != nil {
		return fmt.Errorf("failed to create archive file: %w", err)
	}
	defer file.Close()

	gzipWriter := gzip.NewWriter(file)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	ignorePatterns := ParseKdepsIgnore(componentDir)
	return filepath.Walk(
		componentDir,
		CreateArchiveWalkFunc(componentDir, tarWriter, ignorePatterns),
	)
}

// IsKomponentFile reports whether path points to a .komponent archive.
func IsKomponentFile(path string) bool {
	kdeps_debug.Log("enter: IsKomponentFile")
	return strings.HasSuffix(path, komponentExtension)
}
