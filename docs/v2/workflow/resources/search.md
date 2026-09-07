# Search resources

kdeps provides two native search executors compiled into the binary: `searchLocal` for local file search and `searchWeb` for web search.

## Where it runs

Both [workflow mode](/workflow/) and [agent mode](/agent/). In workflow mode each executor runs as a DAG step. In agent mode, the workflow containing these resources runs as a single callable tool.

## searchLocal

Walks a local directory and returns matching files by filename glob pattern and/or content keyword, with an optional persistent TF-IDF index for ranked results.

```yaml
searchLocal:
  path: "/data/documents"
  query: "invoice total"
  glob: "*.txt"
```

See [searchLocal Resource](/workflow/resources/search-local) for the full field reference, indexed/ranked search, and `graphBoost`.

## searchWeb

Queries the web and returns structured results. Default provider is DuckDuckGo - no connection or API key required; Brave, Bing, and Tavily need a named connection.

```yaml
searchWeb:
  query: "{{ get('query') }}"
  maxResults: 5
```

See [searchWeb Resource](/workflow/resources/search-web) for the full field reference, provider setup, and error handling.

## See also

- [searchLocal resource](/workflow/resources/search-local)
- [searchWeb resource](/workflow/resources/search-web)
- [Embedding resource](/workflow/resources/embedding) - SQLite keyword store for on-prem RAG
- [Scraper resource](/workflow/resources/scraper) - Fetch URL content to feed into search pipelines
- [LLM resource](/workflow/resources/llm) - Use search results as context for chat resources
