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
	"archive/tar"
	"bytes"
	"compress/gzip"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// releaseArchive is a GitHub-style archive: one top folder with kdeps.pkg.yaml.
func releaseArchive(t *testing.T, manifestYAML string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, body := range map[string]string{
		"invoice-bot-1.2.0/kdeps.pkg.yaml": manifestYAML,
		"invoice-bot-1.2.0/workflow.yaml":  "apiVersion: kdeps.io/v1\nkind: Workflow\n",
	} {
		require.NoError(
			t,
			tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(body)), Typeflag: tar.TypeReg}),
		)
		_, err := tw.Write([]byte(body))
		require.NoError(t, err)
	}
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
	return buf.Bytes()
}

func TestCreateFormula_FetchesAndReadsManifest(t *testing.T) {
	body := releaseArchive(
		t,
		"name: invoice-bot\nversion: 1.2.0\ntype: workflow\ndescription: Reads invoices\nlicense: MIT\n",
	)
	srv := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	f, err := createFormula(srv.URL+"/invoice-bot-1.2.0.tar.gz", false)
	require.NoError(t, err)
	assert.Equal(t, "invoice-bot", f.Name)
	assert.Equal(t, "1.2.0", f.Version)
	assert.Equal(t, "workflow", f.Type)
	assert.Equal(t, "MIT", f.License)
	assert.Len(t, f.SHA256, 64)

	var out bytes.Buffer
	require.NoError(t, printFormulaYAML(&out, f))
	assert.Contains(t, out.String(), "# Save as formulas/invoice-bot.yaml")
	assert.Contains(t, out.String(), "https://github.com/kdeps/registry/new/main/formulas?filename=invoice-bot.yaml")
	assert.Contains(t, out.String(), "sha256: "+f.SHA256)
}

func TestCreateFormula_NoFetchGitHubRelease(t *testing.T) {
	f, err := createFormula("https://github.com/acme/invoice-bot/archive/refs/tags/v1.2.0.tar.gz", true)
	require.NoError(t, err)
	assert.Equal(t, "acme/invoice-bot", f.GitHub)
	assert.Equal(t, "invoice-bot", f.Name)
	assert.Equal(t, "1.2.0", f.Version)
	assert.Empty(t, f.SHA256)

	f, err = createFormula("https://codeload.github.com/acme/tool/tar.gz/refs/tags/2.0.0", true)
	require.NoError(t, err)
	assert.Equal(t, "acme/tool", f.GitHub)
	assert.Equal(t, "2.0.0", f.Version)
}

func TestCreateFormula_Errors(t *testing.T) {
	_, err := createFormula("ftp://x/y.tar.gz", false)
	require.ErrorContains(t, err, "not an http(s) archive URL")
	_, err = createFormula("https://example.com/pkg.tar.gz", true)
	require.ErrorContains(t, err, "--no-fetch needs a GitHub release URL")

	srv := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
		w.WriteHeader(stdhttp.StatusNotFound)
	}))
	defer srv.Close()
	_, err = createFormula(srv.URL+"/missing.tar.gz", false)
	require.ErrorContains(t, err, "download")

	body := releaseArchive(t, "name: invoice-bot\n")
	bad := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
		_, _ = w.Write(body)
	}))
	defer bad.Close()
	_, err = createFormula(bad.URL+"/a.tar.gz", false)
	require.Error(t, err)
}

func TestCommandsAndShortHelp(t *testing.T) {
	root := createRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"commands"})
	require.NoError(t, root.Execute())
	assert.Regexp(t, `(?m)^create +Write a formula`, out.String())
	assert.Regexp(t, `(?m)^upgrade +Upgrade`, out.String())
	assert.NotContains(t, out.String(), "completion")

	out.Reset()
	root.SetArgs([]string{"--help"})
	require.NoError(t, root.Execute())
	assert.Contains(t, out.String(), "kdeps create URL [--no-fetch]")

	out.Reset()
	root.SetArgs([]string{"search", "--help"})
	require.NoError(t, root.Execute())
	assert.Contains(t, out.String(), "Usage:")
}
