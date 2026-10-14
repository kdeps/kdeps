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

package assets

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"golang.org/x/mod/semver"
	"gopkg.in/yaml.v3"
)

// ResourcePackageType is the registry package type of a built-in resource
// type (scraper:, httpClient:, ...). They ship in the kdeps binary, so
// their registry entry carries the resource's docs page, not installable code.
const ResourcePackageType = "resource"

// ResourcesDir is the kdeps/packages directory resource docs are published to.
const ResourcesDir = "resources"

// DocsSite is where the docs pages' site-relative links point.
const DocsSite = "https://kdeps.com"

// ResourceDocs maps each resource action key (its YAML key in a resource) to
// its docs page, relative to docs/v2.
//
//nolint:gochecknoglobals // fixed key -> page table
var ResourceDocs = map[string]string{
	"chat":             "workflow/resources/llm.md",
	"httpClient":       "workflow/resources/http-client.md",
	"sql":              "workflow/resources/sql.md",
	"python":           "workflow/resources/python.md",
	"exec":             "workflow/resources/exec.md",
	"agent":            "agencies/agent-resource.md",
	"component":        "agencies/component-resource.md",
	"embedding":        "workflow/resources/embedding.md",
	"searchLocal":      "workflow/resources/search-local.md",
	"telephony":        "workflow/resources/telephony.md",
	"browser":          "workflow/resources/browser.md",
	"botReply":         "workflow/resources/bot-reply.md",
	"email":            "workflow/resources/email.md",
	"codeIntelligence": "workflow/resources/code-intelligence.md",
	"loader":           "workflow/resources/loader.md",
	"vectorStore":      "workflow/resources/vector-store.md",
	"apiResponse":      "workflow/resources/api-response.md",
}

// ResourcePackageName is the registry package name of a resource key:
// kdeps-resource-<kebab key>, e.g. httpClient -> kdeps-resource-http-client.
func ResourcePackageName(key string) string {
	var b strings.Builder
	for i, r := range key {
		if unicode.IsUpper(r) {
			if i > 0 {
				b.WriteByte('-')
			}
			r = unicode.ToLower(r)
		}
		b.WriteRune(r)
	}
	return formulaPrefix + ResourcePackageType + "-" + b.String()
}

var (
	siteLink  = regexp.MustCompile(`\]\(/`)
	vitePress = regexp.MustCompile(`(?m)^</?div( v-pre)?>\n`)
)

// ResourceDoc is a docs page made self-contained for the registry:
// site-relative links point at DocsSite and VitePress-only wrappers are gone.
func ResourceDoc(page []byte) []byte {
	out := vitePress.ReplaceAll(page, nil)
	return siteLink.ReplaceAll(out, []byte("]("+DocsSite+"/"))
}

// WriteResources publishes every resource type at version: its docs page to
// filesDir/<key>/<version>.md (served from fileBase) and its formula to
// formulasDir. docsDir is docs/v2 of a kdeps checkout.
func WriteResources(formulasDir, filesDir, docsDir, fileBase, version string) error {
	if !semver.IsValid("v" + version) {
		return fmt.Errorf("assets: resources: invalid version %q", version)
	}
	keys := make([]string, 0, len(ResourceDocs))
	for k := range ResourceDocs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		page, err := os.ReadFile(filepath.Join(docsDir, filepath.FromSlash(ResourceDocs[key])))
		if err != nil {
			return fmt.Errorf("assets: resource %s: %w", key, err)
		}
		doc := ResourceDoc(page)
		rel := key + "/" + version + ".md"
		dst := filepath.Join(filesDir, filepath.FromSlash(rel))
		if mkErr := os.MkdirAll(filepath.Dir(dst), dirPerm); mkErr != nil {
			return mkErr
		}
		if wErr := os.WriteFile(dst, doc, 0o644); wErr != nil { //nolint:gosec // public docs
			return wErr
		}
		f := Formula{
			Name:        ResourcePackageName(key),
			Version:     version,
			Type:        ResourcePackageType,
			GitHub:      PackagesRepo,
			Tarball:     strings.TrimRight(fileBase, "/") + "/" + rel,
			SHA256:      Sum(doc),
			Description: docSummary(doc, key),
			Tags:        []string{"kdeps", ResourcePackageType, key},
			License:     "Apache-2.0",
		}
		if wErr := writeFormula(formulasDir, f); wErr != nil {
			return wErr
		}
	}
	return pruneResourceFormulas(formulasDir)
}

// pruneResourceFormulas deletes the formulas of resource types kdeps no
// longer has (a type moved to a component, say), so the registry stops
// listing them.
func pruneResourceFormulas(formulasDir string) error {
	entries, err := os.ReadDir(formulasDir)
	if err != nil {
		return err
	}
	keep := make(map[string]bool, len(ResourceDocs))
	for k := range ResourceDocs {
		keep[ResourcePackageName(k)+yamlExt] = true
	}
	prefix := formulaPrefix + ResourcePackageType + "-"
	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), prefix) || keep[e.Name()] {
			continue
		}
		if rmErr := os.Remove(filepath.Join(formulasDir, e.Name())); rmErr != nil {
			return rmErr
		}
	}
	return nil
}

// docSummary is the page's first paragraph line after its title.
func docSummary(doc []byte, key string) string {
	for _, line := range strings.Split(string(doc), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		return line
	}
	return "kdeps resource: " + key
}

func writeFormula(dir string, f Formula) error {
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return err
	}
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2) //nolint:mnd // registry formula indent
	if err := enc.Encode(f); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, f.Name+yamlExt), buf.Bytes(), 0o644) //nolint:gosec // public formula
}
