package docker

import (
	"context"
	"io"
	"net/http"
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
