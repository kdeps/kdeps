# loader component

The `load_document` component reads a file, URL, or directory into structured `Document` objects - plain text plus metadata - and optionally splits them into chunks. It is the ingestion step of a RAG pipeline: load, then feed the output into the `vector_store` or `embedding` component for indexing. It is a kdeps component from the `components` asset set: call it with a `component:` resource, no install step. Its call is the kdeps API's `load_document` tool, which runs the native loader executor. A workflow that uses `loader:` directly - primary or in a `before:`/`after:` step - is rejected.

It works in both modes: as a DAG step in workflow mode, and as the `load_document` built-in tool in agent mode.

```d2
direction: right
call: "component: load_document" {shape: oval}
api: "POST /tools/load_document/call"
pick: "type?" {shape: diamond}
text: "text / csv\nread file"
pdf: "pdf\nextract page text"
web: "html\nfetch URL, extract text"
dir: "directory / notion\nevery file, one Document each"
chunk: "chunkSize > 0?\nrecursive | token | markdown"
out: "documents + count" {shape: oval}
call -> api
api -> pick
pick -> text: "text | csv"
pick -> pdf: "pdf"
pick -> web: "html"
pick -> dir: "directory | notion"
text -> chunk
pdf -> chunk
web -> chunk
dir -> chunk
chunk -> out
```

## Basic usage

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

```yaml
# resources/loadAndChunk.yaml
actionId: loadAndChunk
name: Load and Chunk for RAG
component:
  name: load_document
  with:
    type: text
    source: /data/handbook.txt
    chunkSize: 500          # split into chunks of 500 characters
    chunkOverlap: 50        # 50 characters shared by consecutive chunks
    chunkSplitter: recursive  # recursive (default), token, or markdown
```

## Inputs

| Input | Type | Required | Description |
|---|---|---|---|
| `source` | string | yes | File path, URL (for `html`), or directory path |
| `type` | string | no | `text` (default), `pdf`, `html`, `csv`, `directory`, `notion`. Omitted types follow the file extension |
| `columns` | array | no | CSV only - column filter; empty loads all columns |
| `password` | string | no | PDF only - decryption password for encrypted PDFs |
| `chunkSize` | number | no | When > 0, splits each loaded document into chunks of this size |
| `chunkOverlap` | number | no | Characters of overlap between consecutive chunks |
| `chunkSplitter` | string | no | Splitting strategy: `recursive` (default), `token`, or `markdown` |

## Loader types

| `type` | Reads |
|---|---|
| `text` | Plain text file |
| `pdf` | PDF file, page text extracted |
| `html` | A URL - fetches and extracts readable text |
| `csv` | CSV file, one Document per row (or per `columns` filter) |
| `directory` | Every file in a directory, one Document per file |

## Output

```json
{
  "documents": [
    { "content": "...", "metadata": { "source": "/data/handbook.txt", "chunk": 0 } }
  ],
  "count": 3
}
```

`documents` is directly usable as the `documents:` input to the [`vector_store` component](/workflow/resources/vector-store)'s `add_documents` operation - each `content`/`metadata` pair maps onto a `VectorStoreDocument`. The component returns the tool's body as a string, so read it with `fromJSON`.

## Size limit

Every loader type reads its source fully into memory before returning - fine for a report or a handbook, but a workflow doing bulk ingestion (a large CSV, a directory of documents) risks an OS-level, no-core-dump `SIGKILL` once that content gets copied again downstream (`chunkSize` splitting, `json.Marshal` in the result, a `vector_store` step embedding it). Any single file over **256 MiB** is rejected with a clear error instead - `type: directory`/`notion` skip an oversized file and keep loading the rest (a batch load should not abort over one large file); every other type fails that one load outright, the same way an explicit `--file`/stdin over its own 256 MiB default already does (see [Input sources](/workflow/input-sources)).

Raise or lower it with `KDEPS_LOADER_MAX_BYTES` (bytes; `0` disables the check):

```bash
KDEPS_LOADER_MAX_BYTES=1073741824 kdeps run workflow.yaml  # 1 GiB, for a workflow that loads bulk data
```

## RAG pipeline example

```yaml
# resources/load.yaml
actionId: load
component:
  name: load_document
  with:
    type: pdf
    source: "{{ get('filePath') }}"
    chunkSize: 500
    chunkOverlap: 50
```

```yaml
# resources/index.yaml
actionId: index
requires: [load]
component:
  name: vector_store
  with:
    operation: add_documents
    provider: qdrant
    url: http://localhost:6333
    collection: docs
    embedModel: text-embedding-3-small  # embedding model for the document contents
    documents: "{{ fromJSON(output('load')).documents }}"  # load_document's Document list
```

## See also

- [Component resource](/agencies/component-resource) - how `component:` calls pass `with:` inputs
- [Vector store component](/workflow/resources/vector-store) - index and search the loaded documents
- [Embedding component](/workflow/resources/embedding) - lightweight keyword-based alternative for local, API-key-free RAG
- [Resources overview](/workflow/resources) - all resource types
