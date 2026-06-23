//go:build !js

package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEntryGroupOrder(t *testing.T) {
	assert.Equal(t, groupOrderCached, entryGroupOrder(ModelEntry{Cached: true}))
	assert.Equal(t, groupOrderLLamafile, entryGroupOrder(ModelEntry{ModelType: modelTypeLLamafile}))
	assert.Equal(t, groupOrderGGUF, entryGroupOrder(ModelEntry{ModelType: modelTypeGGUF}))
	assert.Equal(t, groupOrderOllama, entryGroupOrder(ModelEntry{ModelType: modelTypeOllama}))
	assert.Equal(t, groupOrderCloud, entryGroupOrder(ModelEntry{}))
	// Cached beats ModelType
	assert.Equal(
		t,
		groupOrderCached,
		entryGroupOrder(ModelEntry{Cached: true, ModelType: modelTypeGGUF}),
	)
}

func TestSortEntries_CurrentModelFirst(t *testing.T) {
	entries := []ModelEntry{
		{Name: "b", ModelType: modelTypeGGUF},
		{Name: "a", ModelType: modelTypeGGUF},
		{Name: "current", ModelType: modelTypeGGUF},
	}
	sorted := sortEntries(entries, "current")
	assert.Equal(t, "current", sorted[0].Name)
}

func TestSortEntries_Alphabetical(t *testing.T) {
	entries := []ModelEntry{
		{Name: "c", ModelType: modelTypeGGUF},
		{Name: "a", ModelType: modelTypeGGUF},
		{Name: "b", ModelType: modelTypeGGUF},
	}
	sorted := sortEntries(entries, "")
	assert.Equal(t, "a", sorted[0].Name)
	assert.Equal(t, "b", sorted[1].Name)
	assert.Equal(t, "c", sorted[2].Name)
}

func TestSortEntries_GroupsOrdered(t *testing.T) {
	entries := []ModelEntry{
		{Name: "cloud1"},
		{Name: "gguf1", ModelType: modelTypeGGUF},
		{Name: "cached1", Cached: true},
	}
	sorted := sortEntries(entries, "")
	assert.Equal(t, "cached1", sorted[0].Name)
	assert.Equal(t, "gguf1", sorted[1].Name)
	assert.Equal(t, "cloud1", sorted[2].Name)
}

func TestSortEntries_Empty(t *testing.T) {
	sorted := sortEntries(nil, "")
	assert.Empty(t, sorted)
}

func TestBuildGroups(t *testing.T) {
	entries := []ModelEntry{
		{Name: "c1", Cached: true},
		{Name: "g1", ModelType: modelTypeGGUF},
		{Name: "cl1"},
	}
	groups := buildGroups(entries)
	require.Len(t, groups, 5)
	assert.Len(t, groups[groupOrderCached].entries, 1)
	assert.Len(t, groups[groupOrderGGUF].entries, 1)
	assert.Len(t, groups[groupOrderCloud].entries, 1)
	assert.Empty(t, groups[groupOrderLLamafile].entries)
	assert.Empty(t, groups[groupOrderOllama].entries)
}

func TestBuildGroups_AllTypes(t *testing.T) {
	entries := []ModelEntry{
		{Name: "c1", Cached: true},
		{Name: "lf1", ModelType: modelTypeLLamafile},
		{Name: "g1", ModelType: modelTypeGGUF},
		{Name: "o1", ModelType: modelTypeOllama},
		{Name: "cl1"},
	}
	groups := buildGroups(entries)
	for i, g := range groups {
		assert.Len(t, g.entries, 1, "group %d should have 1 entry", i)
	}
}

func TestNewModelPickerModel_CursorOnCurrent(t *testing.T) {
	entries := []ModelEntry{
		{Name: "a", ModelType: modelTypeGGUF},
		{Name: "b", ModelType: modelTypeGGUF},
		{Name: "c", ModelType: modelTypeGGUF},
	}
	m := newModelPickerModel(entries, "b", "")
	flat := m.flatFiltered()
	assert.Equal(t, "b", flat[m.cursor].Name)
}

func TestNewModelPickerModel_NoCurrent(t *testing.T) {
	entries := []ModelEntry{{Name: "x"}}
	m := newModelPickerModel(entries, "", "")
	assert.Equal(t, 0, m.cursor)
}

func TestNewModelPickerModel_WithPreFilter(t *testing.T) {
	entries := []ModelEntry{
		{Name: "llama"},
		{Name: "gemma"},
	}
	m := newModelPickerModel(entries, "", "gem")
	assert.Equal(t, "gem", m.filter)
	flat := m.flatFiltered()
	require.Len(t, flat, 1)
	assert.Equal(t, "gemma", flat[0].Name)
}

