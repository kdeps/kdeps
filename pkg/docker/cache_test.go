package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"runtime"
	"strings"
	"testing"

	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/kdeps/kdeps/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetCurrentArchitectureDup(t *testing.T) {
	ctx := context.Background()

	var expected string
	if archMap, ok := archMappings["apple/pkl"]; ok {
		if mapped, exists := archMap[runtime.GOARCH]; exists {
			expected = mapped
		}
	}
	// Fallback to default mapping only if apple/pkl did not contain entry
	if expected == "" {
		if defaultMap, ok := archMappings["default"]; ok {
			if mapped, exists := defaultMap[runtime.GOARCH]; exists {
				expected = mapped
			}
		}
	}
	if expected == "" {
		expected = runtime.GOARCH
	}

	arch := GetCurrentArchitecture(ctx, "apple/pkl")
	assert.Equal(t, expected, arch)
}

func TestCompareVersionsDup(t *testing.T) {
	ctx := context.Background()

	assert.True(t, CompareVersions(ctx, "2.0.0", "1.9.9"))
	assert.False(t, CompareVersions(ctx, "1.0.0", "1.0.0"))
	assert.False(t, CompareVersions(ctx, "1.2.3", "1.2.4"))
	// Mixed length versions
	assert.True(t, CompareVersions(ctx, "1.2.3", "1.2"))
	assert.False(t, CompareVersions(ctx, "1.2", "1.2.3"))
}

func TestParseVersion(t *testing.T) {
	t.Run("Simple", func(t *testing.T) {
		parts := parseVersion("1.2.3")
		assert.Equal(t, []int{1, 2, 3}, parts)
	})

	t.Run("WithHyphen", func(t *testing.T) {
		parts := parseVersion("1-2-3")
		assert.Equal(t, []int{1, 2, 3}, parts)
	})
}

func TestBuildURL(t *testing.T) {
	base := "https://example.com/download/{version}/app-{arch}"
	url := buildURL(base, "1.0.0", "x86_64")
	assert.Equal(t, "https://example.com/download/1.0.0/app-x86_64", url)
}

func TestGenerateURLs_DefaultVersion(t *testing.T) {
	// Ensure we are not in latest mode to avoid network calls
	schemaUseLatestBackup := schema.UseLatest
	schema.UseLatest = false
	defer func() { schema.UseLatest = schemaUseLatestBackup }()

	ctx := context.Background()
	items, err := GenerateURLs(ctx, true)
	assert.NoError(t, err)
	assert.NotEmpty(t, items)

	// verify each item has URL and LocalName populated
	for _, item := range items {
		assert.NotEmpty(t, item.URL)
		assert.NotEmpty(t, item.LocalName)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// helper to build *http.Response
func buildResp(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     make(http.Header),
	}
}

func TestGetLatestAnacondaVersionsSuccess(t *testing.T) {
	html := `Anaconda3-20.3.1-dev7-1-Linux-x86_64.sh Anaconda3-20.3.1-dev5-1-Linux-aarch64.sh` +
		` Anaconda3-2024.10-1-Linux-x86_64.sh Anaconda3-2024.08-1-Linux-aarch64.sh`

	// mock transport
	old := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host == "repo.anaconda.com" {
			return buildResp(http.StatusOK, html), nil
		}
		return old.RoundTrip(r)
	})
	defer func() { http.DefaultTransport = old }()

	ctx := context.Background()
	versions, err := GetLatestAnacondaVersions(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if versions["x86_64"] != "2024.10-1" || versions["aarch64"] != "2024.08-1" {
		t.Fatalf("unexpected versions: %v", versions)
	}

	_ = schema.SchemaVersion(ctx)
}

func TestGetLatestAnacondaVersionsErrors(t *testing.T) {
	cases := []struct {
		status int
		body   string
		expect string
	}{
		{http.StatusInternalServerError, "", "unexpected status"},
		{http.StatusOK, "no matches", "no Anaconda versions"},
	}

	for _, c := range cases {
		old := http.DefaultTransport
		http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return buildResp(c.status, c.body), nil
		})
		ctx := context.Background()
		_, err := GetLatestAnacondaVersions(ctx)
		if err == nil {
			t.Fatalf("expected error for case %+v", c)
		}
		http.DefaultTransport = old
	}

	_ = schema.SchemaVersion(context.Background())
}

type archHTMLTransport struct{}

func (archHTMLTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	html := `<html><body>
        <a href="Anaconda3-2024.10-1-Linux-x86_64.sh">x</a>
        <a href="Anaconda3-2024.09-1-Linux-aarch64.sh">y</a>
        <a href="Anaconda3-2023.12-0-Linux-x86_64.sh">old-x</a>
        <a href="Anaconda3-20.3.1-dev1-0-Linux-aarch64.sh">old-y</a>
        </body></html>`
	return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewBufferString(html)), Header: make(http.Header)}, nil
}

