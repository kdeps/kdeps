# searchWeb component

The `searchWeb` component queries the web and returns `{results, count, query, provider}`, each result a `{title, url, snippet}`. DuckDuckGo is the default and needs no key; Brave, Bing and Tavily read their API key from `~/.kdeps/config.yaml`. It is a kdeps component from the `components` asset set: call it with a `component:` resource, no install step. Its provider calls are the kdeps API's `http_request` tool. See [Search resources](/workflow/resources/search) for how it relates to `search_local`.

It works in both modes: as a DAG step in workflow mode, and inside any workflow the LLM calls as a tool in agent mode.

```d2
direction: right
call: "component: searchWeb" {shape: oval}
p: "provider?" {shape: diamond}
ddg: "POST /tools/http_request/call\n-> GET html.duckduckgo.com"
api: "POST /tools/http_request/call\n-> Brave / Bing / Tavily\n(key from search_connections)"
norm: "normalize:\nhtmlFind() or the JSON fields"
out: "{results, count, query, provider}" {shape: oval}
call -> p
p -> ddg: ddg
p -> api: brave | bing | tavily
ddg -> norm
api -> norm
norm -> out
```

## Named connections (paid providers)

API keys belong in `~/.kdeps/config.yaml`, not in resource files or `workflow.yaml`:

```yaml
# ~/.kdeps/config.yaml
search_connections:
  brave:
    apiKey: "${BRAVE_API_KEY}"
  bing:
    apiKey: "${BING_API_KEY}"
  tavily:
    apiKey: "${TAVILY_API_KEY}"
```

`kdeps run` asks for a connection that a `connectionName:` names but `config.yaml` does not have (the input is marked `connection: search` in the component).

## Configuration

<div v-pre>

```yaml
# resources/search.yaml
actionId: search
name: Web Search
component:
  name: searchWeb                  # from the components asset set; a components/searchWeb/ of your own wins
  with:
    query: "{{ get('query') }}"    # required
    provider: ddg                  # ddg (default) | brave | bing | tavily
    connectionName: brave          # brave/bing/tavily: search_connections.<name>.apiKey
    maxResults: 5                  # default 5
    timeout: 15s                   # duration, default 15s
```

</div>

| Input | Required | Default | Description |
|-------|----------|---------|-------------|
| `query` | yes | - | Search query |
| `provider` | no | `ddg` | `ddg`, `brave`, `bing`, `tavily` |
| `connectionName` | brave/bing/tavily | `""` | Name under `search_connections` in `~/.kdeps/config.yaml` |
| `maxResults` | no | `5` | Results to return at most |
| `timeout` | no | `15s` | Request timeout as a duration |

## Output

```json
{
  "results": [{"title": "The Go Programming Language", "url": "https://go.dev", "snippet": "..."}],
  "count": 1,
  "query": "golang",
  "provider": "ddg"
}
```

DuckDuckGo results have an empty `snippet`. Read them with `output('search').results`.

## Examples

**Feed results into an LLM:**

<div v-pre>

```yaml
# resources/search.yaml
actionId: search
name: Search
component:
  name: searchWeb
  with:
    query: "{{ get('query') }}"
```

```yaml
# resources/answer.yaml
actionId: answer
name: Answer
requires: [search]
chat:
  model: llama3.2:1b
  prompt: |
    Answer based on these results:
    {{ json(output('search').results) }}
    Question: {{ get('query') }}
```

</div>

## Errors

| Error | Cause |
|-------|-------|
| `searchWeb: connectionName required for brave` (bing, tavily) | No key at `search_connections.<connectionName>.apiKey` |
| `searchWeb: unknown provider` | `provider` is not ddg, brave, bing or tavily |
| `HTTP 4xx/5xx` | The provider rejected the request (bad key, rate limit) |

## Environment variable overrides

| Variable | Description |
|----------|-------------|
| `KDEPS_DDG_URL` | DuckDuckGo base URL (default `https://html.duckduckgo.com`) |
| `KDEPS_BRAVE_URL` | Brave API base URL |
| `KDEPS_BING_URL` | Bing API base URL |
| `KDEPS_TAVILY_URL` | Tavily API base URL |

## How it is built

One `httpClient` POST to the kdeps API's `http_request` tool per provider, each skipped unless its provider is selected (`validations.skip`) and checked for a key (`validations.check`), then a step that maps the response to `{title, url, snippet}` - DuckDuckGo's HTML with the [`htmlFind()`](/reference/expression-functions) expression, the APIs' JSON with `map()`. `http_request` returns the response as `{statusCode, body, headers}`, so each mapping reads `fromJSON(output('<provider>').body)`. Copy `components/searchWeb/` into your project to add a provider or change the mapping.

## See also

- [Search resources](/workflow/resources/search) - overview and `search_local`
- [Scraper component](/workflow/resources/scraper) - fetch the pages a search found
- [LLM resource](/workflow/resources/llm) - use search results as context
