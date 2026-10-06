# Search resources

Two search executors, compiled into the binary. Each has its own reference page.

Works in workflow and agent mode.

| Resource | Does | Reference |
| :--- | :--- | :--- |
| `searchLocal:` | Walk a directory by filename glob and content keyword. Optional TF-IDF index for ranked results. | [searchLocal](/workflow/resources/search-local) |
| `searchWeb:` | Query the web. DuckDuckGo needs no key. Brave, Bing, and Tavily need a named connection. | [searchWeb](/workflow/resources/search-web) |

## See also

- [Embedding resource](/workflow/resources/embedding) - SQLite keyword store for on-prem RAG
- [Scraper resource](/workflow/resources/scraper) - Fetch URL content to feed into search pipelines
- [LLM resource](/workflow/resources/llm) - Use search results as context for chat resources
- [Resources overview](/workflow/resources) - all resource types
