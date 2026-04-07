# scraper

> Web scraper using Python requests and BeautifulSoup.

The scraper component fetches a URL over HTTP and extracts text content from the page. An optional CSS selector narrows extraction to a specific element. Use it to pull data from public web pages, APIs that return HTML, or internal tools with web UIs.

## Inputs

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `url` | string | yes | — | URL to scrape |
| `selector` | string | no | `""` | CSS selector to extract (omit to return full page text) |
| `timeout` | integer | no | `30` | HTTP request timeout in seconds |

## Outputs

Accessed via `output('<your-action-id>')`:

| Field | Type | Description |
|-------|------|-------------|
| `result` | string | Extracted text — either the full page text or the text matched by the CSS selector |

## Usage

Scrape a full page:

```yaml
run:
  component:
    name: scraper
    with:
      url: "https://news.ycombinator.com"
  apiResponse:
    success: true
    response:
      pageText: "{{ output('scrapeHN').result }}"
```

Extract specific elements with a CSS selector:

```yaml
run:
  component:
    name: scraper
    with:
      url: "https://news.ycombinator.com"
      selector: ".titleline > a"
  apiResponse:
    success: true
    response:
      headlines: "{{ output('scrapeHeadlines').result }}"
```

Scrape with a custom timeout:

```yaml
run:
  component:
    name: scraper
    with:
      url: "https://slow-site.example.com/data"
      timeout: 60
```

Pipe scraped content into an LLM:

```yaml
- actionId: scrapeArticle
  run:
    component:
      name: scraper
      with:
        url: "{{ input('articleUrl') }}"
        selector: "article"

- actionId: summarise
  run:
    chat:
      prompt: "Summarise this article in 3 bullet points:\n{{ output('scrapeArticle').result }}"
```

## Requirements

- Python packages `requests` and `beautifulsoup4` — declared in this component's `pythonPackages` and automatically merged into the workflow environment at load time. No manual `agentSettings.pythonPackages` declaration needed.
- Network access from the workflow runtime to the target URL.

## Notes

- When `selector` is provided and matches multiple elements, their text is joined with newlines.
- When `selector` matches nothing, the result is an empty string.
- Without a `selector`, `get_text(separator="\n", strip=True)` is used — HTML tags are stripped and whitespace is normalised.
- HTTP errors (4xx, 5xx) raise an exception and cause the resource to fail.
- Does not support JavaScript-rendered content. Use the `browser` component for pages that require JS execution.
- For authenticated endpoints, use `run.httpClient:` directly with custom headers instead.
