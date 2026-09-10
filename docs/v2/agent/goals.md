# Goal-directed execution

With goal-directed execution on, every prompt in the [agent loop REPL](/agent/) becomes an explicit task list that Go code drives to completion.

*Applies to agent mode.*

**Off by default.** Goal-directed execution adds a planning LLM call and extra
output per turn, so it is opt-in: `/goal on` enables it for the session and
persists the choice; `/goal off` turns it back off. Library and test callers
always run the plain round loop.

The loop walks a cursor through the list that only ever moves forward, so a model
cannot circle back over finished work or stall on a task until a budget expires.

When [prompt refinement](/agent/refine) is on (the default), it runs first and
the task list is built from the refined prompt, not the raw one.

```text
prompt -> decompose into tasks -> confirm -> [task 1] -> [task 2] -> ... -> answer
                                     ^ only the active task is in scope
```

**Decomposing the prompt.** The planner is asked to break the request into its
natural steps - one task per distinct action it names or implies, typically two
to six. If the first attempt just returns the whole request restated as a single
task, the loop retries once with an explicit "break this into at least two
steps" instruction.

If the model is unavailable, times out, or still will not decompose, kdeps
falls back to a **mechanical split** that uses only the request's own wording:
numbered lines (`1. ... 2. ...`), bullet lines (`- ...`), and sequencing phrases
(`; `, `then`, `and then`, `after that`, `finally`) each start a new task. Plain
`and` is not a separator ("read the file and print it" stays one step). A
request with no such structure and a model that will not split it is kept as a
single task - the loop still drives it, and the model breaks it down through its
tool calls. A weak local model (for example the default `llama3.2:1b`) benefits
most from writing the request as a numbered list or switching to a larger model
or the router.

**Skipping decomposition.** A short remark (32 characters or fewer) or any
question under ~120 characters is treated as chat and drives a single-task goal
without a planning call, so ordinary conversation costs nothing extra. A longer
imperative one-liner - "clean up the logging package and wire it into startup" -
is planned even without an explicit "then"/"and then" marker.

**Confirming the plan.** Reaching the original prompt's goal can take several
intermediate tasks, and a single decomposition call can misorder, omit, or
invent a step. Once decomposition produces more than one task, an independent
second LLM call reviews the candidate list against the original request and
either approves it unchanged or returns a corrected list - the plan the loop
actually runs is always the confirmed one. A single-task plan skips this
(nothing to reorder); a confirmation that tries to collapse a multi-step plan
back to one task is ignored; and a failed or unparsable confirmation falls back
to the original candidate rather than blocking the turn.

**How a task is settled.** The model cannot finish a task by saying so in prose.
It calls one of two tools and the code validates the id against the active task:

- `task_complete{id, summary, evidence}` - the objective is met; advance.
- `task_fail{id, reason}` - it cannot be done; advance anyway with the reason recorded.

If a turn ends with a text answer instead of either call, the loop settles the
active task from that text and continues with the next one.

