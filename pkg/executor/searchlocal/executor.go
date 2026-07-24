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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/spf13/afero"

	kdeps_debug "github.com/kdeps/kdeps/v2/pkg/debug"
	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/executor"
	"github.com/kdeps/kdeps/v2/pkg/executor/searchlocal/internal/index"
	"github.com/kdeps/kdeps/v2/pkg/executor/searchlocal/internal/tokenizer"
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

// indexDBPath returns the default index database path for a directory.
func indexDBPath(dir string) string {
	return filepath.Join(dir, ".kdeps", "index.db")
}

// noIndexDirs are directory names that are never indexed.
//
//nolint:gochecknoglobals // static lookup table
var noIndexDirs = map[string]bool{
	// Version control
	".git": true,
	".hg":  true,
	".svn": true,

	// JavaScript / TypeScript
	"node_modules":     true,
	"bower_components": true,
	".next":            true,
	".nuxt":            true,
	".output":          true,
	".turbo":           true,
	".parcel-cache":    true,
	".svelte-kit":      true,
	".angular":         true,
	".yarn":            true,

	// Python
	"__pycache__":        true,
	".venv":              true,
	"venv":               true,
	".env":               true,
	"env":                true,
	".tox":               true,
	".mypy_cache":        true,
	".pytest_cache":      true,
	".ruff_cache":        true,
	"site-packages":      true,
	"dist-packages":      true,
	".eggs":              true,
	"eggs":               true,
	"pip-wheel-metadata": true,

	// Go
	"vendor": true,

	// Rust / Java / C++ / C#
	"target": true,
	"dist":   true,
	"build":  true,
	"obj":    true,
	"bin":    true,

	// Infrastructure / IaC
	".terraform":  true,
	".serverless": true,
	"cdk.out":     true,

	// Coverage / test artifacts
	"coverage":      true,
	"__snapshots__": true,

	// Misc cache / temp
	".cache": true,
	"tmp":    true,
	"temp":   true,
}

// skipWalkEntry returns true if the walk entry should be skipped.
func skipWalkEntry(d os.DirEntry) bool {
	if len(d.Name()) == 0 {
		return true
	}
	if d.Name()[0] == '.' {
		return true
	}
	return noIndexDirs[d.Name()]
}

// indexBatchSize is how many files are collected before flushing to bbolt.
// indexableExtensions are file extensions that are indexed in index mode.
// Files without an extension (e.g. Makefile, README) are always indexed.
//
//nolint:gochecknoglobals // static lookup table
var indexableExtensions = map[string]bool{
	".go": true, ".py": true, ".js": true, ".ts": true, ".tsx": true, ".jsx": true,
	".java": true, ".c": true, ".cpp": true, ".cc": true, ".h": true, ".hpp": true,
	".rs": true, ".rb": true, ".php": true, ".sh": true, ".bash": true, ".zsh": true,
	".yml": true, ".yaml": true, ".json": true, ".xml": true, ".toml": true,
	".ini": true, ".cfg": true, ".conf": true, ".envrc": true,
	".md": true, ".txt": true, ".rst": true, ".adoc": true,
	".html": true, ".css": true, ".scss": true, ".less": true, ".svg": true,
	".sql": true, ".proto": true, ".graphql": true, ".prisma": true,
	".dart": true, ".swift": true, ".kt": true, ".kts": true,
	".scala": true, ".clj": true, ".cljs": true, ".ex": true, ".exs": true, ".erl": true,
	".lua": true, ".zig": true, ".nim": true, ".vue": true, ".svelte": true,
	".tf": true, ".hcl": true, ".nix": true, ".dhall": true,
}

// isIndexableExt returns true if the file extension is indexable.
// Files without an extension (e.g. Makefile, README) are considered indexable.
func isIndexableExt(name string) bool {
	ext := filepath.Ext(name)
	return ext == "" || indexableExtensions[ext]
}

const indexBatchSize = 2000

// maxFileSize is the largest file (bytes) we will read and index.
const maxFileSize = 1 << 20 // 1 MiB