func TestGetLatestAnacondaVersionsMultiArch(t *testing.T) {
	ctx := context.Background()

	oldTransport := http.DefaultTransport
	http.DefaultTransport = archHTMLTransport{}
	defer func() { http.DefaultTransport = oldTransport }()

	versions, err := GetLatestAnacondaVersions(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if versions["x86_64"] != "2024.10-1" {
		t.Fatalf("unexpected version for x86_64: %s", versions["x86_64"])
	}
	if versions["aarch64"] != "2024.09-1" {
		t.Fatalf("unexpected version for aarch64: %s", versions["aarch64"])
	}
}

// mockTransport intercepts HTTP requests to repo.anaconda.com and returns fixed HTML.
type mockHTMLTransport struct{}

func (m mockHTMLTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Host == "repo.anaconda.com" {
		html := `<html><body>
<a href="Anaconda3-2024.10-1-Linux-x86_64.sh">Anaconda3-2024.10-1-Linux-x86_64.sh</a>
<a href="Anaconda3-2024.09-1-Linux-aarch64.sh">Anaconda3-2024.09-1-Linux-aarch64.sh</a>
</body></html>`
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(bytes.NewBufferString(html)),
			Header:     make(http.Header),
		}
		return resp, nil
	}
	return nil, http.ErrUseLastResponse
}

func TestGetLatestAnacondaVersionsMockSimple(t *testing.T) {
	// Replace the default transport
	origTransport := http.DefaultTransport
	http.DefaultTransport = mockHTMLTransport{}
	defer func() { http.DefaultTransport = origTransport }()

	ctx := context.Background()
	vers, err := GetLatestAnacondaVersions(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vers["x86_64"] != "2024.10-1" {
		t.Fatalf("x86_64 version mismatch, got %s", vers["x86_64"])
	}
	if vers["aarch64"] != "2024.09-1" {
		t.Fatalf("aarch64 version mismatch, got %s", vers["aarch64"])
	}
}

func TestCompareVersions_EdgeCases(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		a, b    string
		greater bool // whether a>b expected
	}{
		{"1.0.0-alpha", "1.0.0", false},
		{"1.0.1", "1.0.0-beta", true},
		{"1.0", "1.0.0", false},
		{"2", "10", false},
		{"0.0.0", "0", false},
	}

	for _, c := range cases {
		got := CompareVersions(ctx, c.a, c.b)
		if got != c.greater {
			t.Fatalf("CompareVersions(%s,%s)=%v want %v", c.a, c.b, got, c.greater)
		}
	}
}

func TestCompareVersionsMore(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		v1, v2  string
		greater bool
	}{
		{"1.2.3", "1.2.2", true},
		{"1.2.0", "1.2", false},
		{"1.2.10", "1.3", false},
		{"2.0.0", "2.0.0", false},
		{"1.2.3-alpha", "1.2.3", false},
	}

	for _, c := range cases {
		got := CompareVersions(ctx, c.v1, c.v2)
		if got != c.greater {
			t.Errorf("CompareVersions(%s,%s)=%v, want %v", c.v1, c.v2, got, c.greater)
		}
	}
}

func TestGetCurrentArchitectureMapping(t *testing.T) {
	ctx := context.Background()

	arch := GetCurrentArchitecture(ctx, "apple/pkl")
	_ = schema.SchemaVersion(ctx)
	want := map[string]string{"amd64": "amd64", "arm64": "aarch64"}[runtime.GOARCH]
	if arch != want {
		t.Errorf("mapping mismatch for apple/pkl: got %s want %s", arch, want)
	}

	// default mapping path
	arch2 := GetCurrentArchitecture(ctx, "unknown/repo")
	def := map[string]string{"amd64": "x86_64", "arm64": "aarch64"}[runtime.GOARCH]
	if arch2 != def {
		t.Errorf("default mapping mismatch: got %s want %s", arch2, def)
	}
}

func TestParseVersionParts(t *testing.T) {
	got := parseVersion("1.2.3")
	want := []int{1, 2, 3}
	if len(got) != len(want) {
		t.Fatalf("expected length %d, got %d", len(want), len(got))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("parseVersion mismatch at index %d: want %d got %d", i, want[i], got[i])
		}
	}
}

func TestCompareVersionsEdge(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		v1, v2 string
		want   bool
	}{
		{"1.2.3", "1.2.2", true},  // greater
		{"2.0.0", "2.0.0", false}, // equal
		{"1.0.0", "1.0.1", false}, // less
		{"1.10", "1.9", true},     // numeric compare not lexicographic
	}

	for _, c := range cases {
		got := CompareVersions(ctx, c.v1, c.v2)
		if got != c.want {
			t.Errorf("CompareVersions(%s,%s) = %v, want %v", c.v1, c.v2, got, c.want)
		}
	}
}

