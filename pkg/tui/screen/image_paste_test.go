package screen

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCtrlV_PastesAnImagePathAndLeavesTextAlone(t *testing.T) {
	m := typeText(t, newPromptScreen(t), "see ")
	m.readImage = func() (string, bool) { return "/tmp/pic.png", true }
	m, cmd := typed(t, m, ctrl('v'))
	require.NotNil(t, cmd)
	next, _ := m.Update(cmd())
	m = next.(Model)
	assert.Equal(t, "see @/tmp/pic.png ", m.Prompt())

	m.readImage = func() (string, bool) { return "", false }
	m, cmd = typed(t, m, ctrl('v'))
	require.NotNil(t, cmd)
	assert.Nil(t, cmd(), "an empty clipboard sends no message")
	assert.Equal(t, "see @/tmp/pic.png ", m.Prompt())

	m = typeText(t, m, "v")
	assert.Equal(t, "see @/tmp/pic.png v", m.Prompt(), "v still types")
}

func TestReadClipboardImage_SavesImageBytes(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake clipboard tool is a #!/bin/sh script; not executable on Windows")
	}
	dir := t.TempDir()
	script := filepath.Join(dir, "pngpaste")
	body := "#!/bin/sh\nprintf '\\211PNG\\r\\n\\032\\n'\n"
	require.NoError(t, os.WriteFile(script, []byte(body), 0o755))
	prevB, prevP := imageByteReads, imagePathReads
	imageByteReads = [][]string{{script, "-"}}
	imagePathReads = nil
	t.Cleanup(func() { imageByteReads, imagePathReads = prevB, prevP })

	path, ok := readClipboardImage()
	require.True(t, ok)
	assert.Equal(t, ".png", filepath.Ext(path))
	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.True(t, len(got) >= 4 && got[1] == 'P')
	_ = os.Remove(path)
}

func TestReadClipboardImage_UsesAnImagePathAndStopsOnText(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake clipboard tool is a #!/bin/sh script; not executable on Windows")
	}
	dir := t.TempDir()
	pic := filepath.Join(dir, "shot.png")
	require.NoError(t, os.WriteFile(pic, []byte{0x89, 'P', 'N', 'G'}, 0o644))
	script := filepath.Join(dir, "pbpaste")
	require.NoError(t, os.WriteFile(script, []byte("#!/bin/sh\nprintf '%s' \""+pic+"\"\n"), 0o755))
	prevB, prevP := imageByteReads, imagePathReads
	imageByteReads = [][]string{{filepath.Join(dir, "missing")}}
	imagePathReads = [][]string{{script}}
	t.Cleanup(func() { imageByteReads, imagePathReads = prevB, prevP })

	path, ok := readClipboardImage()
	require.True(t, ok)
	assert.Equal(t, pic, path)

	text := filepath.Join(dir, "words")
	require.NoError(t, os.WriteFile(text, []byte("#!/bin/sh\nprintf 'hello'\n"), 0o755))
	imagePathReads = [][]string{{text}}
	_, ok = readClipboardImage()
	assert.False(t, ok, "clipboard text that is not an image is left alone")
}

func TestImageExt_RecognizesTheCommonTypes(t *testing.T) {
	assert.Equal(t, ".png", imageExt([]byte{0x89, 'P', 'N', 'G'}))
	assert.Equal(t, ".jpg", imageExt([]byte{0xff, 0xd8, 0xff, 0x00}))
	assert.Equal(t, ".gif", imageExt([]byte("GIF89a")))
	assert.Equal(t, "", imageExt([]byte("hello")))
	_ = tea.KeyPressMsg{}
}
