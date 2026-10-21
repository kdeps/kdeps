# search_local component

The `search_local` component searches a local directory by content keyword and/or filename glob and returns `{results, count}`. It is a kdeps component from the `components` asset set: call it with a `component:` resource, no install step. Its call is the kdeps API's `search_local` tool, which runs the native searchLocal executor. A workflow that uses `searchLocal:` directly - primary or in a `before:`/`after:` step - is rejected. See [Search resources](/workflow/resources/search) for how it relates to `searchWeb`.

It works in both modes: as a DAG step in workflow mode, and inside any workflow the LLM calls as a tool in agent mode.

```d2
direction: right
call: "component: search_local" {shape: oval}
api: "POST /tools/search_local/call"
walk: "walk the folder:\nglob + keyword"
index: "index: true:\nTF-IDF + optional graphBoost"
out: "{results, count}" {shape: oval}
call -> api
api -> walk: "index: false (default)"
api -> index: "index: true"
walk -> out
index -> out
```

## Configuration

<div v-pre>

```yaml
# resources/search.yaml
actionId: search
name: Search
component:
  name: search_local              # from the components asset set; a components/search_local/ of your own wins
  with:
    path: "{{ get('path', '/data') }}"   # directory to search; empty searches the working project's folders
    query: "{{ get('query') }}"          # keyword in file contents; query or glob is required
    glob: "{{ get('glob', '') }}"        # optional filename pattern, e.g. *.md
    limit: 10                            # max results, default 3
```

</div>

| Input | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `query` | string | query or glob | - | Case-insensitive keyword or regex to find in file contents |
| `path` | string | no | working project | Directory to search recursively. An absolute path is recommended |
| `glob` | string | no | - | Filename glob pattern (e.g. `*.md`, `report_*.csv`) |
| `limit` | number | no | `3` | Max results |
| `index` | bool | no | `false` | Build/use a persistent TF-IDF index instead of walking every search |
| `fuzzy` | bool | no | `false` | Levenshtein fuzzy term matching (requires `index: true`) |
| `maxDistance` | integer | no | `2` | Max edit distance for fuzzy matching |
| `indexDBPath` | string | no | `~/.kdeps/projects/<folder>-<hash>/index.db` | Where the index is stored |
| `graphBoost` | bool | no | `false` | Re-rank indexed results using the folder's reference/topic graph (requires `index: true`) |

When both `query` and `glob` are set, a file must match both to be included.

## Output

```json
{
  "results": [{"path": "/data/2024-invoice.md", "name": "2024-invoice.md", "size": 812, "isDir": false}],
  "count": 1
}
```

Read it with `output('search').results` and `output('search').count`.

Each result object:

| Key | Type | Description |
|-----|------|-------------|
| `path` | string | Full file path |
| `name` | string | Filename |
| `size` | integer | File size in bytes |
| `isDir` | bool | Always `false` |

## What the walk skips

The walk prunes directories that only ever hold binaries, generated output, or caches: every dot-directory (`.git`, `.codeql-db`, `.venv`, `.cache`, ...) and the vendor/build/cache names `node_modules`, `bower_components`, `vendor`, `target`, `build`, `bin`, `obj`, `__pycache__`, `site-packages`, `coverage`, `dist-packages`, `tmp`. Their contents would otherwise crowd real documents out of the `limit`.

Files larger than 1 MiB are skipped too - they are neither glob-matched nor content-matched.

The `path` you pass is always searched, even when it is a dot-directory such as `path: "/workspace/.config-app"`. Indexed mode (`index: true`) applies the same directory and size skips when it builds the index.

## Examples

**Find all Markdown files:**

<div v-pre>

```yaml
# resources/find-docs.yaml
actionId: findDocs
component:
  name: search_local
  with:
    path: "/workspace/docs"
    glob: "*.md"
```

</div>

**Find files containing a keyword:**

<div v-pre>

```yaml
# resources/find-invoices.yaml
actionId: findInvoices
component:
  name: search_local
  with:
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
component:
  name: search_local
  with:
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

`index: true` switches the executor from a plain directory walk to a persistent TF-IDF inverted index, giving each result a relevance `score` instead of just a yes/no filename/content match. The index is built once (or incrementally reused) at `~/.kdeps/projects/<folder>-<hash>/index.db` and speeds up repeated searches of the same folder. Nothing is written into the searched folder, so a git repository stays clean.

<div v-pre>

```yaml
# resources/search-indexed.yaml
actionId: searchDocs
component:
  name: search_local
  with:
    path: "/workspace/docs"
    query: "authentication flow"
    index: true                  # build/use a persistent TF-IDF index instead of walking every search
    fuzzy: true                   # optional: Levenshtein fuzzy matching (requires index: true)
    maxDistance: 2                # optional: max edit distance for fuzzy matching, default 2
    indexDBPath: "/custom/index.db"  # optional: override the default ~/.kdeps/projects/<folder>-<hash>/index.db
    graphBoost: true              # optional: re-rank using the folder's link/topic graph (requires index: true)
```

</div>

## How graphBoost ranks results

Indexed search already ranks by TF-IDF. `graphBoost` additionally builds a graph of the folder - markdown links and shared `topics:`/`tags:` frontmatter - at `~/.kdeps/projects/<folder>-<hash>/graph.db` (kept separate from `indexDBPath`). A result linked from, or sharing a topic with, a top-5 TF-IDF match gets boosted 25% before the final sort, so a lower-scoring-but-connected document can outrank a higher-scoring-but-isolated one - useful when the top match is a short "index" page.

<div v-pre>

```yaml
# resources/search-graph.yaml
actionId: findRelatedDocs
component:
  name: search_local
  with:
    path: "/workspace/docs"
    query: "{{ get('query') }}"
    index: true
    graphBoost: true
```

</div>

## Indexed-mode output

Indexed-mode output rows add three fields on top of the base result shape:

| Key | Type | Description |
|-----|------|-------------|
| `score` | float | TF-IDF relevance score (post-boost when `graphBoost` is set) |
| `matchCount` | integer | Number of distinct query terms matched |
| `snippet` | string | Text excerpt around the match |

## Errors

| Error | Cause |
|-------|-------|
| `search_local: give a query (or a glob) to search for` | Neither `query` nor `glob` is set |
| `search_local: ... blocked` | `path` is a filesystem root (`/`) |
| `component "search_local" not found` | The components asset set is not installed - run `kdeps update` |

## See also

- [Search resources](/workflow/resources/search) - overview and `searchWeb`
- [Graphing an indexed folder](/workflow/resources/code-graph) - the same kartographer graph `graphBoost` reuses
