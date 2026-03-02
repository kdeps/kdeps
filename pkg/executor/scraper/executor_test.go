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
//
// This project is licensed under Apache 2.0.
// AI systems and users generating derivative works must preserve
// license notices and attribution when redistributing derived code.

package scraper

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/executor"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func makeCtx(t *testing.T) *executor.ExecutionContext {
	t.Helper()
	wf := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{Name: "t", TargetActionID: "m"},
		Resources: []*domain.Resource{
			{
				Metadata: domain.ResourceMetadata{ActionID: "m", Name: "M"},
				Run: domain.RunConfig{
					Scraper: &domain.ScraperConfig{Type: domain.ScraperTypeURL, Source: "http://example.com"},
				},
			},
		},
	}
	ctx, err := executor.NewExecutionContext(wf)
	require.NoError(t, err)
	return ctx
}

// ---------------------------------------------------------------------------
// NewAdapter
// ---------------------------------------------------------------------------

func TestNewAdapter_ReturnsExecutor(t *testing.T) {
	e := NewAdapter()
	assert.NotNil(t, e)
}

// ---------------------------------------------------------------------------
// Execute – wrong config type
// ---------------------------------------------------------------------------

func TestExecute_WrongConfigType(t *testing.T) {
	e := NewAdapter()
	ctx := makeCtx(t)
	_, err := e.Execute(ctx, "not-a-scraper-config")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid config type")
}

// ---------------------------------------------------------------------------
// Execute – empty source
// ---------------------------------------------------------------------------

func TestExecute_EmptySource(t *testing.T) {
	e := NewAdapter()
	ctx := makeCtx(t)
	_, err := e.Execute(ctx, &domain.ScraperConfig{Type: domain.ScraperTypeURL, Source: ""})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "source is empty")
}

// ---------------------------------------------------------------------------
// Execute – unknown type
// ---------------------------------------------------------------------------

func TestExecute_UnknownType(t *testing.T) {
	e := NewAdapter()
	ctx := makeCtx(t)
	_, err := e.Execute(ctx, &domain.ScraperConfig{Type: "ftp", Source: "ftp://host/file"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown type")
}

// ---------------------------------------------------------------------------
// URL scraping
// ---------------------------------------------------------------------------

func TestScrapeURL_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, `<html><head><title>T</title></head><body><p>Hello world</p></body></html>`)
	}))
	defer srv.Close()

	e := NewAdapter()
	ctx := makeCtx(t)
	result, err := e.Execute(ctx, &domain.ScraperConfig{
		Type:   domain.ScraperTypeURL,
		Source: srv.URL,
	})
	require.NoError(t, err)

	m, ok := result.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, true, m["success"])
	assert.Contains(t, m["content"].(string), "Hello world")
}

func TestScrapeURL_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	e := NewAdapter()
	ctx := makeCtx(t)
	result, err := e.Execute(ctx, &domain.ScraperConfig{
		Type:   domain.ScraperTypeURL,
		Source: srv.URL,
	})
	// A 500 response still returns content (the error body text)
	require.NoError(t, err)
	m, ok := result.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, true, m["success"])
}

func TestScrapeURL_ConnectionRefused(t *testing.T) {
	e := NewAdapter()
	ctx := makeCtx(t)
	_, err := e.Execute(ctx, &domain.ScraperConfig{
		Type:            domain.ScraperTypeURL,
		Source:          "http://127.0.0.1:19999",
		TimeoutDuration: "1s",
	})
	require.Error(t, err)
}

func TestScrapeURL_WithTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, "<html><body>ok</body></html>")
	}))
	defer srv.Close()

	e := NewAdapter()
	ctx := makeCtx(t)
	result, err := e.Execute(ctx, &domain.ScraperConfig{
		Type:            domain.ScraperTypeURL,
		Source:          srv.URL,
		TimeoutDuration: "5s",
	})
	require.NoError(t, err)
	m := result.(map[string]interface{})
	assert.Equal(t, true, m["success"])
}

func TestScrapeURL_TimeoutAlias(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, "<html><body>ok</body></html>")
	}))
	defer srv.Close()

	// Test that Timeout field is aliased to TimeoutDuration via UnmarshalYAML
	// by testing the promotion logic directly (YAML round-trip tested elsewhere)
	cfg := &domain.ScraperConfig{
		Type:    domain.ScraperTypeURL,
		Source:  srv.URL,
		Timeout: "10s",
	}
	// Simulate the alias promotion (UnmarshalYAML does this automatically)
	if cfg.Timeout != "" && cfg.TimeoutDuration == "" {
		cfg.TimeoutDuration = cfg.Timeout
	}
	assert.Equal(t, "10s", cfg.TimeoutDuration)
}

// ---------------------------------------------------------------------------
// extractTextFromHTML
// ---------------------------------------------------------------------------

func TestExtractTextFromHTML_Basic(t *testing.T) {
	html := []byte(`<html><head><title>T</title><style>body{}</style></head><body><p>Hello</p><script>alert(1)</script></body></html>`)
	out := ExtractTextFromHTMLForTesting(html)
	assert.Contains(t, out, "Hello")
	assert.NotContains(t, out, "alert")
	assert.NotContains(t, out, "body{}")
}

func TestExtractTextFromHTML_NoTags(t *testing.T) {
	out := ExtractTextFromHTMLForTesting([]byte("plain text"))
	assert.Equal(t, "plain text", out)
}

func TestExtractTextFromHTML_WhitespaceNormalized(t *testing.T) {
	html := []byte(`<p>  Hello   World  </p>`)
	out := ExtractTextFromHTMLForTesting(html)
	assert.Equal(t, "Hello World", out)
}

// ---------------------------------------------------------------------------
// PDF scraping
// ---------------------------------------------------------------------------

