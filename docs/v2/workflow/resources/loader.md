# loader component

The `load_document` component reads a file, URL, or directory into structured `Document` objects (plain text and metadata). It can also split them into chunks. It is the ingestion step of a RAG pipeline. First load the data. Then give the output to the `vector_store` or `embedding` component, which indexes it.

```yaml
# resources/load.yaml
actionId: load
name: Load Report
component:
  name: load_document       # from the components asset set; a components/load_document/ of your own wins
  with:
    type: pdf               # text (default), pdf, html, csv, directory, notion
    source: /data/quarterly-report.pdf
```

## Read next

- [loader component reference](/reference/resources/loader) - basic usage, inputs, loader types, output, size limit, RAG pipeline example
- [Component resource](/agencies/component-resource) - how `component:` calls pass `with:` inputs
- [Vector store component](/workflow/resources/vector-store) - index and search the loaded documents
- [Embedding component](/workflow/resources/embedding) - lightweight keyword-based alternative for local, API-key-free RAG
- [Resources overview](/workflow/resources) - all resource types
