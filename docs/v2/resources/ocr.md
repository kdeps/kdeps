# OCR resource

The `ocr:` resource extracts text from an image using [tesseract](https://github.com/tesseract-ocr/tesseract) - fully local, no API key, no network call.

## Where it runs

Both [workflow mode](/modes/workflow-mode) and [agent mode](/modes/agent-loop-mode). In agent mode, the same executor is available as the `ocr_image` built-in tool.

## Requirements

Requires the `tesseract` binary on `PATH` (`apt install tesseract-ocr`, `brew install tesseract`). kdeps shells out to it - there is no bundled OCR engine.

## Basic usage

```yaml
# resources/ocr.yaml
actionId: ocr
name: Extract Text
ocr:
  file: /data/scanned-invoice.png
```

```yaml
# resources/respond.yaml
actionId: respond
requires: [ocr]
apiResponse:
  success: true
  response:
    text: "{{ output('ocr') }}"
```

## Configuration options

| Option | Description |
|---|---|
| `file` | Path to the image file (required) |
| `language` | Tesseract language code, e.g. `eng` (default), `deu`, `fra`. Requires the matching `tesseract-ocr-<lang>` data pack to be installed |
| `psm` | Page Segmentation Mode (tesseract `--psm`), integer `0`-`13`. Controls how tesseract assumes text is laid out on the page |
| `oem` | OCR Engine Mode (tesseract `--oem`), integer `0`-`3`. Selects the legacy engine, LSTM engine, or both |

```yaml
ocr:
  file: /data/receipt.jpg
  language: eng
  psm: 6   # assume a single uniform block of text
```

## Output

The extracted text, as a plain string:

```json
"kdeps OCR Sample"
```

## See also

- [Transcribe resource](transcribe) - the equivalent offline, no-API-key pattern for audio (`whisper-cpp` backend)
- [Resources overview](overview) - all resource types
