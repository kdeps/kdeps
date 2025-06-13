package docker

import (
	"context"
	"io"
	"net/http"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// mockTransport intercepts HTTP requests and returns a canned HTML directory listing
// that contains a few Anaconda installer filenames for different architectures.
// It enables testing GetLatestAnacondaVersions without real network access.

type mockTransport struct{ html string }

func (m mockTransport) RoundTrip(_ *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(m.html)),
		Header:     make(http.Header),
	}, nil
}

func TestGetLatestAnacondaVersions(t *testing.T) {
	ctx := context.Background()

	// Prepare fake HTML similar to the real repo listing (only the relevant bits).
	html := `
<a href="Anaconda3-2023.11-2-Linux-x86_64.sh">Anaconda3-2023.11-2-Linux-x86_64.sh</a>
<a href="Anaconda3-2023.11-2-Linux-aarch64.sh">Anaconda3-2023.11-2-Linux-aarch64.sh</a>
<a href="Anaconda3-2022.05-0-Linux-x86_64.sh">Anaconda3-2022.05-0-Linux-x86_64.sh</a>`

	// Swap http.DefaultTransport with our mock and restore afterwards.
	origTransport := http.DefaultTransport
	http.DefaultTransport = mockTransport{html: html}
	t.Cleanup(func() { http.DefaultTransport = origTransport })

	versions, err := GetLatestAnacondaVersions(ctx)
	require.NoError(t, err)
	require.Equal(t, "2023.11-2", versions["x86_64"], "latest x86_64 version mismatch")
	require.Equal(t, "2023.11-2", versions["aarch64"], "latest aarch64 version mismatch")
}

func TestCompareVersionsExtended(t *testing.T) {
	ctx := context.Background()
	require.True(t, CompareVersions(ctx, "1.2.3", "1.2.2"))
	require.False(t, CompareVersions(ctx, "1.2.0", "1.2.1"))
	require.False(t, CompareVersions(ctx, "1.2.0", "1.2.0"))
	require.True(t, CompareVersions(ctx, "2023.11-2", "2023.11-1"))
}

func TestBuildURLExtended(t *testing.T) {
	base := "https://example.com/download/{version}/file-{arch}.tar.gz"
	url := buildURL(base, "1.0.0", "x86_64")
	require.Equal(t, "https://example.com/download/1.0.0/file-x86_64.tar.gz", url)
}

func TestParseVersionExtra(t *testing.T) {
	cases := []struct {
		in   string
		want []int
	}{
		{"1.2.3", []int{1, 2, 3}},
		{"2.0.0-alpha", []int{2, 0, 0, 0}},
		{"2024.10-1", []int{2024, 10, 1}},
	}
	for _, c := range cases {
		got := parseVersion(c.in)
		if !reflect.DeepEqual(got, c.want) {
			t.Fatalf("parseVersion(%s)=%v want %v", c.in, got, c.want)
		}
	}
}

func TestCompareVersionsExtra(t *testing.T) {
	ctx := context.Background()
	if !CompareVersions(ctx, "1.2.3", "1.2.0") {
		t.Fatalf("expected 1.2.3 to be greater than 1.2.0")
	}
	if CompareVersions(ctx, "1.2.0", "1.2.3") {
		t.Fatalf("expected 1.2.0 to be less than 1.2.3")
	}
	if CompareVersions(ctx, "1.2.3", "1.2.3") {
		t.Fatalf("expected equal versions to return false")
	}
}

// TestGetCurrentArchitectureExtra verifies mapping for default and specific repos.
func TestGetCurrentArchitectureExtra(t *testing.T) {
	ctx := context.Background()
	goarch := runtime.GOARCH

	// Default mapping: "amd64"->"x86_64", "arm64"->"aarch64", else identity
	var expectedDefault string
	switch goarch {
	case "amd64":
		expectedDefault = "x86_64"
	case "arm64":
		expectedDefault = "aarch64"
	default:
		expectedDefault = goarch
	}
	gotDefault := GetCurrentArchitecture(ctx, "unknown/repo")
	if gotDefault != expectedDefault {
		t.Errorf("GetCurrentArchitecture default: got %s, want %s", gotDefault, expectedDefault)
	}

	// Specific mapping for apple/pkl: "amd64"->"amd64", "arm64"->"aarch64"
	var expectedApple string
	switch goarch {
	case "amd64":
		expectedApple = "amd64"
	case "arm64":
		expectedApple = "aarch64"
	default:
		expectedApple = goarch
	}
	gotApple := GetCurrentArchitecture(ctx, "apple/pkl")
	if gotApple != expectedApple {
		t.Errorf("GetCurrentArchitecture apple/pkl: got %s, want %s", gotApple, expectedApple)
	}
}

// TestBuildURLExtra verifies that buildURL replaces placeholders in the base URL.
func TestBuildURLExtra(t *testing.T) {
	baseURL := "https://example.com/{version}/file-{arch}.bin"
	url := buildURL(baseURL, "1.2.3", "x86_64")
	want := "https://example.com/1.2.3/file-x86_64.bin"
	if url != want {
		t.Errorf("buildURL() = %s, want %s", url, want)
	}
}
