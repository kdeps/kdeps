# Search local files

Build an API that searches a directory on disk by filename pattern and content
keyword. It uses the `search_local` component, with no index and no external
service.

The search resource:

<div v-pre>

```yaml
# resources/search.yaml (part)
component:
  name: search_local
  with:
    path: "{{ get('path', '/data') }}"
    query: "{{ get('query') }}"           # keyword in file contents
    glob: "{{ get('glob', '') }}"         # optional filename pattern
```

</div>

## Read next

- [Full walkthrough](/reference/examples/local-file-search) - every step, file and command
- [search_local component](/workflow/resources/search-local) - `index: true`, fuzzy matching, graph boost
- [Document search tutorial](/examples/rag-search) - semantic search with the `embedding` component
