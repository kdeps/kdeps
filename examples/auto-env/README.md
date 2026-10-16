# auto-env

Demonstrates kdeps **automatic environment variable scoping** for components.

## What It Shows

| Feature | Description |
|---------|-------------|
| Scoped env vars | `TRANSLATOR_OPENAI_API_KEY` overrides `OPENAI_API_KEY` inside `translator` only |
| `.env` fallback | Values in a `.env` you put in a component's folder are the lowest-priority source; kdeps never creates or writes it |

## Structure

```
auto-env/
├── workflow.yaml
├── components/
│   ├── translator/
│   │   └── component.yaml     # uses env('OPENAI_API_KEY')
│   └── summarizer/
│       └── component.yaml     # uses env('OPENAI_API_KEY')
└── resources/
    ├── 01-translate.yaml
    ├── 02-summarize.yaml
    └── 03-response.yaml
```

## Priority: How `env('OPENAI_API_KEY')` Resolves

When the `translator` component calls `env('OPENAI_API_KEY')`:

```
1. TRANSLATOR_OPENAI_API_KEY   ← scoped os env (highest priority)
2. OPENAI_API_KEY              ← plain os env
3. translator/.env             ← component .env file (lowest priority)
```

When the `summarizer` component calls `env('OPENAI_API_KEY')`:

```
1. SUMMARIZER_OPENAI_API_KEY   ← scoped os env (highest priority)
2. OPENAI_API_KEY              ← plain os env
3. summarizer/.env             ← component .env file (lowest priority)
```

No YAML changes needed — the prefix is derived automatically from the component name.

## Setup

```bash
# Global fallback key (used by both components)
export OPENAI_API_KEY=sk-global

# Optional: component-specific overrides
export TRANSLATOR_OPENAI_API_KEY=sk-translate-key   # translator only
export SUMMARIZER_OPENAI_API_KEY=sk-summary-key     # summarizer only
```

If no API key is set:
- `translator` returns an error field instead of a translation
- `summarizer` falls back to a simple extractive summary (no API call)

## Run

```bash
kdeps run examples/auto-env
```

## Optional: a component `.env`

For a fallback value per component, create the file yourself. kdeps only reads it, never creates or changes it. It holds keys, so keep it out of git:

```bash
echo 'OPENAI_API_KEY=sk-translate' > examples/auto-env/components/translator/.env
echo '.env' >> .gitignore
```

## Naming Convention

The scoped prefix is the component name uppercased with non-alphanumeric characters replaced by `_`:

| Component name | Scoped prefix |
|---------------|---------------|
| `scraper` | `SCRAPER_` |
| `my-bot` | `MY_BOT_` |
| `gpt.4` | `GPT_4_` |

## Real-World Use Case

Use scoped env vars to rotate API keys per-component without touching shared config:

```bash
# CI/CD: different keys per service
export SCRAPER_OPENAI_API_KEY=$SCRAPER_KEY
export EMBEDDING_OPENAI_API_KEY=$EMBED_KEY
export SUMMARIZER_OPENAI_API_KEY=$SUMMARY_KEY
export OPENAI_API_KEY=$DEFAULT_KEY   # fallback for everything else

kdeps run my-workflow
```

## Validate

```bash
kdeps validate examples/auto-env
```
