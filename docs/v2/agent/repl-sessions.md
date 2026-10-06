# Sessions

Each directory you launch `kdeps` from has its own conversation history. Nothing is written into the project directory. The other REPL behaviors stay on [REPL features](/agent/repl).

## Where sessions are stored

Conversations and the agent's memory are stored in `~/.kdeps/`, **partitioned by
the directory you launch `kdeps` from** - `~/.kdeps/sessions/` and
`~/.kdeps/memory/` each hold a subfolder per working directory. `cd` into a
project and `kdeps` sees that folder's history; a different folder starts clean.
Nothing is written into the project directory itself.

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
- Resuming, continuing, then exiting updates the same session - it does not
  fork a new one. The session is also saved after every turn, so a crash or
  kill still leaves it in the picker.
- Settings (`/refine`, `/theme`, the default model, tool tuning), the model
  cache, and everything else under `~/.kdeps/` are unchanged.
- `ctrl+d` on a highlighted session deletes it in place (the "Start a new
  session" row cannot be deleted) - same as `/session delete <id>`, just
  reachable from the picker itself.
- Multiple `kdeps` instances can point at the same folder at once, each with
  its own picker and its own chosen (or new) session - the underlying session
  file is opened only for the moment of each read/write, not held open for an
  instance's whole run, so instances do not lock each other out.

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

`/session goto` is non-destructive: the pruned tail is stashed. Use `/session branches` to see stashed entry IDs, then `/session goto <id>` again to navigate back.