**Failed-tool gate (always on).** A task cannot close as *done* while its most
recent work tool (anything other than `task_complete` / `task_fail`) is still
failing: `task_complete` is refused ("your last `edit_file` call failed ... retry
it or call `task_fail`"), and a prose "done" records the task **failed**. A later
successful call to that tool clears the gate. This is independent of
`RequireTaskEvidence` below.

**Evidence-gated completion (`RequireTaskEvidence`).** By default, `task_complete`
only checks *which* task is being closed - the claimed outcome itself is
unverified. With `RequireTaskEvidence: true`, a task that made tool calls must
have run at least one verification-capable tool (`bash_exec`, `read_file`,
`list_files`, `md5_file`, `tail_file`, `search_local`, `sql_query`,
`memory_query`, `code_search`, `code_diagnostics`) before `task_complete` is
accepted - otherwise it is refused with a message naming what to run first.
A task that made *no* tool calls (a direct answer with nothing to verify) is
exempt, and `task_fail` is never gated. The `evidence` argument records what
was checked and what it showed (e.g. `"ran go test ./pkg/foo, 12 passed"`),
stored on the task and queryable later via
[`memory_query`](/agent/memory#relational-query-memory-query)'s
`tasks` relation.

Two tools exist specifically for this: `md5_file` computes a file's MD5
checksum - call it once before and once after a change and compare the two
hashes to prove whether the content actually changed - and `tail_file`
returns the last N lines of a file (default 20) without needing to know its
total length upfront, for checking how a log or command output ends.
`list_files` also defaults to the current working directory when `path` is
omitted, making a quick directory listing a one-argument-free call.

**You can see the plan as soon as it exists.** After decomposition, the REPL
prints the outcome before any tool round runs. Ordinary chat (short, no
multi-step markers) stays silent.

```
[goal] planning...
[goal] plan generated - /goal clear to drop
goal: add the /users endpoint and write tests
  [>] 1. add the /users endpoint
  [ ] 2. write tests for it
[goal] working on task 1/2: add the /users endpoint
```

When the planner could not (or would not) decompose the request - it comes
back as a single task equal to the prompt - the REPL says so instead of
printing a one-line "plan":

```
[goal] planning...
[goal] no sub-tasks - working the request as one goal
[goal] working on: refactor the auth middleware
```

Whenever the cursor then moves onto a task - a fresh plan, a resumed goal, or
an advance via `task_complete`/`task_fail` - the REPL names it, single-task
goals included. The modeline's `task:N/M` counter tracks the same cursor
between prompts.

**When a task stops producing.** A round is unproductive when every tool result
is an error, a convergence block, or a byte-identical repeat. Consecutive
unproductive rounds escalate:

1. Re-anchor - restate the active task and the settled ones ("do not redo these").
2. Narrow - drop the tools that keep failing.
3. Force-close - strip tools and demand the task be closed with what was gathered.
4. Fail forward - mark the task failed and advance the cursor.

Because step 4 always advances, a goal terminates instead of stalling. Work from a
settled task is also refused if reissued, so finished tasks are never re-run.

The plan persists in the memory store, so it survives a `/model` switch and later
turns continue the same goal. When you start the REPL with a goal still carried
over from a previous session, it is shown up front with the commands to steer or
drop it, rather than silently resuming on your next prompt.

```
/goal                # show the plan and each task's status
/goal off            # disable goal-directed execution for the session
/goal on             # re-enable it
/goal new <text>     # replace the active goal
/goal skip           # abandon the active task and move to the next
/goal clear          # drop the current goal (a fresh one starts next prompt)
```

The modeline shows `task:2/5` while a goal is active. `/goal on` and `/goal off`
both persist across sessions. Tuning:
`TaskRoundBudget` (default 25 rounds per task), `MaxUnproductiveRounds`
(default 3), and `RequireTaskEvidence` (default `false`, opt-in).

Small local models sometimes copy the task directive into their reply instead of
acting on it, which would leave the turn with no answer. When that happens the
directive is removed, enforcement is turned off for the rest of the turn, and the
round is retried once as a plain round. The modeline drops `task:n/m` for that
turn.

## Adaptive tool budgets

The per-category caps (`web`, `bash`, `file`, `code`) start at their configured
values and then follow measured yield - the share of distinct calls that returned
something new. The model is never asked to forecast a budget: at plan time it has
seen no results, and a self-granted limit would be exactly the kind of state the
task machine refuses to trust.

- A category still returning new content as it approaches its cap is **extended**
  (up to 3x its starting value).
- A category mostly returning blocks, errors, or duplicates is **cut** to just
  above the calls already made, so the turn stops sinking calls into it.

Adjustments need at least 4 distinct calls in the category, never drop below work
already done, and are reported as `[goal] web budget → 30`.

## See also

- [Agent mode](/agent/) - overview and starting the REPL
- [Judge panel](/agent/judges) - reviews each turn's final output
- [Agent registries](/agent/registries) - TaskRegistry/TeamRegistry tools for multi-agent coordination