func TestBuildURLReplacer(t *testing.T) {
	base := "https://example.com/{version}/{arch}/download"
	url := buildURL(base, "1.0.0", "x86_64")
	expected := "https://example.com/1.0.0/x86_64/download"
	if url != expected {
		t.Fatalf("buildURL mismatch: got %s, want %s", url, expected)
	}
}

func TestGetCurrentArchitectureDefault(t *testing.T) {
	ctx := context.Background()
	arch := GetCurrentArchitecture(ctx, "apple/pkl")
	_ = schema.SchemaVersion(ctx)

	switch runtime.GOARCH {
	case "amd64":
		if arch != "amd64" {
			t.Fatalf("expected amd64 mapping, got %s", arch)
		}
	case "arm64":
		if arch != "aarch64" {
			t.Fatalf("expected aarch64 mapping for arm64, got %s", arch)
		}
	default:
		if arch != runtime.GOARCH {
			t.Fatalf("expected arch to match runtime (%s), got %s", runtime.GOARCH, arch)
		}
	}
}

func TestGetCurrentArchitectureMappingExtra(t *testing.T) {
	ctx := context.Background()
	repo := "apple/pkl"
	arch := GetCurrentArchitecture(ctx, repo)
	// Validate against mapping table.
	goArch := runtime.GOARCH
	expected := archMappings[repo][goArch]
	if expected == "" {
		expected = archMappings["default"][goArch]
		if expected == "" {
			expected = goArch
		}
	}
	if arch != expected {
		t.Fatalf("expected %s, got %s", expected, arch)
	}
}

func TestCompareVersionsExtra(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		v1, v2 string
		newer  bool
	}{
		{"1.2.3", "1.2.2", true},
		{"1.2.3", "1.2.3", false},
		{"1.2.3", "1.3.0", false},
		{"2.0", "1.9.9", true},
		{"1.0.0", "1.0", false},
	}
	for _, c := range cases {
		got := CompareVersions(ctx, c.v1, c.v2)
		if got != c.newer {
			t.Fatalf("CompareVersions(%s,%s)=%v want %v", c.v1, c.v2, got, c.newer)
		}
	}
}

func TestBuildURLExtra(t *testing.T) {
	url := buildURL("https://example.com/{version}/bin-{arch}", "v1.0", "x86_64")
	expected := "https://example.com/v1.0/bin-x86_64"
	if url != expected {
		t.Fatalf("expected %s, got %s", expected, url)
	}
}

func TestGenerateURLs_NoLatest(t *testing.T) {
	ctx := context.Background()
	originalLatest := schema.UseLatest
	schema.UseLatest = false
	defer func() { schema.UseLatest = originalLatest }()

	items, err := GenerateURLs(ctx, true)
	require.NoError(t, err)
	// Expect 2 items for supported architectures (pkl + anaconda) relevant to current arch
	require.Len(t, items, 2)

	// Basic validation each item populated
	for _, it := range items {
		require.NotEmpty(t, it.URL)
		require.NotEmpty(t, it.LocalName)
	}
}

type multiMockTransport struct{}

func (m multiMockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	switch req.URL.Host {
	case "api.github.com":
		body, _ := json.Marshal(map[string]string{"tag_name": "v9.9.9"})
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(body)), Header: make(http.Header)}, nil
	case "repo.anaconda.com":
		html := `<a href="Anaconda3-2025.01-0-Linux-x86_64.sh">Anaconda3-2025.01-0-Linux-x86_64.sh</a>`
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewBufferString(html)), Header: make(http.Header)}, nil
	default:
		return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(bytes.NewBuffer(nil)), Header: make(http.Header)}, nil
	}
}

func TestGenerateURLsLatestMode(t *testing.T) {
	// Enable latest mode
	schema.UseLatest = true
	defer func() { schema.UseLatest = false }()

	origTransport := http.DefaultTransport
	http.DefaultTransport = multiMockTransport{}
	defer func() { http.DefaultTransport = origTransport }()

	ctx := context.Background()
	items, err := GenerateURLs(ctx, true)
	if err != nil {
		t.Fatalf("GenerateURLs latest failed: %v", err)
	}
	if len(items) == 0 {
		t.Fatalf("expected items when latest mode enabled")
	}
	// All LocalName fields should contain "latest" placeholder
	for _, it := range items {
		if it.LocalName == "" {
			t.Fatalf("missing LocalName")
		}
		// When UseLatest=true, filenames should contain "latest" to match Dockerfile template expectations
		if !contains(it.LocalName, "latest") {
			t.Fatalf("LocalName should reference latest: %s", it.LocalName)
		}
	}
}

func contains(s, sub string) bool { return bytes.Contains([]byte(s), []byte(sub)) }

func TestGenerateURLsBasic(t *testing.T) {
	ctx := context.Background()
	// Ensure deterministic behaviour
	schema.UseLatest = false

	items, err := GenerateURLs(ctx, true)
	if err != nil {
		t.Fatalf("GenerateURLs returned error: %v", err)
	}
	if len(items) == 0 {
		t.Fatalf("GenerateURLs returned no items")
	}
	for _, it := range items {
		if it.URL == "" {
			t.Fatalf("item has empty URL")
		}
		if it.LocalName == "" {
			t.Fatalf("item has empty LocalName")
		}
	}
}

