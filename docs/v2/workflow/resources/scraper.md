# Scraper component

The `scraper` component fetches a URL and returns its text - the whole body, or only the elements matching a CSS selector. It is a kdeps component from the `components` asset set: call it with a `component:` resource, no install step. The fetch is the kdeps API's `http_request` tool; the component only shapes the result.

```text
component: scraper -> POST /apis/tools.kdeps.io/v1/tools/http_request/call
                   -> http_request GET url -> htmlText(body, selector) -> {content, url, status}
```

## Where it runs

Both [workflow mode](/workflow/) and [agent mode](/agent/). In agent mode, the workflow containing it runs as a single callable tool; the agent's own `web_scraper` [built-in tool](/agent/tools-web) is separate.

## Configuration

<div v-pre>

```yaml
# resources/fetch.yaml
actionId: fetch
name: Fetch Page
component:
  name: scraper                  # from the components asset set; a components/scraper/ of your own wins
  with:
    url: "https://example.com"   # required, fetched with GET
    selector: "article.content"  # optional CSS selector; empty = the whole body
    timeout: 30s                 # duration (default 30s)
```

</div>

| Input | Required | Default | Description |
|-------|----------|---------|-------------|
| `url` | yes | - | URL to fetch |
| `selector` | no | `""` | CSS selector; `content` becomes the text of each match, one per line |
| `timeout` | no | `30s` | Request timeout as a duration |

## Output

| Key | Type | Description |
|-----|------|-------------|
| `content` | string | The body, or the selector's matched text |
| `url` | string | The URL that was fetched |
| `status` | integer | HTTP status code - a 404 or 500 is reported here, not raised |

```json
{"content": "Integration Test", "url": "https://example.com", "status": 200}
```

## Examples

### Fetch a page and summarize

<div v-pre>

```yaml
# resources/fetch.yaml
actionId: fetch
name: Fetch
component:
  name: scraper
  with:
    url: "{{ get('url') }}"
```

```yaml
# resources/summarize.yaml
actionId: summarize
name: Summarize
requires: [fetch]
chat:
  model: llama3.2:1b
  prompt: "Summarize: {{ output('fetch').content }}"
```

</div>

### Extract with a CSS selector

```yaml
# resources/fetch-article.yaml
actionId: fetchArticle
name: Fetch Article
component:
  name: scraper
  with:
    url: "https://news.example.com/article"
    selector: "article.body"
```

## How it is built

The component is one `httpClient` call to the `http_request` tool plus an `interface.output` block that shapes the result with the [`htmlText()`](/reference/expression-functions) expression function. The tool returns `{statusCode, body, headers}`; `fromJSON` reads it back. Copy the component into `components/scraper/` of your project to change it (add headers, follow a link, keep the raw HTML):

<div v-pre>

```yaml
# components/scraper/component.yaml (as shipped, trimmed)
interface:
  inputs:
    - {name: url, type: string, required: true}
    - {name: selector, type: string, default: ""}
    - {name: timeout, type: string, default: 30s}
  output:                                         # the component's result
    content: "{{ htmlText(fromJSON(output('call').body).body, get('scraper.selector')) }}"
    url: "{{ get('scraper.url') }}"
    status: "{{ fromJSON(output('call').body).statusCode }}"
resources:
  - actionId: call
    name: Fetch page
    httpClient:
      method: POST
      url: http://kdeps.api/apis/tools.kdeps.io/v1/tools/http_request/call
      headers: {Content-Type: application/json}
      failOnError: true
      timeout: 60s
      data:
        url: "{{ get('scraper.url') }}"
        method: GET
        timeout: "{{ get('scraper.timeout') }}"
```

</div>

## Error handling

An unreachable host fails the resource. Use `onError` to continue with a fallback:

```yaml
# resources/example.yaml
actionId: example
name: Example
component:
  name: scraper
  with:
    url: "https://example.com"
onError:
  action: continue
  fallback: ""
```

## See also

- [Embedding component](/workflow/resources/embedding) - store and search scraped content
- [LLM resource](/workflow/resources/llm) - feed scraped content into prompts
- [HTTP client](/workflow/resources/http-client) - lower-level HTTP with auth, headers, caching
- [Browser resource](/workflow/resources/browser) - pages that need JavaScript
