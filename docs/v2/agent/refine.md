# Prompt refinement

*Applies to agent mode.*

A terse or under-specified prompt ("fix that", "add the flag") makes the model
guess, which wastes the first few tool rounds. **Prompt refinement** runs one
cheap LLM call before the turn starts that rewrites your prompt into a clearer,
self-contained version, then runs the turn on the rewrite.

```text
your prompt -> [refine call] -> refined prompt -> normal turn (preamble, tools, goal, judges)
```

**On by default.** It is a small pre-turn cost that makes the turn land better.
Turn it off with `/refine off` if you want prompts sent verbatim.

**You see the rewrite every turn.** When the prompt is changed, the REPL prints
it before any work starts:

```
[refine] -> Add a --dry-run flag to the sync command and add a unit test that exercises it.
```

The refined text is what the model receives, what `/prompt` shows as the user
message, and what is saved to the session and the memory bridge for that turn -
so a later model switch sees exactly what was answered.

**What it will not do.** The rewrite preserves your intent and every concrete
detail (names, paths, numbers, constraints). It does not answer the request, add
requirements you did not imply, or invent facts. An already-clear prompt comes
back essentially unchanged (and no `[refine]` line is printed).

**When it is skipped.**

- Short or trivial prompts (a one-liner, a bare question) - refining them is not
  worth a call.
- No model is served yet (a local backend that has not been downloaded/started).
- The refine call errors, returns nothing, or returns a runaway expansion - the
  original prompt is used unchanged. Refinement can never block a turn.

Refinement runs *before* [goal decomposition](/agent/goals) and the
[judge roster](/agent/judges), so the plan and the panel are built from the
refined prompt.

## From the REPL

```
/refine            # show whether refinement is on or off
/refine on         # enable (persists across sessions)
/refine off        # disable (persists across sessions)
```

The choice is written to `~/.kdeps/agent-loop-settings.yaml` and restored on the
next `kdeps` run.

## See also

- [Agent mode](/agent/) - overview and starting the REPL
- [Goal-directed execution](/agent/goals) - built from the refined prompt
- [Judge panel](/agent/judges) - the other optional per-turn LLM call
- [REPL slash commands](/agent/commands) - full command reference
