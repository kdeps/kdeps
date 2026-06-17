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

// Package vectorstore executes vectorStore: resources, adding documents to
// and searching a vector database. Supported providers: qdrant (default),
// azureaisearch, chroma, pinecone, opensearch, elasticsearch, weaviate,
// mariadb, dolt, mysql, pgvector, postgres, postgresql, alloydb, cloudsql,
// mongodb, mongo.
package vectorstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"

	lcemb "github.com/tmc/langchaingo/embeddings"
	lchemb_bedrock "github.com/tmc/langchaingo/embeddings/bedrock"
	lchemb_cybertron "github.com/tmc/langchaingo/embeddings/cybertron"
	lchemb_hf "github.com/tmc/langchaingo/embeddings/huggingface"
	lchemb_jina "github.com/tmc/langchaingo/embeddings/jina"
	lchemb_voyage "github.com/tmc/langchaingo/embeddings/voyageai"
	lcgoogleai "github.com/tmc/langchaingo/llms/googleai"
	lchf "github.com/tmc/langchaingo/llms/huggingface"
	lcopenai "github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/schema"
	lcvectorstores "github.com/tmc/langchaingo/vectorstores"
	lcazure "github.com/tmc/langchaingo/vectorstores/azureaisearch"
	lcqdrant "github.com/tmc/langchaingo/vectorstores/qdrant"

	kdeps_debug "github.com/kdeps/kdeps/v2/pkg/debug"
	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/executor"
)

// Executor runs vectorStore: resources.
type Executor struct{}

// NewExecutor creates a new vector store executor.
func NewExecutor() *Executor {
	kdeps_debug.Log("enter: vectorstore.NewExecutor")
	return &Executor{}
}

// Execute runs the configured vector store operation.
func (e *Executor) Execute(
	_ *executor.ExecutionContext,
	cfg *domain.VectorStoreConfig,
) (interface{}, error) {
	kdeps_debug.Log("enter: vectorstore.Execute")

	ctx := context.Background()

	switch cfg.Operation {
	case "add_documents":
		return executeAddDocuments(ctx, cfg)
	case "similarity_search":
		return executeSimilaritySearch(ctx, cfg)
	default:
		return nil, fmt.Errorf(
			"vectorstore: unknown operation %q (use add_documents, similarity_search)",
			cfg.Operation,
		)
	}
}

func executeAddDocuments(ctx context.Context, cfg *domain.VectorStoreConfig) (interface{}, error) {
	if len(cfg.Documents) == 0 {
		return nil, errors.New("vectorstore add_documents: documents is required")
	}

	store, err := buildStore(ctx, cfg)
	if err != nil {
		return nil, err
	}

	docs := make([]schema.Document, len(cfg.Documents))
	for i, d := range cfg.Documents {
		docs[i] = schema.Document{
			PageContent: d.Content,
			Metadata:    d.Metadata,
		}
	}

	ids, err := store.AddDocuments(ctx, docs, lcvectorstores.WithNameSpace(cfg.Collection))
	if err != nil {
		return nil, fmt.Errorf("vectorstore add_documents: %w", err)
	}

	return map[string]interface{}{
		"added": len(ids),
		"ids":   ids,
	}, nil
}

func executeSimilaritySearch(
	ctx context.Context,
	cfg *domain.VectorStoreConfig,
) (interface{}, error) {
	if cfg.Query == "" {
		return nil, errors.New("vectorstore similarity_search: query is required")
	}

	store, err := buildStore(ctx, cfg)
	if err != nil {
		return nil, err
	}

	topK := cfg.TopK
	if topK <= 0 {
		topK = 5
	}

	docs, err := store.SimilaritySearch(
		ctx,
		cfg.Query,
		topK,
		lcvectorstores.WithNameSpace(cfg.Collection),
	)
	if err != nil {
		return nil, fmt.Errorf("vectorstore similarity_search: %w", err)
	}

	results := make([]map[string]interface{}, len(docs))
	for i, d := range docs {
		results[i] = map[string]interface{}{
			"content":  d.PageContent,
			"metadata": d.Metadata,
		}
	}

	b, _ := json.Marshal(results)
	return map[string]interface{}{
		"results": results,
		"count":   len(results),
		"json":    string(b),
	}, nil
}

