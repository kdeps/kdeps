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

// Package searchlocal provides local filesystem search for KDeps workflows.
package searchlocal

import (
	"archive/zip"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	kgsearch "github.com/kdeps/kartographer/search"

	"github.com/charmbracelet/log"
	"github.com/ledongthuc/pdf"
	"github.com/spf13/afero"

	"github.com/kdeps/kartographer/graph"

	"github.com/kdeps/kdeps/v2/pkg/datadir"
	kdeps_debug "github.com/kdeps/kdeps/v2/pkg/debug"
	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/executor"
	"github.com/kdeps/kdeps/v2/pkg/knowledge"
)

//nolint:gochecknoglobals // enables test injection for filesystem and output
var (
	AppFS          afero.Fs  = afero.NewOsFs()
	ProgressWriter io.Writer = os.Stderr
)

// Executor executes local filesystem search resources.
type Executor struct{}

// NewExecutor creates a new SearchLocal executor.
func NewExecutor() *Executor {
	kdeps_debug.Log("enter: NewExecutor")
	return &Executor{}
}

// IndexPath is where the search index of dir lives (under ~/.kdeps).
func IndexPath(dir string) string { return indexDBPath(dir) }

// Hit is one file a search index matched.
type Hit struct {
	Path    string
	Preview string
}

// queryWait bounds how long QueryIndex waits for an index being built.
const queryWait = 200 * time.Millisecond

// QueryIndex returns up to limit files of dir's search index that match
// query, best first; nil when dir has no index, it is busy being built, or
// nothing matches. It never builds or changes an index.
func QueryIndex(dir, query string, limit int) []Hit {
	dbPath := indexDBPath(dir)
	if !fileExists(dbPath) || limit <= 0 {
		return nil
	}
	tok, tokErr := knowledge.SearchTokenizer()
	if tokErr != nil {
		return nil
	}
	terms := tok.TokenizeToStrings(query)
	if len(terms) == 0 {
		return nil
	}
	idx, err := kgsearch.OpenReadOnly(dbPath, queryWait)
	if err != nil {
		return nil
	}
	defer idx.Close()
	var out []Hit
	for _, r := range idx.Search(terms) {
		if r.Document == nil || r.MatchCount < (len(terms)+1)/2 {
			continue // most of the request's words must appear
		}
		out = append(out, Hit{Path: r.Document.Path, Preview: r.Document.Preview})
		if len(out) == limit {
			break
		}
	}
	return out
}

// indexDBPath returns the default index database path for a directory. It
// lives in the folder's data dir under ~/.kdeps, never in the folder itself.
func indexDBPath(dir string) string {
	return datadir.File(dir, "index.db")
}

// graphDBPath returns the default kartographer graph database path for a
// directory -- deliberately separate from indexDBPath's TF-IDF index.db.
func graphDBPath(dir string) string {
	return datadir.File(dir, "graph.db")
}

// graphBoostFactor multiplies the score of a result linked from, or sharing a
// topic with, a top-ranked seed result (search.yaml limits.graph_boost).
func graphBoostFactor() float64 { return knowledge.SearchLimitsNow().GraphBoost }

// graphBoostSeedCount is how many top-ranked results seed the graph lookup
// (search.yaml limits.graph_boost_seeds).
func graphBoostSeedCount() int { return knowledge.SearchLimitsNow().GraphBoostSeeds }

// maxFileSize is the largest file (bytes) read and indexed (limits.max_file_size).
func maxFileSize() int64 { return knowledge.SearchLimitsNow().MaxFileSize }

// previewMax is how many characters of a file its index preview keeps (limits.preview_chars).
func previewMax() int { return knowledge.SearchLimitsNow().PreviewChars }

// snippetContext is the characters of context around a match (limits.snippet_context).
func snippetContext() int { return knowledge.SearchLimitsNow().SnippetContext }

