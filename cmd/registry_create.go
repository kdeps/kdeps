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

package cmd

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	kdeps_debug "github.com/kdeps/kdeps/v3/pkg/debug"
	"github.com/kdeps/kdeps/v3/pkg/formulas"
	"github.com/kdeps/kdeps/v3/pkg/manifest"
)

// githubArchive matches a GitHub release archive URL: owner/repo and the tag.
var githubArchive = regexp.MustCompile(
	`^https://(?:github\.com/([^/]+/[^/]+)/archive/refs/tags/|codeload\.github\.com/([^/]+/[^/]+)/tar\.gz/refs/tags/)` +
		`([^/]+?)(?:\.tar\.gz)?$`)

// newRegistryCreateCmd is kdeps create: write a formula for a package archive.
func newRegistryCreateCmd() *cobra.Command {
	kdeps_debug.Log("enter: newRegistryCreateCmd")
	var noFetch bool
	c := &cobra.Command{
		Use:   "create URL",
		Short: "Write a formula for a package archive URL",
		Long: `Write a formula for the package archive at URL and print it.

kdeps downloads the archive, computes its sha256, and reads name, version,
type, description, tags and license from its kdeps.pkg.yaml. --no-fetch
skips the download: the name and version come from a GitHub release URL,
and sha256 and type stay empty for you to fill in.

Add the formula as formulas/<name>.yaml to a registry: open a pull request
to kdeps/registry, or commit it to a registry of your own.

Examples:
  kdeps create https://github.com/acme/invoice-bot/archive/refs/tags/v1.2.0.tar.gz
  kdeps create --no-fetch https://github.com/acme/invoice-bot/archive/refs/tags/v1.2.0.tar.gz`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := createFormula(args[0], noFetch)
			if err != nil {
				return err
			}
			return printFormulaYAML(cmd.OutOrStdout(), f)
		},
	}
	c.Flags().BoolVar(&noFetch, "no-fetch", false, "do not download the archive; leave sha256 and type empty")
	return c
}

// createFormula builds the formula for the archive at rawURL.
func createFormula(rawURL string, noFetch bool) (formulas.Formula, error) {
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
		return formulas.Formula{}, fmt.Errorf("%q is not an http(s) archive URL", rawURL)
	}
	f := formulas.Formula{Tarball: rawURL}
	if m := githubArchive.FindStringSubmatch(rawURL); m != nil {
		f.GitHub = m[1] + m[2]
		f.Name = f.GitHub[strings.Index(f.GitHub, "/")+1:]
		f.Version = strings.TrimPrefix(m[3], "v")
	}
	if noFetch {
		if f.Name == "" {
			return f, errors.New(
				"--no-fetch needs a GitHub release URL (.../archive/refs/tags/<tag>.tar.gz) to name the package",
			)
		}
		return f, nil
	}
	return fetchFormula(f)
}

// fetchFormula downloads f.Tarball, hashes it, and reads its manifest.
func fetchFormula(f formulas.Formula) (formulas.Formula, error) {
	dir, err := os.MkdirTemp("", "kdeps-create-*")
	if err != nil {
		return f, err
	}
	defer os.RemoveAll(dir)
	archive := filepath.Join(dir, "package.tar.gz")
	if err = downloadArchive(f.Tarball, archive); err != nil {
		return f, fmt.Errorf("download %s: %w", f.Tarball, err)
	}
	if f.SHA256, err = fileSHA256(archive); err != nil {
		return f, err
	}
	out := filepath.Join(dir, "x")
	if err = extractArchive(archive, out); err != nil {
		return f, err
	}
	m, err := loadArchiveManifest(out)
	if err != nil {
		return f, err
	}
	f.Name, f.Version, f.Type = m.Name, m.Version, m.Type
	f.Description, f.Tags, f.License = m.Description, m.Tags, m.License
	return f, nil
}

// loadArchiveManifest reads kdeps.pkg.yaml at the archive root, or in its
// single top folder (a GitHub archive has one).
func loadArchiveManifest(root string) (*manifest.Manifest, error) {
	dir := root
	if _, err := os.Stat(filepath.Join(dir, manifest.ManifestFile)); err != nil {
		entries, _ := os.ReadDir(root)
		if len(entries) != 1 || !entries[0].IsDir() {
			return nil, fmt.Errorf("the archive has no %s", manifest.ManifestFile)
		}
		dir = filepath.Join(root, entries[0].Name())
	}
	m, err := manifest.Load(dir)
	if err != nil {
		return nil, err
	}
	if err = manifest.Validate(m); err != nil {
		return nil, err
	}
	return m, nil
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	h := sha256.New()
	if _, err = io.Copy(h, file); err != nil {
		return "", fmt.Errorf("hash %s: %w", path, err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// printFormulaYAML prints f with a comment that says where it goes.
func printFormulaYAML(w io.Writer, f formulas.Formula) error {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2) //nolint:mnd // YAML indent
	if err := enc.Encode(f); err != nil {
		return fmt.Errorf("encode formula: %w", err)
	}
	fmt.Fprintf(w, "# Save as formulas/%s.yaml in a registry, for example:\n", f.Name)
	fmt.Fprintf(w, "# https://github.com/%s/new/main/formulas?filename=%s.yaml\n\n", formulas.DefaultRegistry, f.Name)
	_, err := w.Write(buf.Bytes())
	return err
}