// indexWorkers is the number of parallel file readers. Defaults to GOMAXPROCS.
var indexWorkers = runtime.GOMAXPROCS(0) //nolint:gochecknoglobals // parallel file readers

// fileJob is a file to be read and indexed by a worker.
type fileJob struct {
	path string
	d    os.DirEntry
}

// walkIndex walks the directory tree and adds documents to the index.
// File reading and tokenization runs in parallel across indexWorkers goroutines;
// bbolt writes are serialized in a single goroutine, batched every indexBatchSize.
// Incremental: files with unchanged ModTime are skipped; deleted files are purged.
func (e *Executor) walkIndex(root string, glob string, idx *index.InvertedIndex) (int, error) {
	jobs := make(chan fileJob, indexBatchSize)
	results := make(chan index.DocWithTokens, indexBatchSize)
	done := make(chan error, 1)

	// Track seen docIDs for stale-document cleanup after the walk.
	seenDocIDs := make(map[string]bool)
	var seenMu sync.Mutex

	// Single writer: batches results and flushes to bbolt.
	var indexedCount int
	go func() {
		batch := make([]index.DocWithTokens, 0, indexBatchSize)
		for doc := range results {
			batch = append(batch, doc)
			indexedCount++
			if len(batch) >= indexBatchSize {
				idx.BatchAddDocuments(batch)
				batch = batch[:0]
			}
			if indexedCount%100 == 0 {
				fmt.Fprintf(ProgressWriter, "\rsearchLocal: indexed %d files ...", indexedCount)
			}
		}
		if len(batch) > 0 {
			idx.BatchAddDocuments(batch)
		}
		done <- nil
	}()

	// Worker pool: reads and tokenizes files in parallel.
	var wg sync.WaitGroup
	for range indexWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tok := tokenizer.NewTokenizer()
			for job := range jobs {
				if doc, tokens, positions := e.prepareDoc(job.path, job.d, tok); doc != nil {
					results <- index.DocWithTokens{Doc: doc, Tokens: tokens, Positions: positions}
				}
			}
		}()
	}

	// Walk the directory tree and feed jobs.
	var skippedIncremental int
	walkErr := filepath.WalkDir(root,
		e.makeWalkCallback(root, glob, idx, jobs, &seenDocIDs, &seenMu, &skippedIncremental))
	close(jobs)
	wg.Wait()
	close(results)
	<-done

	// Clean up stale documents (files that were deleted since last index).
	staleRemoved := idx.PurgeDocs(seenDocIDs)
	if staleRemoved > 0 {
		fmt.Fprintf(ProgressWriter, "\rsearchLocal: removed %d stale docs from index\n", staleRemoved)
	}

	return indexedCount, walkErr
}

// walkCallback is the signature for filepath.WalkDir callbacks.
type walkCallback = func(string, os.DirEntry, error) error

// makeWalkCallback returns a WalkDir callback that feeds indexable files into the jobs channel.
func (e *Executor) makeWalkCallback(
	_ string, glob string, idx *index.InvertedIndex,
	jobs chan<- fileJob,
	seenDocIDs *map[string]bool, seenMu *sync.Mutex,
	skippedIncremental *int,
) walkCallback {
	return func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil //nolint:nilerr // skip unreadable entries during walk
		}
		if d.IsDir() {
			if skipWalkEntry(d) {
				return fs.SkipDir
			}
			return nil
		}
		if skipWalkEntry(d) || !isIndexableExt(d.Name()) {
			return nil
		}
		if !matchGlob(glob, d.Name()) {
			return nil
		}

		docID := fmt.Sprintf("%x", p)
		seenMu.Lock()
		(*seenDocIDs)[docID] = true
		seenMu.Unlock()

		if e.skipUnchanged(d, docID, idx, skippedIncremental) {
			return nil
		}

		jobs <- fileJob{path: p, d: d}
		return nil
	}
}

// matchGlob returns true if the filename matches the glob pattern.
// An empty glob matches everything.
func matchGlob(glob, name string) bool {
	if glob == "" {
		return true
	}
	matched, err := filepath.Match(glob, name)
	return err == nil && matched
}