type stubRoundTrip func(*http.Request) (*http.Response, error)

func (f stubRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestGenerateURLs_UseLatestWithStubsLow(t *testing.T) {
	// Stub GitHub release fetcher to avoid network
	origFetcher := utils.GitHubReleaseFetcher
	utils.GitHubReleaseFetcher = func(ctx context.Context, repo, baseURL string) (string, error) {
		return "99.99.99", nil
	}
	defer func() { utils.GitHubReleaseFetcher = origFetcher }()

	// Intercept HTTP requests for both Anaconda archive and GitHub API
	origTransport := http.DefaultTransport
	http.DefaultTransport = stubRoundTrip(func(req *http.Request) (*http.Response, error) {
		var body string
		if strings.Contains(req.URL.Host, "repo.anaconda.com") {
			body = `Anaconda3-2024.10-1-Linux-x86_64.sh Anaconda3-2024.10-1-Linux-aarch64.sh`
		} else {
			body = `{"tag_name":"v99.99.99"}`
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
		}, nil
	})
	defer func() { http.DefaultTransport = origTransport }()

	schema.UseLatest = true
	defer func() { schema.UseLatest = false }()

	items, err := GenerateURLs(context.Background(), true)
	if err != nil {
		t.Fatalf("GenerateURLs error: %v", err)
	}
	if len(items) == 0 {
		t.Fatalf("expected non-empty items")
	}
	for _, it := range items {
		// When UseLatest=true, filenames should contain "latest" to match Dockerfile template expectations
		if !strings.Contains(it.LocalName, "latest") {
			t.Fatalf("expected LocalName to contain latest, got %s", it.LocalName)
		}
	}
}

// mockTransport intercepts HTTP requests and serves canned responses.
type mockTransport struct{}

func (m mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var body string
	if strings.Contains(req.URL.Path, "/releases/latest") { // GitHub API
		body = `{"tag_name":"v1.2.3"}`
	} else { // Anaconda archive listing
		body = `Anaconda3-2024.05-0-Linux-x86_64.sh
Anaconda3-2024.05-0-Linux-aarch64.sh`
	}
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
	return resp, nil
}

func TestGenerateURLs_UseLatest(t *testing.T) {
	// Save and restore globals we mutate.
	origLatest := schema.UseLatest
	origFetcher := utils.GitHubReleaseFetcher
	origTransport := http.DefaultTransport
	defer func() {
		schema.UseLatest = origLatest
		utils.GitHubReleaseFetcher = origFetcher
		http.DefaultTransport = origTransport
	}()

	schema.UseLatest = true

	// Stub GitHub release fetcher.
	utils.GitHubReleaseFetcher = func(ctx context.Context, repo, baseURL string) (string, error) {
		return "v9.9.9", nil
	}

	// Intercept Anaconda archive request.
	http.DefaultTransport = mockTransport{}

	items, err := GenerateURLs(context.Background(), true)
	assert.NoError(t, err)
	assert.NotEmpty(t, items)

	// Ensure an item for pkl latest and anaconda latest exist.
	var gotPkl, gotAnaconda bool
	for _, it := range items {
		if strings.Contains(it.LocalName, "pkl-linux-latest") {
			gotPkl = true
		}
		if strings.Contains(it.LocalName, "anaconda-linux-latest") {
			gotAnaconda = true
		}
	}
	assert.True(t, gotPkl, "expected pkl latest item")
	assert.True(t, gotAnaconda, "expected anaconda latest item")
}

type roundTripFuncAnaconda func(*http.Request) (*http.Response, error)

func (f roundTripFuncAnaconda) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestGetLatestAnacondaVersions(t *testing.T) {
	// sample HTML page snippet with versions
	html := `
        <a href="Anaconda3-2024.10-1-Linux-x86_64.sh">x86</a>
        <a href="Anaconda3-2023.12-0-Linux-x86_64.sh">old</a>
        <a href="Anaconda3-2024.10-1-Linux-aarch64.sh">arm</a>
    `

	// Mock transport to return above HTML for any request
	origTransport := http.DefaultTransport
	http.DefaultTransport = roundTripFuncAnaconda(func(r *http.Request) (*http.Response, error) {
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(html)),
			Header:     make(http.Header),
		}
		return resp, nil
	})
	defer func() { http.DefaultTransport = origTransport }()

	versions, err := GetLatestAnacondaVersions(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, "2024.10-1", versions["x86_64"])
	assert.Equal(t, "2024.10-1", versions["aarch64"])
}

