# markdown-renderer

Renders Markdown to plain text or HTML, with optional section extraction.
Demonstrates component setup: installs the `mistune` Python package once.


Version: 1.0.0

## Usage

```yaml
run:
  component:
    name: markdown-renderer
    with:
      text: "" # Markdown source text to render.  # required
      output_format: "" # Output format: 'plain' (default) or 'html'.
```

## Install

```bash
kdeps component install markdown-renderer
```
