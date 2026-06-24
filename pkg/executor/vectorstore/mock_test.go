// Copyright 2026 Kdeps, KvK 94834768
// Licensed under the Apache License, Version 2.0

package vectorstore

import (
	"context"

	"github.com/tmc/langchaingo/schema"
	lcvectorstores "github.com/tmc/langchaingo/vectorstores"
)

// mockVectorStore implements lcvectorstores.VectorStore with controllable results.
type mockVectorStore struct {
	addDocsResult []string
	addDocsErr    error
	searchResult  []schema.Document
	searchErr     error
}

func (m *mockVectorStore) AddDocuments(
	_ context.Context, _ []schema.Document, _ ...lcvectorstores.Option,
) ([]string, error) {
	return m.addDocsResult, m.addDocsErr
}

func (m *mockVectorStore) SimilaritySearch(
	_ context.Context, _ string, _ int, _ ...lcvectorstores.Option,
) ([]schema.Document, error) {
	return m.searchResult, m.searchErr
}