func buildStore(
	ctx context.Context,
	cfg *domain.VectorStoreConfig,
) (lcvectorstores.VectorStore, error) {
	if cfg.Collection == "" {
		return nil, errors.New("vectorstore: collection is required")
	}
	if cfg.EmbedModel == "" && cfg.Provider != "bedrock" {
		return nil, errors.New("vectorstore: embedModel is required")
	}

	switch cfg.Provider {
	case "azureaisearch":
		return buildAzureAISearchStore(ctx, cfg)
	case "bedrock":
		return buildBedrockStore(ctx, cfg)
	case "chroma":
		return buildChromaStore(ctx, cfg)
	case "pinecone":
		return buildPineconeStore(ctx, cfg)
	case "opensearch", "elasticsearch":
		return buildOpenSearchStore(ctx, cfg)
	case "weaviate":
		return buildWeaviateStore(ctx, cfg)
	case "mariadb", "dolt", "mysql":
		return buildMySQLStore(ctx, cfg)
	case "pgvector", "postgres", "postgresql", "alloydb", "cloudsql":
		return buildPostgresStore(ctx, cfg)
	case "mongodb", "mongo":
		return buildMongoStore(ctx, cfg)
	default:
		return buildQdrantStore(ctx, cfg)
	}
}

func buildBedrockStore(
	ctx context.Context,
	cfg *domain.VectorStoreConfig,
) (lcvectorstores.VectorStore, error) {
	// Bedrock KB handles embedding server-side — no local embedder needed.
	return newBedrockStore(ctx, cfg)
}

func buildQdrantStore(
	ctx context.Context,
	cfg *domain.VectorStoreConfig,
) (lcvectorstores.VectorStore, error) {
	if cfg.URL == "" {
		return nil, errors.New("vectorstore: url is required")
	}

	qdrantURL, err := url.Parse(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("vectorstore: invalid url %q: %w", cfg.URL, err)
	}

	embedder, err := buildEmbedder(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("vectorstore: build embedder: %w", err)
	}

	opts := []lcqdrant.Option{
		lcqdrant.WithURL(*qdrantURL),
		lcqdrant.WithCollectionName(cfg.Collection),
		lcqdrant.WithEmbedder(embedder),
	}
	if cfg.APIKey != "" {
		opts = append(opts, lcqdrant.WithAPIKey(cfg.APIKey))
	}

	store, err := lcqdrant.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("vectorstore: qdrant: %w", err)
	}
	return &store, nil
}

func buildChromaStore(ctx context.Context, cfg *domain.VectorStoreConfig) (lcvectorstores.VectorStore, error) {
	embedder, err := buildEmbedder(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("vectorstore: build embedder: %w", err)
	}
	return newChromaStore(cfg, embedder), nil
}

func buildPineconeStore(ctx context.Context, cfg *domain.VectorStoreConfig) (lcvectorstores.VectorStore, error) {
	embedder, err := buildEmbedder(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("vectorstore: build embedder: %w", err)
	}
	return newPineconeStore(cfg, embedder)
}

func buildOpenSearchStore(ctx context.Context, cfg *domain.VectorStoreConfig) (lcvectorstores.VectorStore, error) {
	embedder, err := buildEmbedder(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("vectorstore: build embedder: %w", err)
	}
	return newOpenSearchStore(cfg, embedder)
}

func buildWeaviateStore(ctx context.Context, cfg *domain.VectorStoreConfig) (lcvectorstores.VectorStore, error) {
	embedder, err := buildEmbedder(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("vectorstore: build embedder: %w", err)
	}
	return newWeaviateStore(cfg, embedder)
}

func buildMySQLStore(ctx context.Context, cfg *domain.VectorStoreConfig) (lcvectorstores.VectorStore, error) {
	embedder, err := buildEmbedder(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("vectorstore: build embedder: %w", err)
	}
	return newMySQLStore(cfg, embedder)
}

func buildPostgresStore(
	ctx context.Context,
	cfg *domain.VectorStoreConfig,
) (lcvectorstores.VectorStore, error) {
	embedder, err := buildEmbedder(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("vectorstore: build embedder: %w", err)
	}
	return newPostgresStore(cfg, embedder)
}

func buildMongoStore(
	ctx context.Context,
	cfg *domain.VectorStoreConfig,
) (lcvectorstores.VectorStore, error) {
	embedder, err := buildEmbedder(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("vectorstore: build embedder: %w", err)
	}
	return newMongoStore(ctx, cfg, embedder)
}

func buildAzureAISearchStore(
	ctx context.Context,
	cfg *domain.VectorStoreConfig,
) (lcvectorstores.VectorStore, error) {
	embedder, err := buildEmbedder(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("vectorstore: build embedder: %w", err)
	}

	opts := []lcazure.Option{
		lcazure.WithEmbedder(embedder),
	}
	if cfg.URL != "" {
		opts = append(opts, lcazure.WithEndpoint(cfg.URL))
	}
	if cfg.APIKey != "" {
		opts = append(opts, lcazure.WithAPIKey(cfg.APIKey))
	}

	store, err := lcazure.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("vectorstore: azureaisearch: %w", err)
	}
	return &store, nil
}

