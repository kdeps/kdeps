# component-setup-teardown

Demonstrates **component setup and teardown lifecycle hooks** — automatic dependency installation and per-run cleanup.

## Structure

```
component-setup-teardown/
├── workflow.yaml
├── components/
│   └── word-counter/
│       └── component.yaml    # setup + teardown hooks
└── resources/
    ├── 01-count-intro.yaml
    ├── 02-count-poem.yaml
    └── 03-response.yaml
```

## Usage

```bash
kdeps run examples/component-setup-teardown
```

## What Happens on First Run

kdeps reads the `setup` block and automatically installs dependencies:

```
[setup] word-counter v1.0.0
  • pip install nltk             ← pythonPackages
  • apt-get install -y wc        ← osPackages (skipped if already present)
  • python3 -c "import nltk; nltk.download('punkt', quiet=True)"  ← commands
[setup] complete — cached for this version
```

On subsequent runs the setup cache is hit and setup is skipped entirely.

## Setup Block

```yaml
setup:
  pythonPackages:
    - nltk                        # installed via uv pip into isolated venv
  osPackages:
    - wc                          # installed via apt-get / apk / brew
  commands:
    - "python3 -c \"import nltk; nltk.download('punkt', quiet=True)\""
```

- **`pythonPackages`** — pip packages installed into the kdeps-managed Python venv
- **`osPackages`** — system packages; installer is auto-detected (apt-get / apk / brew); already-installed packages are skipped
- **`commands`** — shell commands run once after packages are ready

## Teardown Block

```yaml
teardown:
  commands:
    - "rm -f /tmp/word-counter-*.tmp"
```

Teardown runs **after every invocation**, not just the first. Use it for:
- Deleting temporary files
- Releasing locks
- Resetting per-run state

## Setup vs Teardown Timing

| Hook | When it runs |
|------|-------------|
| `setup` | Once per component version (cached) |
| `teardown` | After every component invocation |

## Sample Output

```json
{
  "intro_stats": "{\"label\":\"intro\",\"word_count\":18,\"sentence_count\":2,\"unique_words\":18,\"avg_word_len\":4.28}",
  "poem_stats":  "{\"label\":\"poem\",\"word_count\":22,\"sentence_count\":3,\"unique_words\":19,\"avg_word_len\":3.86}"
}
```

## Validate

```bash
kdeps validate examples/component-setup-teardown
```
