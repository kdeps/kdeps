# Scraper Resource

The `scraper` executor is built into the `kdeps` binary — no installation required. It fetches a URL and returns the text content, with optional CSS selector filtering.

## Configuration

```yaml
run:
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
metadata:
  actionId: fetch
run:
  scraper:
    url: "{{ get('url') }}"

---
metadata:
  actionId: summarize
  requires: [fetch]
run:
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
metadata:
  actionId: fetchArticle
run:
  scraper:
    url: "https://news.example.com/article"
    selector: "article.body"
```

</div>

## Error Handling

Use `onError` to handle unreachable URLs gracefully:

```yaml
run:
  scraper:
    url: "https://example.com"
  onError:
    action: continue
    fallback: ""
```

---

> **Need more?** For PDF extraction, OCR, and document types (.docx, .xlsx), install the component:
> ```bash
> kdeps component install scraper
> ```

## Next Steps

- [Embedding Resource](embedding) - Store and search scraped content
- [LLM Resource](llm) - Feed scraped content into prompts
- [HTTP Client](http-client) - Lower-level HTTP with auth, headers, caching