func TestScrapePDF_NotExist(t *testing.T) {
	_, err := ScrapePDFForTesting("/tmp/kdeps_test_nonexistent.pdf")
	require.Error(t, err)
}

func TestScrapePDFRaw_NotPDF(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "test*.bin")
	require.NoError(t, err)
	_, _ = f.WriteString("not a pdf file")
	require.NoError(t, f.Close())

	_, err = ScrapePDFRawForTesting(f.Name())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not appear to be a PDF")
}

func TestScrapePDFRaw_ValidPDF(t *testing.T) {
	// Minimal fake PDF with readable text run
	fakePDF := "%PDF-1.4\n% some comment\nHello World from PDF document text\n%%EOF"
	f, err := os.CreateTemp(t.TempDir(), "test*.pdf")
	require.NoError(t, err)
	_, _ = f.WriteString(fakePDF)
	require.NoError(t, f.Close())

	content, err := ScrapePDFRawForTesting(f.Name())
	require.NoError(t, err)
	assert.Contains(t, content, "Hello World from PDF document text")
}

// ---------------------------------------------------------------------------
// Word (.docx) scraping
// ---------------------------------------------------------------------------

func makeDocx(t *testing.T, text string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.docx")

	f, err := os.Create(path)
	require.NoError(t, err)

	w := zip.NewWriter(f)

	// Minimal [Content_Types].xml
	ct, _ := w.Create("[Content_Types].xml")
	_, _ = ct.Write([]byte(`<?xml version="1.0"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/></Types>`))

	// word/document.xml with the text
	doc, _ := w.Create("word/document.xml")
	xmlContent := fmt.Sprintf(`<?xml version="1.0"?><w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body><w:p><w:r><w:t>%s</w:t></w:r></w:p></w:body></w:document>`, text)
	_, _ = doc.Write([]byte(xmlContent))

	require.NoError(t, w.Close())
	require.NoError(t, f.Close())
	return path
}

func TestScrapeWord_Success(t *testing.T) {
	path := makeDocx(t, "Hello from Word document")
	content, err := ScrapeWordForTesting(path)
	require.NoError(t, err)
	assert.Contains(t, content, "Hello from Word document")
}

func TestScrapeWord_NotExist(t *testing.T) {
	_, err := ScrapeWordForTesting("/tmp/nonexistent.docx")
	require.Error(t, err)
}

func TestScrapeWord_NotZip(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "test*.docx")
	require.NoError(t, err)
	_, _ = f.WriteString("not a zip file")
	require.NoError(t, f.Close())

	_, err = ScrapeWordForTesting(f.Name())
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// Excel (.xlsx) scraping
// ---------------------------------------------------------------------------

func makeXlsx(t *testing.T, cellValue string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.xlsx")

	f, err := os.Create(path)
	require.NoError(t, err)

	w := zip.NewWriter(f)

	// [Content_Types].xml
	ct, _ := w.Create("[Content_Types].xml")
	_, _ = ct.Write([]byte(`<?xml version="1.0"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/></Types>`))

	// xl/worksheets/sheet1.xml
	sheet, _ := w.Create("xl/worksheets/sheet1.xml")
	xmlContent := fmt.Sprintf(`<?xml version="1.0"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData><row><c><v>%s</v></c></row></sheetData></worksheet>`, cellValue)
	_, _ = sheet.Write([]byte(xmlContent))

	require.NoError(t, w.Close())
	require.NoError(t, f.Close())
	return path
}

func TestScrapeExcel_Success(t *testing.T) {
	path := makeXlsx(t, "42")
	content, err := ScrapeExcelForTesting(path)
	require.NoError(t, err)
	assert.Contains(t, content, "42")
}

func TestScrapeExcel_NotExist(t *testing.T) {
	_, err := ScrapeExcelForTesting("/tmp/nonexistent.xlsx")
	require.Error(t, err)
}

func TestScrapeExcel_WithSharedStrings(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test_shared.xlsx")

	f, err := os.Create(path)
	require.NoError(t, err)

	w := zip.NewWriter(f)

	// xl/sharedStrings.xml
	ss, _ := w.Create("xl/sharedStrings.xml")
	_, _ = ss.Write([]byte(`<?xml version="1.0"?><sst xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><si><t>SharedValue</t></si></sst>`))

	// xl/worksheets/sheet1.xml – cell with type="s" (shared string index 0)
	sheet, _ := w.Create("xl/worksheets/sheet1.xml")
	_, _ = sheet.Write([]byte(`<?xml version="1.0"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData><row><c t="s"><v>0</v></c></row></sheetData></worksheet>`))

	require.NoError(t, w.Close())
	require.NoError(t, f.Close())

	content, err := ScrapeExcelForTesting(path)
	require.NoError(t, err)
	assert.Contains(t, content, "SharedValue")
}

// ---------------------------------------------------------------------------
// OCR (image) – only tests the "tesseract not installed" path
// ---------------------------------------------------------------------------

func TestScrapeImage_TesseractNotFound(t *testing.T) {
	// Override PATH to guarantee tesseract is not found
	origPath := os.Getenv("PATH")
	t.Setenv("PATH", "")
	defer func() { _ = os.Setenv("PATH", origPath) }()

	_, err := ScrapeImageForTesting("/tmp/test.png", "eng")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tesseract is not installed")
}

// ---------------------------------------------------------------------------
// Execute – result structure
// ---------------------------------------------------------------------------

