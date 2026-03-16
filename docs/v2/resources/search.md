# Search Resource

The search resource discovers content — either from the web via a provider API or from the
local filesystem — and returns a list of results (title, URL, snippet) for downstream
processing, typically by a [`scraper`](./scraper.md) or [`llm`](./llm.md) resource.

All string fields support [KDeps expressions](../concepts/expressions) such as
<span v-pre>`{{get(...)}}` and `{{env(...)}}`</span>.

---

## Quick Reference

| Provider | Auth | Description |
|---|---|---|
| `brave` | `BRAVE_API_KEY` or `apiKey` | Brave Search API (recommended) |
| `serpapi` | `SERPAPI_KEY` or `apiKey` | SerpAPI (Google, Bing, and more) |
| `duckduckgo` | none | DuckDuckGo HTML scrape (no key required) |
| `tavily` | `TAVILY_API_KEY` or `apiKey` | Tavily AI-optimised search |
| `local` | none | Walk the local filesystem and match files |

---

## Basic Usage

### Web search (Brave)

```yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: findArticles
  name: Find Articles

run:
  search:
    provider: brave
    query: "{{get('topic')}}"
    apiKey: "{{env('BRAVE_API_KEY')}}"
    limit: 5
```

### Local filesystem search

```yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: findDocs
  name: Find Documentation Files

run:
  search:
    provider: local
    glob: "**/*.md"
    query: "getting started"
    limit: 20
```

---

## Configuration Fields

| Field | Type | Default | Description |
|---|---|---|---|
| `provider` | string | — | **Required.** One of: `brave`, `serpapi`, `duckduckgo`, `tavily`, `local`. |
| `query` | string | `""` | Search query. Required for all web providers. Used as content filter for `local`. |
| `apiKey` | string | `""` | API key for the provider. Falls back to the provider-specific env var (see below). |
| `limit` | int | `10` | Maximum number of results to return. |
| `safeSearch` | bool | `false` | Enable safe-search filtering (Brave, Tavily). |
| `region` | string | `""` | Search region/locale hint (Brave: e.g. `"us"`, SerpAPI: `"gl"` param). |
| `timeout` | string | `"30s"` | HTTP request timeout as a Go duration string, e.g. `"15s"`, `"1m"`. |
| `path` | string | FSRoot | **Local only.** Root directory to walk. Defaults to the agent's FSRoot. |
| `glob` | string | `""` | **Local only.** Filename glob pattern, e.g. `"*.go"`, `"**/*.md"`. |

---

## Provider Details

### Brave

