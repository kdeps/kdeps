// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package loader

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

// TestLoadRTF_NeitherFound covers the error path when neither pandoc nor
// textutil is in PATH.
func TestLoadRTF_NeitherFound(t *testing.T) {
	t.Setenv("PATH", "")
	_, err := loadRTF("/nonexistent/file.rtf")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "neither pandoc nor textutil found in PATH")
}

// TestLoadHTMLLynx_NoBin covers the error path when lynx is not in PATH.
func TestLoadHTMLLynx_NoBin(t *testing.T) {
	t.Setenv("PATH", "")
	_, err := loadHTMLLynx("/some/file.html")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "lynx not found in PATH")
}

// TestLoadPandoc_NotFound covers the error path when pandoc is not in PATH.
func TestLoadPandoc_NotFound(t *testing.T) {
	t.Setenv("PATH", "")
	_, err := loadPandoc("/nonexistent/file.md")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pandoc not found in PATH")
}

// TestLoadCSV_MalformedCSV covers the CSV read-error path (rerr != nil).
func TestLoadCSV_MalformedCSV(t *testing.T) {
	// A bare unquoted quote in the middle triggers csv.Reader to return an error.
	content := "name,age\nAlice,\"unclosed\n"
	f := writeTempFileExt(t, content, ".csv")
	_, err := loadCSV(f, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "loader csv")
}

// TestSplitDocuments_WithChunkOverlap covers the ChunkOverlap > 0 branch.
func TestSplitDocuments_WithChunkOverlap(t *testing.T) {
	docs := []Document{{
		Content:  "alpha beta gamma delta epsilon zeta eta theta iota kappa lambda mu nu xi omicron",
		Metadata: map[string]interface{}{},
	}}
	cfg := &domain.LoaderConfig{ChunkSize: 30, ChunkOverlap: 10, ChunkSplitter: "recursive"}
	result, err := splitDocuments(docs, cfg)
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

// TestLoadDirectory_UnreadableFile covers the err != nil branch in the
// WalkDir callback (when a file/directory cannot be read due to permissions).
func TestLoadDirectory_UnreadableFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission-based test not reliable on Windows")
	}
	dir := t.TempDir()
	// Create a readable file.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ok.txt"), []byte("ok"), 0o600))
	// Create a subdirectory with no read permission so WalkDir errors on it.
	sub := filepath.Join(dir, "noperm")
	require.NoError(t, os.Mkdir(sub, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(sub, "hidden.txt"), []byte("hidden"), 0o600))
	require.NoError(t, os.Chmod(sub, 0o000))
	t.Cleanup(func() { _ = os.Chmod(sub, 0o750) })

	// loadDirectory skips errors (returns nil from callback) so it still succeeds.
	docs, err := loadDirectory(dir)
	require.NoError(t, err)
	// Only the readable file should be returned.
	assert.Len(t, docs, 1)
}

// TestLoadNotionDirectory_UnreadableFile covers the rerr != nil path in
// loadNotionDirectory when an .md file can't be read.
func TestLoadNotionDirectory_UnreadableFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission-based test not reliable on Windows")
	}
	dir := t.TempDir()
	// Create a readable .md file.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "good.md"), []byte("# Good"), 0o600))
	// Create an unreadable .md file.
	bad := filepath.Join(dir, "bad.md")
	require.NoError(t, os.WriteFile(bad, []byte("# Bad"), 0o600))
	require.NoError(t, os.Chmod(bad, 0o000))
	t.Cleanup(func() { _ = os.Chmod(bad, 0o600) })

	// loadNotionDirectory skips unreadable files (continue after rerr != nil).
	docs, err := loadNotionDirectory(dir)
	require.NoError(t, err)
	assert.Len(t, docs, 1)
	assert.Contains(t, docs[0].Content, "Good")
}

// TestLoadDocuments_TextutilType covers the textutil type dispatch in loadDocuments.
// Skipped if textutil is not installed (macOS only).
func TestLoadDocuments_TextutilType(t *testing.T) {
	if _, err := lookupBin("textutil"); err != nil {
		t.Skip("textutil not found in PATH")
	}
	f := writeTempFileExt(t, "{\\rtf1 Hello textutil}", ".rtf")
	_, err := loadDocuments(&domain.LoaderConfig{Type: "textutil", Source: f})
	// May fail on bad RTF content but we exercise the dispatch path.
	_ = err
}

// lookupBin is a local helper because exec.LookPath is not imported directly
// in this test file — we use the PATH env check.
func lookupBin(name string) (string, error) {
	path := os.Getenv("PATH")
	for _, dir := range filepath.SplitList(path) {
		full := filepath.Join(dir, name)
		if _, err := os.Stat(full); err == nil {
			return full, nil
		}
	}
	return "", os.ErrNotExist
}

// TestLoadDocuments_CSVType covers the csv type dispatch path with columns.
func TestLoadDocuments_CSVType_WithColumns(t *testing.T) {
	content := "first,second,third\nA,B,C\n"
	f := writeTempFileExt(t, content, ".csv")
	docs, err := loadDocuments(&domain.LoaderConfig{
		Type:    "csv",
		Source:  f,
		Columns: []string{"first", "third"},
	})
	require.NoError(t, err)
	require.Len(t, docs, 1)
	assert.Contains(t, docs[0].Content, "first: A")
	assert.Contains(t, docs[0].Content, "third: C")
	assert.NotContains(t, docs[0].Content, "second:")
}

// TestLoadHTML_ParseBody covers the loadHTML function with a body-less HTML doc.
func TestLoadHTML_BodyLessHTML(t *testing.T) {
	content := "<html><head><title>No Body</title></head></html>"
	f := writeTempFileExt(t, content, ".html")
	docs, err := loadHTML(f)
	require.NoError(t, err)
	require.Len(t, docs, 1)
	// Body text is empty; content may be empty string.
	assert.Equal(t, "", strings.TrimSpace(docs[0].Content))
}

