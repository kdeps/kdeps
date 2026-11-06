package agent

import (
	"context"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/tui/screen"
)

func TestSubmitQueue_PopsInPushOrder(t *testing.T) {
	q := newSubmitQueue()
	q.push("one")
	q.push("two")
	line, ok := q.pop()
	assert.True(t, ok)
	assert.Equal(t, "one", line)
	line, _ = q.pop()
	assert.Equal(t, "two", line)
}

func TestSubmitQueue_CloseWakesAWaitingPop(t *testing.T) {
	q := newSubmitQueue()
	got := make(chan bool)
	go func() {
		_, ok := q.pop()
		got <- ok
	}()
	q.close()
	select {
	case ok := <-got:
		assert.False(t, ok)
	case <-time.After(5 * time.Second):
		t.Fatal("pop kept waiting after close")
	}
	q.push("late")
	_, ok := q.pop()
	assert.False(t, ok, "a line pushed after close is dropped")
}

// TestRunSubmitted_RunsLinesOneAtATimeInOrder checks a prompt sent during a
// turn waits for that turn instead of running beside it.
func TestRunSubmitted_RunsLinesOneAtATimeInOrder(t *testing.T) {
	repl := NewREPL(context.Background(), makeTestLoop(nil))
	defer repl.cancel()

	var (
		mu      sync.Mutex
		order   []string
		running atomic.Int32
		overlap atomic.Bool
	)
	release := make(chan struct{})
	repl.runFn = func(_ context.Context, prompt string) (string, error) {
		if running.Add(1) > 1 {
			overlap.Store(true)
		}
		defer running.Add(-1)
		mu.Lock()
		order = append(order, prompt)
		first := len(order) == 1
		mu.Unlock()
		if first {
			<-release
		}
		return "ok", nil
	}

	q := newSubmitQueue()
	pump := screen.NewPump()
	defer pump.Close()
	done := make(chan struct{})
	go func() {
		defer close(done)
		repl.runSubmitted(q, pump, io.Discard, func() {})
	}()
	for _, line := range []string{"alpha", "beta", "gamma"} {
		q.push(line)
	}
	close(release)
	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(order) == 3
	}, 10*time.Second, 10*time.Millisecond)
	q.close()
	<-done

	assert.False(t, overlap.Load(), "two turns ran at once")
	assert.Equal(t, []string{"alpha", "beta", "gamma"}, order)
}

// TestRunSubmitted_QuitsWhenTheLoopEnds checks /exit closes the view.
func TestRunSubmitted_QuitsWhenTheLoopEnds(t *testing.T) {
	repl := NewREPL(context.Background(), makeTestLoop(nil))
	repl.runFn = func(context.Context, string) (string, error) {
		repl.loopCancel()
		return "ok", nil
	}
	q := newSubmitQueue()
	defer q.close()
	pump := screen.NewPump()
	defer pump.Close()
	quit := make(chan struct{})
	go repl.runSubmitted(q, pump, io.Discard, func() { close(quit) })
	q.push("bye")
	select {
	case <-quit:
	case <-time.After(10 * time.Second):
		t.Fatal("the view was not closed after the loop ended")
	}
}

// TestAttachScreenModel_CoalescesThroughThePump checks model, pill and redraw
// updates reach the program as the newest value per kind.
func TestAttachScreenModel_CoalescesThroughThePump(t *testing.T) {
	pump := screen.NewPump()
	detach := attachScreenModel(pump)
	t.Cleanup(func() { setScreenModel("") })
	setScreenModel("first")
	setScreenModel("second")
	sendScreenPill("backend", "ollama")
	sendScreenPill("backend", "openrouter")
	tuiFeedSink.Lock()
	redraw := tuiFeedSink.redraw
	tuiFeedSink.Unlock()
	require.NotNil(t, redraw)
	redraw()
	redraw()
	detach()

	var got []tea.Msg
	pump.Close()
	pump.Run(func(msg tea.Msg) { got = append(got, msg) })
	assert.Equal(t, []tea.Msg{
		screen.ModelMsg{Name: "second"},
		screen.PillMsg{Name: "backend", Value: "openrouter"},
		screen.RedrawMsg{},
	}, got)
}
