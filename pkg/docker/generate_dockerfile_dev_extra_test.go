package docker

import (
	"strings"
	"testing"

	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/kdeps/kdeps/pkg/utils"
	"github.com/stretchr/testify/assert"
)

func TestGenerateDockerfile_DevBuildAndAPIServer(t *testing.T) {
	df := generateDockerfile(
		"1.2.3",              // image version
		"2.0",                // schema version
		"0.0.0.0",            // host IP
		"11434",              // ollama port
		"0.0.0.0:11434",      // kdeps host
		"ARG SAMPLE=1",       // args section
		"ENV FOO=bar",        // envs section
		"RUN apt-get update", // pkg section
		"RUN pip install x",  // python section
		"",                   // conda pkg section
		"2024.01-1",          // anaconda version
		"0.28.1",             // pkl version
		"UTC",                // timezone
		"8080",               // expose port
		false,                // installAnaconda
		true,                 // devBuildMode (exercise branch)
		true,                 // apiServerMode (expose port branch)
		false,                // useLatest
	)

	if !has(df, "cp /cache/kdeps /bin/kdeps") {
		t.Fatalf("expected dev build copy line")
	}
	if !has(df, "EXPOSE 8080") {
		t.Fatalf("expected expose port line")
	}
}

// small helper to avoid importing strings each time
func has(haystack, needle string) bool { return strings.Contains(haystack, needle) }

func TestGenerateDockerfileEdgeCasesNew(t *testing.T) {
	baseArgs := []interface{}{
		"latest",                     // imageVersion
		"1.0",                        // schemaVersion
		"127.0.0.1",                  // hostIP
		"11435",                      // ollamaPortNum
		"127.0.0.1:9090",             // kdepsHost
		"ARG FOO=bar",                // argsSection
		"ENV BAR=baz",                // envsSection
		"RUN apt-get install -y gcc", // pkgSection
		"",                           // pythonPkgSection
		"",                           // condaPkgSection
		"2024.10-1",                  // anacondaVersion
		"0.28.1",                     // pklVersion
		"UTC",                        // timezone
		"8080",                       // exposedPort
	}

	t.Run("devBuildMode", func(t *testing.T) {
		params := append(baseArgs, true /* installAnaconda */, true /* devBuildMode */, true /* apiServerMode */, false /* useLatest */)
		dockerfile := generateDockerfile(params[0].(string), params[1].(string), params[2].(string), params[3].(string), params[4].(string), params[5].(string), params[6].(string), params[7].(string), params[8].(string), params[9].(string), params[10].(string), params[11].(string), params[12].(string), params[13].(string), params[14].(bool), params[15].(bool), params[16].(bool), params[17].(bool))

		// Expect copy of kdeps binary due to devBuildMode true
		if !strings.Contains(dockerfile, "cp /cache/kdeps /bin/kdeps") {
			t.Fatalf("expected dev build copy step, got:\n%s", dockerfile)
		}
		// Anaconda installer should be present because installAnaconda true
		if !strings.Contains(dockerfile, "anaconda-linux-") {
			t.Fatalf("expected anaconda install snippet")
		}
		// Should expose port 8080 because apiServerMode true
		if !strings.Contains(dockerfile, "EXPOSE 8080") {
			t.Fatalf("expected EXPOSE directive")
		}
	})

	t.Run("prodBuildMode", func(t *testing.T) {
		params := append(baseArgs, false /* installAnaconda */, false /* devBuildMode */, false /* apiServerMode */, false /* useLatest */)
		dockerfile := generateDockerfile(params[0].(string), params[1].(string), params[2].(string), params[3].(string), params[4].(string), params[5].(string), params[6].(string), params[7].(string), params[8].(string), params[9].(string), params[10].(string), params[11].(string), params[12].(string), "", params[14].(bool), params[15].(bool), params[16].(bool), params[17].(bool))

		// Should pull kdeps via curl (not copy) because devBuildMode false
		if !strings.Contains(dockerfile, "raw.githubusercontent.com") {
			t.Fatalf("expected install kdeps via curl in prod build")
		}
		// Should not contain EXPOSE when apiServerMode false
		if strings.Contains(dockerfile, "EXPOSE") {
			t.Fatalf("did not expect EXPOSE directive when apiServerMode false")
		}
	})
}