// skipUnchanged returns true if the file's ModTime matches the index, meaning it hasn't changed.
func (e *Executor) skipUnchanged(d os.DirEntry, docID string, idx *index.InvertedIndex, skippedIncremental *int) bool {
	info, statErr := d.Info()
	if statErr != nil {
		return true
	}
	if info.ModTime().Unix() == idx.LookupModTime(docID) {
		*skippedIncremental++
		if *skippedIncremental%1000 == 0 {
			fmt.Fprintf(ProgressWriter, "\rsearchLocal: %d files unchanged, skipped ...", *skippedIncremental)
		}
		return true
	}
	return false
}

// prepareDoc reads and tokenizes a file, returning a ready-to-index document or nil.
// Files larger than maxFileSize are skipped. The first ~240 chars of content are
// captured as a preview so snippet generation never re-reads files from disk.
func (e *Executor) prepareDoc(
	p string, d os.DirEntry, tok *tokenizer.Tokenizer,
) (*index.Document, []string, []int) {
	info, statErr := d.Info()
	if statErr != nil {
		return nil, nil, nil
	}
	if info.Size() > maxFileSize {
		return nil, nil, nil
	}

	content, readErr := os.ReadFile(p)
	if readErr != nil {
		return nil, nil, nil
	}

	docID := fmt.Sprintf("%x", p)
	tokens := tok.Tokenize(string(content))
	tokenStrings := make([]string, len(tokens))
	positions := make([]int, len(tokens))
	for i, token := range tokens {
		tokenStrings[i] = token.Text
		positions[i] = token.Position
	}

	preview := content
	const previewMax = 240
	if len(preview) > previewMax {
		preview = preview[:previewMax]
	}

	return &index.Document{
		ID:      docID,
		Path:    p,
		ModTime: info.ModTime().Unix(),
		Size:    info.Size(),
		Preview: string(preview),
	}, tokenStrings, positions
}

// openOrCreateIndex opens an existing bbolt index or creates a new one,
// ensuring the parent directory exists. If the file exists but is not a
// valid bbolt database (e.g. a leftover gob-format index or corrupted file),
// it is removed and a fresh database is created. Returns the index and
// whether a valid database already existed.
func openOrCreateIndex(dbPath string) (*index.InvertedIndex, bool, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return nil, false, err
	}

	existed := fileExists(dbPath)

	idx, err := index.NewInvertedIndex(dbPath)
	if err == nil {
		return idx, existed, nil
	}

	// Remove the invalid/corrupt file and create a fresh database.
	_ = os.Remove(dbPath)
	idx, createErr := index.NewInvertedIndex(dbPath)
	if createErr != nil {
		return nil, false, createErr
	}
	return idx, false, nil
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
	idx, existed, err := openOrCreateIndex(dbPath)
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

// snippetContext is the number of characters of context around a query match.
const snippetContext = 120

