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

// Package index provides a bbolt-backed inverted index with TF-IDF ranking
// for the searchlocal executor.
package index

import (
	"bytes"
	"encoding/gob"
	"math"
	"sort"

	bolt "go.etcd.io/bbolt"

	"github.com/kdeps/kdeps/v2/pkg/executor/searchlocal/internal/util"
)

//nolint:gochecknoglobals // bbolt bucket names, static byte slices
var (
	bucketMeta  = []byte("__meta__")
	bucketTerms = []byte("terms")
	bucketDocs  = []byte("docs")
	keyDocCount = []byte("docCount")
)

// Document represents an indexed document.
type Document struct {
	ID      string
	Path    string
	Content string
	ModTime int64
	Size    int64
}

// Posting represents a document that contains a term.
type Posting struct {
	DocID     string
	Frequency int
	Positions []int
}

// InvertedIndex is a bbolt-backed inverted index.
type InvertedIndex struct {
	db     *bolt.DB
	dbPath string
}

const defaultDBMode = 0600

// NewInvertedIndex opens or creates a bbolt-backed inverted index at dbPath.
// The caller must call Close() when done.
func NewInvertedIndex(dbPath string) (*InvertedIndex, error) {
	db, openErr := bolt.Open(dbPath, defaultDBMode, nil)
	if openErr != nil {
		return nil, openErr
	}
	if bucketErr := db.Update(func(tx *bolt.Tx) error {
		for _, b := range [][]byte{bucketMeta, bucketTerms, bucketDocs} {
			if _, err := tx.CreateBucketIfNotExists(b); err != nil {
				return err
			}
		}
		return nil
	}); bucketErr != nil {
		_ = db.Close()
		return nil, bucketErr
	}
	return &InvertedIndex{db: db, dbPath: dbPath}, nil
}

// Close closes the bbolt database.
func (idx *InvertedIndex) Close() error {
	return idx.db.Close()
}

// DBPath returns the path to the bbolt database file.
func (idx *InvertedIndex) DBPath() string {
	return idx.dbPath
}

// AddDocument adds or updates a document in the index.
func (idx *InvertedIndex) AddDocument(doc *Document, tokens []string, positions []int) {
	termFreq := make(map[string]int)
	termPositions := make(map[string][]int)
	for i, token := range tokens {
		termFreq[token]++
		termPositions[token] = append(termPositions[token], positions[i])
	}

	_ = idx.db.Update(func(tx *bolt.Tx) error {
		// Remove old document if it exists.
		if oldData := tx.Bucket(bucketDocs).Get([]byte(doc.ID)); oldData != nil {
			idx.removeDocumentInTx(tx, doc.ID)
		}

		// Store the document.
		var docBuf bytes.Buffer
		if err := gob.NewEncoder(&docBuf).Encode(doc); err != nil {
			return err
		}
		if err := tx.Bucket(bucketDocs).Put([]byte(doc.ID), docBuf.Bytes()); err != nil {
			return err
		}

		// Update term postings.
		termsB := tx.Bucket(bucketTerms)
		for term, freq := range termFreq {
			p := Posting{DocID: doc.ID, Frequency: freq, Positions: termPositions[term]}
			postings := readPostings(termsB, term)
			postings = append(postings, p)
			if err := writePostings(termsB, term, postings); err != nil {
				return err
			}
		}

		// Increment doc count.
		return incDocCount(tx, 1)
	})
}

// removeDocumentInTx removes a document from the index within a transaction.
func (idx *InvertedIndex) removeDocumentInTx(tx *bolt.Tx, docID string) {
	termsB := tx.Bucket(bucketTerms)
	c := termsB.Cursor()
	for k, v := c.First(); k != nil; k, v = c.Next() {
		postings := decodePostings(v)
		filtered := make([]Posting, 0, len(postings))
		for _, p := range postings {
			if p.DocID != docID {
				filtered = append(filtered, p)
			}
		}
		if len(filtered) == 0 {
			_ = termsB.Delete(k)
		} else {
			var buf bytes.Buffer
			if err := gob.NewEncoder(&buf).Encode(filtered); err == nil {
				_ = termsB.Put(k, buf.Bytes())
			}
		}
	}
}

// scoreDocs computes TF-IDF scores for query terms against the index.
func (idx *InvertedIndex) scoreDocs(tx *bolt.Tx, queryTerms []string) (map[string]float64, map[string]int) {
	docCount := readDocCount(tx)
	docScores := make(map[string]float64)
	docMatches := make(map[string]int)
	termsB := tx.Bucket(bucketTerms)

	for _, term := range queryTerms {
		postings := readPostings(termsB, term)
		if len(postings) == 0 {
			continue
		}
		idf := calculateIDF(len(postings), docCount)
		for _, p := range postings {
			docScores[p.DocID] += float64(p.Frequency) * idf
			docMatches[p.DocID]++
		}
	}
	return docScores, docMatches
}

