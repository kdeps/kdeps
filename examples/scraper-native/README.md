# scraper-native

Web scraping + AI summarization with the **scraper component** (`component: scraper`) - an `httpClient` GET plus `htmlText()` CSS-selector extraction, no Python dependencies.

## Usage

```bash
kdeps run examples/scraper-native/workflow.yaml --dev
```

Scrape and summarize a URL:
```bash
curl -X POST http://localhost:16402/summarize \
  -H "Content-Type: application/json" \
  -d '{"url": "https://go.dev/doc/"}'
```

With CSS selector to extract specific content:
```bash
curl -X POST http://localhost:16402/summarize \
  -H "Content-Type: application/json" \
  -d '{"url": "https://go.dev/doc/", "selector": "p"}'
```

## How it works

1. **fetch** -- `component: scraper` fetches the URL and extracts text (optionally filtered by CSS selector)
2. **summarize** -- LLM condenses the scraped content into 3-5 sentences
3. **response** -- returns the URL and summary as a JSON API response

## Structure

```
scraper-native/
├── workflow.yaml
└── resources/
    ├── fetch.yaml      # component: scraper - httpClient + htmlText()
    ├── summarize.yaml  # LLM summarization
    └── response.yaml   # API response
```
