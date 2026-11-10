package screen

import (
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v3/pkg/tui/feed"
)

// recorder collects what a pump sends. block, when set, holds every send until
// it is closed, like a program busy drawing a frame.
type recorder struct {
	mu    sync.Mutex
	got   []tea.Msg
	block chan struct{}
}

func (r *recorder) send(msg tea.Msg) {
	if r.block != nil {
		<-r.block
	}
	r.mu.Lock()
	r.got = append(r.got, msg)
	r.mu.Unlock()
}

func (r *recorder) msgs() []tea.Msg {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]tea.Msg(nil), r.got...)
}

// runPump starts p and returns a wait that closes it and waits for Run.
func runPump(p *Pump, send func(tea.Msg)) func() {
	done := make(chan struct{})
	go func() {
		defer close(done)
		p.Run(send)
	}()
	return func() {
		p.Close()
		<-done
	}
}

func TestPump_SendersNeverBlockOnASlowProgram(t *testing.T) {
	rec := &recorder{block: make(chan struct{})}
	p := NewPump()
	wait := runPump(p, rec.send)

	finished := make(chan struct{})
	go func() {
		defer close(finished)
		for range 10000 {
			p.Output("line")
			p.Latest("status", StatusTextMsg{Text: "busy"})
		}
	}()
	select {
	case <-finished:
	case <-time.After(5 * time.Second):
		t.Fatal("a sender blocked while the program was not receiving")
	}
	close(rec.block)
	wait()
}

func TestPump_KeepsEveryOutputInOrder(t *testing.T) {
	rec := &recorder{block: make(chan struct{})}
	p := NewPump()
	wait := runPump(p, rec.send)
	for _, text := range []string{"a", "b", "c"} {
		p.Output(text)
	}
	close(rec.block)
	wait()

	var texts []string
	for _, msg := range rec.msgs() {
		if b, ok := msg.(OutputBatchMsg); ok {
			texts = append(texts, b.Texts...)
		}
	}
	assert.Equal(t, []string{"a", "b", "c"}, texts)
}

func TestPump_SendsOnlyTheNewestValuePerKey(t *testing.T) {
	rec := &recorder{}
	p := NewPump()
	p.Latest("status", StatusTextMsg{Text: "one"})
	p.Latest("pill:backend", PillMsg{Name: "backend", Value: "ollama"})
	p.Latest("status", StatusTextMsg{Text: "two"})
	wait := runPump(p, rec.send)
	wait()

	assert.Equal(t, []tea.Msg{
		StatusTextMsg{Text: "two"},
		PillMsg{Name: "backend", Value: "ollama"},
	}, rec.msgs(), "one send per key, newest value, first-queued order")
}

func TestPump_IgnoresMessagesAfterClose(t *testing.T) {
	rec := &recorder{}
	p := NewPump()
	wait := runPump(p, rec.send)
	wait()
	p.Output("late")
	p.Latest("status", StatusTextMsg{Text: "late"})
	assert.Empty(t, rec.msgs())
}

func TestOutputBatchMsg_AppendsEachTextAsAnOutput(t *testing.T) {
	m := resized(t, New(&feed.Feed{}))
	m, _ = typed(t, m, OutputBatchMsg{Texts: []string{"one", "two\nthree"}})
	require.Len(t, m.outEnds, 2, "each text is its own output")
	assert.Equal(t, []string{"one", "two", "three"}, m.main)
}
