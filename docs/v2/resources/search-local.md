# Search Local Resource

The `searchLocal` executor is a native capability compiled into the `kdeps` binary. It walks a local directory and returns matching files by filename glob pattern and/or content keyword.

## Configuration

```yaml
run:
  searchLocal:
    path: "/data/documents"    # required: directory to search
    query: "invoice total"     # optional: keyword in file contents
    glob: "*.txt"              # optional: filename pattern
    limit: 10                  # optional: max results (0 = unlimited)
```

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `path` | string | yes | — | Directory to search recursively |
| `query` | string | no | — | Case-insensitive keyword to find in file contents |
| `glob` | string | no | — | Filename glob pattern (e.g. `*.md`, `report_*.csv`) |
| `limit` | integer | no | `0` | Max results (0 = unlimited) |

When both `query` and `glob` are set, a file must match **both** to be included.

## Output

| Key | Type | Description |
|-----|------|-------------|
| `results` | array | List of matching file objects |
| `count` | integer | Number of results |
| `path` | string | The search root used |
| `json` | string | Full result as JSON string |

Each result object:

| Key | Type | Description |
|-----|------|-------------|
| `path` | string | Full file path |
| `name` | string | Filename |
| `size` | integer | File size in bytes |
| `isDir` | bool | Always `false` (directories are skipped) |

## Examples

### Find all Markdown files

<div v-pre>

```yaml
metadata:
  actionId: findDocs
run:
  searchLocal:
    path: "/workspace/docs"
    glob: "*.md"
```

</div>

### Find files containing a keyword

<div v-pre>

```yaml
metadata:
  actionId: findInvoices
run:
  searchLocal:
    path: "/data/uploads"
    query: "overdue"
    limit: 20
```

</div>

### Combine glob and keyword

<div v-pre>

```yaml
metadata:
  actionId: findContracts
run:
  searchLocal:
    path: "/data"
    glob: "*.txt"
    query: "termination clause"
```

</div>

### Feed results into an LLM

<div v-pre>

```yaml
metadata:
  actionId: findFiles
run:
  searchLocal:
    path: "/data/reports"
    query: "{{ get('query') }}"

---
metadata:
  actionId: answer
  requires: [findFiles]
run:
  chat:
    model: llama3.2:1b
    prompt: "Files found: {{ output('findFiles').results }}. Summarize."
  apiResponse:
    response: "{{ output('answer') }}"
```

</div>

## Error Handling

```yaml
run:
  searchLocal:
    path: "/data"
    query: "keyword"
  onError:
    action: continue
```

## Next Steps

- [Embedding Resource](embedding) - Index and keyword-search document content
- [Scraper Resource](scraper) - Fetch web content
- [Python Resource](python) - Advanced file processing
