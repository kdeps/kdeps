# Search Resources

kdeps provides two native search executors compiled into the binary: `searchLocal` for local file search and `searchWeb` for web search.

## Where it runs

Both [workflow mode](/modes/workflow-mode) and [agent mode](/modes/agent-mode). In workflow mode each executor runs as a DAG step. In agent mode both are auto-registered as callable tools.

---

## searchLocal

The `searchLocal` executor walks a local directory and returns matching files by filename glob pattern and/or content keyword.

### Configuration

```yaml
searchLocal:
  path: "/data/documents"    # required: directory to search
  query: "invoice total"     # optional: keyword in file contents
  glob: "*.txt"              # optional: filename pattern
  limit: 10                  # optional: max results (0 = unlimited)
```

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `path` | string | yes | - | Directory to search recursively |
| `query` | string | no | - | Case-insensitive keyword to find in file contents |
| `glob` | string | no | - | Filename glob pattern (e.g. `*.md`, `report_*.csv`) |
| `limit` | integer | no | `0` | Max results (0 = unlimited) |

When both `query` and `glob` are set, a file must match **both** to be included.

### Output

| Key | Type | Description |
|-----|------|-------------|
| `results` | array | List of matching file objects |
| `count` | integer | Number of results |
| `path` | string | The search root used |
| `json` | string | Full result as JSON string |

Each result object:

| Key | Type | Description |
|-----|------|-------------|
| `path` | string | Full file path |
| `name` | string | Filename |
| `size` | integer | File size in bytes |
| `isDir` | bool | Always `false` |

### Examples

**Find all Markdown files:**

<div v-pre>

```yaml
actionId: findDocs
searchLocal:
  path: "/workspace/docs"
  glob: "*.md"
```

</div>

**Find files containing a keyword:**

<div v-pre>

```yaml
actionId: findInvoices
searchLocal:
  path: "/data/uploads"
  query: "overdue"
  limit: 20
```

</div>

**Feed results into an LLM:**

<div v-pre>

```yaml
actionId: findFiles
searchLocal:
  path: "/data/reports"
  query: "{{ get('query') }}"

---
actionId: answer
requires: [findFiles]
chat:
  model: llama3.2:1b
  prompt: "Files found: {{ output('findFiles').results }}. Summarize."
```

</div>

---

## searchWeb

The `searchWeb` executor queries the web and returns structured results. The default provider is DuckDuckGo - no API key required.

### Configuration

```yaml
searchWeb:
  query: "{{ get('query') }}"     # required
  provider: ddg                    # optional: ddg (default) | brave | bing | tavily
  apiKey: "{{ get('apiKey') }}"   # required for brave/bing/tavily
  maxResults: 5                    # optional, default 5
  timeout: 15                      # optional, seconds, default 15
```

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `query` | string | yes | - | Search query |
| `provider` | string | no | `ddg` | Search provider |
| `apiKey` | string | no* | - | API key (required for non-DDG providers) |
| `maxResults` | integer | no | `5` | Maximum number of results |
| `timeout` | integer | no | `15` | HTTP request timeout in seconds |

*Required when `provider` is `brave`, `bing`, or `tavily`.

### Providers

| Provider | Value | API Key |
|----------|-------|---------|
| DuckDuckGo | `ddg` | Not required |
| Brave Search | `brave` | Required |
| Bing | `bing` | Required |
| Tavily | `tavily` | Required |

### Output

```yaml
output('search').results    # list of result objects
output('search').count      # number of results returned
output('search').query      # the query string
output('search').provider   # provider used
output('search').json       # JSON string of the full result
```

Result object fields:

| Field | Type | Description |
|-------|------|-------------|
| `title` | string | Page title |
| `url` | string | Page URL |
| `snippet` | string | Description or summary snippet |

### Examples

**DuckDuckGo (no API key):**

<div v-pre>

```yaml
actionId: search
searchWeb:
  query: "{{ get('query') }}"
  maxResults: 5
```

</div>

**Brave Search:**

<div v-pre>

```yaml
actionId: search
searchWeb:
  query: "{{ get('query') }}"
  provider: brave
  apiKey: "{{ env('BRAVE_API_KEY') }}"
  maxResults: 10
```

</div>

**Feed results into an LLM:**

<div v-pre>

```yaml
actionId: search
searchWeb:
  query: "{{ get('query') }}"

---
actionId: answer
requires: [search]
chat:
  model: llama3.2:1b
  prompt: |
    Answer based on these results:
    {% for r in output('search').results %}
    - {{ r.title }}: {{ r.snippet }}
    {% endfor %}
    Question: {{ get('query') }}
```

</div>

### Error handling

| Error | Cause |
|-------|-------|
| `query is required` | Empty query string |
| `apiKey required for brave provider` | Missing API key for Brave |
| `apiKey required for bing provider` | Missing API key for Bing |
| `apiKey required for tavily provider` | Missing API key for Tavily |
| `unknown provider "x"` | Invalid provider value |

### Environment variable overrides

| Variable | Description |
|----------|-------------|
| `KDEPS_DDG_URL` | Override DuckDuckGo base URL |
| `KDEPS_BRAVE_URL` | Override Brave API base URL |
| `KDEPS_BING_URL` | Override Bing API base URL |
| `KDEPS_TAVILY_URL` | Override Tavily API base URL |

---

## See Also

- [Embedding Resource](embedding) - SQLite keyword store for on-prem RAG
- [Scraper Resource](scraper) - Fetch URL content to feed into search pipelines
- [LLM Resource](llm) - Use search results as context for chat resources
