# search-web-native

Web search using the **built-in native searchWeb executor** -- uses DuckDuckGo by default, no API key needed. Optionally supports Brave, Bing, and Tavily.

## Usage

```bash
kdeps run examples/search-web-native/workflow.yaml --dev
```

Search and get an AI-powered answer:
```bash
curl -X POST http://localhost:16398/search \
  -H "Content-Type: application/json" \
  -d '{"query": "What is the Go programming language?"}'
```

With Brave Search (requires API key):
```bash
curl -X POST http://localhost:16398/search \
  -H "Content-Type: application/json" \
  -d '{"query": "latest Go release", "provider": "brave", "apiKey": "YOUR_BRAVE_KEY"}'
```

## How it works

1. **search** -- `run.searchWeb` queries DuckDuckGo (or Brave/Bing/Tavily) and returns structured results
2. **answer** -- LLM uses the results to answer the question
3. **response** -- returns the AI answer as an API response

## Providers

| Provider | `provider` value | API Key Required |
|----------|-----------------|------------------|
| DuckDuckGo | `ddg` (default) | No |
| Brave Search | `brave` | Yes |
| Bing | `bing` | Yes |
| Tavily | `tavily` | Yes |