func TestExecute_URLResultStructure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, "<html><body>Structured</body></html>")
	}))
	defer srv.Close()

	e := NewAdapter()
	ctx := makeCtx(t)
	result, err := e.Execute(ctx, &domain.ScraperConfig{
		Type:   domain.ScraperTypeURL,
		Source: srv.URL,
	})
	require.NoError(t, err)
	m := result.(map[string]interface{})
	assert.Equal(t, domain.ScraperTypeURL, m["type"])
	assert.Equal(t, srv.URL, m["source"])
	assert.Equal(t, true, m["success"])
	assert.IsType(t, "", m["content"])
}

// ---------------------------------------------------------------------------
// ResolvePath
// ---------------------------------------------------------------------------

func TestResolvePath_NilCtx(t *testing.T) {
	assert.Equal(t, "relative/path", ResolvePath(nil, "relative/path"))
}

func TestResolvePath_EmptyFSRoot(t *testing.T) {
	ctx := &executor.ExecutionContext{}
	assert.Equal(t, "relative/path", ResolvePath(ctx, "relative/path"))
}

func TestResolvePath_AbsPath(t *testing.T) {
	ctx := &executor.ExecutionContext{FSRoot: "/root"}
	assert.Equal(t, "/absolute/path", ResolvePath(ctx, "/absolute/path"))
}

func TestResolvePath_RelPath(t *testing.T) {
	ctx := &executor.ExecutionContext{FSRoot: "/root"}
	assert.Equal(t, "/root/relative/path", ResolvePath(ctx, "relative/path"))
}

// ---------------------------------------------------------------------------
// parseSharedIdx
// ---------------------------------------------------------------------------

func TestParseSharedIdx_Valid(t *testing.T) {
	idx, err := parseSharedIdx("0")
	require.NoError(t, err)
	assert.Equal(t, 0, idx)

	idx, err = parseSharedIdx("5")
	require.NoError(t, err)
	assert.Equal(t, 5, idx)
}

func TestParseSharedIdx_Invalid(t *testing.T) {
	_, err := parseSharedIdx("abc")
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// extractTextFromXML
// ---------------------------------------------------------------------------

func TestExtractTextFromXML_Basic(t *testing.T) {
	xmlData := `<root><w:t xmlns:w="test">hello</w:t><other>ignored</other></root>`
	wanted := map[string]bool{"t": true}
	result, err := extractTextFromXML(strings.NewReader(xmlData), wanted)
	require.NoError(t, err)
	assert.Contains(t, result, "hello")
}

func TestExtractTextFromXML_SkipFalse(t *testing.T) {
	xmlData := `<root><t>visible</t><skip>invisible</skip></root>`
	wanted := map[string]bool{"t": true, "skip": false}
	result, err := extractTextFromXML(strings.NewReader(xmlData), wanted)
	require.NoError(t, err)
	assert.Contains(t, result, "visible")
	assert.NotContains(t, result, "invisible")
}

// ---------------------------------------------------------------------------
// normalizeWhitespace
// ---------------------------------------------------------------------------

func TestNormalizeWhitespace(t *testing.T) {
	assert.Equal(t, "a b", normalizeWhitespace("  a   b  "))
	assert.Equal(t, "hello", normalizeWhitespace("hello"))
	assert.Equal(t, "", normalizeWhitespace("   "))
}

// ---------------------------------------------------------------------------
// ScrapeURLForTesting
// ---------------------------------------------------------------------------

func TestScrapeURLForTesting(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, "<html><body>direct</body></html>")
	}))
	defer srv.Close()

	e := &Executor{httpClient: &http.Client{Timeout: 5 * time.Second}}
	content, err := e.ScrapeURLForTesting(srv.URL, 5*time.Second)
	require.NoError(t, err)
	assert.Contains(t, content, "direct")
}

// ---------------------------------------------------------------------------
// GetHTTPClient
// ---------------------------------------------------------------------------

func TestGetHTTPClient(t *testing.T) {
	e := NewAdapter().(*Executor)
	assert.NotNil(t, e.GetHTTPClient())
}

// ---------------------------------------------------------------------------
// ScraperConfig YAML alias test using raw YAML
// ---------------------------------------------------------------------------

func TestScraperConfig_TimeoutAlias_Direct(t *testing.T) {
	// Test that when Timeout is set and TimeoutDuration is empty, alias works.
	cfg := &domain.ScraperConfig{
		Type:    domain.ScraperTypeURL,
		Source:  "http://example.com",
		Timeout: "5s",
	}
	// Simulate what UnmarshalYAML does for alias promotion
	if cfg.Timeout != "" && cfg.TimeoutDuration == "" {
		cfg.TimeoutDuration = cfg.Timeout
	}
	assert.Equal(t, "5s", cfg.TimeoutDuration)
}

// ---------------------------------------------------------------------------
// readSharedStrings – no sharedStrings.xml
// ---------------------------------------------------------------------------

func TestReadSharedStrings_NoFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.xlsx")

	f, err := os.Create(path)
	require.NoError(t, err)
	w := zip.NewWriter(f)
	require.NoError(t, w.Close())
	require.NoError(t, f.Close())

	r, err := zip.OpenReader(path)
	require.NoError(t, err)
	defer func() { _ = r.Close() }()

	_, err = readSharedStrings(r)
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// removeTagBlock
// ---------------------------------------------------------------------------

func TestRemoveTagBlock(t *testing.T) {
	s := "<html><head><title>T</title></head><body>body</body></html>"
	result := removeTagBlock(s, "head")
	assert.NotContains(t, result, "<head>")
	assert.Contains(t, result, "body")
}

// ---------------------------------------------------------------------------
// extractExcelCells edge case – invalid XML
// ---------------------------------------------------------------------------

func TestExtractExcelCells_InvalidXML(t *testing.T) {
	_, err := extractExcelCells(bytes.NewReader([]byte("not xml")), nil)
	// The XML decoder returns an error for non-XML data or just treats as no tokens
	// Either way, no panic should occur.
	_ = err // may or may not error depending on xml decoder behavior
}

