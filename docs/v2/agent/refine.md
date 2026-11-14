# Prompt refinement

A short or incomplete prompt ("fix that", "add the flag") makes the model
guess. This uses the first tool rounds without result. Before the turn starts,
prompt refinement runs one low-cost LLM call. This call rewrites your prompt
into a clearer, self-contained version. Then the turn runs on the rewrite. The refine call uses the [subagent channel](/agent/subagent), which is the session model unless you point it at another model.

```text
your prompt -> [refine call] -> refined prompt -> normal turn (preamble, tools, judges)
```

**Off by default.** Turn it on with `/refine on`. The choice persists. A
session that already stored refine-on stays on until `/refine off`.

**You see the rewrite on each turn.** When refinement changes the prompt, the
REPL prints it before the work starts:

```
[refine] -> Add a --dry-run flag to the sync command and add a unit test that exercises it.
```

The model receives the refined text. `/prompt` shows it as the user message.
kdeps saves it to the session and the memory bridge for that turn. Then, after
a model switch, the new model sees the exact question that the turn answered.

**It sees the conversation up to this point.** The refine call gets the same
history of prior turns as the real turn. Then it can find the meaning of a
reference. For example, "change that to use port 465" becomes specific when the
refiner sees the configuration from the prior turns. The history has only prior
turns. At this point, the prompt that the refiner rewrites is not in the history.

**What it will not do.** The rewrite preserves your intent and every concrete
detail (names, paths, numbers, constraints). It does not answer the request, add
requirements you did not imply, or invent facts. An already-clear prompt comes
back almost unchanged (and the REPL prints no `[refine]` line).

**When kdeps skips it.**

- Short or trivial prompts (a one-liner, a bare question) - refining them is not
  worth a call.
- No model is available yet (a local backend that you did not download or start).
- The refine call gives an error, returns nothing, or returns a text that is
  too long. kdeps uses the original prompt with no change. Refinement cannot
  stop a turn.

Refinement runs *before* the [judge roster](/agent/judges). Then kdeps makes
the panel from the refined prompt.

## From the REPL

```
/refine            # show whether refinement is on or off
/refine on         # enable (persists across sessions)
/refine off        # disable (persists across sessions)
```

kdeps writes the choice to `~/.kdeps/agent-loop-settings.yaml` and uses it
again on the next `kdeps` run.

## See also

- [Agent mode](/agent/) - overview and starting the REPL
- Goal-directed execution - built from the refined prompt
- [Judge panel](/agent/judges) - the other optional per-turn LLM call
- [REPL slash commands](/agent/commands) - full command reference
