# Sessions

Each directory where you start `kdeps` has its own conversation history. kdeps writes nothing into the project directory. The other REPL behaviors stay on [REPL features](/agent/repl).

## Where sessions are stored

kdeps keeps conversations and agent memory **separately for each directory where
you start `kdeps`**. Conversations go in `~/.kdeps/sessions/<folder>/`. Memory
goes as notes in `Memory/<project>/` of the knowledge vault (see
[Persistent memory](/agent/memory)). When you `cd` into a project, `kdeps` sees
the history of that folder. A different folder starts with no history. kdeps
writes nothing into the project directory.

## The resume picker

When you start `kdeps` in a folder that has saved sessions - and you did not pass
`--resume` or `--new` - a picker lists them:

```d2
direction: down
start: "kdeps  (no --resume, no --new)" {shape: oval}
has: "this folder has saved sessions?" {shape: diamond}
picker: "picker: each session's first prompt,\nturn count, last-active time\n+ 'Start a new session'"
resume: "chosen session's history restored" {shape: oval}
fresh: "clean session" {shape: oval}
start -> has
has -> picker: "yes (interactive)"
has -> fresh: "no / piped input"
picker -> resume: "pick a session"
picker -> fresh: "'Start a new session' / esc"
```

```text
kdeps                     # in a folder with history -> resume picker
kdeps --new               # skip the picker, start a clean session
kdeps --resume <id>       # resume a specific session directly (no picker)
```

## Picker rules

- The picker only appears in an interactive terminal with at least one saved
  session for this folder and no `--resume` / `--new`. Piped input starts clean.
- The first turn after a resume tells the model when the session last ran
  (for example "2026-11-04 07:17 CET, 30 hours ago"). It also tells the model
  to check the repository, files, and task status again with a tool before it
  reports them. The old turns describe the project at that earlier time. The
  text is the `session-resumed` harness entry
  (`assets/harness/session-resumed.yaml`).
- Resuming, continuing, then exiting updates the same session - it does not
  fork a new one. The session is also saved after every turn, so a crash or
  kill still leaves it in the picker.
- Settings (`/refine`, `/theme`, the default model, tool tuning), the model
  cache, and all other files under `~/.kdeps/` do not change.
- `ctrl+d` on a highlighted session deletes it. You cannot delete the "Start a
  new session" row. This is the same as `/session delete <id>`, but you do it
  from the picker.
- Many `kdeps` instances can use the same folder at the same time. Each
  instance has its own picker and its own selected (or new) session. kdeps
  opens the session file only for each read or write. It does not keep the
  file open for the full run of an instance. For this reason, instances do not lock each
  other out.

## Session commands

```
/session list                  # this folder's saved sessions
/session save [name]           # save a named snapshot of the current session
/session load <id>             # restore a saved session
/session delete <id>           # delete a saved session
/session checkpoint            # print the current entry ID (for /session goto)
/session goto <entry-id>       # restore session to the turn at that entry ID
/session branches              # list stashed (pruned) turns from prior /session goto calls
/session import <path>         # load a JSONL session file exported from another run
```

`/session goto` does not delete data. kdeps keeps the removed tail in a stash. Use `/session branches` to see stashed entry IDs, then `/session goto <id>` again to navigate back.

`/session` is a YAML command (`commands/session.yaml`) with [subcommands](/agent/commands#subcommands) over the `session_ops` operations `sessions`, `save`, `load`, `delete`, `import`, `checkpoint`, `goto` and `branches`. Every printed line is in that file, and any kdeps API caller can run the same operations.