Calls the [Brave Search API](https://brave.com/search/api/) and parses `web.results`.

```yaml
search:
  provider: brave
  query: "kdeps AI workflow"
  apiKey: "{{env('BRAVE_API_KEY')}}"   # or set BRAVE_API_KEY env var
  limit: 10
  safeSearch: false
  region: "us"
  timeout: "30s"
```

**Environment variable:** `BRAVE_API_KEY`

### SerpAPI

Calls the [SerpAPI](https://serpapi.com/) search endpoint, which supports Google, Bing,
and many other engines.

```yaml
search:
  provider: serpapi
  query: "machine learning tutorials"
  apiKey: "{{env('SERPAPI_KEY')}}"   # or set SERPAPI_KEY env var
  limit: 10
  region: "us"
```

**Environment variable:** `SERPAPI_KEY`

### DuckDuckGo

Scrapes DuckDuckGo HTML results — no API key required. Best-effort; results may vary.

```yaml
search:
  provider: duckduckgo
  query: "open source AI frameworks"
  limit: 10
  timeout: "30s"
```

### Tavily

Calls the [Tavily](https://tavily.com/) AI-optimised search API.

```yaml
search:
  provider: tavily
  query: "latest LLM papers"
  apiKey: "{{env('TAVILY_API_KEY')}}"   # or set TAVILY_API_KEY env var
  limit: 10
  safeSearch: false
```

**Environment variable:** `TAVILY_API_KEY`

### Local

Walks a directory tree, optionally filtering by filename glob and/or content match.
Results are returned as `file://`-prefixed URLs.

```yaml
search:
  provider: local
  path: "/data/knowledge-base"   # absolute, or relative to FSRoot
  glob: "**/*.md"                # filename filter (supports **)
  query: "authentication"        # optional content grep (case-insensitive)
  limit: 50
```

When `path` is omitted, the agent's `FSRoot` is used.  The `**` glob pattern matches
any number of nested directories.

---

## Result Map

All providers return the same envelope:

```json
{
  "query": "kdeps AI workflow",
  "provider": "brave",
  "results": [
    {
      "title": "Introduction to kdeps",
      "url": "https://kdeps.io/docs/intro",
      "snippet": "kdeps is an open-source AI workflow engine..."
    }
  ],
  "count": 1,
  "success": true
}
```

On error, an `"error"` key is added and `"success"` is `false`:

```json
{
  "query": "test",
  "provider": "brave",
  "results": [],
  "count": 0,
  "success": false,
  "error": "search executor: brave: HTTP 401: ..."
}
```

| Field | Type | Description |
|---|---|---|
| `query` | string | The evaluated query string that was used. |
| `provider` | string | The provider that was called. |
| `results` | []object | Array of result items (see below). |
| `count` | int | Number of results returned. |
| `success` | bool | `true` when the search completed without error. |
| `error` | string | Error message (only present when `success` is `false`). |

**Result item fields:**

| Field | Type | Description |
|---|---|---|
| `title` | string | Page title or filename. |
| `url` | string | Page URL or `file://` path (local). |
| `snippet` | string | Excerpt or description. |

---

## Expression Support

All string fields are evaluated through the KDeps expression engine before use:

```yaml
search:
  provider: "{{get('search_provider')}}"
  query: "{{get('user_query')}}"
  apiKey: "{{env('BRAVE_API_KEY')}}"
  limit: 10
  region: "{{get('user_region', 'us')}}"
  timeout: "{{get('search_timeout', '30s')}}"
```

---

## Search as an Inline Resource

Run a quick search **before** or **after** the primary resource action using the
`before` / `after` inline blocks.  This avoids creating a dedicated resource for a
one-off lookup:

```yaml
run:
  before:
    - search:
        provider: local
        glob: "**/*.txt"
        query: "{{get('topic')}}"
        limit: 5

  llm:
    prompt: |
      Here are relevant files I found: {{get('search.results')}}

      Answer the user's question: {{get('question')}}
```

---

## Full Example: Web Research Pipeline

This pipeline searches the web for a topic, scrapes the top result, and asks an LLM to
summarise it.

```yaml
# Step 1: Search for the topic
- apiVersion: kdeps.io/v1
  kind: Resource

  metadata:
    actionId: webSearch
    name: Web Search

  run:
    search:
      provider: brave
      query: "{{get('research_topic')}}"
      apiKey: "{{env('BRAVE_API_KEY')}}"
      limit: 3
      timeout: "30s"

# Step 2: Scrape the first result
- apiVersion: kdeps.io/v1
  kind: Resource

  metadata:
    actionId: scrapeTop
    name: Scrape Top Result
    requires:
      - webSearch

  run:
    scraper:
      url: "{{get('webSearch.results[0].url')}}"
      timeout: "30s"

# Step 3: LLM summarises the scraped content
- apiVersion: kdeps.io/v1
  kind: Resource

  metadata:
    actionId: summarise
    name: Summarise Research
    requires:
      - scrapeTop

  run:
    llm:
      prompt: |
        You are a research assistant. Summarise the following content about
        "{{get('research_topic')}}":

        {{get('scrapeTop.content')}}

        Provide a concise 3-paragraph summary.

# Step 4: Return the summary
- apiVersion: kdeps.io/v1
  kind: Resource

  metadata:
    actionId: researchResponse
    name: Research Response
    requires:
      - summarise

  run:
    apiResponse:
      success: true
      response:
        summary: "{{get('summarise')}}"
        source_url: "{{get('webSearch.results[0].url')}}"
        source_title: "{{get('webSearch.results[0].title')}}"
```

---

## Full Example: Local Documentation Search

This pipeline finds documentation files matching a query and generates a structured
answer using an LLM.

```yaml
# Step 1: Find relevant docs
- apiVersion: kdeps.io/v1
  kind: Resource

  metadata:
    actionId: findDocs
    name: Find Relevant Documentation

  run:
    search:
      provider: local
      path: "/data/docs"
      glob: "**/*.md"
      query: "{{get('question')}}"
      limit: 5

# Step 2: Ask the LLM
- apiVersion: kdeps.io/v1
  kind: Resource

  metadata:
    actionId: answerQuestion
    name: Answer Question
    requires:
      - findDocs

  run:
    llm:
      prompt: |
        You are a documentation assistant.
        The user asked: {{get('question')}}

        Relevant files found:
        {{get('findDocs.results')}}

        Answer based on these files. If you cannot find the answer, say so.

# Step 3: Return the answer
- apiVersion: kdeps.io/v1
  kind: Resource

  metadata:
    actionId: docAnswer
    name: Documentation Answer
    requires:
      - answerQuestion

  run:
    apiResponse:
      success: true
      response:
        answer: "{{get('answerQuestion')}}"
        sources_count: "{{get('findDocs.count')}}"
```