// boostByGraph re-ranks results before the final sort. When config.Path is
// inside the knowledge vault, the structural graph is brought current (no
// model call) and the boost reads those edges. It does not build graph.db:
// an extracted reference multiplies the score by graphBoostFactor(), and a
// weaker edge multiplies by less (search.Boost scales by its weight). Any other folder, or
// a vault graph that cannot be built, uses a kartographer reference/topic
// graph, and every connected document gets the full factor. Never returns
// an error -- graph boosting is a ranking enhancement, not a correctness
// requirement, so any graph failure just leaves the TF-IDF ranking untouched.
func boostByGraph(
	ctx context.Context,
	config *domain.SearchLocalConfig,
	results []kgsearch.SearchResult,
) []kgsearch.SearchResult {
	if len(results) == 0 {
		return results
	}
	if boosted, ok := boostByKnowledge(ctx, config, results); ok {
		return boosted
	}

	dbPath := graphDBPath(config.Path)
	if err := AppFS.MkdirAll(filepath.Dir(dbPath), 0o700); err != nil {
		return results
	}

	ig, err := graph.NewIndexedGraph(AppFS, log.Default(), dbPath)
	if err != nil {
		return results
	}
	defer func() { _ = ig.Close() }()

	if _, indexErr := ig.IndexFolder(config.Path, nil); indexErr != nil {
		return results
	}

	connected := graphConnectedSet(ig, results)
	kgsearch.Boost(results, graphBoostFactor(), func(path string) float64 {
		if connected[path] {
			return 1
		}
		return 0
	})
	return results
}

// boostByKnowledge re-ranks results from the vault graph. ok is false when
// this folder is outside the vault or the structural graph cannot be built,
// so the caller falls back to kartographer.
func boostByKnowledge(
	ctx context.Context,
	config *domain.SearchLocalConfig,
	results []kgsearch.SearchResult,
) ([]kgsearch.SearchResult, bool) {
	vault, _ := knowledge.Configured()
	if !knowledge.PathInVault(vault, config.Path) {
		return nil, false
	}
	if err := knowledge.Ensure(ctx, vault); err != nil {
		return nil, false
	}
	ix := knowledge.SharedIndex(knowledge.GraphDir(vault))
	if ix == nil {
		return nil, false
	}
	seedCount := graphBoostSeedCount()
	if seedCount > len(results) {
		seedCount = len(results)
	}
	weight := map[string]float64{}
	for _, seed := range results[:seedCount] {
		for _, link := range knowledge.LinksFrom(ix, vault, seed.Document.Path) {
			if link.Weight > weight[link.Path] {
				weight[link.Path] = link.Weight
			}
		}
	}
	// A tag edge (weight 0.3) or a low-confidence inferred edge moves the score
	// less than an extracted reference (weight 1).
	kgsearch.Boost(results, graphBoostFactor(), func(path string) float64 {
		return weight[filepath.Clean(path)]
	})
	return results, true
}

// graphConnectedSet returns every document path linked from, or sharing a
// topic with, one of the top graphBoostSeedCount() results. Split out from
// boostByGraph so a query against an already-open (or already-closed) graph
// can be tested independently of the open/index setup around it.
func graphConnectedSet(ig *graph.IndexedGraph, results []kgsearch.SearchResult) map[string]bool {
	seedCount := graphBoostSeedCount()
	if seedCount > len(results) {
		seedCount = len(results)
	}

	connected := make(map[string]bool)
	for _, seed := range results[:seedCount] {
		references, relatedByTopic, err := ig.GraphFile(seed.Document.Path)
		if err != nil {
			continue
		}
		for _, path := range references[seed.Document.Path] {
			connected[path] = true
		}
		for _, path := range relatedByTopic {
			connected[path] = true
		}
	}
	return connected
}

// skipWalkEntry returns true if the walk entry should be skipped (search.yaml's
// skip rules). Without the rules nothing is walked.
func skipWalkEntry(d os.DirEntry) bool {
	f, err := knowledge.SearchFilter()
	return err != nil || f.SkipEntry(d.Name())
}

// isIndexableExt returns true if a file is indexed (search.yaml's extensions,
// never a secret). Without the rules nothing is.
func isIndexableExt(name string) bool {
	f, err := knowledge.SearchFilter()
	return err == nil && f.Indexable(name)
}

// IsSecretFile reports a file that holds keys or credentials (search.yaml's
// secret rules). Search and indexing never return one, so its content never
// reaches a prompt through search_local or recall. Without the rules every
// file counts as one.
func IsSecretFile(path string) bool {
	f, err := knowledge.SearchFilter()
	return err != nil || f.Secret(path)
}