func TestBuildURLAndArchMappingLow(t *testing.T) {
	_ = schema.SchemaVersion(context.Background())

	base := "https://example.com/{version}/{arch}/binary"
	url := buildURL(base, "1.2.3", "x86_64")
	want := "https://example.com/1.2.3/x86_64/binary"
	if url != want {
		t.Fatalf("buildURL mismatch: got %s want %s", url, want)
	}

	arch := runtime.GOARCH // expect mapping fall-through works
	ctx := context.Background()
	got := GetCurrentArchitecture(ctx, "unknown/repo")
	var expect string
	if m, ok := archMappings["default"]; ok {
		if v, ok2 := m[arch]; ok2 {
			expect = v
		} else {
			expect = arch
		}
	}
	if got != expect {
		t.Fatalf("GetCurrentArchitecture fallback = %s; want %s", got, expect)
	}
}

func TestGenerateURLs_NoLatestLow(t *testing.T) {
	// Ensure UseLatest is false for deterministic output
	schema.UseLatest = false
	ctx := context.Background()
	urls, err := GenerateURLs(ctx, true)
	if err != nil {
		t.Fatalf("GenerateURLs error: %v", err)
	}
	if len(urls) == 0 {
		t.Fatalf("expected some URLs")
	}

	// Each item should have LocalName containing version, not "latest"
	for _, it := range urls {
		if strings.Contains(it.LocalName, "latest") {
			t.Fatalf("LocalName should not contain 'latest' when UseLatest=false: %s", it.LocalName)
		}
		if it.URL == "" || it.LocalName == "" {
			t.Fatalf("got empty fields in item %+v", it)
		}
	}
}

// TestGenerateURLsDefault verifies that GenerateURLs returns the expected
// download items when schema.UseLatest is false.
func TestGenerateURLsDefault(t *testing.T) {
	ctx := context.Background()

	// Ensure we are testing the static version path.
	original := schema.UseLatest
	schema.UseLatest = false
	defer func() { schema.UseLatest = original }()

	items, err := GenerateURLs(ctx, true)
	if err != nil {
		t.Fatalf("GenerateURLs returned error: %v", err)
	}

	// We expect exactly two download targets (PKL + Anaconda).
	if len(items) != 2 {
		t.Fatalf("expected 2 download items, got %d", len(items))
	}

	// Basic sanity checks on the returned structure.
	for _, itm := range items {
		if !strings.HasPrefix(itm.URL, "https://") {
			t.Errorf("URL does not start with https: %s", itm.URL)
		}
		if itm.LocalName == "" {
			t.Errorf("LocalName should not be empty for item %+v", itm)
		}
	}

	// Reference the schema version as required by testing rules.
	_ = schema.SchemaVersion(ctx)
}

func TestBuildURLAndArchMapping(t *testing.T) {
	ctx := context.Background()

	// Verify buildURL replaces tokens correctly.
	input := "https://example.com/{version}/{arch}"
	got := buildURL(input, "v1", "x86_64")
	want := "https://example.com/v1/x86_64"
	if got != want {
		t.Fatalf("buildURL mismatch: got %s want %s", got, want)
	}

	// Check architecture mapping for apple/pkl and default.
	apple := GetCurrentArchitecture(ctx, "apple/pkl")
	def := GetCurrentArchitecture(ctx, "some/repo")

	switch runtime.GOARCH {
	case "amd64":
		if apple != "amd64" {
			t.Fatalf("expected amd64 for apple mapping, got %s", apple)
		}
		if def != "x86_64" {
			t.Fatalf("expected x86_64 for default mapping, got %s", def)
		}
	case "arm64":
		if apple != "aarch64" {
			t.Fatalf("expected aarch64 for apple mapping, got %s", apple)
		}
		if def != "aarch64" {
			t.Fatalf("expected aarch64 for default mapping, got %s", def)
		}
	}
}

func TestCompareVersionsAndParse(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		a, b    string
		greater bool
	}{
		{"1.2.3", "1.2.2", true},
		{"1.2", "1.2.0", false},
		{"2.0.0", "2.0.0", false},
		{"1.10", "1.9", true}, // numeric comparison not lexicographic
	}

	for _, c := range cases {
		got := CompareVersions(ctx, c.a, c.b)
		if got != c.greater {
			t.Fatalf("CompareVersions(%s,%s) = %v want %v", c.a, c.b, got, c.greater)
		}
	}

	// parseVersion edge validation
	parts := parseVersion("10.20.3-alpha")
	if len(parts) < 3 || parts[0] != 10 || parts[1] != 20 {
		t.Fatalf("parseVersion unexpected result: %v", parts)
	}
}

func TestGenerateURLsStaticQuick(t *testing.T) {
	schema.UseLatest = false
	items, err := GenerateURLs(context.Background(), true)
	assert.NoError(t, err)
	assert.NotEmpty(t, items)
	// Ensure each local name contains arch or version placeholders replaced
	for _, it := range items {
		assert.NotContains(t, it.LocalName, "{", "template placeholders should be resolved")
		assert.NotEmpty(t, it.URL)
	}
}

