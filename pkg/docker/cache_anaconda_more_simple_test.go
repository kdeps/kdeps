package docker

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"testing"
)

type archHTMLTransport struct{}

func (archHTMLTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	html := `<html><body>
        <a href="Anaconda3-2024.10-1-Linux-x86_64.sh">x</a>
        <a href="Anaconda3-2024.09-1-Linux-aarch64.sh">y</a>
        <a href="Anaconda3-2023.12-0-Linux-x86_64.sh">old-x</a>
        <a href="Anaconda3-2023.01-0-Linux-aarch64.sh">old-y</a>
        </body></html>`
	return &http.Response{StatusCode: 200, Body: ioutil.NopCloser(bytes.NewBufferString(html)), Header: make(http.Header)}, nil
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
