package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScreenStatusSink_OnlyWhileTheScreenRuns(t *testing.T) {
	assert.False(t, sendScreenStatus("x"), "no sink: the caller draws the rows itself")

	var got []string
	setScreenStatusSink(func(text string) { got = append(got, text) })
	t.Cleanup(func() { setScreenStatusSink(nil) })

	assert.True(t, sendScreenStatus("[turn 1k]  [session 2k]"))
	assert.Equal(t, []string{"[turn 1k]  [session 2k]"}, got)
}
