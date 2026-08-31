# Search resources

kdeps provides two native search executors compiled into the binary: `searchLocal` for local file search and `searchWeb` for web search.

## Where it runs

Both [workflow mode](/modes/workflow-mode) and [agent mode](/modes/agent-loop-mode). In workflow mode each executor runs as a DAG step. In agent mode, the workflow containing these resources runs as a single callable tool.

## searchLocal

Walks a local directory and returns matching files by filename glob pattern and/or content keyword, with an optional persistent TF-IDF index for ranked results.

```yaml
searchLocal:
  path: "/data/documents"
  query: "invoice total"
  glob: "*.txt"
```

See [searchLocal Resource](/resources/searchlocal) for the full field reference, indexed/ranked search, and `graphBoost`.

## searchWeb

Queries the web and returns structured results. Default provider is DuckDuckGo - no connection or API key required; Brave, Bing, and Tavily need a named connection.

```yaml
searchWeb:
  query: "{{ get('query') }}"
  maxResults: 5
```

See [searchWeb Resource](/resources/searchweb) for the full field reference, provider setup, and error handling.

## See also

- [searchLocal resource](/resources/searchlocal)
- [searchWeb resource](/resources/searchweb)
- [Embedding resource](embedding) - SQLite keyword store for on-prem RAG
- [Scraper resource](scraper) - Fetch URL content to feed into search pipelines
- [LLM resource](llm) - Use search results as context for chat resources
