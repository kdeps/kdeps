package screen

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
)

// imagePasteMsg is a clipboard image saved or named for the prompt.
type imagePasteMsg struct {
	path string
}

const (
	imageReadLimit = 2 * time.Second
	imageByteCap   = 5 << 20
)

// imageByteReads are tools that write image bytes to stdout. A missing tool
// is skipped. The first bytes that are an image are saved.
//
//nolint:gochecknoglobals // clipboard tools; tests swap them for scripts
var imageByteReads = [][]string{
	{"pngpaste", "-"},
	{"wl-paste", "--type", "image/png"},
	{"xclip", "-selection", "clipboard", "-t", "image/png", "-o"},
}

// imagePathReads are tools that write clipboard text. The first text is used.
// It is pasted only when it is a path to an image file.
//
//nolint:gochecknoglobals // clipboard tools; tests swap them for scripts
var imagePathReads = [][]string{
	{"pbpaste"},
	{"wl-paste", "--no-newline"},
	{"xclip", "-selection", "clipboard", "-o"},
}

// pasteImage reads a clipboard image off the UI loop. Nothing is inserted
// when the clipboard has no image.
func (m Model) pasteImage() tea.Cmd {
	read := m.readImage
	if read == nil {
		read = readClipboardImage
	}
	return func() tea.Msg {
		path, ok := read()
		if !ok || path == "" {
			return nil
		}
		return imagePasteMsg{path: path}
	}
}

// readClipboardImage saves clipboard image bytes, or returns a clipboard
// path that names an image file.
func readClipboardImage() (string, bool) {
	for _, argv := range imageByteReads {
		out, err := limitedOutput(argv, imageReadLimit)
		if err != nil || len(out) == 0 {
			continue
		}
		if len(out) > imageByteCap {
			return "", false
		}
		ext := imageExt(out)
		if ext == "" {
			continue
		}
		path, err := writePasteFile(out, ext)
		if err != nil {
			return "", false
		}
		return path, true
	}
	for _, argv := range imagePathReads {
		out, err := limitedOutput(argv, imageReadLimit)
		if err != nil || len(out) == 0 {
			continue
		}
		path := strings.TrimSpace(string(out))
		if isImageFile(path) {
			return path, true
		}
		return "", false
	}
	return "", false
}

// imageExt is the extension for recognized image bytes.
func imageExt(b []byte) string {
	switch {
	case bytes.HasPrefix(b, []byte{0x89, 'P', 'N', 'G'}):
		return ".png"
	case bytes.HasPrefix(b, []byte{0xff, 0xd8, 0xff}):
		return ".jpg"
	case bytes.HasPrefix(b, []byte("GIF87a")), bytes.HasPrefix(b, []byte("GIF89a")):
		return ".gif"
	case len(b) >= 12 && bytes.Equal(b[:4], []byte("RIFF")) && bytes.Equal(b[8:12], []byte("WEBP")):
		return ".webp"
	default:
		return ""
	}
}

// isImageFile reports whether path is a regular file with an image extension.
func isImageFile(path string) bool {
	if path == "" || strings.ContainsAny(path, "\r\n") {
		return false
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp", ".tiff":
	default:
		return false
	}
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

// writePasteFile stores data where the prompt can refer to it.
func writePasteFile(data []byte, ext string) (string, error) {
	f, err := os.CreateTemp("", "kdeps-paste-*"+ext)
	if err != nil {
		return "", err
	}
	name := f.Name()
	if _, werr := f.Write(data); werr != nil {
		_ = f.Close()
		_ = os.Remove(name)
		return "", werr
	}
	if cerr := f.Close(); cerr != nil {
		_ = os.Remove(name)
		return "", cerr
	}
	return name, nil
}

// limitedOutput runs argv and returns stdout. The process is killed at limit.
func limitedOutput(argv []string, limit time.Duration) ([]byte, error) {
	if len(argv) == 0 {
		return nil, errors.New("no command")
	}
	cmd := exec.Command(argv[0], argv[1:]...) //nolint:gosec,noctx // clipboard tool; no shell
	var buf bytes.Buffer
	cmd.Stdout = &buf
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	case <-time.After(limit):
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		<-done
		return nil, errors.New("clipboard read timed out")
	}
}
