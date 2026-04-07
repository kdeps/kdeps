# csv-analyzer

Parses and analyzes CSV data: row/column counts, data types, summary stats,
and tabular display. Demonstrates component setup with `tabulate`.


Version: 1.0.0

## Usage

```yaml
run:
  component:
    name: csv-analyzer
    with:
      data: "" # Raw CSV text to analyze.  # required
      table_format: "" # Table style for display: 'simple' (default), 'grid', 'pipe', 'github', 'html'.
      max_rows: "" # Maximum rows to display (default: 10).
```

## Install

```bash
kdeps component install csv-analyzer
```
