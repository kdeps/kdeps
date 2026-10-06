# Themes

The REPL look is `normal` or a disguise. It changes rendering only. Prompts, responses, and logs stay the same. The other REPL behaviors stay on [REPL features](/agent/repl).

## Picking a theme

`/theme <name>` (or `--theme <name>`, or `KDEPS_THEME=<name>`) changes the REPL's entire look - banner, prompt, the text you type, model name, streamed responses, thinking blocks, tool summaries. `normal` is the bright default; every other theme is a disguise that no longer reads as "an AI session on model X" to anyone glancing at your screen in a cafe, on a plane, or in an open office. There's no separate on/off flag - picking a theme takes effect immediately.

The `/model` picker, the `/settings` picker, and the startup resume picker (`--resume`/session history) run outside the REPL's own text rendering, in their own full-screen views - but they pick up the exact same theme colors (accent, success, warning, dim, bold), not just a muted/bright toggle. Switching to `vim` recolors those pickers with vim's yellow/green/red accents; switching to `black` collapses them to the same flat gray as everything else.

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

Every theme but `normal` renders the model-name color at full legibility, so the literal model name is shortened to initials in the status line by default - `claude-sonnet-5` becomes `CS5`, `llama3.2:1b` becomes `L21` - hidden by content, not color. Override this with `/model name`:

```text
/model name              show the current mode
/model name show         always show the literal model name, regardless of theme
/model name hide         omit the model name from the modeline entirely
/model name abbreviate   always abbreviate, even under normal
/model name auto         back to the theme-based default (the factory setting)
```

`/model name <mode>` persists to `~/.kdeps/agent-loop-settings.yaml` the same way `/theme` does.

## The spinner

The spinner that appears while waiting for a response never carries a descriptive word in any theme - no "generating," no "thinking" - just the animated glyph and the token counter.

## Context-path status line

Directly above the session counter, whenever the running model's context window is known, a line shows what this turn's prompt is made of:

```
[turn 12.4k/200k | sys 8.1k | mem 1.2k | hist 2.4k | bash_exec 700]
[session sent 1.2m | generated 30 | total 1.2m | this turn +47.2k sent +400 generated]
```

`turn A/B` is this prompt's size over the model's context window. The groups after it are that same prompt, split: `sys` system prompt, `mem` memory, `hist` conversation history, `goal` the active task, and one entry per tool name. Repeated calls to the same tool add to that one number. Past 5 groups the oldest collapse into `+N`. The line resets with each new prompt.

## The session row

The `session` row is cumulative and is never measured against the window; only `turn` is. `sent` is the running total of prompt tokens handed to the model this session, `generated` is the running total the model wrote back, and `total` is their sum (everything spent this session, which is why it can exceed the window). `this turn` is the sent/generated delta since your last message; it appears once the turn has spent anything and resets on the next message. History is sent again on every round, so `sent` grows faster than one turn's window and is not "how full the context is". `generated` is the running total of tokens the model wrote back, including reasoning. Both move while a reply is still streaming and while a tool is running; they are not stuck at 0 until the call finishes. `/compact` and `/fold` replace both with the context that remains: `sent` is that context, `generated` is the model-written part still in it. `web`, `sh`, `file`, and `src` appear beside them only after a call in that budget, as `used/limit`. The turn line and this counter are one frame: each tick rewrites those rows in place.

## The turn figure

The `turn` figure is the prompt size the model actually received on its most recent call: the provider-reported count when the backend returns one, otherwise the size of the messages sent. The per-part numbers (`sys`, `mem`, `hist`, tools) are estimates taken before the call, so they are scaled to add up to that total, and anything they do not cover (your message, tool schemas, framing) shows as `other`. Before the first call of a turn the parts are shown as plain estimates. The modeline `used/limit` shows the same figure against the same window, in the same decimal format (`20.7k/128.0k`).

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

A custom theme can reuse a built-in's name (e.g. your own `vim.yaml`) to override it. An invalid color value in one field is dropped with a warning; the rest of the file still loads.

Every `palette:` field is optional; the full set is `heading`, `link`, `code`, `codeBlock`, `text`, `thinking`, `muted`, `bullet`, `quote`, `borderHr`, `synKeyword`, `synFunc`, `synStr`, `synComment`, `synNum`, `synType`, `synOp`, `replError`, `replMeta`, `replHeading`, `replSuccess`, `replPrompt`, `replInfo`, `replDim`, `bannerText`, `bannerBorder`, `modelsReady`, `modelsNoKey`, `modelsCurrent`, and `modelName` (the model name shown in the status line).
