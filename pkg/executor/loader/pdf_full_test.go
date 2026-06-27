package loader

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/assert"
)

func TestLoadPDF_FullPath(t *testing.T) {
	if _, err := exec.LookPath("pandoc"); err != nil {
		t.Skip("pandoc not available, cannot generate test PDF")
	}
	dir := t.TempDir()
	pdfPath := filepath.Join(dir, "test.pdf")
	cmd := exec.Command("pandoc", "-f", "markdown", "-o", pdfPath)
	cmd.Stdin = strings.NewReader("# Hello\n\nThis is a PDF test.\n")
	require.NoError(t, cmd.Run())

	docs, err := loadPDF(pdfPath, "")
	require.NoError(t, err)
	require.NotEmpty(t, docs)
	for _, doc := range docs {
		assert.Contains(t, doc.Metadata, "page")
		assert.Contains(t, doc.Metadata, "total_pages")
	}
}
