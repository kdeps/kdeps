package statuswindow

import (
	"bytes"
	"testing"
	"time"

	"github.com/charmbracelet/x/exp/teatest/v2"
)

// TestTeatest_StatusWindowDrawsItsRows runs the status window as a program:
// a status message reaches the drawn rows, then the program quits.
func TestTeatest_StatusWindowDrawsItsRows(t *testing.T) {
	tm := teatest.NewTestModel(t, New(), teatest.WithInitialTermSize(100, 30))
	tm.Send(StatusMsg{Turn: "[turn 5.6k]", Session: "[session sent 1k]", Spinner: "⠋"})
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("[turn 5.6k]"))
	}, teatest.WithDuration(5*time.Second))
	tm.Quit()
	tm.WaitFinished(t, teatest.WithFinalTimeout(5*time.Second))
}