func TestCompareVersionsAdditional(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		v1, v2 string
		expect bool // true if v1 > v2
	}{
		{"1.2.3", "1.2.2", true},
		{"1.2.3", "1.2.3", false},
		{"1.2.3", "1.3.0", false},
		{"2.0", "1.999.999", true},
		{"1.2.3-alpha", "1.2.3", false},
	}
	for _, c := range cases {
		got := CompareVersions(ctx, c.v1, c.v2)
		if got != c.expect {
			t.Fatalf("CompareVersions(%s,%s)=%v want %v", c.v1, c.v2, got, c.expect)
		}
	}
}

func TestGetCurrentArchitectureAdditional(t *testing.T) {
	ctx := context.Background()
	arch := GetCurrentArchitecture(ctx, "apple/pkl")
	if runtime.GOARCH == "amd64" {
		if arch != "amd64" {
			t.Fatalf("expected amd64 mapping for amd64 runtime, got %s", arch)
		}
	}
	// arm64 maps to aarch64 for apple/pkl mapping, verify deterministically
	fakeCtx := context.Background()
	expectedDefault := runtime.GOARCH
	if mapping, ok := archMappings["default"]; ok {
		if mapped, ok2 := mapping[runtime.GOARCH]; ok2 {
			expectedDefault = mapped
		}
	}
	got := GetCurrentArchitecture(fakeCtx, "unknown/repo")
	if got != expectedDefault {
		t.Fatalf("unexpected default mapping: got %s want %s", got, expectedDefault)
	}
}

func TestBuildURLAdditional(t *testing.T) {
	base := "https://example.com/{version}/{arch}/bin"
	out := buildURL(base, "v1.0.0", "x86_64")
	expected := "https://example.com/v1.0.0/x86_64/bin"
	if out != expected {
		t.Fatalf("buildURL mismatch got %s want %s", out, expected)
	}
}

func TestCompareVersionsUnit(t *testing.T) {
	ctx := context.Background()
	assert.True(t, CompareVersions(ctx, "1.2.3", "1.2.0"))
	assert.False(t, CompareVersions(ctx, "1.2.0", "1.2.3"))
	assert.False(t, CompareVersions(ctx, "1.2.3", "1.2.3"))
}

func TestGetCurrentArchitectureMappingUnit(t *testing.T) {
	ctx := context.Background()
	arch := GetCurrentArchitecture(ctx, "apple/pkl")
	switch runtime.GOARCH {
	case "amd64":
		assert.Equal(t, "amd64", arch)
	case "arm64":
		assert.Equal(t, "aarch64", arch)
	default:
		assert.Equal(t, runtime.GOARCH, arch)
	}
}

func TestCompareVersionsOrdering(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		a, b          string
		expectABigger bool
	}{
		{"1.2.3", "1.2.2", true},
		{"2.0.0", "1.9.9", true},
		{"1.0.0", "1.0.0", false},
		{"1.0.0", "1.0.1", false},
		{"1.10.0", "1.9.9", true},
	}

	for _, c := range cases {
		got := CompareVersions(ctx, c.a, c.b)
		if got != c.expectABigger {
			t.Fatalf("CompareVersions(%s,%s) = %v, want %v", c.a, c.b, got, c.expectABigger)
		}
	}
}

func TestGetCurrentArchitectureMappingCov(t *testing.T) {
	ctx := context.Background()

	arch := GetCurrentArchitecture(ctx, "apple/pkl")

	switch runtime.GOARCH {
	case "amd64":
		if arch != "amd64" {
			t.Fatalf("expected amd64 mapping, got %s", arch)
		}
	case "arm64":
		if arch != "aarch64" {
			t.Fatalf("expected aarch64 mapping, got %s", arch)
		}
	}
}

func TestBuildURLTemplateSubstitution(t *testing.T) {
	base := "https://example.com/download/{version}/bin-{arch}"
	url := buildURL(base, "v1.2.3", "x86_64")
	expected := "https://example.com/download/v1.2.3/bin-x86_64"
	if url != expected {
		t.Fatalf("buildURL produced %s, want %s", url, expected)
	}
}

func TestGetCurrentArchitecture(t *testing.T) {
	ctx := context.Background()

	arch := GetCurrentArchitecture(ctx, "apple/pkl")
	switch runtime.GOARCH {
	case "amd64":
		if arch != "amd64" {
			t.Fatalf("expected amd64 mapping, got %s", arch)
		}
	case "arm64":
		if arch != "aarch64" {
			t.Fatalf("expected aarch64 mapping for arm64 host, got %s", arch)
		}
	default:
		if arch != runtime.GOARCH {
			t.Fatalf("expected passthrough architecture %s, got %s", runtime.GOARCH, arch)
		}
	}

	// Unknown repo should fallback to default mapping
	arch = GetCurrentArchitecture(ctx, "some/unknown")
	expected := runtime.GOARCH
	if runtime.GOARCH == "amd64" {
		expected = "x86_64"
	} else if runtime.GOARCH == "arm64" {
		expected = "aarch64"
	}
	if arch != expected {
		t.Fatalf("expected %s for default mapping, got %s", expected, arch)
	}
}

