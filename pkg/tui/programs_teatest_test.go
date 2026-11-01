package tui

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/exp/teatest/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The programs below run as real bubbletea programs under teatest: keys go in
// through the program, and the final model is checked.

func TestTeatest_TextInputTypesAndSubmits(t *testing.T) {
	tm := teatest.NewTestModel(t, newTextInputModel("Name", "> ", ""), teatest.WithInitialTermSize(80, 20))
	tm.Type("kdeps")
	tm.Send(tea.KeyPressMsg{Code: tea.KeyEnter})
	tm.WaitFinished(t, teatest.WithFinalTimeout(5*time.Second))

	final, ok := tm.FinalModel(t).(textInputModel)
	require.True(t, ok)
	assert.Equal(t, "kdeps", final.value)
	assert.False(t, final.cancelled)
}

func TestTeatest_TextInputEscCancels(t *testing.T) {
	tm := teatest.NewTestModel(t, newTextInputModel("Name", "> ", "draft"), teatest.WithInitialTermSize(80, 20))
	tm.Send(tea.KeyPressMsg{Code: tea.KeyEsc})
	tm.WaitFinished(t, teatest.WithFinalTimeout(5*time.Second))

	final, ok := tm.FinalModel(t).(textInputModel)
	require.True(t, ok)
	assert.True(t, final.cancelled)
}

func TestTeatest_ModelPickerEnterPicksTheHighlightedModel(t *testing.T) {
	entries := []ModelEntry{{Name: "alpha"}, {Name: "beta"}}
	tm := teatest.NewTestModel(t, newModelPickerModel(entries, "", ""), teatest.WithInitialTermSize(100, 30))
	tm.Send(tea.KeyPressMsg{Code: tea.KeyDown})
	tm.Send(tea.KeyPressMsg{Code: tea.KeyEnter})
	tm.WaitFinished(t, teatest.WithFinalTimeout(5*time.Second))

	final, ok := tm.FinalModel(t).(modelPickerModel)
	require.True(t, ok)
	flat := final.flatFiltered()
	require.NotEmpty(t, flat)
	assert.Equal(t, "beta", flat[final.cursor].Name)
}

func TestTeatest_SelectorEnterConfirmsAndQQuits(t *testing.T) {
	items := [numTabs][]Item{{{Name: "skill-a", Enabled: true}}}
	tm := teatest.NewTestModel(t, newModel(items), teatest.WithInitialTermSize(100, 30))
	tm.Send(tea.KeyPressMsg{Code: tea.KeyEnter})
	tm.WaitFinished(t, teatest.WithFinalTimeout(5*time.Second))

	final, ok := tm.FinalModel(t).(model)
	require.True(t, ok)
	assert.False(t, final.quitted, "enter confirms; it is not a quit without saving")

	tm = teatest.NewTestModel(t, newModel(items), teatest.WithInitialTermSize(100, 30))
	tm.Type("q")
	tm.WaitFinished(t, teatest.WithFinalTimeout(5*time.Second))
	final, ok = tm.FinalModel(t).(model)
	require.True(t, ok)
	assert.True(t, final.quitted, "q quits the selector")
}
