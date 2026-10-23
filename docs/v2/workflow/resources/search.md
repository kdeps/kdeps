# Search resources

Three components search: `web_search` finds pages on the web, `search_local` walks files on disk, and `embedding` ranks text by keyword in a local SQLite store. Each has its own reference page.

Works in workflow and agent mode.

| Resource | Does | Reference |
| :--- | :--- | :--- |
| `component: web_search` | Query the web via DuckDuckGo, return titles, URLs, and snippets - a kdeps component, YAML only. | [web_search](#web_search) |
| `component: search_local` | Walk a directory by filename glob and content keyword. Optional TF-IDF index for ranked results - a kdeps component, YAML only. | [search_local](/workflow/resources/search-local) |

## web_search

```yaml
# resources/find.yaml
actionId: find
name: Find Pages
component:
  name: web_search
  with:
    query: "kdeps workflow engine"   # required
```

```yaml
# resources/respond.yaml
actionId: respond
requires: [find]
apiResponse:
  response:
    results: "{{ output('find') }}"
```

The output is the search engine's readable text - titles, URLs, and snippets. Feed a URL into `web_scraper` for the full page.

In agent mode the same tool is the `web_search` built-in, and the loop caps `web_search`/`web_scraper` calls per user request (20 by default; `/model tool set web-limit <n>` changes it).

## See also

- [Web scraper](/workflow/resources/web-scraper) - read the text of a URL the search returned
- [Embedding component](/workflow/resources/embedding) - SQLite keyword store for on-prem RAG
- [LLM resource](/workflow/resources/llm) - use search results as context for chat resources
- [Resources overview](/workflow/resources) - all resource types
