# loader component

The `load_document` component reads a file, URL, or directory into structured `Document` objects (plain text and metadata). It is the first step of a RAG pipeline.

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

- [loader component reference](/reference/resources/loader) - every key, option and detail
- [Component resource](/agencies/component-resource) - how `component:` calls pass `with:` inputs
- [Vector store component](/workflow/resources/vector-store) - index and search the loaded documents