func buildEmbedder(ctx context.Context, cfg *domain.VectorStoreConfig) (lcemb.Embedder, error) {
	switch cfg.EmbedBackend {
	case "google":
		apiKey := os.Getenv("GOOGLE_API_KEY")
		client, err := lcgoogleai.New(ctx,
			lcgoogleai.WithAPIKey(apiKey),
			lcgoogleai.WithDefaultModel(cfg.EmbedModel),
		)
		if err != nil {
			return nil, err
		}
		return lcemb.NewEmbedder(client)
	case "huggingface":
		token := os.Getenv("HF_TOKEN")
		if token == "" {
			token = os.Getenv("HUGGINGFACEHUB_API_TOKEN")
		}
		llmClient, hfErr := lchf.New(lchf.WithToken(token))
		if hfErr != nil {
			return nil, fmt.Errorf("vectorstore: build huggingface client: %w", hfErr)
		}
		opts := []lchemb_hf.Option{lchemb_hf.WithClient(*llmClient)}
		if cfg.EmbedModel != "" {
			opts = append(opts, lchemb_hf.WithModel(cfg.EmbedModel))
		}
		return lchemb_hf.NewHuggingface(opts...)
	case "jina":
		return lchemb_jina.NewJina(
			lchemb_jina.WithAPIKey(os.Getenv("JINA_API_KEY")),
			lchemb_jina.WithModel(cfg.EmbedModel),
		)
	case "voyageai":
		return lchemb_voyage.NewVoyageAI(
			lchemb_voyage.WithToken(os.Getenv("VOYAGEAI_API_KEY")),
			lchemb_voyage.WithModel(cfg.EmbedModel),
		)
	case "bedrock":
		return lchemb_bedrock.NewBedrock(
			lchemb_bedrock.WithModel(cfg.EmbedModel),
		)
	case "cybertron":
		client, err := lchemb_cybertron.NewCybertron(
			lchemb_cybertron.WithModel(cfg.EmbedModel),
		)
		if err != nil {
			return nil, fmt.Errorf("vectorstore: build cybertron client: %w", err)
		}
		return lcemb.NewEmbedder(client)
	default:
		return buildOpenAICompatEmbedder(cfg)
	}
}

func buildOpenAICompatEmbedder(cfg *domain.VectorStoreConfig) (lcemb.Embedder, error) {
	apiKey := os.Getenv(providerEnvKey(cfg.EmbedBackend))
	baseURL := cfg.EmbedBaseURL
	if baseURL == "" {
		baseURL = openAICompatBaseURL(cfg.EmbedBackend)
	}

	opts := []lcopenai.Option{
		lcopenai.WithToken(apiKey),
		lcopenai.WithModel(cfg.EmbedModel),
	}
	if baseURL != "" {
		opts = append(opts, lcopenai.WithBaseURL(baseURL))
	}

	client, err := lcopenai.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("build openai compat embedder: %w", err)
	}
	return lcemb.NewEmbedder(client)
}

func providerEnvKey(backend string) string {
	switch backend {
	case "openai":
		return "OPENAI_API_KEY"
	case "google":
		return "GOOGLE_API_KEY"
	case "groq":
		return "GROQ_API_KEY"
	case "mistral":
		return "MISTRAL_API_KEY"
	case "deepseek":
		return "DEEPSEEK_API_KEY"
	case "openrouter":
		return "OPENROUTER_API_KEY"
	case "together":
		return "TOGETHERAI_API_KEY"
	case "cohere":
		return "COHERE_API_KEY"
	case "xai":
		return "XAI_API_KEY"
	case "perplexity":
		return "PERPLEXITY_API_KEY"
	default:
		return ""
	}
}

func openAICompatBaseURL(backend string) string {
	urls := map[string]string{
		"openai":     "https://api.openai.com/v1",
		"ollama":     "http://localhost:11434/v1",
		"groq":       "https://api.groq.com/openai/v1",
		"mistral":    "https://api.mistral.ai/v1",
		"deepseek":   "https://api.deepseek.com/v1",
		"openrouter": "https://openrouter.ai/api/v1",
		"together":   "https://api.together.xyz/v1",
		"cohere":     "https://api.cohere.com/compatibility/v1",
		"xai":        "https://api.x.ai/v1",
		"perplexity": "https://api.perplexity.ai",
	}
	if u, ok := urls[backend]; ok {
		return u
	}
	return ""
}
