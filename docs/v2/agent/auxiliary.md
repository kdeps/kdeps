# Auxiliary model channel

Knowledge-graph extraction, prompt refinement, and judges call a model of their own. That channel is the session model until you point it somewhere else. The turn itself, compaction, the handshake, and branch summaries stay on the session model.

```text
/aux                         # session model (the default)
/aux current                 # back to the session model
/aux free                    # a large OpenRouter free model, chosen per call
/aux google/gemma-4-31b-it:free   # pin one model, every auxiliary call
/aux gpt-4o                        # any model, not only OpenRouter
```

`free` picks from the larger half of the OpenRouter free catalog (`models/openrouter`), one id per call. Any other id is a pin. A catalog id uses that provider. A slash id that is not in the catalog goes to OpenRouter. Any other name uses the session backend. The choice persists in `~/.kdeps/agent-loop-settings.yaml` and wins over the config file once you have set it.

```yaml
# ~/.kdeps/config.yaml
llm:
  aux: current    # current | free | a model id
```

`kdeps knowledge build --extract` uses the same order: `--model`, then `llm.aux` when it names a model, then `knowledge.extract.model`, then the current model. `/knowledge build` chooses when that pass runs: `manual` (when you ask), `auto` (also at session end and before a background build), or `off`. The link graph still rebuilds without a model.

```bash
kdeps knowledge build --extract
kdeps knowledge build --extract --model google/gemma-4-31b-it:free
```

An unserved local session model still skips extraction, refinement, and roster generation. Pointing the channel at another provider runs those calls without that local server.
