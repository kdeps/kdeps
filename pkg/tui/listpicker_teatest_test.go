package tui

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/exp/teatest/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// pickerItems is the session selector's shape: a "new" row first, then saved sessions.
func pickerItems() []ListItem {
	return []ListItem{
		{ID: "__new__", Title: "Start a new session"},
		{ID: "s1", Title: "last week's refactor"},
		{ID: "s2", Title: "debugging the judge"},
	}
}

// runPicker runs the picker as a program and returns its final model.
func runPicker(t *testing.T, keys ...tea.KeyMsg) listPickerModel {
	t.Helper()
	tm := teatest.NewTestModel(t, newListPickerModel("Resume a session in this folder?", pickerItems(), nil),
		teatest.WithInitialTermSize(100, 30))
	for _, k := range keys {
		tm.Send(k)
	}
	tm.WaitFinished(t, teatest.WithFinalTimeout(5*time.Second))
	final, ok := tm.FinalModel(t).(listPickerModel)
	require.True(t, ok)
	return final
}

func TestTeatest_EnterOnTheFirstRowStartsANewSession(t *testing.T) {
	final := runPicker(t, tea.KeyPressMsg{Code: tea.KeyEnter})
	require.True(t, final.quitted)
	assert.False(t, final.cancelled)
	assert.Equal(t, "__new__", final.filtered()[final.cursor].ID)
}

func TestTeatest_DownThenEnterResumesThatSession(t *testing.T) {
	final := runPicker(t, tea.KeyPressMsg{Code: tea.KeyDown}, tea.KeyPressMsg{Code: tea.KeyEnter})
	require.True(t, final.quitted)
	assert.Equal(t, "s1", final.filtered()[final.cursor].ID)
}

func TestTeatest_EscCancelsThePicker(t *testing.T) {
	final := runPicker(t, tea.KeyPressMsg{Code: tea.KeyEsc})
	assert.True(t, final.cancelled)
}