// docOptions is how a file becomes a search document: search.yaml's tokenizer,
// size limit and preview, with PDF and office documents read as text.
func docOptions() (kgsearch.DocOptions, error) {
	tok, err := knowledge.SearchTokenizer()
	if err != nil {
		return kgsearch.DocOptions{}, err
	}
	return kgsearch.DocOptions{
		Tokenizer: tok,
		MaxSize:   maxFileSize(),
		Preview:   previewMax(),
		Read:      readIndexableText,
	}, nil
}

// indexProgressEvery is how often (files read) indexing reports progress;
// kartographer reports unchanged files ten times less often.
const (
	indexProgressEvery  = 100
	indexUnchangedEvery = indexProgressEvery * 10
)

// walkIndex brings idx up to date with root through kartographer's
// search.IndexFolder (search.yaml's filter, tokenizer and limits), printing
// progress to ProgressWriter.
func (e *Executor) walkIndex(root string, glob string, idx *kgsearch.InvertedIndex) (int, error) {
	opts, err := docOptions()
	if err != nil {
		return 0, err
	}
	filter, err := knowledge.SearchFilter()
	if err != nil {
		return 0, err
	}
	var read, unchanged int
	return kgsearch.IndexFolder(idx, root, kgsearch.FolderOptions{
		DocOptions: opts, Filter: filter, Glob: glob, ProgressEvery: indexProgressEvery,
		Progress: func(p kgsearch.FolderProgress) {
			switch {
			case p.Read != read && p.Read%indexProgressEvery == 0:
				fmt.Fprintf(ProgressWriter, "\rsearchLocal: indexed %d files ...", p.Read)
			case p.Unchanged != unchanged && p.Unchanged%indexUnchangedEvery == 0:
				fmt.Fprintf(ProgressWriter, "\rsearchLocal: %d files unchanged, skipped ...", p.Unchanged)
			case p.Purged > 0:
				fmt.Fprintf(ProgressWriter, "\rsearchLocal: removed %d stale docs from index\n", p.Purged)
			}
			read, unchanged = p.Read, p.Unchanged
		},
	})
}

// readIndexableText returns a file's indexable text content. PDFs and
// ZIP-based office formats are extracted to plain text (see below); every
// other indexable extension is read as-is, since prior extension gating
// already limits this to text-shaped formats.
func readIndexableText(p string) (string, error) {
	switch strings.ToLower(filepath.Ext(p)) {
	case ".pdf":
		return extractPDFText(p)
	case ".docx":
		return extractZippedXMLText(p, "word/document.xml")
	case ".xlsx":
		return extractZippedXMLText(p, "xl/sharedStrings.xml")
	case ".odt", ".ods", ".odp":
		return extractZippedXMLText(p, "content.xml")
	case ".pptx":
		return extractPptxText(p)
	default:
		content, err := os.ReadFile(p)
		if err != nil {
			return "", err
		}
		return string(content), nil
	}
}

// extractZippedXMLText opens a ZIP-based document and extracts the text
// content of the given member file(s) (docx's word/document.xml, xlsx's
// shared-string table, or an ODF package's single content.xml), stripping
// XML markup down to the visible text nodes. Missing members are skipped
// rather than erroring, since a genuinely empty or unusually-structured
// document should still index as "no text found", not fail the whole file.
func extractZippedXMLText(p string, members ...string) (string, error) {
	r, err := zip.OpenReader(p)
	if err != nil {
		return "", err
	}
	defer r.Close()

	var sb strings.Builder
	for _, f := range r.File {
		if !slices.Contains(members, f.Name) {
			continue
		}
		if text, terr := extractXMLText(f); terr == nil {
			sb.WriteString(text)
			sb.WriteString("\n")
		}
	}
	return sb.String(), nil
}

