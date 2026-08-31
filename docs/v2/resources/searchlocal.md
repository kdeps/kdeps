# searchLocal resource

The `searchLocal` executor walks a local directory and returns matching files by filename glob pattern and/or content keyword. See [Search resources](/resources/search) for how it relates to `searchWeb`.

It works in both modes: as a DAG step in workflow mode, and inside any workflow the LLM calls as a tool in agent mode.

## Configuration

```yaml
# resources/search.yaml
searchLocal:
  path: "/data/documents"    # required: directory to search
  query: "invoice total"     # optional: keyword in file contents
  glob: "*.txt"              # optional: filename pattern
  limit: 10                  # optional: max results (0 = unlimited)
```

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `path` | string | yes | - | Directory to search recursively |
| `query` | string | no | - | Case-insensitive keyword to find in file contents |
| `glob` | string | no | - | Filename glob pattern (e.g. `*.md`, `report_*.csv`) |
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
| `isDir` | bool | Always `false` |

## Examples

**Find all Markdown files:**

<div v-pre>

```yaml
# resources/find-docs.yaml
actionId: findDocs
searchLocal:
  path: "/workspace/docs"
  glob: "*.md"
```

</div>

**Find files containing a keyword:**

<div v-pre>

```yaml
# resources/find-invoices.yaml
actionId: findInvoices
searchLocal:
  path: "/data/uploads"
  query: "overdue"
  limit: 20
```

</div>

**Feed results into an LLM:**

<div v-pre>

```yaml
# resources/find-files.yaml
actionId: findFiles
searchLocal:
  path: "/data/reports"
  query: "{{ get('query') }}"

---
actionId: answer
requires: [findFiles]
chat:
  model: llama3.2:1b
  prompt: "Files found: {{ output('findFiles').results }}. Summarize."
```

</div>

## Indexed search (ranked results)

`index: true` switches `searchLocal` from a plain directory walk to a persistent TF-IDF inverted index, giving each result a relevance `score` instead of just a yes/no filename/content match. The index is built once (or incrementally reused) at `<path>/.kdeps/index.db` and speeds up repeated searches of the same folder.

```yaml
searchLocal:
  path: "/workspace/docs"
  query: "authentication flow"
  index: true                # build/use a persistent TF-IDF index instead of walking every search
  fuzzy: true                 # optional: Levenshtein fuzzy matching (requires index: true)
  maxDistance: 2               # optional: max edit distance for fuzzy matching, default 2
  indexDBPath: "/custom/index.db"  # optional: override the default <path>/.kdeps/index.db
  graphBoost: true             # optional: re-rank using the folder's link/topic graph (requires index: true)
```

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `index` | bool | no | `false` | Build/use a persistent TF-IDF inverted index instead of a plain walk |
| `fuzzy` | bool | no | `false` | Enable Levenshtein fuzzy term matching (requires `index: true`) |
| `maxDistance` | integer | no | `2` | Max edit distance for fuzzy matching |
| `indexDBPath` | string | no | `<path>/.kdeps/index.db` | Where the TF-IDF index is stored |
| `graphBoost` | bool | no | `false` | Re-rank results using the same [kartographer](https://github.com/kdeps/kartographer) reference/topic graph `codeIntelligence`'s [`indexFolder`/`graphAll`](codeintelligence-graph) builds - requires `index: true` |

**How `graphBoost` ranks results:** `searchLocal` already ranks by TF-IDF. `graphBoost` additionally builds a graph of the folder - markdown links and shared `topics:`/`tags:` frontmatter - at `<path>/.kdeps/graph.db` (kept separate from `indexDBPath`). A result linked from, or sharing a topic with, a top-5 TF-IDF match gets boosted 25% before the final sort, so a lower-scoring-but-connected document can outrank a higher-scoring-but-isolated one - useful when the top match is a short "index" page.

```yaml
# resources/search-graph.yaml
actionId: findRelatedDocs
searchLocal:
  path: "/workspace/docs"
  query: "{{ get('query') }}"
  index: true
  graphBoost: true
```

Indexed-mode output rows add two fields on top of the base result shape:

| Key | Type | Description |
|-----|------|-------------|
| `score` | float | TF-IDF relevance score (post-boost when `graphBoost` is set) |
| `matchCount` | integer | Number of distinct query terms matched |
| `snippet` | string | Text excerpt around the match |

## See also

- [Search resources](/resources/search) - overview and `searchWeb`
- [Graphing an indexed folder](/resources/codeintelligence-graph) - the same kartographer graph `graphBoost` reuses
