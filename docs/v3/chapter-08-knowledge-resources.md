# Chapter 8: Knowledge Resources

Knowledge resources connect your workflows to unstructured information: web pages, local files, search results, and vector stores. kdeps bundles three knowledge executors directly into the binary — no external services required for basic use.

## Scraper Resource

The `scraper:` resource fetches a URL and returns its text content. It strips HTML to plain text, optionally filtered by a CSS selector.

### Basic Usage

```yaml
# resources/fetch-page.yaml
actionId: fetchPage
scraper:
  url: "&#123;&#123; get('url') &#125;&#125;"
  timeout: 30
```

After execution, `get('fetchPage')` contains the page's text content.

### CSS Selector Filtering

When you only need part of a page, use `selector:` to limit extraction to a specific element:

```yaml
# resources/fetch-article.yaml
actionId: fetchArticle
scraper:
  url: "&#123;&#123; get('articleUrl') &#125;&#125;"
  selector: "article.content"    # extracts only the article body
  timeout: 30
```

This is important for noisy pages where the navigation, ads, and footers would otherwise pollute your LLM prompt. A CSS selector like `main`, `article`, `.post-content`, or `#main-content` extracts only the relevant content.

### Full Configuration

```yaml
scraper:
  url: "https://example.com"
  selector: ".content"      # optional CSS selector
  timeout: 30               # seconds; default 30
```

| Field | Required | Default | Description |
|---|---|---|---|
| `url` | yes | — | URL to fetch |
| `selector` | no | none | CSS selector to filter content |
| `timeout` | no | 30 | Timeout in seconds |

### Common Use Cases

**Ingest a documentation page:**
```yaml
scraper:
  url: "https://docs.example.com/api-reference"
  selector: ".docs-content"
```

**Extract a news article:**
```yaml
scraper:
  url: "&#123;&#123; get('articleUrl') &#125;&#125;"
  selector: "article"
```

**Fetch a product page:**
```yaml
scraper:
  url: "&#123;&#123; get('productUrl') &#125;&#125;"
  selector: ".product-description"
```

### Combining with LLM

The classic pattern: fetch content, then ask the LLM about it:

```yaml
# resources/fetch.yaml
actionId: fetch
scraper:
  url: "&#123;&#123; get('url') &#125;&#125;"
  selector: "article"

# resources/qa.yaml
actionId: qa
requires: [fetch]
chat:
  model: llama3.2:1b
  prompt: |
    Based on the following content, answer this question:
    Question: &#123;&#123; get('question') &#125;&#125;
    
    Content:
    &#123;&#123; get('fetch') &#125;&#125;
```

## Search Resources

kdeps provides two search executors: `searchWeb:` for internet search and `searchLocal:` for local file search.

### Web Search

```yaml
# resources/web-search.yaml
actionId: webSearch
searchWeb:
  query: "&#123;&#123; get('topic') &#125;&#125;"
  maxResults: 5
```

`searchWeb:` queries a search index and returns a list of results, each with `title`, `url`, and `snippet`. No external search API key is required by default — kdeps uses a built-in search capability.

The output format:

```json
[
  {"title": "...", "url": "https://...", "snippet": "..."},
  {"title": "...", "url": "https://...", "snippet": "..."}
]
```

Access results in downstream resources:

```yaml
after:
  - set('first_url', get('webSearch')[0].url)
  - set('urls', map(get('webSearch'), {.url}))
```

### Web Search + Scrape + LLM Pattern

A complete research pipeline:

```yaml
# resources/search.yaml
actionId: search
searchWeb:
  query: "&#123;&#123; get('q') &#125;&#125;"
  maxResults: 3

# resources/scrape-first.yaml
actionId: scrapeFirst
requires: [search]
scraper:
  url: "&#123;&#123; get('search')[0].url &#125;&#125;"
  timeout: 30

# resources/answer.yaml
actionId: answer
requires: [search, scrapeFirst]
chat:
  model: llama3.2:1b
  prompt: |
    Search results:
    {% for r in get('search') %}
    - &#123;&#123; r.title &#125;&#125;: &#123;&#123; r.snippet &#125;&#125;
    {% endfor %}
    
    Full content of first result:
    &#123;&#123; get('scrapeFirst') &#125;&#125;
    
    Question: &#123;&#123; get('q') &#125;&#125;
    Answer based on the above information:
```

### Local File Search

`searchLocal:` walks a directory and returns matching files by filename glob and/or content keyword:

```yaml
# resources/find-docs.yaml
actionId: findDocs
searchLocal:
  path: "/data/documents"
  pattern: "*.txt"           # filename glob
  keyword: "&#123;&#123; get('term') &#125;&#125;"  # content keyword (optional)
  maxResults: 10
```

Output format:

```json
[
  {
    "path": "/data/documents/report-2024.txt",
    "filename": "report-2024.txt",
    "size": 4096,
    "matches": ["line 12: ...term found here...", "line 47: ...term found here..."]
  }
]
```

Useful for workflows that process a local document corpus: legal document review, codebase analysis, file organization.

## Embedding Resource

The `embedding:` resource provides a SQLite-backed keyword store for indexing, searching, upserting, and deleting text documents. It is the storage layer for RAG (retrieval-augmented generation) pipelines that run fully on-premise.

The embedding executor is compiled into the kdeps binary. It requires no external vector database, no embedding model endpoint, no additional infrastructure. It works offline.

### Operations

The embedding resource has four operations: `index`, `search`, `upsert`, and `delete`.

#### Indexing Documents