// extractPptxText extracts every slide's text in a PowerPoint package.
// Slides are numbered member files (ppt/slides/slide1.xml, slide2.xml, ...)
// rather than one fixed path like docx/xlsx/odf, so this walks the archive
// instead of using extractZippedXMLText's fixed member list.
func extractPptxText(p string) (string, error) {
	r, err := zip.OpenReader(p)
	if err != nil {
		return "", err
	}
	defer r.Close()

	var sb strings.Builder
	for _, f := range r.File {
		if !strings.HasPrefix(f.Name, "ppt/slides/slide") || !strings.HasSuffix(f.Name, ".xml") {
			continue
		}
		if text, terr := extractXMLText(f); terr == nil {
			sb.WriteString(text)
			sb.WriteString("\n")
		}
	}
	return sb.String(), nil
}

// extractXMLText reads a zipped XML member and concatenates its character
// data nodes, discarding markup -- good enough for search indexing, not a
// faithful document reconstruction.
func extractXMLText(f *zip.File) (string, error) {
	rc, err := f.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()

	dec := xml.NewDecoder(rc)
	var sb strings.Builder
	for {
		tok, terr := dec.Token()
		if errors.Is(terr, io.EOF) {
			break
		}
		if terr != nil {
			break
		}
		if cd, ok := tok.(xml.CharData); ok {
			sb.Write(cd)
			sb.WriteByte(' ')
		}
	}
	return sb.String(), nil
}

