# Search resources

Three components search: `web_search` searches the web, `search_local` searches files on disk, and `embedding` searches a local store.

```yaml
# resources/find.yaml
actionId: find
name: Find Pages
component:
  name: web_search
  with:
    query: "kdeps workflow engine"   # required
```

## Read next

- [Search resources reference](/reference/resources/search) - every key, option and detail
- [Web scraper](/workflow/resources/web-scraper) - read the text of a URL the search returned
- [Embedding component](/workflow/resources/embedding) - SQLite keyword store for on-prem RAG