func TestModelPickerModel_FlatFiltered_AllWithEmptyFilter(t *testing.T) {
	entries := []ModelEntry{{Name: "a"}, {Name: "b"}}
	m := newModelPickerModel(entries, "", "")
	assert.Len(t, m.flatFiltered(), 2)
}

func TestModelPickerModel_Init(t *testing.T) {
	m := newModelPickerModel(nil, "", "")
	cmd := m.Init()
	assert.Nil(t, cmd)
}

func TestModelPickerModel_Update_WindowSize(t *testing.T) {
	m := newModelPickerModel(nil, "", "")
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	pm := updated.(modelPickerModel)
	assert.Equal(t, 120, pm.termWidth)
}

func TestModelPickerModel_Update_UnknownMsg(t *testing.T) {
	m := newModelPickerModel(nil, "", "")
	updated, cmd := m.Update("unknown-message-type")
	assert.NotNil(t, updated)
	assert.Nil(t, cmd)
}

func TestHandleKey_CtrlC(t *testing.T) {
	m := newModelPickerModel([]ModelEntry{{Name: "a"}}, "", "")
	updated, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyCtrlC})
	pm := updated.(modelPickerModel)
	assert.True(t, pm.cancelled)
	assert.True(t, pm.quitted)
	assert.NotNil(t, cmd)
}

func TestHandleKey_Esc(t *testing.T) {
	m := newModelPickerModel([]ModelEntry{{Name: "a"}}, "", "")
	updated, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
	pm := updated.(modelPickerModel)
	assert.True(t, pm.cancelled)
	assert.True(t, pm.quitted)
	assert.NotNil(t, cmd)
}

func TestHandleKey_Enter_WithItems(t *testing.T) {
	m := newModelPickerModel([]ModelEntry{{Name: "a"}}, "", "")
	updated, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	pm := updated.(modelPickerModel)
	assert.True(t, pm.quitted)
	assert.False(t, pm.cancelled)
	assert.NotNil(t, cmd)
}

func TestHandleKey_Enter_NoItems(t *testing.T) {
	m := newModelPickerModel(nil, "", "nomatch_xyz")
	updated, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	pm := updated.(modelPickerModel)
	assert.False(t, pm.quitted)
	assert.Nil(t, cmd)
}

func TestHandleKey_Down(t *testing.T) {
	entries := []ModelEntry{{Name: "a"}, {Name: "b"}, {Name: "c"}}
	m := newModelPickerModel(entries, "", "")
	m.cursor = 0
	updated, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyDown})
	pm := updated.(modelPickerModel)
	assert.Equal(t, 1, pm.cursor)
}

func TestHandleKey_Down_Wraps(t *testing.T) {
	entries := []ModelEntry{{Name: "a"}, {Name: "b"}}
	m := newModelPickerModel(entries, "", "")
	m.cursor = 1
	updated, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyDown})
	pm := updated.(modelPickerModel)
	assert.Equal(t, 0, pm.cursor)
}

func TestHandleKey_Up(t *testing.T) {
	entries := []ModelEntry{{Name: "a"}, {Name: "b"}}
	m := newModelPickerModel(entries, "", "")
	m.cursor = 1
	updated, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyUp})
	pm := updated.(modelPickerModel)
	assert.Equal(t, 0, pm.cursor)
}

func TestHandleKey_Up_Wraps(t *testing.T) {
	entries := []ModelEntry{{Name: "a"}, {Name: "b"}}
	m := newModelPickerModel(entries, "", "")
	m.cursor = 0
	updated, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyUp})
	pm := updated.(modelPickerModel)
	assert.Equal(t, 1, pm.cursor)
}

func TestHandleKey_j_Down(t *testing.T) {
	entries := []ModelEntry{{Name: "a"}, {Name: "b"}}
	m := newModelPickerModel(entries, "", "")
	m.cursor = 0
	updated, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	pm := updated.(modelPickerModel)
	assert.Equal(t, 1, pm.cursor)
}

func TestHandleKey_k_Up(t *testing.T) {
	entries := []ModelEntry{{Name: "a"}, {Name: "b"}}
	m := newModelPickerModel(entries, "", "")
	m.cursor = 1
	updated, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	pm := updated.(modelPickerModel)
	assert.Equal(t, 0, pm.cursor)
}

func TestHandleKey_Backspace(t *testing.T) {
	m := newModelPickerModel([]ModelEntry{{Name: "a"}}, "", "abc")
	updated, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyBackspace})
	pm := updated.(modelPickerModel)
	assert.Equal(t, "ab", pm.filter)
	assert.Equal(t, 0, pm.cursor)
}

func TestHandleKey_Backspace_EmptyFilter(t *testing.T) {
	m := newModelPickerModel([]ModelEntry{{Name: "a"}}, "", "")
	updated, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyBackspace})
	pm := updated.(modelPickerModel)
	assert.Equal(t, "", pm.filter)
}

