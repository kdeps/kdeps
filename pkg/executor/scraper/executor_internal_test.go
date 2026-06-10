package scraper

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

func TestExecute_MarshalError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<html><body>test</body></html>"))
	}))
	defer srv.Close()

	orig := jsonMarshal
	t.Cleanup(func() { jsonMarshal = orig })
	jsonMarshal = func(_ any) ([]byte, error) {
		return nil, errors.New("injected marshal error")
	}

	e := NewExecutor()
	config := &domain.ScraperConfig{URL: srv.URL}
	_, err := e.Execute(nil, config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "scraper: failed to marshal result")
}