// extractPDFText extracts plain text from a PDF using a pure-Go parser (no
// pdftotext/pdfcpu binary required), so indexing works out of the box.
// Unparsable pages are skipped rather than failing the whole file -- a
// best-effort partial index beats no index for a PDF with one bad page.
func extractPDFText(p string) (string, error) {
	f, err := os.Open(p)
	if err != nil {
		return "", err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return "", err
	}

	reader, err := pdf.NewReader(f, info.Size())
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	fonts := make(map[string]*pdf.Font)
	for i := 1; i <= reader.NumPage(); i++ {
		page := reader.Page(i)
		for _, name := range page.Fonts() {
			if _, ok := fonts[name]; !ok {
				fnt := page.Font(name)
				fonts[name] = &fnt
			}
		}
		text, perr := page.GetPlainText(fonts)
		if perr != nil {
			continue
		}
		sb.WriteString(text)
		sb.WriteString("\n")
	}
	return sb.String(), nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// StartIndex builds or updates an inverted index of the given directory.
// It is called when the user runs '/search index'. Only changed files
// (different ModTime) are re-indexed; unchanged files are skipped.
// Returns the number of files indexed (new or changed).
func (e *Executor) StartIndex(path string) int {
	if path == "" {
		wd, err := os.Getwd()
		if err != nil {
			kdeps_debug.Log("searchLocal: StartIndex: getwd failed: " + err.Error())
			return 0
		}
		path = wd
	}

	// Verify path exists before attempting to create the index directory.
	if _, statErr := os.Stat(path); statErr != nil {
		fmt.Fprintf(ProgressWriter, "searchLocal: indexing %s ...\n", path)
		fmt.Fprintf(ProgressWriter, "searchLocal: no files indexed in %s\n", path)
		return 0
	}

	dbPath := indexDBPath(path)
	idx, existed, err := kgsearch.OpenOrCreate(dbPath)
	if err != nil {
		kdeps_debug.Log("searchLocal: failed to open index: " + err.Error())
		fmt.Fprintf(ProgressWriter, "searchLocal: failed to open index: %v\n", err)
		return 0
	}
	defer idx.Close()

	if existed {
		stats := idx.GetStats()
		if stats.DocumentCount > 0 {
			kdeps_debug.Log("searchLocal: updating existing index from " +
				dbPath + fmt.Sprintf(" (%d docs)", stats.DocumentCount))
		} else {
			kdeps_debug.Log("searchLocal: rebuilding index (empty)")
		}
	}

	fmt.Fprintf(ProgressWriter, "searchLocal: indexing %s ...\n", path)

	indexedCount, walkErr := e.walkIndex(path, "", idx)
	if walkErr != nil {
		fmt.Fprintf(ProgressWriter, "searchLocal: walk failed: %v\n", walkErr)
		return indexedCount
	}

	if indexedCount == 0 {
		fmt.Fprintf(ProgressWriter, "searchLocal: no files indexed in %s\n", path)
		kdeps_debug.Log(fmt.Sprintf("searchLocal: indexed 0 files in %s", path))
		return 0
	}

	fmt.Fprintf(ProgressWriter, "searchLocal: index saved to %s (%d files)\n", dbPath, indexedCount)
	kdeps_debug.Log(fmt.Sprintf("searchLocal: indexed %d files in %s", indexedCount, path))
	return indexedCount
}

// snippetMaxPreview is the max characters for a fallback snippet when query is
// not found: twice the snippet context.
func snippetMaxPreview() int { return snippetContext() * 2 } //nolint:mnd // both sides of a match

// generateSnippet reads a file and returns a snippet of context around the first
// occurrence of query. Returns empty string if the file can't be read or the
// query isn't found (keeps results compact but meaningful for the LLM).
func generateSnippet(filePath, query string) string {
	f, err := os.Open(filePath)
	if err != nil {
		return ""
	}
	defer f.Close()

	// Read up to 512 KiB — enough to find the query in any text file.
	buf := make([]byte, knowledge.SearchLimitsNow().SnippetReadBytes)
	n, _ := io.ReadFull(f, buf)
	if n == 0 {
		return ""
	}
	content := string(buf[:n])
	findStart, findEnd := newQueryMatcher(query).find(content)
	if findStart < 0 {
		// Query not found in content (token mismatch). Return first ~200 chars.
		end := len(content)
		if end > snippetMaxPreview() {
			end = snippetMaxPreview()
		}
		if end < len(content) {
			return content[:end] + "..."
		}
		return content[:end]
	}

	start := findStart - snippetContext()
	if start < 0 {
		start = 0
	}
	end := findEnd + snippetContext()
	if end > len(content) {
		end = len(content)
	}
	// Defend against Unicode case-expansion: strings.ToLower may produce a
	// longer byte string than the original content, so the literal-match index
	// can point past content's bounds after start/end clamping. Fall back to a
	// preview when indices are inverted or start exceeds content length.
	if start >= len(content) || end < start {
		previewEnd := len(content)
		if previewEnd > snippetMaxPreview() {
			previewEnd = snippetMaxPreview()
		}
		if previewEnd < len(content) {
			return content[:previewEnd] + "..."
		}
		return content[:previewEnd]
	}

	var sb strings.Builder
	if start > 0 {
		sb.WriteString("...")
	}
	sb.WriteString(content[start:end])
	if end < len(content) {
		sb.WriteString("...")
	}
	return sb.String()
}

// IndexFile indexes a single file into the CWD's search index. It opens the
// index DB, reads the file, tokenizes it, and closes the DB. If the index
// doesn't exist yet, it is created. Errors are silent (best-effort).
func (e *Executor) IndexFile(path string) {
	wd, err := os.Getwd()
	if err != nil {
		return
	}
	if abs, absErr := filepath.Abs(path); absErr == nil {
		e.indexInto(wd, []string{abs}, false)
	}
}

// IndexFiles indexes multiple files into root's search index in a single
// batch (root "" is the working directory). Files outside root are skipped:
// each folder's index holds only that folder's files.
func (e *Executor) IndexFiles(root string, paths []string) {
	if root == "" {
		wd, err := os.Getwd()
		if err != nil {
			return
		}
		root = wd
	}
	e.indexInto(filepath.Clean(root), paths, true)
}

// indexInto adds paths to root's search index in one batch through
// kartographer's search.IndexFiles. inRootOnly leaves out files outside root;
// IndexFile's working-directory index takes any path.
func (e *Executor) indexInto(root string, paths []string, inRootOnly bool) {
	idx, _, err := kgsearch.OpenOrCreate(indexDBPath(root))
	if err != nil {
		return
	}
	defer idx.Close()
	opts, err := docOptions()
	if err != nil {
		return
	}
	filter, err := knowledge.SearchFilter()
	if err != nil {
		return
	}
	within := root
	if !inRootOnly {
		within = filepath.VolumeName(root) + string(filepath.Separator)
	}
	kgsearch.IndexFiles(idx, within, paths, filter, opts)
}

func buildSearchResult(path string, results []map[string]interface{}) map[string]interface{} {
	if results == nil {
		results = []map[string]interface{}{}
	}
	result := map[string]interface{}{
		"results": results,
		"count":   len(results),
		"path":    path,
	}
	jsonBytes, _ := json.Marshal(result)
	result["json"] = string(jsonBytes)
	return result
}

// Execute performs a local filesystem search.
// When config.Index is true, it builds/uses a persistent TF-IDF inverted index
// for ranked search results with optional fuzzy matching.
// Otherwise, it uses the existing filepath.WalkDir approach with glob and content matching.
func (e *Executor) Execute(
	execCtx *executor.ExecutionContext,
	config *domain.SearchLocalConfig,
) (interface{}, error) {
	kdeps_debug.Log("enter: Execute")

	config, err := resolveConfig(execCtx, config)
	if err != nil {
		return nil, err
	}

	if config.Path == "" {
		return nil, errors.New("searchLocal: path is required")
	}
	if config.GraphBoost && !config.Index {
		return nil, errors.New("searchLocal: graphBoost requires index: true")
	}

	if config.Index {
		return e.executeIndexed(execCtx, config)
	}

	results, err := e.walk(config)
	if err != nil {
		return nil, err
	}

	return buildSearchResult(config.Path, results), nil
}

func execContext(execCtx *executor.ExecutionContext) context.Context {
	if execCtx == nil {
		return nil
	}
	return execCtx.Ctx
}

// executeIndexed builds or loads an inverted index and performs a ranked search.
func (e *Executor) executeIndexed(
	execCtx *executor.ExecutionContext,
	config *domain.SearchLocalConfig,
) (interface{}, error) {
	dbPath := config.IndexDBPath
	if dbPath == "" {
		dbPath = indexDBPath(config.Path)
	}

	idx, _, err := kgsearch.OpenOrCreate(dbPath)
	if err != nil {
		return nil, fmt.Errorf("searchLocal: failed to open index: %w", err)
	}
	defer idx.Close()

	// Only walk/index if the DB is empty — avoid catastrophic O(N*M) re-index.
	stats := idx.GetStats()
	indexedCount := stats.DocumentCount
	if indexedCount == 0 {
		var walkErr error
		indexedCount, walkErr = e.walkIndex(config.Path, config.Glob, idx)
		if walkErr != nil {
			return nil, fmt.Errorf("searchLocal: index walk failed: %w", walkErr)
		}
		kdeps_debug.Log(fmt.Sprintf("searchLocal: indexed %d files", indexedCount))
	}
	stats = idx.GetStats()

	// If no query, return index stats
	if config.Query == "" {
		return buildSearchResult(config.Path, []map[string]interface{}{
			{
				"indexedFiles": indexedCount,
				"totalDocs":    stats.DocumentCount,
				"totalTerms":   stats.TermCount,
				"indexDB":      dbPath,
			},
		}), nil
	}

	tok, tokErr := knowledge.SearchTokenizer()
	if tokErr != nil {
		return nil, tokErr
	}
	queryTerms := tok.TokenizeToStrings(config.Query)
	if len(queryTerms) == 0 {
		return buildSearchResult(config.Path, nil), nil
	}

	var results []kgsearch.SearchResult
	maxDist := config.MaxDistance
	if maxDist <= 0 {
		maxDist = 2
	}

	if config.Fuzzy {
		results = idx.FuzzySearch(queryTerms, maxDist)
	} else {
		results = idx.Search(queryTerms)
	}

	if config.GraphBoost {
		results = boostByGraph(execContext(execCtx), config, results)
	}

	// Apply limit
	limit := config.Limit
	if limit <= 0 || limit > len(results) {
		limit = len(results)
	}

	output := make([]map[string]interface{}, 0, limit)
	for _, r := range results[:limit] {
		snippet := r.Document.Preview
		// Fall back to file read only when the index predates the Preview field.
		if snippet == "" {
			snippet = generateSnippet(r.Document.Path, config.Query)
		}
		output = append(output, map[string]interface{}{
			"path":       r.Document.Path,
			"name":       filepath.Base(r.Document.Path),
			"size":       r.Document.Size,
			"isDir":      false,
			"score":      r.Score,
			"matchCount": r.MatchCount,
			"snippet":    snippet,
		})
	}

	return buildSearchResult(config.Path, output), nil
}