func TestHandleKey_PrintableChar(t *testing.T) {
	m := newModelPickerModel([]ModelEntry{{Name: "llama"}}, "", "")
	updated, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	pm := updated.(modelPickerModel)
	assert.Equal(t, "l", pm.filter)
	assert.Equal(t, 0, pm.cursor)
}

func TestHandleKey_PrintableChar_ResetsCursor(t *testing.T) {
	entries := []ModelEntry{{Name: "llama"}, {Name: "gemma"}}
	m := newModelPickerModel(entries, "", "")
	m.cursor = 1
	updated, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	pm := updated.(modelPickerModel)
	assert.Equal(t, 0, pm.cursor)
}

func TestModelPickerModel_View_NonEmpty(t *testing.T) {
	entries := []ModelEntry{{Name: "llama3.2", ModelType: modelTypeGGUF}}
	m := newModelPickerModel(entries, "llama3.2", "")
	v := m.View()
	assert.NotEmpty(t, v)
}

func TestModelPickerModel_View_SmallTermWidth(t *testing.T) {
	m := newModelPickerModel(nil, "", "")
	m.termWidth = 10 // less than pickerMinWidth
	v := m.View()
	assert.NotEmpty(t, v)
}

func TestModelPickerModel_ViewHeader_NoFilter(t *testing.T) {
	m := newModelPickerModel(nil, "", "")
	h := m.viewHeader(60)
	assert.NotEmpty(t, h)
	assert.Contains(t, h, "Model Picker")
}

func TestModelPickerModel_ViewHeader_WithFilter(t *testing.T) {
	m := newModelPickerModel(nil, "", "myfilter")
	h := m.viewHeader(60)
	assert.Contains(t, h, "myfilter")
}

func TestModelPickerModel_ViewList_Empty(t *testing.T) {
	m := newModelPickerModel(nil, "", "nomatch_xyz_zz")
	v := m.viewList(60)
	assert.Contains(t, v, "No matching")
}

func TestModelPickerModel_ViewList_WithEntries(t *testing.T) {
	entries := []ModelEntry{{Name: "llama3.2", ModelType: modelTypeGGUF}}
	m := newModelPickerModel(entries, "", "")
	v := m.viewList(60)
	assert.NotEmpty(t, v)
}

func TestModelPickerModel_ViewList_CurrentMarked(_ *testing.T) {
	entries := []ModelEntry{{Name: "llama3.2", ModelType: modelTypeGGUF}}
	m := newModelPickerModel(entries, "llama3.2", "")
	_ = m.viewList(60) // just verify no panic
}

func TestModelPickerModel_ViewFooter_WithEntries(t *testing.T) {
	entries := []ModelEntry{{Name: "llama3.2"}}
	m := newModelPickerModel(entries, "", "")
	f := m.viewFooter(60)
	assert.NotEmpty(t, f)
	assert.Contains(t, f, "llama3.2")
}

func TestModelPickerModel_ViewFooter_NoEntries(t *testing.T) {
	m := newModelPickerModel(nil, "", "nomatch_xyz")
	f := m.viewFooter(60)
	assert.NotEmpty(t, f) // has separator even with no entries
}

func TestModelPickerModel_ViewFooter_SizeGB(t *testing.T) {
	entries := []ModelEntry{{Name: "llama3.2", SizeGB: "4.2"}}
	m := newModelPickerModel(entries, "", "")
	f := m.viewFooter(60)
	assert.Contains(t, f, "4.2")
}

func TestScrollWindow_SmallList(t *testing.T) {
	m := newModelPickerModel(nil, "", "")
	start, end := m.scrollWindow(5)
	assert.Equal(t, 0, start)
	assert.Equal(t, 5, end)
}

func TestScrollWindow_ExactMaxVisible(t *testing.T) {
	m := newModelPickerModel(nil, "", "")
	start, end := m.scrollWindow(pickerMaxVisible)
	assert.Equal(t, 0, start)
	assert.Equal(t, pickerMaxVisible, end)
}

func TestScrollWindow_LargeList_CursorMiddle(t *testing.T) {
	entries := make([]ModelEntry, 20)
	for i := range entries {
		entries[i] = ModelEntry{Name: string(rune('a' + i))}
	}
	m := newModelPickerModel(entries, "", "")
	m.cursor = 15
	start, end := m.scrollWindow(20)
	assert.GreaterOrEqual(t, start, 0)
	assert.LessOrEqual(t, end-start, pickerMaxVisible)
	assert.GreaterOrEqual(t, m.cursor, start)
	assert.Less(t, m.cursor, end)
}

