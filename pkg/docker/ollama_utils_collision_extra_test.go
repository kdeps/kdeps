package docker

import (
	crand "crypto/rand"
	"testing"
)

// stubReader allows us to control the bytes returned by crypto/rand.Reader so we can
// force generateUniqueOllamaPort to hit its collision branch once.
// It will return all-zero bytes on the first call, then all-0xFF bytes afterwards.
// This causes the first generated port to equal minPort and collide with existingPort,
// ensuring the loop executes at least twice.

type stubReader struct{ call int }

func (s *stubReader) Read(p []byte) (int, error) {
	// crypto/rand.Int reads len(m.Bytes()) bytes (here 2). Provide deterministic data:
	// First call -> 0x0000 to generate num=0 (collision). Second call -> 0x0002 to generate num=2 (unique).
	val := byte(0x00)
	if s.call > 0 {
		val = 0x02
	}
	for i := range p {
		p[i] = val
	}
	s.call++
	return len(p), nil
}

func TestGenerateUniqueOllamaPort_CollisionLoop(t *testing.T) {
	// Swap out crypto/rand.Reader with our stub and restore afterwards.
	orig := crand.Reader
	crand.Reader = &stubReader{}
	t.Cleanup(func() { crand.Reader = orig })

	// existingPort set to minPort so first generated port collides.
	existing := uint16(minPort)

	portStr := generateUniqueOllamaPort(existing)

	if portStr == "" || portStr == "11435" { // 11435 == minPort
		t.Fatalf("expected non-empty unique port different from minPort, got %s", portStr)
	}
}
