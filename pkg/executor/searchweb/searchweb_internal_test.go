package searchweb

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestExecute_MarshalError(t *testing.T) {
	origClient := httpClientFactory
	t.Cleanup(func() { httpClientFactory = origClient })
	httpClientFactory = func(_ time.Duration) *http.Client {
		return &http.Client{
			Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       http.NoBody,
				}, nil
			}),
		}
	}

	orig := jsonMarshal
	t.Cleanup(func() { jsonMarshal = orig })
	jsonMarshal = func(_ any) ([]byte, error) {
		return nil, errors.New("injected marshal error")
	}

	e := NewExecutor()
	config := &domain.SearchWebConfig{Query: "test", Provider: "ddg"}
	_, err := e.Execute(nil, config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "marshal")
}