func TestCompareVersions(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		v1, v2  string
		greater bool
	}{
		{"1.2.3", "1.2.2", true},  // higher patch
		{"1.3.0", "1.2.9", true},  // higher minor
		{"2.0.0", "1.9.9", true},  // higher major
		{"1.0.0", "1.0.0", false}, // equal
		{"1.2.3", "2.0.0", false}, // lower major
		{"1.2", "1.2.1", false},   // shorter version string
	}

	for _, c := range cases {
		got := CompareVersions(ctx, c.v1, c.v2)
		if got != c.greater {
			t.Fatalf("CompareVersions(%s,%s) = %v, want %v", c.v1, c.v2, got, c.greater)
		}
	}
}

// No test for buildURL because it is an unexported helper; its
// behaviour is implicitly covered by higher-level GenerateURLs tests.

func TestCompareAndParseVersion(t *testing.T) {
	ctx := context.Background()
	assert.True(t, CompareVersions(ctx, "2.0.0", "1.9.9"))
	assert.False(t, CompareVersions(ctx, "1.0.0", "1.0.1"))
	// equal
	assert.False(t, CompareVersions(ctx, "1.0.0", "1.0.0"))

	got := parseVersion("1.2.3-alpha")
	assert.Equal(t, []int{1, 2, 3, 0}, got, "non numeric suffixed parts become 0")
}

func TestGenerateURLs_Static(t *testing.T) {
	schema.UseLatest = false
	items, err := GenerateURLs(context.Background(), true)
	assert.NoError(t, err)
	assert.NotEmpty(t, items)
	// Ensure each local name contains arch or version placeholders replaced
	for _, it := range items {
		assert.NotContains(t, it.LocalName, "{", "template placeholders should be resolved")
		assert.NotEmpty(t, it.URL)
	}
}

// mockRoundTripper implements http.RoundTripper to stub external calls made by
// GetLatestAnacondaVersions. It always returns a fixed HTML listing that
// contains multiple Anaconda installer filenames so that the version parsing
// logic is fully exercised.

type mockRoundTripper struct{}

func (m mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Minimal HTML directory index with two entries for different archs.
	body := `
<html><body>
<a href="Anaconda3-2024.05-0-Linux-x86_64.sh">Anaconda3-2024.05-0-Linux-x86_64.sh</a><br>
<a href="Anaconda3-2024.10-1-Linux-aarch64.sh">Anaconda3-2024.10-1-Linux-aarch64.sh</a><br>
</body></html>`

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     make(http.Header),
	}
	return resp, nil
}

func TestGetLatestAnacondaVersionsMocked(t *testing.T) {
	// Swap the default transport for our mock and restore afterwards.
	origTransport := http.DefaultTransport
	http.DefaultTransport = mockRoundTripper{}
	defer func() { http.DefaultTransport = origTransport }()

	ctx := context.Background()
	versions, err := GetLatestAnacondaVersions(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// We expect to get both architectures with their respective versions.
	if versions["x86_64"] != "2024.05-0" {
		t.Fatalf("expected x86_64 version '2024.05-0', got %s", versions["x86_64"])
	}
	if versions["aarch64"] != "2024.10-1" {
		t.Fatalf("expected aarch64 version '2024.10-1', got %s", versions["aarch64"])
	}
}

// TestGetLatestAnacondaVersions_StatusError ensures non-200 response returns error.
func TestGetLatestAnacondaVersions_StatusError(t *testing.T) {
	ctx := context.Background()
	original := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusInternalServerError, Header: make(http.Header), Body: io.NopCloser(bytes.NewBufferString(""))}, nil
	})
	defer func() { http.DefaultTransport = original }()

	if _, err := GetLatestAnacondaVersions(ctx); err == nil {
		t.Fatalf("expected error for non-OK status")
	}
}

// TestGetLatestAnacondaVersions_NoMatches ensures HTML without matches returns error.
func TestGetLatestAnacondaVersions_NoMatches(t *testing.T) {
	ctx := context.Background()
	html := "<html><body>no versions here</body></html>"
	original := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(bytes.NewBufferString(html))}, nil
	})
	defer func() { http.DefaultTransport = original }()

	if _, err := GetLatestAnacondaVersions(ctx); err == nil {
		t.Fatalf("expected error when no versions found")
	}
}

// TestGetLatestAnacondaVersions_NetworkError simulates transport failure.
func TestGetLatestAnacondaVersions_NetworkError(t *testing.T) {
	ctx := context.Background()
	original := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return nil, context.DeadlineExceeded
	})
	defer func() { http.DefaultTransport = original }()

	if _, err := GetLatestAnacondaVersions(ctx); err == nil {
		t.Fatalf("expected network error")
	}
}

