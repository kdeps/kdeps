# File tools

`read_file`, `write_file`, `edit_file`, `list_files`, `md5_file`, and `tail_file` run in the agent REPL with no YAML and no API key. Permission modes, lean mode, the handshake, and harness reminders stay on [Built-in tools](/agent/tools).

## File operations

Always available. No environment variables required.

| Tool | Description |
|------|-------------|
| `read_file` | Read file contents with a 1-based line number on every line (plain text, plus PDF/DOCX/EPUB/RTF/ODT extraction) |
| `write_file` | Write or overwrite a file |
| `edit_file` | `command`-dispatched editor: `view`, `str_replace`, `insert`, `patch`, `replace_symbol`, `undo_edit` |
| `list_files` | List directory contents |
| `md5_file` | Compute a file's MD5 hash - cheap way to check whether content actually changed |
| `tail_file` | Read the last N lines of a file without loading the whole thing |

## Line numbers

Every line `read_file` returns is prefixed with its 1-based line number, and the line itself is rendered `cat -A` style: tabs show as `^I`, other control characters as `^X`, and a `$` marks the true end of the line (`  42⇥code$`) - so indentation, trailing whitespace, and stray control characters are visible instead of silently hidden. Those line numbers are what `edit_file`'s `insert` and `view` commands point at.

## Optional file_path

`read_file`, `tail_file`, and `md5_file` treat `file_path` as optional: omit it and the tool operates on the file most recently read, edited, or written this session. This covers the common slip where the model means "the file I was just looking at" and calls `read_file` with only `offset`/`limit`. `write_file` and `edit_file` always require an explicit path.

## Colored diff

`write_file` and `edit_file` print a **colored diff** of what changed under the tool call - removed lines in red, added lines in green, with a couple of context lines - so you can see every change the agent makes at a glance. Large diffs (e.g. writing a whole new file) are capped. The diff is shown in the terminal only; the model receives a concise result, not the ANSI-colored text.

## Jumping from search_local to read_file with match_id

Every `search_local` result that finds its query in the file carries `match_id`, `line`, and `revision` alongside the usual `path`/`snippet` fields. Pass `match_id` to `read_file` instead of `file_path` to jump straight to that hit - `context_before`/`context_after` (default 20 each) control how much surrounding code comes back, the same defaults `edit_file`'s `anchor` view uses:

```xml
<invoke name="search_local">
  <parameter name="path">/app</parameter>
  <parameter name="query">func handleLogin</parameter>
</invoke>
```

```json
{"results": [{"path": "/app/server.go", "match_id": "match-4a9f21bc", "line": 112, "revision": "sha256:...", "snippet": "..."}]}
```

```xml
<invoke name="read_file">
  <parameter name="match_id">match-4a9f21bc</parameter>
  <parameter name="context_after">40</parameter>
</invoke>
```

A `match_id` is process-local and capped in count (oldest evicted first) - it's for chaining a search into a read within the same session, not a durable reference. An unresolvable one (evicted, or from a different process) is a clear error naming it; `file_path` given explicitly always takes priority over `match_id`.

The six `edit_file` commands are on [Edit file](/agent/tools-edit).
