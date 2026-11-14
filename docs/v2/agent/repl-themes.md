# Themes

The REPL look is `normal` or a disguise. It changes rendering only. Prompts, responses, and logs stay the same. The other REPL behaviors stay on [REPL features](/agent/repl).

## Picking a theme

`/theme <name>` (or `--theme <name>`, or `KDEPS_THEME=<name>`) changes the whole look immediately: banner, prompt, typed text, model name, responses, thinking blocks, tool summaries. `normal` is the bright default. Every other theme is a disguise that does not read as "an AI session on model X" to someone glancing at your screen.

The `/model`, `/settings` and startup resume pickers are separate full-screen views but use the same theme colors (accent, success, warning, dim, bold). `vim` gives them yellow/green/red accents; `black` makes them flat gray.

```bash
kdeps --theme black             # start disguised
```

```text
/theme              show the current theme and the list of valid names
/theme list         same as bare /theme - built-in and custom names shown separately
/theme vim          switch themes - normal, black, linux, vim, emacs, or a custom name
```

| Theme | Look |
|---|---|
| `normal` (default) | The bright default palette - no disguise |
| `black` | A single flat, legible dark gray (`#767676`) for every element - muted and monochrome, but readable, not near-invisible |
| `linux` | Plain, monochrome-ish light-gray-on-black, like a default terminal with no syntax highlighting |
| `vim` | vim's classic default colorscheme conventions (yellow keywords, cyan identifiers, red strings) and a `: ` command-line prompt |
| `emacs` | A common terminal-Emacs highlight set (purple keywords, blue functions, salmon strings) and an `M-x ` prompt |

## Precedence

`/theme <name>` writes `theme: <name>` to `~/.kdeps/agent-loop-settings.yaml`, so the next `kdeps` starts with it too. Precedence: `--theme` flag, then `KDEPS_THEME`, then the persisted setting, then `normal`.

## Model name

Every theme but `normal` shortens the model name in the status line to initials: `claude-sonnet-5` becomes `CS5`, `llama3.2:1b` becomes `L21`. Override it with `/model name`:

```text
/model name              show the current mode
/model name show         always show the literal model name, regardless of theme
/model name hide         omit the model name from the modeline entirely
/model name abbreviate   always abbreviate, even under normal
/model name auto         back to the theme-based default (the factory setting)
```

`/model name <mode>` persists to `~/.kdeps/agent-loop-settings.yaml` the same way `/theme` does.

## The spinner

The spinner never shows a word like "generating" or "thinking" in any theme: only the glyph and the token counter.

## Context-path status line

If kdeps knows the context window of the current model, a line above the session counter shows the parts of the prompt for this turn:

```
[turn 12.4k/200k | sys 8.1k | mem 1.2k | hist 2.4k | bash_exec 700]
[session sent 1.2m | generated 30 | total 1.2m | this turn +47.2k sent +400 generated]
```

`turn A/B` is this prompt's size over the model's context window. The groups after it are that same prompt, split: `sys` system prompt, `mem` memory, `hist` conversation history, and one entry per tool name. Repeated calls to the same tool add to that one number. Past 5 groups the oldest collapse into `+N`. The line resets with each new prompt.

## The session row

The `session` row is cumulative and never measured against the window; only `turn` is.

| Field | Meaning |
|---|---|
| `sent` | prompt tokens sent this session. History is resent every round, so it outgrows the window; it is not context fullness |
| `generated` | tokens the model wrote this session, including reasoning |
| `total` | `sent` + `generated` |
| `this turn` | the sent/generated delta since your last message; shown once the turn spends anything |
| `web`, `sh`, `file`, `src` | `used/limit` of a call budget, shown after its first call |

- `sent` and `generated` update when a reply streams and when a tool runs.
- `/compact` and `/fold` reset both to the remaining context: `sent` is that context, `generated` the model-written part of it.
- The two rows are one frame, rewritten in place each tick. Rows wider than the terminal are cut. Non-terminal output reprints the rows only when numbers change.

## The turn figure

`turn` is the prompt size of the latest call: the provider's count when the backend reports one, else the size of the messages sent. The parts (`sys`, `mem`, `hist`, tools) are pre-call estimates scaled to that total; the rest (your message, tool schemas, framing) shows as `other`. Before a turn's first call, parts are plain estimates. The modeline `used/limit` shows the same figure in the same format (`20.7k/128.0k`).

## Custom themes

Every theme - built-in or not - is a YAML file. Drop your own into `~/.kdeps/themes/<name>.yaml` and it shows up in `/theme`'s list immediately:

```yaml
# ~/.kdeps/themes/solarized.yaml
name: solarized     # optional - defaults to the filename without its extension
prompt: "$ "         # the literal prompt text this theme renders
bold: true
palette:
  heading: "#b58900"
  text: "#839496"
  # any field you omit falls back to the "normal" theme's value, so a
  # custom theme can override just a couple of accent colors and leave
  # everything else alone
```

To override a built-in theme, give a custom theme the same name (for example, your own `vim.yaml`). If one field has an invalid color value, kdeps ignores that field and shows a warning. The remaining part of the file loads.

Every `palette:` field is optional: `heading`, `link`, `code`, `codeBlock`, `text`, `thinking`, `muted`, `bullet`, `quote`, `borderHr`, `synKeyword`, `synFunc`, `synStr`, `synComment`, `synNum`, `synType`, `synOp`, `replError`, `replMeta`, `replHeading`, `replSuccess`, `replPrompt`, `replInfo`, `replDim`, `bannerText`, `bannerBorder`, `modelsReady`, `modelsNoKey`, `modelsCurrent`, and `modelName` (the model name shown in the status line).