// snippetMaxPreview is the max characters for a fallback snippet when query is not found.
const snippetMaxPreview = snippetContext * 2

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
	const maxRead = 512 << 10
	buf := make([]byte, maxRead)
	n, _ := io.ReadFull(f, buf)
	if n == 0 {
		return ""
	}
	content := string(buf[:n])
	lower := strings.ToLower(content)
	queryLower := strings.ToLower(query)

	idx := strings.Index(lower, queryLower)
	if idx < 0 {
		// Query not found in content (token mismatch). Return first ~200 chars.
		end := len(content)
		if end > snippetMaxPreview {
			end = snippetMaxPreview
		}
		if end < len(content) {
			return content[:end] + "..."
		}
		return content[:end]
	}

	start := idx - snippetContext
	if start < 0 {
		start = 0
	}
	end := idx + len(query) + snippetContext
	if end > len(content) {
		end = len(content)
	}
	// Defend against Unicode case-expansion: strings.ToLower may produce a
	// longer byte string than the original content, so idx (from lower) can
	// point past content's bounds after start/end clamping. Fall back to a
	// preview when indices are inverted or start exceeds content length.
	if start >= len(content) || end < start {
		previewEnd := len(content)
		if previewEnd > snippetMaxPreview {
			previewEnd = snippetMaxPreview
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
	dbPath := indexDBPath(wd)
	idx, _, openErr := openOrCreateIndex(dbPath)
	if openErr != nil {
		return
	}
	defer idx.Close()

	info, statErr := os.Stat(path)
	if statErr != nil || info.IsDir() || info.Size() > maxFileSize {
		return
	}
	if !isIndexableExt(filepath.Base(path)) {
		return
	}

	content, readErr := os.ReadFile(path)
	if readErr != nil {
		return
	}

	docID := fmt.Sprintf("%x", path)
	tok := tokenizer.NewTokenizer()
	tokens := tok.Tokenize(string(content))
	tokenStrings := make([]string, len(tokens))
	positions := make([]int, len(tokens))
	for i, token := range tokens {
		tokenStrings[i] = token.Text
		positions[i] = token.Position
	}

	preview := content
	const previewMax = 240
	if len(preview) > previewMax {
		preview = preview[:previewMax]
	}

	idx.BatchAddDocuments([]index.DocWithTokens{{
		Doc: &index.Document{
			ID:      docID,
			Path:    path,
			ModTime: info.ModTime().Unix(),
			Size:    info.Size(),
			Preview: string(preview),
		},
		Tokens:    tokenStrings,
		Positions: positions,
	}})
}

// IndexFiles indexes multiple files into the CWD's search index in a single batch.
func (e *Executor) IndexFiles(paths []string) {
	wd, err := os.Getwd()
	if err != nil {
		return
	}
	dbPath := indexDBPath(wd)
	idx, _, openErr := openOrCreateIndex(dbPath)
	if openErr != nil {
		return
	}
	defer idx.Close()

	tok := tokenizer.NewTokenizer()
	docs := make([]index.DocWithTokens, 0, len(paths))
	for _, path := range paths {
		info, statErr := os.Stat(path)
		if statErr != nil || info.IsDir() || info.Size() > maxFileSize {
			continue
		}
		if !isIndexableExt(filepath.Base(path)) {
			continue
		}

		content, readErr := os.ReadFile(path)
		if readErr != nil {
			continue
		}

		docID := fmt.Sprintf("%x", path)
		tokens := tok.Tokenize(string(content))
		tokenStrings := make([]string, len(tokens))
		positions := make([]int, len(tokens))
		for i, token := range tokens {
			tokenStrings[i] = token.Text
			positions[i] = token.Position
		}

		preview := content
		const previewMax = 240
		if len(preview) > previewMax {
			preview = preview[:previewMax]
		}

		docs = append(docs, index.DocWithTokens{
			Doc: &index.Document{
				ID:      docID,
				Path:    path,
				ModTime: info.ModTime().Unix(),
				Size:    info.Size(),
				Preview: string(preview),
			},
			Tokens:    tokenStrings,
			Positions: positions,
		})
		idx.BatchAddDocuments(docs)
	}
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
	_ *executor.ExecutionContext,
	config *domain.SearchLocalConfig,
) (interface{}, error) {
	kdeps_debug.Log("enter: Execute")

	if config.Path == "" {
		return nil, errors.New("searchLocal: path is required")
	}

	if config.Index {
		return e.executeIndexed(config)
	}

	results, err := e.walk(config)
	if err != nil {
		return nil, err
	}

	return buildSearchResult(config.Path, results), nil
}

// executeIndexed builds or loads an inverted index and performs a ranked search.
func (e *Executor) executeIndexed(config *domain.SearchLocalConfig) (interface{}, error) {
	dbPath := config.IndexDBPath
	if dbPath == "" {
		dbPath = indexDBPath(config.Path)
	}

	idx, _, err := openOrCreateIndex(dbPath)
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

	tok := tokenizer.NewTokenizer()
	queryTerms := tok.TokenizeToStrings(config.Query)
	if len(queryTerms) == 0 {
		return buildSearchResult(config.Path, nil), nil
	}

	var results []index.SearchResult
	maxDist := config.MaxDistance
	if maxDist <= 0 {
		maxDist = 2
	}

	if config.Fuzzy {
		results = idx.FuzzySearch(queryTerms, maxDist)
	} else {
		results = idx.Search(queryTerms)
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
