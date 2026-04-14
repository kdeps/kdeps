# Search Web

The `searchWeb` executor is a native capability compiled into the `kdeps` binary.

It queries the web and returns structured results. The default provider is **DuckDuckGo** (no API key required).

## Configuration

```yaml
run:
  searchWeb:
    query: "{{ get('query') }}"     # required
    provider: ddg                    # optional: ddg (default) | brave | bing | tavily
    apiKey: "{{ get('apiKey') }}"   # required for brave/bing/tavily
    maxResults: 5                    # optional, default 5
    timeout: 15                      # optional, seconds, default 15
```

### Config fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `query` | string | Yes | -- | Search query |
| `provider` | string | No | `ddg` | Search provider |
| `apiKey` | string | No* | -- | API key (required for non-DDG providers) |
| `maxResults` | int | No | `5` | Maximum number of results to return |
| `timeout` | int | No | `15` | HTTP request timeout in seconds |

*Required when `provider` is `brave`, `bing`, or `tavily`.

## Providers

| Provider | Value | API Key | Endpoint |
|----------|-------|---------|----------|
| DuckDuckGo | `ddg` | Not required | HTML scraping |
| Brave Search | `brave` | Required | REST API |
| Bing | `bing` | Required | REST API |
| Tavily | `tavily` | Required | REST API |

## Output

```yaml
output('search').results    # list of result objects
output('search').count      # number of results returned
output('search').query      # the query string
output('search').provider   # provider used
output('search').json       # JSON string of the full result
```

### Result object fields

| Field | Type | Description |
|-------|------|-------------|
| `title` | string | Page title |
| `url` | string | Page URL |
| `snippet` | string | Description or summary snippet |

## Examples

### DuckDuckGo (no API key)

```yaml
metadata:
  actionId: search
run:
  searchWeb:
    query: "{{ get('query') }}"
    maxResults: 5
```

### Brave Search

```yaml
metadata:
  actionId: search
run:
  searchWeb:
    query: "{{ get('query') }}"
    provider: brave
    apiKey: "{{ env('BRAVE_API_KEY') }}"
    maxResults: 10
```

### Feed results into an LLM

```yaml
metadata:
  actionId: search
run:
  searchWeb:
    query: "{{ get('query') }}"

---
metadata:
  actionId: answer
  requires: [search]
run:
  chat:
    model: llama3.2:1b
    prompt: |
      Answer based on these results:
      {% for r in output('search').results %}
      - {{ r.title }}: {{ r.snippet }}
      {% endfor %}
      Question: {{ get('query') }}
```

## Error handling

| Error | Cause |
|-------|-------|
| `query is required` | Empty query string |
| `apiKey required for brave provider` | Missing API key for Brave |
| `apiKey required for bing provider` | Missing API key for Bing |
| `apiKey required for tavily provider` | Missing API key for Tavily |
| `unknown provider "x"` | Invalid provider value |

## Environment variable overrides (for testing)

| Variable | Description |
|----------|-------------|
| `KDEPS_DDG_URL` | Override DuckDuckGo base URL |
| `KDEPS_BRAVE_URL` | Override Brave API base URL |
| `KDEPS_BING_URL` | Override Bing API base URL |
| `KDEPS_TAVILY_URL` | Override Tavily API base URL |
