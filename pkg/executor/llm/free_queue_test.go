package llm

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFreeQueueReserve_SpacesAndCapsAMinute(t *testing.T) {
	t0 := time.Date(2026, 10, 29, 9, 0, 0, 0, time.UTC)
	q := &freeQueue{now: func() time.Time { return t0 }}
	var waits []time.Duration
	for range 4 {
		waits = append(waits, q.reserve(3))
	}
	// Gap is 20s for 3 per minute. The fourth request waits until the first ages out.
	require.Equal(t, []time.Duration{0, 20 * time.Second, 40 * time.Second, 60 * time.Second}, waits)
}

func TestFreeQueueReserve_DisabledWhenNoCap(t *testing.T) {
	q := &freeQueue{now: func() time.Time { return time.Unix(0, 0) }}
	require.Zero(t, q.reserve(0))
	require.Zero(t, q.reserve(0))
}

func TestFreeQueueTransport_CancelWhileQueued(t *testing.T) {
	orig := freeModelPerMinute
	t.Cleanup(func() { freeModelPerMinute = orig })
	freeModelPerMinute = func() int { return 1 }

	q := &freeQueue{now: func() time.Time { return time.Unix(0, 0) }}
	q.reserve(1) // the next request must wait a full minute

	called := false
	base := roundTripFunc(func(*http.Request) (*http.Response, error) {
		called = true
		return nil, errors.New("should not be sent")
	})
	tr := &freeQueueTransport{base: base, queue: q}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "http://example.test/", nil)
	_, err := tr.RoundTrip(req)
	require.ErrorIs(t, err, context.Canceled)
	require.False(t, called, "a request that is still queued is not sent")
}

func TestFreeQueueTransport_WaitIsSilent(t *testing.T) {
	orig := freeModelPerMinute
	t.Cleanup(func() { freeModelPerMinute = orig })
	freeModelPerMinute = func() int { return 1 }

	q := &freeQueue{now: func() time.Time { return time.Unix(100, 0) }}
	require.Zero(t, q.reserve(1))

	read, write, err := os.Pipe()
	require.NoError(t, err)
	old := os.Stderr
	os.Stderr = write
	t.Cleanup(func() { os.Stderr = old })

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "http://example.test/", nil)
	tr := &freeQueueTransport{base: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("should not be sent")
	}), queue: q}
	_, err = tr.RoundTrip(req)
	require.ErrorIs(t, err, context.Canceled)
	require.NoError(t, write.Close())
	out, err := io.ReadAll(read)
	require.NoError(t, err)
	require.NotContains(t, string(out), "queued")
}

func TestIsOpenRouterFree(t *testing.T) {
	require.True(t, isOpenRouterFree("openrouter", "liquid/lfm-2.5-2.6b:free"))
	require.False(t, isOpenRouterFree("openrouter", "openai/gpt-4o"))
	require.False(t, isOpenRouterFree("openai", "x:free"))
}