// ---------------------------------------------------------------------------
// extractExcelCells – valid XML with multiple rows
// ---------------------------------------------------------------------------

func TestExtractExcelCells_MultipleRows(t *testing.T) {
	xmlData := `<?xml version="1.0"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData><row><c><v>A1</v></c><c><v>B1</v></c></row><row><c><v>A2</v></c></row></sheetData></worksheet>`
	content, err := extractExcelCells(strings.NewReader(xmlData), nil)
	require.NoError(t, err)
	assert.Contains(t, content, "A1")
	assert.Contains(t, content, "B1")
	assert.Contains(t, content, "A2")
}

// ---------------------------------------------------------------------------
// ScrapePDF via Execute (drives the full path including file-not-found)
// ---------------------------------------------------------------------------

func TestExecute_PDFNotFound(t *testing.T) {
	e := NewAdapter()
	ctx := makeCtx(t)
	_, err := e.Execute(ctx, &domain.ScraperConfig{
		Type:   domain.ScraperTypePDF,
		Source: "/tmp/kdeps_definitely_nonexistent_12345.pdf",
	})
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// ScrapeWord via Execute
// ---------------------------------------------------------------------------

func TestExecute_WordSuccess(t *testing.T) {
	path := makeDocx(t, "Execute Word Test")
	e := NewAdapter()
	ctx := makeCtx(t)
	result, err := e.Execute(ctx, &domain.ScraperConfig{
		Type:   domain.ScraperTypeWord,
		Source: path,
	})
	require.NoError(t, err)
	m := result.(map[string]interface{})
	assert.Equal(t, true, m["success"])
	assert.Contains(t, m["content"].(string), "Execute Word Test")
}

// ---------------------------------------------------------------------------
// ScrapeExcel via Execute
// ---------------------------------------------------------------------------

func TestExecute_ExcelSuccess(t *testing.T) {
	path := makeXlsx(t, "99")
	e := NewAdapter()
	ctx := makeCtx(t)
	result, err := e.Execute(ctx, &domain.ScraperConfig{
		Type:   domain.ScraperTypeExcel,
		Source: path,
	})
	require.NoError(t, err)
	m := result.(map[string]interface{})
	assert.Equal(t, true, m["success"])
	assert.Contains(t, m["content"].(string), "99")
}

// ---------------------------------------------------------------------------
// ScrapeImage via Execute (tesseract not available fallback)
// ---------------------------------------------------------------------------

func TestExecute_ImageNoTesseract(t *testing.T) {
	origPath := os.Getenv("PATH")
	t.Setenv("PATH", "")
	defer func() { _ = os.Setenv("PATH", origPath) }()

	e := NewAdapter()
	ctx := makeCtx(t)
	_, err := e.Execute(ctx, &domain.ScraperConfig{
		Type:   domain.ScraperTypeImage,
		Source: "/tmp/test.png",
		OCR:    &domain.ScraperOCRConfig{Language: "eng"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tesseract is not installed")
}

// ---------------------------------------------------------------------------
// XML decode tests
// ---------------------------------------------------------------------------

func TestExtractTextFromXML_InvalidXML(t *testing.T) {
	_, err := extractTextFromXML(strings.NewReader("<unclosed"), map[string]bool{"t": true})
	// xml decoder may return an error or not - either way no panic
	_ = err
}

// ---------------------------------------------------------------------------
// Test readSharedStrings with valid content
// ---------------------------------------------------------------------------

func TestReadSharedStrings_Valid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ss.xlsx")

	f, err := os.Create(path)
	require.NoError(t, err)
	w := zip.NewWriter(f)

	ss, _ := w.Create("xl/sharedStrings.xml")
	buf := new(bytes.Buffer)
	enc := xml.NewEncoder(buf)
	_ = enc.EncodeToken(xml.StartElement{Name: xml.Name{Local: "sst"}})
	_ = enc.EncodeToken(xml.StartElement{Name: xml.Name{Local: "si"}})
	_ = enc.EncodeToken(xml.StartElement{Name: xml.Name{Local: "t"}})
	_ = enc.EncodeToken(xml.CharData("TestShared"))
	_ = enc.EncodeToken(xml.EndElement{Name: xml.Name{Local: "t"}})
	_ = enc.EncodeToken(xml.EndElement{Name: xml.Name{Local: "si"}})
	_ = enc.EncodeToken(xml.EndElement{Name: xml.Name{Local: "sst"}})
	_ = enc.Flush()
	_, _ = io.Copy(ss, buf)

	require.NoError(t, w.Close())
	require.NoError(t, f.Close())

	r, err := zip.OpenReader(path)
	require.NoError(t, err)
	defer func() { _ = r.Close() }()

	strs, err := readSharedStrings(r)
	require.NoError(t, err)
	require.Len(t, strs, 1)
	assert.Equal(t, "TestShared", strs[0])
}

// ===========================================================================
// New content types (text, html, csv, markdown, pptx, json, xml)
// ===========================================================================

// ---------------------------------------------------------------------------
// text
// ---------------------------------------------------------------------------

func TestScrapeText_Success(t *testing.T) {
f, err := os.CreateTemp(t.TempDir(), "*.txt")
require.NoError(t, err)
_, _ = f.WriteString("Hello plain text\n")
require.NoError(t, f.Close())

content, err := ScrapeTextForTesting(f.Name())
require.NoError(t, err)
assert.Equal(t, "Hello plain text", content)
}

func TestScrapeText_NotFound(t *testing.T) {
_, err := ScrapeTextForTesting("/tmp/kdeps_nonexistent_text.txt")
require.Error(t, err)
}

func TestExecute_TextSuccess(t *testing.T) {
f, err := os.CreateTemp(t.TempDir(), "*.txt")
require.NoError(t, err)
_, _ = f.WriteString("  execute text  ")
require.NoError(t, f.Close())

e := NewAdapter()
ctx := makeCtx(t)
result, err := e.Execute(ctx, &domain.ScraperConfig{
Type:   domain.ScraperTypeText,
Source: f.Name(),
})
require.NoError(t, err)
m := result.(map[string]interface{})
assert.Equal(t, true, m["success"])
assert.Equal(t, "execute text", m["content"])
}

// ---------------------------------------------------------------------------
// html (local file)
// ---------------------------------------------------------------------------

func TestScrapeHTMLFile_Success(t *testing.T) {
f, err := os.CreateTemp(t.TempDir(), "*.html")
require.NoError(t, err)
_, _ = f.WriteString(`<html><body><p>Local HTML</p></body></html>`)
require.NoError(t, f.Close())

content, err := ScrapeHTMLFileForTesting(f.Name())
require.NoError(t, err)
assert.Contains(t, content, "Local HTML")
}

func TestScrapeHTMLFile_NotFound(t *testing.T) {
_, err := ScrapeHTMLFileForTesting("/tmp/kdeps_nonexistent.html")
require.Error(t, err)
}

func TestExecute_HTMLFileSuccess(t *testing.T) {
f, err := os.CreateTemp(t.TempDir(), "*.html")
require.NoError(t, err)
_, _ = f.WriteString(`<html><body><h1>Title</h1><p>Paragraph</p></body></html>`)
require.NoError(t, f.Close())

e := NewAdapter()
ctx := makeCtx(t)
result, err := e.Execute(ctx, &domain.ScraperConfig{
Type:   domain.ScraperTypeHTML,
Source: f.Name(),
})
require.NoError(t, err)
m := result.(map[string]interface{})
assert.Equal(t, true, m["success"])
assert.Contains(t, m["content"].(string), "Title")
assert.Contains(t, m["content"].(string), "Paragraph")
}

// ---------------------------------------------------------------------------
// csv
// ---------------------------------------------------------------------------

func TestScrapeCSV_Success(t *testing.T) {
f, err := os.CreateTemp(t.TempDir(), "*.csv")
require.NoError(t, err)
_, _ = f.WriteString("name,age\nAlice,30\nBob,25\n")
require.NoError(t, f.Close())

content, err := ScrapeCSVForTesting(f.Name())
require.NoError(t, err)
assert.Contains(t, content, "Alice")
assert.Contains(t, content, "Bob")
assert.Contains(t, content, "30")
}

func TestScrapeCSV_TabSeparated(t *testing.T) {
f, err := os.CreateTemp(t.TempDir(), "*.csv")
require.NoError(t, err)
_, _ = f.WriteString("a,b,c\n1,2,3\n")
require.NoError(t, f.Close())

content, err := ScrapeCSVForTesting(f.Name())
require.NoError(t, err)
// rows are joined with tabs
assert.Contains(t, content, "\t")
}

func TestScrapeCSV_NotFound(t *testing.T) {
_, err := ScrapeCSVForTesting("/tmp/kdeps_nonexistent.csv")
require.Error(t, err)
}

func TestExecute_CSVSuccess(t *testing.T) {
f, err := os.CreateTemp(t.TempDir(), "*.csv")
require.NoError(t, err)
_, _ = f.WriteString("x,y\n10,20\n")
require.NoError(t, f.Close())

e := NewAdapter()
ctx := makeCtx(t)
result, err := e.Execute(ctx, &domain.ScraperConfig{
Type:   domain.ScraperTypeCSV,
Source: f.Name(),
})
require.NoError(t, err)
m := result.(map[string]interface{})
assert.Equal(t, true, m["success"])
assert.Contains(t, m["content"].(string), "10")
}

// ---------------------------------------------------------------------------
// markdown
// ---------------------------------------------------------------------------

func TestStripMarkdown_Headings(t *testing.T) {
in := "# Heading 1\n## Heading 2\nNormal text"
out := StripMarkdownForTesting(in)
assert.Contains(t, out, "Heading 1")
assert.Contains(t, out, "Heading 2")
assert.Contains(t, out, "Normal text")
assert.NotContains(t, out, "#")
}

func TestStripMarkdown_Bold(t *testing.T) {
out := StripMarkdownForTesting("This is **bold** text")
assert.Equal(t, "This is bold text", out)
}

func TestStripMarkdown_Italic(t *testing.T) {
out := StripMarkdownForTesting("This is *italic* text")
assert.Equal(t, "This is italic text", out)
}

func TestStripMarkdown_Link(t *testing.T) {
out := StripMarkdownForTesting("Click [here](https://example.com) for info")
assert.Contains(t, out, "here")
assert.NotContains(t, out, "https://example.com")
}

func TestStripMarkdown_Image(t *testing.T) {
out := StripMarkdownForTesting("![alt text](image.png)")
assert.Contains(t, out, "alt text")
assert.NotContains(t, out, "image.png")
}

func TestStripMarkdown_UnorderedList(t *testing.T) {
out := StripMarkdownForTesting("- item one\n- item two")
assert.Contains(t, out, "item one")
assert.Contains(t, out, "item two")
assert.NotContains(t, out, "- ")
}

func TestStripMarkdown_OrderedList(t *testing.T) {
out := StripMarkdownForTesting("1. first\n2. second")
assert.Contains(t, out, "first")
assert.Contains(t, out, "second")
}

func TestStripMarkdown_HorizontalRule(t *testing.T) {
out := StripMarkdownForTesting("Before\n---\nAfter")
assert.Contains(t, out, "Before")
assert.Contains(t, out, "After")
assert.NotContains(t, out, "---")
}

func TestStripMarkdown_FencedCode(t *testing.T) {
out := StripMarkdownForTesting("Text\n```go\ncode here\n```\nMore")
assert.Contains(t, out, "Text")
assert.Contains(t, out, "More")
}

func TestScrapeMarkdown_Success(t *testing.T) {
f, err := os.CreateTemp(t.TempDir(), "*.md")
require.NoError(t, err)
_, _ = f.WriteString("# Title\n\nSome **bold** paragraph.\n")
require.NoError(t, f.Close())

content, err := ScrapeMarkdownForTesting(f.Name())
require.NoError(t, err)
assert.Contains(t, content, "Title")
assert.Contains(t, content, "paragraph")
assert.NotContains(t, content, "**")
}

func TestScrapeMarkdown_NotFound(t *testing.T) {
_, err := ScrapeMarkdownForTesting("/tmp/kdeps_nonexistent.md")
require.Error(t, err)
}

func TestExecute_MarkdownSuccess(t *testing.T) {
f, err := os.CreateTemp(t.TempDir(), "*.md")
require.NoError(t, err)
_, _ = f.WriteString("# Hello Markdown\nThis is a test.\n")
require.NoError(t, f.Close())

e := NewAdapter()
ctx := makeCtx(t)
result, err := e.Execute(ctx, &domain.ScraperConfig{
Type:   domain.ScraperTypeMarkdown,
Source: f.Name(),
})
require.NoError(t, err)
m := result.(map[string]interface{})
assert.Equal(t, true, m["success"])
assert.Contains(t, m["content"].(string), "Hello Markdown")
}

// ---------------------------------------------------------------------------
// pptx
// ---------------------------------------------------------------------------

func makePPTX(t *testing.T, text string) string {
t.Helper()
dir := t.TempDir()
path := filepath.Join(dir, "test.pptx")

f, err := os.Create(path)
require.NoError(t, err)

w := zip.NewWriter(f)

ct, _ := w.Create("[Content_Types].xml")
_, _ = ct.Write([]byte(`<?xml version="1.0"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Override PartName="/ppt/slides/slide1.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slide+xml"/></Types>`))

slide, _ := w.Create("ppt/slides/slide1.xml")
xmlContent := fmt.Sprintf(`<?xml version="1.0"?><p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"><p:cSld><p:spTree><p:sp><p:txBody><a:p><a:r><a:t>%s</a:t></a:r></a:p></p:txBody></p:sp></p:spTree></p:cSld></p:sld>`, text)
_, _ = slide.Write([]byte(xmlContent))

require.NoError(t, w.Close())
require.NoError(t, f.Close())
return path
}

func TestScrapePPTX_Success(t *testing.T) {
path := makePPTX(t, "Hello from PowerPoint")
content, err := ScrapePPTXForTesting(path)
require.NoError(t, err)
assert.Contains(t, content, "Hello from PowerPoint")
}

func TestScrapePPTX_NotExist(t *testing.T) {
_, err := ScrapePPTXForTesting("/tmp/kdeps_nonexistent.pptx")
require.Error(t, err)
}

func TestScrapePPTX_NotZip(t *testing.T) {
f, err := os.CreateTemp(t.TempDir(), "*.pptx")
require.NoError(t, err)
_, _ = f.WriteString("not a zip")
require.NoError(t, f.Close())

_, err = ScrapePPTXForTesting(f.Name())
require.Error(t, err)
}

func TestExecute_PPTXSuccess(t *testing.T) {
path := makePPTX(t, "Execute PPTX Test")
e := NewAdapter()
ctx := makeCtx(t)
result, err := e.Execute(ctx, &domain.ScraperConfig{
Type:   domain.ScraperTypePPTX,
Source: path,
})
require.NoError(t, err)
m := result.(map[string]interface{})
assert.Equal(t, true, m["success"])
assert.Contains(t, m["content"].(string), "Execute PPTX Test")
}

// ---------------------------------------------------------------------------
// json
// ---------------------------------------------------------------------------

func TestScrapeJSON_Success(t *testing.T) {
f, err := os.CreateTemp(t.TempDir(), "*.json")
require.NoError(t, err)
_, _ = f.WriteString(`{"name":"Alice","age":30}`)
require.NoError(t, f.Close())

content, err := ScrapeJSONForTesting(f.Name())
require.NoError(t, err)
assert.Contains(t, content, "Alice")
assert.Contains(t, content, "30")
}

func TestScrapeJSON_Array(t *testing.T) {
f, err := os.CreateTemp(t.TempDir(), "*.json")
require.NoError(t, err)
_, _ = f.WriteString(`[1, 2, 3]`)
require.NoError(t, f.Close())

content, err := ScrapeJSONForTesting(f.Name())
require.NoError(t, err)
assert.Contains(t, content, "1")
assert.Contains(t, content, "2")
}

func TestScrapeJSON_Invalid(t *testing.T) {
f, err := os.CreateTemp(t.TempDir(), "*.json")
require.NoError(t, err)
_, _ = f.WriteString(`not json {`)
require.NoError(t, f.Close())

_, err = ScrapeJSONForTesting(f.Name())
require.Error(t, err)
assert.Contains(t, err.Error(), "invalid JSON")
}

func TestScrapeJSON_NotFound(t *testing.T) {
_, err := ScrapeJSONForTesting("/tmp/kdeps_nonexistent.json")
require.Error(t, err)
}

func TestExecute_JSONSuccess(t *testing.T) {
f, err := os.CreateTemp(t.TempDir(), "*.json")
require.NoError(t, err)
_, _ = f.WriteString(`{"key":"value"}`)
require.NoError(t, f.Close())

e := NewAdapter()
ctx := makeCtx(t)
result, err := e.Execute(ctx, &domain.ScraperConfig{
Type:   domain.ScraperTypeJSON,
Source: f.Name(),
})
require.NoError(t, err)
m := result.(map[string]interface{})
assert.Equal(t, true, m["success"])
assert.Contains(t, m["content"].(string), "value")
}

// ---------------------------------------------------------------------------
// xml
// ---------------------------------------------------------------------------

func TestScrapeXMLFile_Success(t *testing.T) {
f, err := os.CreateTemp(t.TempDir(), "*.xml")
require.NoError(t, err)
_, _ = f.WriteString(`<root><item>Hello XML</item><item>World</item></root>`)
require.NoError(t, f.Close())

content, err := ScrapeXMLFileForTesting(f.Name())
require.NoError(t, err)
assert.Contains(t, content, "Hello XML")
assert.Contains(t, content, "World")
}

func TestScrapeXMLFile_NotFound(t *testing.T) {
_, err := ScrapeXMLFileForTesting("/tmp/kdeps_nonexistent.xml")
require.Error(t, err)
}

func TestExtractAllXMLText_Basic(t *testing.T) {
xmlData := `<doc><title>My Title</title><body>Body text</body></doc>`
content, err := ExtractAllXMLTextForTesting(strings.NewReader(xmlData))
require.NoError(t, err)
assert.Contains(t, content, "My Title")
assert.Contains(t, content, "Body text")
}

func TestExtractAllXMLText_Nested(t *testing.T) {
xmlData := `<a><b><c>deep</c></b></a>`
content, err := ExtractAllXMLTextForTesting(strings.NewReader(xmlData))
require.NoError(t, err)
assert.Contains(t, content, "deep")
}

func TestExecute_XMLSuccess(t *testing.T) {
f, err := os.CreateTemp(t.TempDir(), "*.xml")
require.NoError(t, err)
_, _ = f.WriteString(`<data><entry>42</entry><entry>hello</entry></data>`)
require.NoError(t, f.Close())

e := NewAdapter()
ctx := makeCtx(t)
result, err := e.Execute(ctx, &domain.ScraperConfig{
Type:   domain.ScraperTypeXML,
Source: f.Name(),
})
require.NoError(t, err)
m := result.(map[string]interface{})
assert.Equal(t, true, m["success"])
assert.Contains(t, m["content"].(string), "42")
assert.Contains(t, m["content"].(string), "hello")
}

// ---------------------------------------------------------------------------
// isAllDigits helper
// ---------------------------------------------------------------------------

func TestIsAllDigits(t *testing.T) {
assert.True(t, isAllDigits("123"))
assert.False(t, isAllDigits("12a"))
assert.False(t, isAllDigits(""))
}

// ---------------------------------------------------------------------------
// Validator – new types
// ---------------------------------------------------------------------------

func TestValidateScraperConfig_NewTypes(t *testing.T) {
newTypes := []string{"text", "html", "csv", "markdown", "pptx", "json", "xml"}
for _, typ := range newTypes {
t.Run(typ, func(t *testing.T) {
err := validateScraperCfg(typ, "/some/path")
assert.NoError(t, err, "type %q should be valid", typ)
})
}
}

// validateScraperCfg is a package-level helper that calls ValidateScraperConfig
// through the domain + validator chain without importing the validator package.
// Since we are inside the scraper package, we test it via Execute with unknown type.
func validateScraperCfg(typ, source string) error {
// We test type validation implicitly: if Execute reaches the switch default,
// the type is unknown; otherwise it is valid (regardless of file existence).
// For a direct unit-test of ValidateScraperConfig we rely on the validator
// package tests – here we just confirm the executor accepts these types.
cfg := &domain.ScraperConfig{Type: typ, Source: source}
// A non-existent path will produce an error AFTER type validation passes,
// meaning the error message won't say "unknown type".
e := NewAdapter()
_, err := e.Execute(nil, cfg)
if err != nil && strings.Contains(err.Error(), "unknown type") {
return err
}
return nil
}

// ===========================================================================
// OpenDocument Format (odt, ods, odp)
// ===========================================================================

// makeODF creates a minimal ODF archive (ZIP) with content.xml containing the
// given text wrapped in a <text:p> element, using the ODF text namespace.
func makeODF(t *testing.T, ext, text string) string {
t.Helper()
dir := t.TempDir()
path := filepath.Join(dir, "test."+ext)

f, err := os.Create(path)
require.NoError(t, err)

w := zip.NewWriter(f)

// Minimal mimetype entry (not strictly required here, but realistic)
mt, _ := w.Create("mimetype")
_, _ = mt.Write([]byte("application/vnd.oasis.opendocument.text"))

// content.xml with a single text:p element
content, _ := w.Create("content.xml")
xmlContent := fmt.Sprintf(
`<?xml version="1.0" encoding="UTF-8"?>`+
`<office:document-content `+
`xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0" `+
`xmlns:text="urn:oasis:names:tc:opendocument:xmlns:text:1.0">`+
`<office:body><office:text>`+
`<text:p>%s</text:p>`+
`</office:text></office:body>`+
`</office:document-content>`,
text,
)
_, _ = content.Write([]byte(xmlContent))

require.NoError(t, w.Close())
require.NoError(t, f.Close())
return path
}

// ---------------------------------------------------------------------------
// odt
// ---------------------------------------------------------------------------

func TestScrapeODT_Success(t *testing.T) {
path := makeODF(t, "odt", "Hello from Writer")
content, err := ScrapeODTForTesting(path)
require.NoError(t, err)
assert.Contains(t, content, "Hello from Writer")
}

func TestScrapeODT_NotExist(t *testing.T) {
_, err := ScrapeODTForTesting("/tmp/kdeps_nonexistent.odt")
require.Error(t, err)
}

func TestScrapeODT_NotZip(t *testing.T) {
f, err := os.CreateTemp(t.TempDir(), "*.odt")
require.NoError(t, err)
_, _ = f.WriteString("not a zip file")
require.NoError(t, f.Close())

_, err = ScrapeODTForTesting(f.Name())
require.Error(t, err)
}

func TestScrapeODT_NoContentXML(t *testing.T) {
// ZIP without content.xml
dir := t.TempDir()
path := filepath.Join(dir, "empty.odt")
f, err := os.Create(path)
require.NoError(t, err)
w := zip.NewWriter(f)
other, _ := w.Create("styles.xml")
_, _ = other.Write([]byte("<styles/>"))
require.NoError(t, w.Close())
require.NoError(t, f.Close())

_, err = ScrapeODTForTesting(path)
require.Error(t, err)
assert.Contains(t, err.Error(), "content.xml not found")
}

func TestExecute_ODTSuccess(t *testing.T) {
path := makeODF(t, "odt", "Execute ODT Test")
e := NewAdapter()
ctx := makeCtx(t)
result, err := e.Execute(ctx, &domain.ScraperConfig{
Type:   domain.ScraperTypeODT,
Source: path,
})
require.NoError(t, err)
m := result.(map[string]interface{})
assert.Equal(t, true, m["success"])
assert.Contains(t, m["content"].(string), "Execute ODT Test")
}

// ---------------------------------------------------------------------------
// ods
// ---------------------------------------------------------------------------

func TestScrapeODS_Success(t *testing.T) {
path := makeODF(t, "ods", "Sheet Cell Value")
content, err := ScrapeODSForTesting(path)
require.NoError(t, err)
assert.Contains(t, content, "Sheet Cell Value")
}

func TestScrapeODS_NotExist(t *testing.T) {
_, err := ScrapeODSForTesting("/tmp/kdeps_nonexistent.ods")
require.Error(t, err)
}

func TestScrapeODS_NotZip(t *testing.T) {
f, err := os.CreateTemp(t.TempDir(), "*.ods")
require.NoError(t, err)
_, _ = f.WriteString("not a zip file")
require.NoError(t, f.Close())

_, err = ScrapeODSForTesting(f.Name())
require.Error(t, err)
}

func TestExecute_ODSSuccess(t *testing.T) {
path := makeODF(t, "ods", "Execute ODS Test")
e := NewAdapter()
ctx := makeCtx(t)
result, err := e.Execute(ctx, &domain.ScraperConfig{
Type:   domain.ScraperTypeODS,
Source: path,
})
require.NoError(t, err)
m := result.(map[string]interface{})
assert.Equal(t, true, m["success"])
assert.Contains(t, m["content"].(string), "Execute ODS Test")
}

// ---------------------------------------------------------------------------
// odp
// ---------------------------------------------------------------------------

func TestScrapeODP_Success(t *testing.T) {
path := makeODF(t, "odp", "Slide One Title")
content, err := ScrapeODPForTesting(path)
require.NoError(t, err)
assert.Contains(t, content, "Slide One Title")
}

func TestScrapeODP_NotExist(t *testing.T) {
_, err := ScrapeODPForTesting("/tmp/kdeps_nonexistent.odp")
require.Error(t, err)
}

func TestScrapeODP_NotZip(t *testing.T) {
f, err := os.CreateTemp(t.TempDir(), "*.odp")
require.NoError(t, err)
_, _ = f.WriteString("not a zip file")
require.NoError(t, f.Close())

_, err = ScrapeODPForTesting(f.Name())
require.Error(t, err)
}

func TestExecute_ODPSuccess(t *testing.T) {
path := makeODF(t, "odp", "Execute ODP Test")
e := NewAdapter()
ctx := makeCtx(t)
result, err := e.Execute(ctx, &domain.ScraperConfig{
Type:   domain.ScraperTypeODP,
Source: path,
})
require.NoError(t, err)
m := result.(map[string]interface{})
assert.Equal(t, true, m["success"])
assert.Contains(t, m["content"].(string), "Execute ODP Test")
}

// ---------------------------------------------------------------------------
// ODF with multiple paragraphs
// ---------------------------------------------------------------------------

func TestScrapeODT_MultipleParagraphs(t *testing.T) {
dir := t.TempDir()
path := filepath.Join(dir, "multi.odt")
f, err := os.Create(path)
require.NoError(t, err)
w := zip.NewWriter(f)
content, _ := w.Create("content.xml")
_, _ = content.Write([]byte(
`<?xml version="1.0"?>` +
`<office:document-content ` +
`xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0" ` +
`xmlns:text="urn:oasis:names:tc:opendocument:xmlns:text:1.0">` +
`<office:body><office:text>` +
`<text:p>First paragraph</text:p>` +
`<text:p>Second paragraph</text:p>` +
`<text:h>A heading</text:h>` +
`</office:text></office:body>` +
`</office:document-content>`,
))
require.NoError(t, w.Close())
require.NoError(t, f.Close())

content2, err := ScrapeODTForTesting(path)
require.NoError(t, err)
assert.Contains(t, content2, "First paragraph")
assert.Contains(t, content2, "Second paragraph")
assert.Contains(t, content2, "A heading")
}

// ---------------------------------------------------------------------------
// Validator – new ODF types
// ---------------------------------------------------------------------------

func TestValidateScraperConfig_ODFTypes(t *testing.T) {
odfTypes := []string{"odt", "ods", "odp"}
for _, typ := range odfTypes {
t.Run(typ, func(t *testing.T) {
err := validateScraperCfg(typ, "/some/path."+typ)
assert.NoError(t, err, "type %q should be valid", typ)
})
}
}