// TestGenerateDockerfileAdditionalCases exercises seldom-hit branches in generateDockerfile so that
// coverage reflects real-world usage scenarios.
func TestGenerateDockerfileAdditionalCases(t *testing.T) {
	t.Run("DevBuildModeWithLatestAndExpose", func(t *testing.T) {
		result := generateDockerfile(
			"v1.2.3",                      // imageVersion
			"2.0",                         // schemaVersion
			"0.0.0.0",                     // hostIP
			"9999",                        // ollamaPortNum
			"kdeps.example",               // kdepsHost
			"ARG SAMPLE=1",                // argsSection
			"ENV FOO=bar",                 // envsSection
			"RUN apt-get -y install curl", // pkgSection
			"RUN pip install pytest",      // pythonPkgSection
			"",                            // condaPkgSection (none)
			"2024.10-1",                   // anacondaVersion (overwritten by useLatest=true below)
			"0.28.1",                      // pklVersion   (ditto)
			"UTC",                         // timezone
			"8080",                        // exposedPort
			true,                          // installAnaconda
			true,                          // devBuildMode  – should copy local kdeps binary
			true,                          // apiServerMode – should add EXPOSE line
			true,                          // useLatest     – should convert version marks to "latest"
		)

		// Ensure dev build mode path is present.
		assert.Contains(t, result, "cp /cache/kdeps /bin/kdeps", "expected dev build mode copy command")
		// When useLatest==true we expect the placeholder 'latest' to appear in pkl download section.
		assert.Contains(t, result, "pkl-linux-latest", "expected latest pkl artifact reference")
		// installAnaconda==true should result in anaconda installer copy logic.
		assert.Contains(t, result, "anaconda-linux-latest", "expected latest anaconda artifact reference")
		// apiServerMode==true adds an EXPOSE directive for provided port(s).
		assert.Contains(t, result, "EXPOSE 8080", "expected expose directive present")
	})

	t.Run("NonDevNoAnaconda", func(t *testing.T) {
		result := generateDockerfile(
			"stable",    // imageVersion
			"1.1",       // schemaVersion
			"127.0.0.1", // hostIP
			"1234",      // ollamaPortNum
			"host:1234", // kdepsHost
			"",          // argsSection
			"",          // envsSection
			"",          // pkgSection
			"",          // pythonPkgSection
			"",          // condaPkgSection
			"2024.10-1", // anacondaVersion
			"0.28.1",    // pklVersion
			"UTC",       // timezone
			"",          // exposedPort (no api server)
			false,       // installAnaconda
			false,       // devBuildMode
			false,       // apiServerMode – no EXPOSE
			false,       // useLatest
		)

		// Non-dev build should use install script instead of local binary.
		assert.Contains(t, result, "raw.githubusercontent.com/kdeps/kdeps", "expected remote install script usage")
		// Should NOT contain cp of anaconda because installAnaconda==false.
		assert.NotContains(t, result, "anaconda-linux", "unexpected anaconda installation commands present")
		// Should not contain EXPOSE directive.
		assert.NotContains(t, result, "EXPOSE", "unexpected expose directive present")
	})
}

func TestGenerateDockerfileContent(t *testing.T) {
	df := generateDockerfile(
		"10.1",          // imageVersion
		"v1",            // schemaVersion
		"127.0.0.1",     // hostIP
		"8000",          // ollamaPortNum
		"localhost",     // kdepsHost
		"ARG FOO=bar",   // argsSection
		"ENV BAR=baz",   // envsSection
		"# pkg section", // pkgSection
		"# python pkgs", // pythonPkgSection
		"# conda pkgs",  // condaPkgSection
		"2024.10-1",     // anacondaVersion
		"0.28.1",        // pklVersion
		"UTC",           // timezone
		"8080",          // exposedPort
		true,            // installAnaconda
		true,            // devBuildMode
		true,            // apiServerMode
		false,           // useLatest
	)

	// basic sanity checks on returned content
	assert.True(t, strings.Contains(df, "FROM ollama/ollama:10.1"))
	assert.True(t, strings.Contains(df, "ENV SCHEMA_VERSION=v1"))
	assert.True(t, strings.Contains(df, "EXPOSE 8080"))
	assert.True(t, strings.Contains(df, "ARG FOO=bar"))
	assert.True(t, strings.Contains(df, "ENV BAR=baz"))
}

