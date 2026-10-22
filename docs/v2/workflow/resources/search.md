# Search resources

Two search components: `search_local` for files on disk and `web_search` for the web. Each has its own reference page.

Works in workflow and agent mode.

| Resource | Does | Reference |
| :--- | :--- | :--- |
| `component: search_local` | Walk a directory by filename glob and content keyword. Optional TF-IDF index for ranked results - a kdeps component, YAML only. | [search_local](/workflow/resources/search-local) |

## See also

- [Embedding component](/workflow/resources/embedding) - SQLite keyword store for on-prem RAG
- Scraper component - Fetch URL content to feed into search pipelines
- [LLM resource](/workflow/resources/llm) - Use search results as context for chat resources
- [Resources overview](/workflow/resources) - all resource types
