# search-local

> Search local files by keyword or glob pattern.

The search-local component searches a local directory for files matching a filename glob pattern or containing a keyword in their contents. Use it to build file discovery tools, local RAG pipelines, or code search utilities within your agent's working directory.

## Inputs

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `path` | string | yes | — | Directory path to search in |
| `query` | string | no | `""` | Keyword to search for within file contents |
| `glob` | string | no | `""` | Glob pattern to match filenames (e.g. `*.md`, `**/*.py`) |
| `limit` | integer | no | `0` | Maximum number of results (`0` = unlimited) |

> Either `query` or `glob` should be provided. If both are given, `glob` takes precedence.

## Outputs

Accessed via `output('<your-action-id>')`:

```json
{
  "results": [
    { "file": "/path/to/file.md" },
    { "file": "/path/to/other.md", "snippet": "first 200 chars of content..." }
  ],
  "count": 2
}
```

| Field | Type | Description |
|-------|------|-------------|
| `results` | array | Matched files. Glob results include only `file`; keyword results include `file` and `snippet`. |
| `count` | integer | Number of matched files |

## Usage

Find all Markdown files in a directory:

```yaml
run:
  component:
    name: search-local
    with:
      path: /workspace/docs
      glob: "**/*.md"
  apiResponse:
    success: true
    response:
      files: "{{ output('findDocs').results }}"
      total: "{{ output('findDocs').count }}"
```

Search file contents for a keyword:

```yaml
run:
  component:
    name: search-local
    with:
      path: /workspace/src
      query: "TODO"
      limit: 20
  apiResponse:
    success: true
    response:
      todos: "{{ output('findTodos').results }}"
```

List Python files only:

```yaml
run:
  component:
    name: search-local
    with:
      path: "{{ input('projectDir') }}"
      glob: "*.py"
      limit: 50
```

## Requirements

- Python stdlib only (`os`, `json`, `glob`) — no additional packages needed.
- Read access to the target directory and its files.

## Notes

- The `glob` pattern is applied relative to `path` using Python's `glob.glob(..., recursive=True)`. Use `**/*.ext` for recursive matching.
- Content search (`query`) reads all files under `path` recursively and checks for case-insensitive substring matches. Binary files are read with `errors="replace"` to avoid failures.
- The `snippet` in keyword results is the first 200 characters of the file content (not just the matching line).
- For large directories, set `limit` to avoid returning thousands of results.
- Symlinks are followed; hidden files (`.dotfiles`) are included.