// TestGenerateDockerfileBranchCoverage exercises additional parameter combinations
func TestGenerateDockerfileBranchCoverage(t *testing.T) {
	combos := []struct {
		installAnaconda bool
		devBuildMode    bool
		apiServerMode   bool
		useLatest       bool
	}{
		{false, false, false, true},
		{true, false, true, true},
		{false, true, false, false},
	}

	for _, c := range combos {
		df := generateDockerfile(
			"10.1",
			"v1",
			"127.0.0.1",
			"8000",
			"localhost",
			"",
			"",
			"",
			"",
			"",
			"2024.10-1",
			"0.28.1",
			"UTC",
			"8080",
			c.installAnaconda,
			c.devBuildMode,
			c.apiServerMode,
			c.useLatest,
		)
		// simple assertion to ensure function returns non-empty string
		assert.NotEmpty(t, df)
	}
}

// TestGenerateURLsHappyPath exercises the default code path where UseLatest is
// false. This avoids external HTTP requests yet covers several branches inside
// GenerateURLs including architecture substitution and local-name template
// logic.
func TestGenerateURLsHappyPath(t *testing.T) {
	ctx := context.Background()

	// Ensure the package-level flag is in the expected default state.
	schema.UseLatest = false

	items, err := GenerateURLs(ctx)
	if err != nil {
		t.Fatalf("GenerateURLs returned error: %v", err)
	}

	// We expect two items (one for Pkl and one for Anaconda).
	if len(items) != 2 {
		t.Fatalf("expected 2 download items, got %d", len(items))
	}

	// Basic sanity checks on the generated URLs/local names – just ensure they
	// contain expected substrings so that we're not overly sensitive to exact
	// versions or architecture values.
	for _, itm := range items {
		if itm.URL == "" {
			t.Fatalf("item URL is empty: %+v", itm)
		}
		if itm.LocalName == "" {
			t.Fatalf("item LocalName is empty: %+v", itm)
		}
	}
}

// rtFunc already declared in another test file; reuse that type here without redefining.

func TestGenerateURLs_GitHubError(t *testing.T) {
	ctx := context.Background()

	// Save globals and transport.
	origLatest := schema.UseLatest
	origTransport := http.DefaultTransport
	defer func() {
		schema.UseLatest = origLatest
		http.DefaultTransport = origTransport
	}()

	schema.UseLatest = true

	// Force GitHub API request to return HTTP 403.
	http.DefaultTransport = rtFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host == "api.github.com" {
			return &http.Response{
				StatusCode: 403,
				Body:       ioutil.NopCloser(bytes.NewBufferString("forbidden")),
				Header:     make(http.Header),
			}, nil
		}
		return origTransport.RoundTrip(r)
	})

	if _, err := GenerateURLs(ctx); err == nil {
		t.Fatalf("expected error when GitHub API returns forbidden")
	}
}

func TestGenerateURLs_AnacondaError(t *testing.T) {
	ctx := context.Background()

	// Save and restore globals and transport.
	origLatest := schema.UseLatest
	origFetcher := utils.GitHubReleaseFetcher
	origTransport := http.DefaultTransport
	defer func() {
		schema.UseLatest = origLatest
		utils.GitHubReleaseFetcher = origFetcher
		http.DefaultTransport = origTransport
	}()

	// GitHub fetch succeeds to move past first item.
	schema.UseLatest = true
	utils.GitHubReleaseFetcher = func(ctx context.Context, repo, base string) (string, error) {
		return "0.28.1", nil
	}

	// Make Anaconda request return HTTP 500.
	http.DefaultTransport = rtFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host == "repo.anaconda.com" {
			return &http.Response{
				StatusCode: 500,
				Body:       ioutil.NopCloser(bytes.NewBufferString("server error")),
				Header:     make(http.Header),
			}, nil
		}
		return origTransport.RoundTrip(r)
	})

	if _, err := GenerateURLs(ctx); err == nil {
		t.Fatalf("expected error when Anaconda version fetch fails")
	}
}

