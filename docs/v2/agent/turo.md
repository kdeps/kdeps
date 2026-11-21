# Prompt reduction (turo)

`turo` is an optional token reducer. A workflow chat turns it on with `harness.turo: true` (see [Answer quality](/workflow/quality)). When it is on and the `turo` binary is on `PATH`, kdeps sends all LLM input through `turo` first. This input is the system preamble, your input, tool results, and conversation history. `turo` keeps code, file paths, and identifiers exactly as they are. If a reduction is not smaller than the input, the original passes through unchanged.

```text
system preamble + input + tool results + history  ->  turo  ->  LLM
```

turo runs a five-stage pipeline, all on by default, repeating until the output stops shrinking:

1. **Filler deletion** - strips pleasantries, hedges, leaders, and articles (`please`, `I think`, `of course`).
2. **Defmatch** - collapses a definition-like phrase into the word it defines (`the state of disorder and lawlessness` -> `anarchy`). This stage has strict conditions. It operates only when all keywords of the definition of a headword are present and the headword uses fewer tokens. On technical text, it usually makes no replacements. It is useful on natural prose.
3. **Gloss swap** - replaces words with the shortest defining word from their dictionary definition (`approach` -> `come`). The lossiest stage.
4. **Synonym swap** - replaces words with a WordNet synonym that uses fewer tokens (`utilize` -> `use`).
5. **Reduction** - keeps content words by part of speech, deduplicates, and (ultra) collapses inflections by lemma.

Phrase stages run before word stages. A phrase matcher must see the full phrase. One earlier swap in the phrase can stop the match. The later swaps do not change the headwords that defmatch makes. Otherwise, they can change the match back.

One more stage runs before the rest, also on by default:

- **Arrows** - replaces multi-word causal connectives (`leads to`, `results in`, `gives rise to`) with `->` (`cache miss leads to slow query` -> `cache miss -> slow query`). Only multi-word phrases qualify, so it always saves a token. Disable with `TURO_ARROWS=off`.

Stages 2-5 and arrows are lossy. They change words, and do not only remove filler words. For this reason, the context that the model gets is compressed. It is not the original prose. To disable one stage, use the `TURO_*` environment variables below. You can also turn off turo fully.

Turo is optional. If the binary is not installed, kdeps sends everything unreduced.

Environment variables that control the installation:

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

## See also

- [Agent loop](/agent/) - overview
