# Prompt Reduction (turo)

`turo` is an optional token reducer for [agent loop mode](/modes/agent-loop-mode). When the `turo` binary is on `PATH`, kdeps pipes everything it sends to the LLM through it first - system preamble, your input, tool results, and conversation history. Code, file paths, and identifiers are preserved verbatim. If a reduction is not smaller than the input, the original passes through unchanged. Applies to agent mode only.

```text
system preamble + input + tool results + history  ->  turo  ->  LLM
```

turo runs a five-stage pipeline, all on by default, repeating until the output stops shrinking:

1. **Filler deletion** - strips pleasantries, hedges, leaders, and articles (`please`, `I think`, `of course`).
2. **Defmatch** - collapses a definition-like phrase into the word it defines (`the state of disorder and lawlessness` -> `anarchy`). Strictly gated: it fires only when every keyword of a headword's definition is present and the headword is cheaper in tokens, so on technical text it makes zero replacements. It earns its keep on natural prose.
3. **Gloss swap** - replaces words with the shortest defining word from their dictionary definition (`approach` -> `come`). The lossiest stage.
4. **Synonym swap** - replaces words with a fewer-token WordNet synonym (`utilize` -> `use`).
5. **Reduction** - keeps content words by part of speech, deduplicates, and (ultra) collapses inflections by lemma.

Phrase-level stages run before word-level ones: a phrase matcher has to see the whole phrase, so a single earlier swap inside it is enough to lose the match. Headwords defmatch produces are held back from the later swaps, which would otherwise walk the match straight back.

One more stage runs before the rest, also on by default:

- **Arrows** - replaces multi-word causal/sequential connectives (`leads to`, `results in`, `gives rise to`) with a single `->` token (`cache miss leads to slow query` -> `cache miss -> slow query`). Only multi-word phrases qualify, so it always saves at least one token. Disable with `/turo arrows off` or `TURO_ARROWS=off`.

Stages 2-5 and arrows are lossy - they change wording, not just drop filler - so agent context sent to the model is compressed but no longer verbatim prose. Disable individual stages with the `TURO_*` environment variables below, or turn turo off entirely.

Turo is entirely optional: if the binary is not installed, kdeps sends everything unreduced and the `/turo` command reports that it is unavailable.

Control it at runtime with `/turo`:

```
/turo                # show status: state, level, and stage toggles
/turo off            # send content unreduced (disable)
/turo on             # re-enable
/turo ultra          # set level: lite | full | ultra | wenyan
/turo wenyan         # ultra reduction + swap words for Classical Chinese chars (CJK-tokenizer models only)
/turo gloss off      # disable a lossy stage: filler | synonyms | gloss | defmatch | arrows
/turo synonyms on    # re-enable a stage
/turo defmatch off   # disable the defmatch stage (definition-like phrase -> headword)
/turo arrows off     # disable the arrow stage (connective phrases -> "->")
```

Install-time controls via environment variables:

```yaml
TURO_LEVEL: ultra    # default compression level (lite, full, ultra)
TURO_FILLER: "off"   # skip stage 1 (filler deletion)
TURO_DEFMATCH: "off" # skip stage 2 (defmatch) - keeps definition-like phrases intact
TURO_GLOSS: "off"    # skip stage 3 (gloss swap) - the lossiest stage
TURO_SYNONYMS: "off" # skip stage 4 (synonym swap) - keeps wording closer to source
TURO_ARROWS: "off"   # skip the arrow stage (connective phrases -> "->")
KDEPS_TURO: "off"    # disable turo entirely (also TURO_DISABLED=1)
KDEPS_TURO_PATH: /custom/path/to/turo  # override binary discovery
```

To keep agent context faithful (drop filler only, no wording changes), set `TURO_SYNONYMS=off` and `TURO_GLOSS=off`.

## See Also

- [Agent Loop Mode](/modes/agent-loop-mode) -- overview and starting the REPL
- [REPL Slash Commands](/modes/agent-loop-commands) -- full command reference
