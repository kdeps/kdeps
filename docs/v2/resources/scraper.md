# Scraper Resource

The `scraper` executor is a native capability compiled into the `kdeps` binary. It fetches a URL and returns the text content, with optional CSS selector filtering.

## Where it runs

Both [workflow mode](/modes/workflow-mode) and [agent mode](/modes/agent-mode). In workflow mode it executes as a DAG step. In agent mode, the workflow containing this resource runs as a single callable tool.

## Configuration

```yaml
# resources/fetch.yaml
scraper:
  url: "https://example.com"     # required
  selector: "article.content"    # optional CSS selector
  timeout: 30                    # seconds (default: 30)
```

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `url` | string | yes | — | URL to fetch |
| `selector` | string | no | — | CSS selector to scope extraction |
| `timeout` | integer | no | `30` | Request timeout in seconds |

## Output

| Key | Type | Description |
|-----|------|-------------|
| `content` | string | Extracted text (full body or selector match) |
| `url` | string | The URL that was fetched |
| `status` | integer | HTTP status code |
| `json` | string | Full result as a JSON string |

Access fields with `output('actionId').content` etc.

## Examples

### Fetch a page and summarize

<div v-pre>

```yaml
# resources/fetch.yaml
actionId: fetch
scraper:
  url: "{{ get('url') }}"

---
actionId: summarize
requires: [fetch]
chat:
  model: llama3.2:1b
  prompt: "Summarize: {{ output('fetch').content }}"
apiResponse:
  response: "{{ output('summarize') }}"
```

</div>

### Extract with CSS selector

<div v-pre>

```yaml
# resources/fetch-article.yaml
actionId: fetchArticle
scraper:
  url: "https://news.example.com/article"
  selector: "article.body"
```

</div>

## Error Handling

Use `onError` to handle unreachable URLs gracefully:

```yaml
# resources/example.yaml
scraper:
  url: "https://example.com"
onError:
  action: continue
  fallback: ""
```

---

> **Need more?** For PDF extraction, OCR, and document types (.docx, .xlsx), install the component:
> ```bash
> kdeps registry install scraper
> ```

## See Also

- [Embedding Resource](embedding) -- store and search scraped content
- [LLM Resource](llm) -- feed scraped content into prompts
- [HTTP Client](http-client) -- lower-level HTTP with auth, headers, caching
