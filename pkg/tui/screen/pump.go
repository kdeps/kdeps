package screen

import (
	"sync"

	tea "charm.land/bubbletea/v2"
)

// OutputBatchMsg appends several outputs at once, in order. Each text is its
// own output, the same as one OutputMsg per text.
type OutputBatchMsg struct{ Texts []string }

// Pump hands messages to a running program without blocking the sender.
// Senders only append under a lock. One goroutine (Run) does the blocking
// sends, so a slow frame never stalls the agent loop.
//
// Output is never dropped: every text queued between two sends arrives as one
// OutputBatchMsg. Other messages are coalesced by key: only the newest value
// per key is sent, so a burst of status or redraw updates costs one update.
type Pump struct {
	mu     sync.Mutex
	out    []string
	latest map[string]tea.Msg
	keys   []string // keys in first-queued order
	closed bool
	wake   chan struct{}
}

// NewPump returns an empty pump. Start it with Run.
func NewPump() *Pump {
	return &Pump{latest: map[string]tea.Msg{}, wake: make(chan struct{}, 1)}
}

// Output queues one output for the main pane.
func (p *Pump) Output(text string) {
	p.mu.Lock()
	if !p.closed {
		p.out = append(p.out, text)
	}
	p.mu.Unlock()
	p.signal()
}

// Latest queues msg under key, replacing a value for that key not sent yet.
func (p *Pump) Latest(key string, msg tea.Msg) {
	p.mu.Lock()
	if !p.closed {
		if _, ok := p.latest[key]; !ok {
			p.keys = append(p.keys, key)
		}
		p.latest[key] = msg
	}
	p.mu.Unlock()
	p.signal()
}

// Close stops the pump. Run sends what is queued, then returns. Later calls to
// Output and Latest are ignored.
func (p *Pump) Close() {
	p.mu.Lock()
	p.closed = true
	p.mu.Unlock()
	p.signal()
}

// Run sends queued messages with send until Close. Call it on its own goroutine.
func (p *Pump) Run(send func(tea.Msg)) {
	for range p.wake {
		out, msgs, closed := p.take()
		if len(out) > 0 {
			send(OutputBatchMsg{Texts: out})
		}
		for _, msg := range msgs {
			send(msg)
		}
		if closed {
			return
		}
	}
}

// signal wakes Run. A wake already pending covers this one.
func (p *Pump) signal() {
	select {
	case p.wake <- struct{}{}:
	default:
	}
}

// take empties the queue.
func (p *Pump) take() ([]string, []tea.Msg, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := p.out
	p.out = nil
	msgs := make([]tea.Msg, 0, len(p.keys))
	for _, k := range p.keys {
		msgs = append(msgs, p.latest[k])
	}
	p.keys = nil
	clear(p.latest)
	return out, msgs, p.closed
}
