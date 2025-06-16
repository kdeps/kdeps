package docker

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