// TestBuildURLPlaceholders verifies placeholder interpolation.
func TestBuildURLPlaceholders(t *testing.T) {
	base := "https://repo/{version}/file-{arch}.sh"
	got := buildURL(base, "v2.0", "x86_64")
	want := "https://repo/v2.0/file-x86_64.sh"
	if got != want {
		t.Fatalf("buildURL returned %s, want %s", got, want)
	}
}

type rtFunc func(*http.Request) (*http.Response, error)

func (f rtFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestGetLatestAnacondaVersionsMock(t *testing.T) {
	ctx := context.Background()

	// HTML snippet with two architectures
	html := `<!DOCTYPE html><html><body>
    <a href="Anaconda3-2024.10-1-Linux-x86_64.sh">x</a>
    <a href="Anaconda3-2024.05-1-Linux-aarch64.sh">y</a>
    </body></html>`

	// Save original transport and replace
	orig := http.DefaultTransport
	http.DefaultTransport = rtFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host == "repo.anaconda.com" {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(bytes.NewBufferString(html)),
			}, nil
		}
		return orig.RoundTrip(r)
	})
	defer func() { http.DefaultTransport = orig }()

	versions, err := GetLatestAnacondaVersions(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if versions["x86_64"] == "" || versions["aarch64"] == "" {
		t.Fatalf("expected versions for both architectures: %+v", versions)
	}
}

// TestCompareVersions covers several version comparison scenarios including
// differing lengths and prerelease identifiers to raise coverage for the helper.
func TestCompareVersionsExtraCases(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		v1   string
		v2   string
		want bool
	}{
		{"1.2.3", "1.2.2", true},       // patch greater
		{"2.0.0", "2.0.0", false},      // equal
		{"1.2.2", "1.2.3", false},      // smaller
		{"1.2.3-alpha", "1.2.2", true}, // prerelease ignored by atoi (becomes 0)
	}

	for _, tc := range cases {
		got := CompareVersions(ctx, tc.v1, tc.v2)
		if got != tc.want {
			t.Fatalf("CompareVersions(%s,%s) = %v, want %v", tc.v1, tc.v2, got, tc.want)
		}
	}
}

func TestGetCurrentArchitectureMappingNew(t *testing.T) {
	ctx := context.Background()

	// When repo matches mapping for apple/pkl
	arch := GetCurrentArchitecture(ctx, "apple/pkl")
	if runtime.GOARCH == "amd64" && arch != "amd64" {
		t.Fatalf("expected amd64 mapping, got %s", arch)
	}
	if runtime.GOARCH == "arm64" && arch != "aarch64" {
		t.Fatalf("expected aarch64 mapping, got %s", arch)
	}

	// Default mapping for unknown repo; should fall back to x86_64 mapping
	arch2 := GetCurrentArchitecture(ctx, "unknown/repo")
	expected := map[string]string{"amd64": "x86_64", "arm64": "aarch64"}
	if got := expected[runtime.GOARCH]; arch2 != got {
		t.Fatalf("expected %s, got %s", got, arch2)
	}
}

func TestCompareVersionsOrderBasic(t *testing.T) {
	ctx := context.Background()
	if !CompareVersions(ctx, "2.0.0", "1.9.9") {
		t.Fatalf("expected 2.0.0 to be greater than 1.9.9")
	}
	if CompareVersions(ctx, "1.0.0", "1.0.0") {
		t.Fatalf("equal versions should return false")
	}
}

func TestBuildURLTemplate(t *testing.T) {
	out := buildURL("https://x/{version}/{arch}", "v1", "amd64")
	if out != "https://x/v1/amd64" {
		t.Fatalf("unexpected url %s", out)
	}
}

func TestGenerateURLsStatic(t *testing.T) {
	ctx := context.Background()
	items, err := GenerateURLs(ctx, true)
	if err != nil {
		t.Fatalf("GenerateURLs unexpected error: %v", err)
	}
	if len(items) == 0 {
		t.Fatalf("expected some download items")
	}
	// Ensure placeholders were substituted.
	for _, it := range items {
		if strings.Contains(it.URL, "{version}") || strings.Contains(it.URL, "{arch}") {
			t.Fatalf("placeholders not replaced in %s", it.URL)
		}
	}
}

func TestGenerateURLs_NoAnaconda(t *testing.T) {
	ctx := context.Background()
	originalLatest := schema.UseLatest
	schema.UseLatest = false
	defer func() { schema.UseLatest = originalLatest }()

	items, err := GenerateURLs(ctx, false) // installAnaconda = false
	require.NoError(t, err)
	// Expect only 1 item (pkl) since anaconda should be excluded
	require.Len(t, items, 1)

	// Verify the single item is pkl, not anaconda
	item := items[0]
	require.Contains(t, item.URL, "pkl")
	require.NotContains(t, item.URL, "anaconda")
	require.Contains(t, item.LocalName, "pkl")
	require.NotContains(t, item.LocalName, "anaconda")
}
