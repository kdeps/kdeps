package docker

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/kdeps/kdeps/pkg/schema"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// helper to build *http.Response
func buildResp(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       ioutil.NopCloser(bytes.NewBufferString(body)),
		Header:     make(http.Header),
	}
}

func TestGetLatestAnacondaVersionsSuccess(t *testing.T) {
	html := `Anaconda3-2023.07-1-Linux-x86_64.sh Anaconda3-2023.05-1-Linux-aarch64.sh` +
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