func TestScrollWindow_LargeList_CursorAtEnd(t *testing.T) {
	total := 30
	m := newModelPickerModel(nil, "", "")
	m.cursor = total - 1
	start, end := m.scrollWindow(total)
	assert.Equal(t, total, end)
	assert.LessOrEqual(t, end-start, pickerMaxVisible)
}

func TestGroupLabel(t *testing.T) {
	assert.Equal(t, groupLabelCached, groupLabel(groupOrderCached))
	assert.Equal(t, groupLabelLLamafile, groupLabel(groupOrderLLamafile))
	assert.Equal(t, groupLabelGGUF, groupLabel(groupOrderGGUF))
	assert.Equal(t, groupLabelOllama, groupLabel(groupOrderOllama))
	assert.Equal(t, groupLabelCloud, groupLabel(groupOrderCloud))
	assert.Equal(t, groupLabelCloud, groupLabel(99)) // unknown -> Cloud
}

func TestRenderRow_Cursor(t *testing.T) {
	m := newModelPickerModel([]ModelEntry{{Name: "llama"}}, "llama", "")
	e := ModelEntry{Name: "llama", ModelType: modelTypeGGUF}
	row := m.renderRow(e, true, 60)
	assert.NotEmpty(t, row)
}

func TestRenderRow_NotCursor(t *testing.T) {
	m := newModelPickerModel(nil, "", "")
	e := ModelEntry{Name: "other"}
	row := m.renderRow(e, false, 60)
	assert.NotEmpty(t, row)
}

func TestRenderRow_LongName(t *testing.T) {
	m := newModelPickerModel(nil, "", "")
	e := ModelEntry{Name: strings.Repeat("x", 200)}
	row := m.renderRow(e, false, 20)
	assert.NotEmpty(t, row)
}

func TestRenderRow_CurrentModelCheckmark(t *testing.T) {
	m := newModelPickerModel(nil, "mymodel", "")
	e := ModelEntry{Name: "mymodel"}
	row := m.renderRow(e, false, 80)
	assert.Contains(t, row, "✓")
}

func TestRunModelPicker_EmptyEntries(t *testing.T) {
	result, err := RunModelPicker(nil, "", "")
	require.NoError(t, err)
	assert.Equal(t, "", result)
}

func TestTagForEntry(t *testing.T) {
	tests := []struct {
		name  string
		entry ModelEntry
		want  string
	}{
		// cached variants
		{
			name:  "cached_llamafile",
			entry: ModelEntry{Cached: true, ModelType: modelTypeLLamafile},
			want:  "[llamafile installed]",
		},
		{
			name:  "cached_gguf",
			entry: ModelEntry{Cached: true, ModelType: modelTypeGGUF},
			want:  "[gguf installed]",
		},
		{
			name:  "cached_ollama",
			entry: ModelEntry{Cached: true, ModelType: modelTypeOllama},
			want:  "[ollama installed]",
		},
		{name: "cached_default", entry: ModelEntry{Cached: true}, want: "[installed]"},
		// cached variants with repo suffix
		{
			name:  "cached_llamafile_repo",
			entry: ModelEntry{Cached: true, ModelType: modelTypeLLamafile, Repo: "org/model"},
			want:  "[llamafile installed org/model]",
		},
		{
			name:  "cached_gguf_repo",
			entry: ModelEntry{Cached: true, ModelType: modelTypeGGUF, Repo: "org/model"},
			want:  "[gguf installed org/model]",
		},
		// non-cached llamafile and gguf
		{
			name:  "uncached_llamafile",
			entry: ModelEntry{ModelType: modelTypeLLamafile},
			want:  "[llamafile]",
		},
		{name: "uncached_gguf", entry: ModelEntry{ModelType: modelTypeGGUF}, want: "[gguf]"},
		{name: "uncached_ollama", entry: ModelEntry{ModelType: modelTypeOllama}, want: "[ollama]"},
		// non-cached with repo suffix
		{
			name:  "uncached_llamafile_repo",
			entry: ModelEntry{ModelType: modelTypeLLamafile, Repo: "org/model"},
			want:  "[llamafile org/model]",
		},
		{
			name:  "uncached_gguf_repo",
			entry: ModelEntry{ModelType: modelTypeGGUF, Repo: "org/model"},
			want:  "[gguf org/model]",
		},
		// cloud variants
		{name: "cloud_enabled", entry: ModelEntry{Enabled: true}, want: "[cloud enabled]"},
		{name: "cloud_disabled", entry: ModelEntry{}, want: "[cloud]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tagForEntry(tt.entry)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestModelPickerModel_Update_KeyMsg(t *testing.T) {
	m := newModelPickerModel([]ModelEntry{{Name: "llama"}}, "", "")
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	pm := updated.(modelPickerModel)
	assert.Equal(t, "l", pm.filter)
	assert.Nil(t, cmd)
}
