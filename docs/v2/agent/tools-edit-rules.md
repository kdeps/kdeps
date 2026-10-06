# Edit rules

A mutating `edit_file` has to follow a read in the same turn. Pass the revision token back to refuse a stale write. The six commands stay on [Edit file](/agent/tools-edit).

## Read before you edit

`str_replace`, `insert`, `patch`, and
`replace_symbol` refuse to touch a file that was not read this turn
(`read_file`, or `edit_file command: view`) - editing a file you have not
looked at is how a change lands in the wrong place. A file you just wrote
with `write_file`, or just edited, counts as read. The file's line-ending
style is preserved on write.

## Revision checks

Every `view` and every successful mutation reports a
`[revision sha256:...]` token. Pass it back as `revision` on a later
`str_replace`/`insert`/`patch`/`replace_symbol` to reject the edit if the file
changed since you read it, instead of silently overwriting someone else's
change:

```json
{"error": "edit_file: /app/server.go changed since revision sha256:1a2b3c4d5e6f was read (it is now sha256:9f8e7d6c5b4a) - view the file again and retry"}
```

Omitting `revision` skips the check entirely - it's an extra safeguard on top
of the read-this-turn gate above, not a replacement for it. The token is
matched on its hex digits, so `sha256:1a2b3c4d5e6f`, bare `1a2b3c4d5e6f`, and
`[revision sha256:1a2b3c4d5e6f]` are all accepted (6+ hex digits).

## Verified by md5, retried until it lands

The loop hashes (md5) the file
before and after every mutating `edit_file` call (`str_replace`, `insert`,
`patch`, `replace_symbol`, `undo_edit`; `dry_run` and `view` are skipped) and
appends the result to the tool output: `[EDIT OK] path md5 a1b2c3d4e5f6 ->
0f9e8d7c6b5a`, or `[EDIT NOT APPLIED] md5 ... is still ...` when nothing changed
(an error, or a no-op such as `old_str` equal to `new_str`). A turn cannot end
while the last edit left the md5 unchanged: the model is pushed back (harness
entry `nudge-edit-unchanged`, at most 4 times) to re-view the file and retry,
and if every retry is spent the turn ends with a `[kdeps: the last edit to ...
never changed the file]` notice instead of a false "done".

## Edit tasks conclude early

The `edit-workflow` preamble section tells the
model to go to the edit once per task in this order: `memory_search` (fuzzy: where
do things stand) -> `memory_query` (precise, based on the search) -> `read_file`
(or `edit_file view`) -> `edit_file`, then finish in one short line instead of
exploring with other tools. Turn it off with `/harness disable edit-workflow`.

## Persistent tool status

Every tool result carries kdeps' own record of what
happened, so the model never hashes a file or guesses whether a call worked. The
status is also written to memory (`status:<tool>:<id>`, last 60 kept) so it can
be looked up later.

```text
edit applied   -> [EDIT OK] path md5 A -> B + memory id + [files edited this session]
edit unchanged -> [EDIT NOT APPLIED] + the error + "read the file first" + view block + retry block
tool ok        -> [STATUS ok] tool + memory id
tool failed    -> [STATUS FAILED] tool: error + memory id + exact <invoke> to retry
```

Each memory id comes with the usage that reads it back, for example
`memory_query query=filter(memory, .key == "status:bash_exec:1790787027780")`.
The `[files edited this session]` list (last 5 files, md5 and applied/not
applied) stays in every edit result. Task-state and memory tools are not
tracked.

## Preview first with `dry_run`

`str_replace`, `insert`, `patch`, and
`replace_symbol` all accept `dry_run: true` - the same diff and would-be
`[revision ...]` are returned, but nothing is written to disk and nothing is
pushed to the undo history.

## Reject a broken edit with `validate_syntax`

`str_replace`, `insert`,
`patch`, and `replace_symbol` accept `validate_syntax: true` - the resulting
file is checked *before* anything is written, and the edit is refused
outright on failure, so a malformed change is never written and never needs
rolling back. Combine with `dry_run` to check without applying. What "check"
means depends on the file extension:

| Extension | Check |
|-----------|-------|
| `.go` | Real parse via Go's standard `go/parser` |
| `.json` | Real parse via `encoding/json` |
| `.yaml`/`.yml` | Real parse via `yaml.v3` |
| `.js`/`.ts`/`.java`/`.c`/`.cpp`/`.cs`/`.rs`/`.php`/`.swift`/`.kt`/etc. | Balanced `()[]{}` scan (string/comment-aware) - lexical, not a real parse |
| `.py`/`.rb`/`.sh`/etc. | Same balanced-delimiter scan, `#` treated as a comment |
| anything else | No-op - `validate_syntax` never blocks a file type it can't check |