// resolveDoc loads a document from the docs bucket.
func resolveDoc(docsB *bolt.Bucket, docID string) *Document {
	data := docsB.Get([]byte(docID))
	if data == nil {
		return nil
	}
	var doc Document
	if gob.NewDecoder(bytes.NewReader(data)).Decode(&doc) != nil {
		return nil
	}
	return &doc
}

// Search performs a search query and returns ranked results.
func (idx *InvertedIndex) Search(queryTerms []string) []SearchResult {
	docScores := make(map[string]float64)
	docMatches := make(map[string]int)

	_ = idx.db.View(func(tx *bolt.Tx) error {
		docScores, docMatches = idx.scoreDocs(tx, queryTerms)
		// Prune scores for documents that no longer exist.
		docsB := tx.Bucket(bucketDocs)
		for docID := range docScores {
			if resolveDoc(docsB, docID) == nil {
				delete(docScores, docID)
				delete(docMatches, docID)
			}
		}
		return nil
	})

	results := make([]SearchResult, 0, len(docScores))
	_ = idx.db.View(func(tx *bolt.Tx) error {
		docsB := tx.Bucket(bucketDocs)
		for docID, score := range docScores {
			if doc := resolveDoc(docsB, docID); doc != nil {
				results = append(results, SearchResult{
					Document:   doc,
					Score:      score,
					MatchCount: docMatches[docID],
					QueryTerms: len(queryTerms),
				})
			}
		}
		return nil
	})

	sort.Slice(results, func(i, j int) bool {
		if results[i].MatchCount != results[j].MatchCount {
			return results[i].MatchCount > results[j].MatchCount
		}
		return results[i].Score > results[j].Score
	})

	return results
}

// FuzzySearch performs fuzzy matching search.
func (idx *InvertedIndex) FuzzySearch(queryTerms []string, maxDistance int) []SearchResult {
	expandedTerms := make(map[string]bool)
	for _, qt := range queryTerms {
		expandedTerms[qt] = true
	}

	// Iterate all index terms and match by Levenshtein distance.
	_ = idx.db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket(bucketTerms).Cursor()
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			indexTerm := string(k)
			for _, qt := range queryTerms {
				if levenshteinDistance(qt, indexTerm) <= maxDistance {
					expandedTerms[indexTerm] = true
				}
			}
		}
		return nil
	})

	terms := make([]string, 0, len(expandedTerms))
	for t := range expandedTerms {
		terms = append(terms, t)
	}
	return idx.Search(terms)
}

// GetStats returns index statistics.
func (idx *InvertedIndex) GetStats() Stats {
	var s Stats
	_ = idx.db.View(func(tx *bolt.Tx) error {
		s.DocumentCount = readDocCount(tx)
		s.TermCount = tx.Bucket(bucketTerms).Stats().KeyN
		return nil
	})
	return s
}

// SearchResult represents a search result with scoring.
type SearchResult struct {
	Document   *Document
	Score      float64
	MatchCount int
	QueryTerms int
}

// Stats contains index statistics.
type Stats struct {
	DocumentCount int
	TermCount     int
}

// --- bbolt helpers ---

func readPostings(b *bolt.Bucket, term string) []Posting {
	data := b.Get([]byte(term))
	if data == nil {
		return nil
	}
	return decodePostings(data)
}

func decodePostings(data []byte) []Posting {
	var p []Posting
	if err := gob.NewDecoder(bytes.NewReader(data)).Decode(&p); err != nil {
		return nil
	}
	return p
}

func writePostings(b *bolt.Bucket, term string, p []Posting) error {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(p); err != nil {
		return err
	}
	return b.Put([]byte(term), buf.Bytes())
}

func readDocCount(tx *bolt.Tx) int {
	data := tx.Bucket(bucketMeta).Get(keyDocCount)
	if data == nil {
		return 0
	}
	var count int
	if err := gob.NewDecoder(bytes.NewReader(data)).Decode(&count); err != nil {
		return 0
	}
	return count
}

func incDocCount(tx *bolt.Tx, delta int) error {
	current := readDocCount(tx)
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(current + delta); err != nil {
		return err
	}
	return tx.Bucket(bucketMeta).Put(keyDocCount, buf.Bytes())
}

func calculateIDF(docFreq, docCount int) float64 {
	if docCount == 0 || docFreq == 0 {
		return 0
	}
	return math.Log(float64(docCount) / float64(docFreq))
}

// levenshteinDistance calculates the edit distance between two strings.
func levenshteinDistance(s1, s2 string) int {
	if s1 == s2 {
		return 0
	}
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	prev := make([]int, len(s2)+1)
	curr := make([]int, len(s2)+1)
	for j := 0; j <= len(s2); j++ {
		prev[j] = j
	}

	for i := 1; i <= len(s1); i++ {
		curr[0] = i
		for j := 1; j <= len(s2); j++ {
			cost := 1
			if s1[i-1] == s2[j-1] {
				cost = 0
			}
			curr[j] = util.MinInt3(prev[j]+1, curr[j-1]+1, prev[j-1]+cost)
		}
		prev, curr = curr, prev
	}
	return prev[len(s2)]
}
