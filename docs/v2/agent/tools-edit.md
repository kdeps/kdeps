# Edit file

`edit_file` changes a file the agent has already read. Six commands: `view`, `str_replace`, `insert`, `patch`, `replace_symbol`, and `undo_edit`. Reads, writes, and `match_id` stay on [File tools](/agent/tools-files).

## edit_file - six commands

`edit_file` takes a `command`. The reliable primitive is still an exact,
byte-verified string swap - there is no fuzzy matching - but `str_replace` can
now resolve that exact text from an anchor pair instead of requiring you to
retype the whole block, and `occurrence` picks a specific match when the text
repeats.

## view

`view` - `edit_file` with `command: view` and a `file_path` prints the file
with a 1-based line number on every line (plain text, not `read_file`'s
visible-whitespace rendering), followed by the file's `[revision
sha256:...]`. Pass `view_range: [start, end]` (1-based
inclusive, `end` `-1` = to end of file) for a slice, or `anchor` (a unique
string) to show the region around it instead - `context_before`/
`context_after` control how many lines of context (default 20 each):

```xml
<invoke name="edit_file">
  <parameter name="command">view</parameter>
  <parameter name="file_path">/app/server.go</parameter>
  <parameter name="anchor">func handleLogin</parameter>
  <parameter name="context_after">40</parameter>
</invoke>
```

A `view` also satisfies the [read-before-edit](/agent/tools-edit-rules#read-before-you-edit) requirement.

## str_replace

`str_replace` - pass `old_str` (the exact current text) and `new_str`.
`old_str` must match the file byte-for-byte - indentation and all:

- 0 matches -> `old_str did not appear verbatim` (copy it from a `view`).
- 2+ matches -> the error lists every line the text starts on; either add
  surrounding lines to make it unique, or pass `occurrence: N` (1-based) to
  pick one of the listed matches directly.
- 1 match (or a resolved `occurrence`) -> the swap is written and the result
  is a numbered snippet of the changed region plus the file's new
  `[revision ...]`.

Instead of `old_str`, pass `start_anchor`/`end_anchor` to replace everything
from one unique string through another (inclusive) - useful for a large block
you do not want to retype:

```xml
<invoke name="edit_file">
  <parameter name="command">str_replace</parameter>
  <parameter name="file_path">/app/server.go</parameter>
  <parameter name="start_anchor">func handleLogin</parameter>
  <parameter name="end_anchor">\n}\n</parameter>
  <parameter name="new_str">func handleLogin(w http.ResponseWriter, r *http.Request) {\n\t// ...\n}</parameter>
</invoke>
```

## insert

`insert` - pass `insert_line` (0 = before the first line, N = after line N)
and `new_str`. Returns the same numbered snippet. Instead of `insert_line`,
pass `insert_before_anchor`/`insert_after_anchor` (a unique string) to
position by content instead of a line number you would otherwise have to look up.

## patch

`patch` - pass `patch`, a standard unified diff with one or more `@@` hunks.
Each hunk's context and removed lines must match the file byte-for-byte and
appear exactly once; hunks are resolved against the file's original
content and applied atomically - if any hunk fails to resolve, nothing is
written:

```xml
<invoke name="edit_file">
  <parameter name="command">patch</parameter>
  <parameter name="file_path">/app/server.go</parameter>
  <parameter name="patch">@@ -10,3 +10,3 @@
 func main() {
-	log.Println("starting")
+	log.Println("starting v2")
 }
</parameter>
</invoke>
```

## replace_symbol

`replace_symbol` - pass `symbol` (a function/type/class/etc. name) and
`new_str` to replace its whole declaration. The symbol must be declared
**exactly once** in the file - same "ambiguous, add specificity" rule as
`str_replace`, with the error listing every declaration line. `symbol_kind`
(e.g. `"function"` or `"type"`) narrows which declaration keywords count,
useful only when a function and a type share a name:

```xml
<invoke name="edit_file">
  <parameter name="command">replace_symbol</parameter>
  <parameter name="file_path">/app/server.go</parameter>
  <parameter name="symbol">handleLogin</parameter>
  <parameter name="new_str">func handleLogin(w http.ResponseWriter, r *http.Request) {
	// ...
}</parameter>
</invoke>
```

Where the symbol's block *ends* is found **lexically, not by a language
parser**: a brace-depth scan for C-like bodies (Go, JS/TS, Java, C/C++/C#,
Rust - braces inside string/comment text are correctly ignored), or an
indentation scan when there is no opening brace nearby (Python `def`/`class`).
This covers ordinary top-level declarations reliably without requiring
gopls/pyright/tree-sitter or any other external parser to be installed, but
it is not full-language-semantics: it does not parse JavaScript embedded
inside an HTML `<script>` tag, and it cannot distinguish two symbols that are
only disambiguated by real scoping (that surfaces as the same "ambiguous"
error `str_replace` gives for a repeated string). `view` accepts the same
`symbol`/`symbol_kind` (plus `context_before`/`context_after`) to show a
declaration without knowing its line range first.

## undo_edit

`undo_edit` - reverts the last mutation on that file (a per-file history
kept for the session).

Read-first, revision, and syntax checks are on [Edit rules](/agent/tools-edit-rules).
