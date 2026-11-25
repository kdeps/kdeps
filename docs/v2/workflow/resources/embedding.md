# embedding component

The `embedding` component indexes and searches text in a local SQLite store. It needs no external service.

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
```

</div>

## Read next

- [embedding component reference](/reference/resources/embedding) - every key, option and detail
- [Search resource](/workflow/resources/search) - search local files and the web
- [LLM resource](/workflow/resources/llm) - use search results as context
