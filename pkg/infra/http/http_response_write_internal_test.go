package http

import (
	stdhttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRespondPlainHTTPError(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	respondPlainHTTPError(w, "bad request", stdhttp.StatusBadRequest)
	assert.Equal(t, stdhttp.StatusBadRequest, w.Code)
}

func TestRespondWebServerNotFound(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	respondWebServerNotFound(w)
	assert.Equal(t, stdhttp.StatusNotFound, w.Code)
}

func TestRespondWebServerInternalError(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	respondWebServerInternalError(w)
	assert.Equal(t, stdhttp.StatusInternalServerError, w.Code)
}

func TestRespondBadGateway(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	respondBadGateway(w, "upstream error")
	assert.Equal(t, stdhttp.StatusBadGateway, w.Code)
}

func TestRespondMethodNotAllowed(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	respondMethodNotAllowed(w, []string{stdhttp.MethodGet, stdhttp.MethodPost})
	assert.Equal(t, stdhttp.StatusMethodNotAllowed, w.Code)
	assert.Contains(t, w.Header().Get("Allow"), "GET")
}

func TestWritePreflightOK(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	writePreflightOK(w)
	assert.Equal(t, stdhttp.StatusOK, w.Code)
}

func TestSetJSONContentType(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	setJSONContentType(w)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
}

func TestWriteJSONResponse(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	writeJSONResponse(w, stdhttp.StatusCreated, map[string]string{"key": "val"})
	assert.Equal(t, stdhttp.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "key")
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
}
