# Search resources

Two search components: `search_local` for files on disk and `searchWeb` for the web. Each has its own reference page.

Works in workflow and agent mode.

| Resource | Does | Reference |
| :--- | :--- | :--- |
| `component: search_local` | Walk a directory by filename glob and content keyword. Optional TF-IDF index for ranked results - a kdeps component, YAML only. | [search_local](/workflow/resources/search-local) |
| `component: searchWeb` | Query the web. DuckDuckGo needs no key. Brave, Bing, and Tavily need a named connection. | searchWeb |

## See also

- [Embedding component](/workflow/resources/embedding) - SQLite keyword store for on-prem RAG
- [Scraper component](/workflow/resources/scraper) - Fetch URL content to feed into search pipelines
- [LLM resource](/workflow/resources/llm) - Use search results as context for chat resources
- [Resources overview](/workflow/resources) - all resource types