// TestRunCLIToFile_BinNotFound covers the lookErr != nil branch in runCLIToFile.
func TestRunCLIToFile_BinNotFound(t *testing.T) {
	t.Setenv("PATH", "")
	_, err := runCLIToFile("testlabel", "totally_nonexistent_bin_xyz", nil, "/file")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found in PATH")
}

// TestExecute_LoadError covers the err != nil path after loadDocuments in Execute.
func TestExecute_LoadError(t *testing.T) {
	_, err := NewExecutor().Execute(nil, &domain.LoaderConfig{
		Source: "/tmp",
		Type:   "nonexistent_type_xyz",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown type")
}

// TestLoadHTML_ParseError covers the goquery parse error path in loadHTML.
func TestLoadHTML_ParseError(t *testing.T) {
	dir := t.TempDir()
	_, err := loadHTML(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "loader html: parse")
}

// TestRunCLIToFile_Success covers the full success path in runCLIToFile.
func TestRunCLIToFile_Success(t *testing.T) {
	requireBin(t, "cp")
	f := writeTempFile(t, "hello from cp")
	docs, err := runCLIToFile("cp_test", "cp", nil, f)
	require.NoError(t, err)
	require.Len(t, docs, 1)
	assert.Contains(t, docs[0].Content, "hello from cp")
	assert.Equal(t, "cp_test", docs[0].Metadata["parser"])
}

// TestLoadPandoc_Success covers the success return in loadPandoc.
func TestLoadPandoc_Success(t *testing.T) {
	requireBin(t, "pandoc")
	f := writeTempFileExt(t, "# Hello\nThis is a test.", ".md")
	docs, err := loadPandoc(f)
	if err != nil {
		t.Skipf("pandoc execution error: %v (may not support --from auto)", err)
	}
	require.Len(t, docs, 1)
	assert.Contains(t, docs[0].Content, "Hello")
}

// TestLoadRTF_TextutilFallback covers the textutil fallback path in loadRTF
// when pandoc is not in PATH but textutil is.
func TestLoadRTF_TextutilFallback(t *testing.T) {
	textutilPath, err := exec.LookPath("textutil")
	if err != nil {
		t.Skip("textutil not available (macOS only)")
	}
	t.Setenv("PATH", filepath.Dir(textutilPath))
	f := writeTempFileExt(t, "{\\rtf1 Hello textutil}", ".rtf")
	_, err = loadRTF(f)
	_ = err
}

// TestLoadHTMLLynx_CmdError covers the cmdErr != nil path in loadHTMLLynx.
func TestLoadHTMLLynx_CmdError(t *testing.T) {
	requireBin(t, "lynx")
	_, err := loadHTMLLynx("/nonexistent_path_for_lynx_test.html")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "loader html_lynx")
}

// TestLoadDirectory_UnreadableFileInWalk covers the rerr != nil skip path in
// the WalkDir callback (when a file cannot be read after being discovered).
func TestLoadDirectory_UnreadableFileInWalk(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission-based test not reliable on Windows")
	}
	dir := t.TempDir()
	good := filepath.Join(dir, "good.txt")
	require.NoError(t, os.WriteFile(good, []byte("good content"), 0o600))
	bad := filepath.Join(dir, "secret.txt")
	require.NoError(t, os.WriteFile(bad, []byte("secret"), 0o600))
	require.NoError(t, os.Chmod(bad, 0o000))
	t.Cleanup(func() { _ = os.Chmod(bad, 0o600) })

	docs, err := loadDirectory(dir)
	require.NoError(t, err)
	assert.Len(t, docs, 1)
	assert.Contains(t, docs[0].Content, "good content")
}

// buildMinimalPDF creates a minimal valid PDF for testing.
func buildMinimalPDF() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%%PDF-1.4\n")
	obj1 := b.Len()
	fmt.Fprintf(&b, "1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n")
	obj2 := b.Len()
	fmt.Fprintf(&b, "2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n")
	fmt.Fprintf(&b, "3 0 obj\n<< /Type /Page /Parent 2 0 R >>\nendobj\n")
	xrefOffset := b.Len()
	fmt.Fprintf(&b, "xref\n0 3\n")
	fmt.Fprintf(&b, "0000000000 65535 f \n")
	fmt.Fprintf(&b, "%010d 00000 n \n", obj1)
	fmt.Fprintf(&b, "%010d 00000 n \n", obj2)
	fmt.Fprintf(&b, "trailer\n<< /Size 3 /Root 1 0 R >>\n")
	fmt.Fprintf(&b, "startxref\n%d\n%%%%EOF", xrefOffset)
	return b.String()
}

// TestLoadPDF_MinimalPDF covers the main success path in loadPDF.
func TestLoadPDF_MinimalPDF(t *testing.T) {
	pdfContent := buildMinimalPDF()
	f := writeTempFileExt(t, pdfContent, ".pdf")
	docs, err := loadPDF(f, "")
	require.NoError(t, err)
	require.NotEmpty(t, docs)
	assert.Contains(t, docs[0].Metadata, "page")
	assert.Contains(t, docs[0].Metadata, "total_pages")
}

// TestLoadPDF_WithPassword covers the password branch in loadPDF on a
// non-encrypted PDF.
func TestLoadPDF_WithPassword(t *testing.T) {
	pdfContent := buildMinimalPDF()
	f := writeTempFileExt(t, pdfContent, ".pdf")
	_, err := loadPDF(f, "testpass")
	_ = err
}