```yaml
# resources/index.yaml
actionId: indexDoc
embedding:
  operation: index
  text: "&#123;&#123; get('documentContent') &#125;&#125;"
  collection: "docs"           # namespace; defaults to "default"
  dbPath: "/data/store.db"    # SQLite file path
```

Each indexed document is stored in the named collection. Multiple collections can exist in the same database file.

#### Searching

```yaml
# resources/retrieve.yaml
actionId: retrieve
embedding:
  operation: search
  text: "&#123;&#123; get('query') &#125;&#125;"   # search query
  collection: "docs"
  dbPath: "/data/store.db"
  limit: 5                     # max results; default 10
```

Search results are returned as an array of matching documents with relevance scores:

```json
[
  {"text": "matching document content...", "score": 0.87, "collection": "docs"},
  {"text": "another matching document...", "score": 0.72, "collection": "docs"}
]
```

#### Upsert

Update an existing document or insert if it does not exist:

```yaml
embedding:
  operation: upsert
  text: "&#123;&#123; get('updatedContent') &#125;&#125;"
  collection: "docs"
  dbPath: "/data/store.db"
```

#### Delete

```yaml
# Delete a specific document
embedding:
  operation: delete
  text: "&#123;&#123; get('documentToDelete') &#125;&#125;"
  collection: "docs"
  dbPath: "/data/store.db"

# Delete all documents in a collection
embedding:
  operation: delete
  collection: "docs"
  dbPath: "/data/store.db"
```

### Building a RAG Pipeline

Retrieval-augmented generation gives an LLM access to a knowledge base without requiring the full knowledge base to fit in the context window. The pattern:

1. Index documents at ingestion time
2. At query time, search for relevant documents
3. Pass the retrieved documents as context to the LLM

**Indexing workflow** (run once to populate the knowledge base):

```yaml
# workflow.yaml for indexer
metadata:
  name: doc-indexer
  targetActionId: confirm
settings:
  apiServer:
    routes:
      - path: /api/v1/index
        methods: [POST]
```

```yaml
# resources/index.yaml
actionId: indexDoc
validations:
  check:
    - get('content') != ''
    - get('id') != ''
  error:
    code: 400
    message: "content and id are required"
embedding:
  operation: upsert
  text: "&#123;&#123; get('content') &#125;&#125;"
  collection: "knowledge-base"
  dbPath: "/data/kb.db"

# resources/confirm.yaml
actionId: confirm
requires: [indexDoc]
apiResponse:
  success: true
  response:
    indexed: true
    collection: "knowledge-base"
```

**Query workflow** (run on each user question):

```yaml
# resources/retrieve.yaml
actionId: retrieve
requires: [validate]
embedding:
  operation: search
  text: "&#123;&#123; get('question') &#125;&#125;"
  collection: "knowledge-base"
  dbPath: "/data/kb.db"
  limit: 3

# resources/answer.yaml
actionId: answer
requires: [retrieve]
chat:
  model: llama3.2:1b
  systemPrompt: "Answer only from the provided context. If the answer is not in the context, say so."
  prompt: |
    Context:
    {% for doc in get('retrieve') %}
    ---
    &#123;&#123; doc.text &#125;&#125;
    {% endfor %}
    
    Question: &#123;&#123; get('question') &#125;&#125;

# resources/respond.yaml
actionId: respond
requires: [answer]
apiResponse:
  success: true
  response:
    answer: get('answer')
    sources_used: len(get('retrieve'))
```

This pipeline runs entirely on-premise. No external vector database. No embedding API. No cloud dependency. The SQLite store can handle tens of thousands of documents efficiently for most production use cases.

### When to Use Each Knowledge Resource

| Resource | Use when |
|---|---|
| `scraper:` | You have a specific URL and need its text content |
| `searchWeb:` | You need to find relevant URLs for a topic at query time |
| `searchLocal:` | You need to find relevant files in a local directory |
| `embedding:` | You have a large, queryable knowledge base that lives on your infrastructure |

For production RAG systems where you need semantic similarity rather than keyword matching, consider adding a dedicated embedding model and vector database (Chroma, Qdrant, Weaviate) via the `httpClient:` resource. The built-in `embedding:` resource is keyword-based and sufficient for most teams starting out.

In the next chapter, we cover browser automation — for when you need to interact with dynamic web applications, not just fetch static pages.

X> ## Exercise
X>
X> Build a simple research assistant that answers questions about a topic by first scraping a relevant web page and then using the scraped content as context for the LLM.
X>
X> 1. Create a `scraper:` resource that fetches a Wikipedia article URL provided by the user in the request body.
X> 2. Pass the scraped text to a `chat:` resource with a prompt like: `"Based only on this text, answer the question '&#123;&#123; get('q') &#125;&#125;':\n\n&#123;&#123; get('page') &#125;&#125;"`.
X> 3. Return the answer and the source URL in the response.
X>
X> Test it:
X> ```bash
X> curl -X POST localhost:16395/api/v1/research \
X>   -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN" \
X>   -H "Content-Type: application/json" \
X>   -d '{"url":"https://en.wikipedia.org/wiki/Photosynthesis","q":"What is the role of chlorophyll?"}'
X> ```
X>
X> Add a validation that rejects non-Wikipedia URLs (`validations.check: get('url') startsWith 'https://en.wikipedia.org/'`). Verify the rejection works.
X>
X> **Stretch goal:** Add an `embedding:` resource that stores the scraped text in a local vector store so subsequent questions about the same URL skip the scraper and query the stored embeddings instead.
