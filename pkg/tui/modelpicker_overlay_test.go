package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModelPickerOverlay_EnterPicksTheHighlightedModel(t *testing.T) {
	entries := []ModelEntry{{Name: "alpha"}, {Name: "beta"}}
	p := NewModelPickerOverlay(entries, "", "")
	next, _ := p.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	next, cmd := next.(ModelPickerOverlay).Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	assert.Nil(t, cmd, "an embedded picker does not quit the program it runs in")

	picker := next.(ModelPickerOverlay)
	require.True(t, picker.Done())
	name, ok := picker.Result()
	require.True(t, ok)
	assert.Equal(t, "beta", name)
}

func TestModelPickerOverlay_EscCancels(t *testing.T) {
	p := NewModelPickerOverlay([]ModelEntry{{Name: "alpha"}}, "", "")
	next, _ := p.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	picker := next.(ModelPickerOverlay)
	assert.True(t, picker.Done())
	_, ok := picker.Result()
	assert.False(t, ok)
}
