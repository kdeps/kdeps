# Goal-directed execution

Every prompt in the [agent loop REPL](/modes/agent-loop-mode) becomes an explicit task list that Go code drives to completion. The
loop walks a cursor through the list that only ever moves forward, so a model
cannot circle back over finished work or stall on a task until a budget expires.

```text
prompt -> decompose into tasks -> confirm -> [task 1] -> [task 2] -> ... -> answer
                                     ^ only the active task is in scope
```

**Confirming the plan.** Reaching the original prompt's goal can take several
intermediate tasks, and a single decomposition call can misorder, omit, or
invent a step. Once decomposition produces more than one task, an independent
second LLM call reviews the candidate list against the original request and
either approves it unchanged or returns a corrected list - the plan the loop
actually runs is always the confirmed one. A single-task plan skips this
(nothing to reorder), and a failed or unparsable confirmation falls back to
the original candidate rather than blocking the turn.

**How a task is settled.** The model cannot finish a task by saying so in prose.
It calls one of two tools and the code validates the id against the active task:

- `task_complete{id, summary, evidence}` - the objective is met; advance.
- `task_fail{id, reason}` - it cannot be done; advance anyway with the reason recorded.

If a turn ends with a text answer instead of either call, the loop settles the
active task from that text and continues with the next one.

**Evidence-gated completion (`RequireTaskEvidence`).** By default, `task_complete`
only checks *which* task is being closed -- the claimed outcome itself is
unverified. With `RequireTaskEvidence: true`, a task that made tool calls must
have run at least one verification-capable tool (`bash_exec`, `read_file`,
`list_files`, `md5_file`, `tail_file`, `search_local`, `sql_query`,
`memory_query`, `code_search`, `code_diagnostics`) before `task_complete` is
accepted -- otherwise it is refused with a message naming what to run first.
A task that made *no* tool calls (a direct answer with nothing to verify) is
exempt, and `task_fail` is never gated. The `evidence` argument records what
was checked and what it showed (e.g. `"ran go test ./pkg/foo, 12 passed"`),
stored on the task and queryable later via
[`memory_query`](/concepts/memory#relational-query-memory-query)'s
`tasks` relation.

Two tools exist specifically for this: `md5_file` computes a file's MD5
checksum -- call it once before and once after a change and compare the two
hashes to prove whether the content actually changed -- and `tail_file`
returns the last N lines of a file (default 20) without needing to know its
total length upfront, for checking how a log or command output ends.
`list_files` also defaults to the current working directory when `path` is
omitted, making a quick directory listing a one-argument-free call.

**You can see which task is active.** Whenever the cursor moves onto a task - a fresh multi-task plan, a resumed goal, or an advance via `task_complete`/
`task_fail` - the REPL prints which one, so you always know what the loop is
doing without running `/goal`:

```
[goal] plan: 3 tasks - /goal to inspect, /goal clear to drop
[goal] working on task 1/3: add the /users endpoint
...
[goal] working on task 2/3: write tests for it
```

Silent for a single-task goal - there is nothing to disambiguate. The
modeline's `task:N/M` counter tracks the same cursor between prompts.

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
/goal new <text>     # replace the active goal
/goal skip           # abandon the active task and move to the next
/goal clear          # drop the goal entirely
```

The modeline shows `task:2/5` while a goal is active. Enforcement is on in the
interactive REPL; library and test callers keep the plain round loop. Tuning:
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

- [Agent Loop Mode](/modes/agent-loop-mode) -- overview and starting the REPL
- [Judge Panel](/modes/agent-loop-judges) -- reviews each turn's final output
- [Agent Registries](/modes/agent-loop-registries) -- TaskRegistry/TeamRegistry tools for multi-agent coordination
