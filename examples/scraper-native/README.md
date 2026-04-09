# scraper-native

Web scraping + AI summarization using the **built-in native scraper executor** -- pure Go, no Python dependencies.

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

1. **fetch** -- `run.scraper` fetches the URL and extracts text (optionally filtered by CSS selector)
2. **summarize** -- LLM condenses the scraped content into 3-5 sentences
3. **response** -- returns the URL and summary as a JSON API response

## Structure

```
scraper-native/
├── workflow.yaml
└── resources/
    ├── fetch.yaml      # run.scraper - native HTTP + goquery
    ├── summarize.yaml  # LLM summarization
    └── response.yaml   # API response
```
