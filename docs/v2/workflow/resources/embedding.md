# embedding component

The `embedding` component runs one embedding operation and returns a structured result. It uses a SQLite keyword store for `index`, `search`, `upsert`, and `delete`. It uses an embedding backend for `vectorize` and `embed_query`, and Cohere for `rerank`.

<div v-pre>

```yaml
# resources/store.yaml
actionId: store
name: Store Document
component:
  name: embedding                   # from the components asset set; a components/embedding/ of your own wins
  with:
    operation: index                # required: index | search | upsert | delete | vectorize | embed_query | rerank
    text: "{{ get('text') }}"       # text to index, search for, upsert or delete
    collection: docs                # optional namespace (default: "default")
    dbPath: /data/store.db          # optional SQLite path; default from the kdeps embedding defaults
    limit: 5                        # optional max search results
```

</div>

## Read next

- [embedding component reference](/reference/resources/embedding) - configuration, operations, output, examples, collections
- [Search resource](/workflow/resources/search) - search local files and the web
- [LLM resource](/workflow/resources/llm) - use search results as context
