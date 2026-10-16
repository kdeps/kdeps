# Search resources

Two ways to search: the `searchLocal` resource and the `searchWeb` component. Each has its own reference page.

Works in workflow and agent mode.

| Resource | Does | Reference |
| :--- | :--- | :--- |
| `searchLocal:` | Walk a directory by filename glob and content keyword. Optional TF-IDF index for ranked results. | [searchLocal](/workflow/resources/search-local) |
| `component: searchWeb` | Query the web. DuckDuckGo needs no key. Brave, Bing, and Tavily need a named connection. | [searchWeb](/workflow/resources/search-web) |

## See also

- [Embedding resource](/workflow/resources/embedding) - SQLite keyword store for on-prem RAG
- [Scraper component](/workflow/resources/scraper) - Fetch URL content to feed into search pipelines
- [LLM resource](/workflow/resources/llm) - Use search results as context for chat resources
- [Resources overview](/workflow/resources) - all resource types