func TestGenerateURLs(t *testing.T) {
	ctx := context.Background()

	items, err := GenerateURLs(ctx)
	if err != nil {
		t.Fatalf("unexpected error generating urls: %v", err)
	}
	if len(items) == 0 {
		t.Fatalf("expected at least one download item")
	}

	for _, itm := range items {
		if itm.URL == "" || itm.LocalName == "" {
			t.Errorf("item fields should not be empty: %+v", itm)
		}
	}
}

func TestGenerateURLsDefaultExtra(t *testing.T) {
	ctx := context.Background()
	items, err := GenerateURLs(ctx)
	if err != nil {
		t.Fatalf("GenerateURLs returned error: %v", err)
	}
	if len(items) == 0 {
		t.Fatalf("expected at least one download item")
	}
	for _, it := range items {
		if it.URL == "" || it.LocalName == "" {
			t.Fatalf("invalid item %+v", it)
		}
	}
}

type roundTripperLatest struct{}

func (roundTripperLatest) RoundTrip(req *http.Request) (*http.Response, error) {
	// Distinguish responses based on requested URL path.
	switch {
	case req.URL.Host == "api.github.com":
		// Fake GitHub release JSON.
		body, _ := json.Marshal(map[string]string{"tag_name": "v0.29.0"})
		return &http.Response{StatusCode: http.StatusOK, Body: ioNopCloser(bytes.NewReader(body)), Header: make(http.Header)}, nil
	case req.URL.Host == "repo.anaconda.com":
		html := `<a href="Anaconda3-2024.05-0-Linux-x86_64.sh">file</a><a href="Anaconda3-2024.05-0-Linux-aarch64.sh">file</a>`
		return &http.Response{StatusCode: http.StatusOK, Body: ioNopCloser(bytes.NewReader([]byte(html))), Header: make(http.Header)}, nil
	default:
		return &http.Response{StatusCode: http.StatusOK, Body: ioNopCloser(bytes.NewReader([]byte(""))), Header: make(http.Header)}, nil
	}
}

type nopCloser struct{ *bytes.Reader }

func (n nopCloser) Close() error { return nil }

func ioNopCloser(r *bytes.Reader) io.ReadCloser { return nopCloser{r} }

func TestGenerateURLsUseLatest(t *testing.T) {
	// Mock HTTP.
	origTransport := http.DefaultTransport
	http.DefaultTransport = roundTripperLatest{}
	defer func() { http.DefaultTransport = origTransport }()

	// Enable latest mode.
	origLatest := schema.UseLatest
	schema.UseLatest = true
	defer func() { schema.UseLatest = origLatest }()

	ctx := context.Background()
	items, err := GenerateURLs(ctx)
	if err != nil {
		t.Fatalf("GenerateURLs returned error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	for _, itm := range items {
		if itm.LocalName == "" || itm.URL == "" {
			t.Fatalf("GenerateURLs produced empty fields: %+v", itm)
		}
		if !schema.UseLatest {
			t.Fatalf("schema.UseLatest should still be true inside loop")
		}
	}
}

func TestGenerateURLsLatestUsesFetcher(t *testing.T) {
	ctx := context.Background()

	// Save globals and restore afterwards
	orig := schema.UseLatest
	fetchOrig := utils.GitHubReleaseFetcher
	defer func() {
		schema.UseLatest = orig
		utils.GitHubReleaseFetcher = fetchOrig
	}()

	schema.UseLatest = true
	utils.GitHubReleaseFetcher = func(ctx context.Context, repo string, baseURL string) (string, error) {
		return "0.99.0", nil
	}

	items, err := GenerateURLs(ctx)
	if err != nil {
		t.Fatalf("GenerateURLs error: %v", err)
	}
	found := false
	for _, it := range items {
		if it.LocalName == "pkl-linux-latest-"+GetCurrentArchitecture(ctx, "apple/pkl") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected pkl latest local name element, got %+v", items)
	}
}
