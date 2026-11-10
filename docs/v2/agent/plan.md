# Planning council

Before a task that is worth planning, a small council agrees on how to do the work. The turn then carries that plan. The prompt you typed is left as you typed it.

**On by default.** `/plan off` skips the council and is remembered. `/plan` shows the current choice. `/plan on` turns it back on.

```text
your prompt -> [refine, if on] -> [council] -> turn, with the plan as a system message
```

The council uses the [subagent channel](/agent/subagent): the session model, unless `/subagent` points it at another one. A short or trivial prompt, a local model that is not running yet, or a council that produces no job orders leaves the turn to run on the prompt alone. A failure does the same.

## The board

Three seats sit unless you change it: architect, implementer, and reviewer. Their names and what each one watches are the `plan-council-members` harness entry, one `name: focus` line per seat. The `plan-council-size` event is how many of those lines sit (3).

The first round is a draft. Later rounds are revisions (`plan-council-rounds`, 2 in total): each member reads the others' latest answers and the board, then may append job orders the board is missing. A title already there is skipped. Nothing on the board is edited or removed. The board stops at `plan-council-max-orders` (24).

The members then rank the board. kdeps sorts by their combined rank: a first place scores the most, and an order a member left out scores nothing from that member. A tie keeps the order the orders were added. The sorted list is the plan.

## Progress file

The plan is a markdown file in the project's data folder:

```text
~/.kdeps/projects/<folder>-<hash>/plans/<time>.md
```

One checkbox per job order, in priority order, then a progress log that only grows. The turn is told to call `plan_update` after each job order:

- `doing`, `blocked`, or `dropped`, with the job order's id and a short note, appends a line to the log.
- `done` ticks that checkbox and appends a line.
- `added`, with a title, appends a new job order and a line. The file keeps every order it already had.

The path is also stored in memory under `plan-progress`. A later session that has not planned yet can keep updating that file: `plan_update` reads the memory entry and continues it.

## From the REPL

```text
/plan            # show whether the council is on
/plan on         # enable (persists across sessions)
/plan off        # disable (persists across sessions)
```

The choice is written to `~/.kdeps/agent-loop-settings.yaml` and restored on the next `kdeps` run.

## See also

- [Prompt refinement](/agent/refine) - the rewrite that runs first, when it is on
- [Subagents](/agent/subagent) - the model the council members use
- [REPL slash commands](/agent/commands) - full command reference
- [Built-in tools](/agent/tools) - `plan_update` with the other tools
