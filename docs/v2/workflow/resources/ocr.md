# OCR component

The `ocr` component extracts text from an image using [tesseract](https://github.com/tesseract-ocr/tesseract) - fully local, no API key, no network call. It is a kdeps component (YAML, no Go code) from the `components` asset set: call it with a `component:` resource, no install step.

## Where it runs

Both [workflow mode](/workflow/) and [agent mode](/agent/). In agent mode the same job is the `ocr_image` built-in tool, a [declared exec tool](/agent/tools#declared-exec-tools).

## Requirements

The `tesseract` binary on `PATH` (`apt install tesseract-ocr`, `brew install tesseract`). `kdeps build` installs `tesseract-ocr` in the image for you: the component's `setup.osPackages` lists it.

## Basic usage

```text
component: ocr  ->  exec: tesseract <file> stdout -l <language> --psm <psm> --oem <oem>  ->  .stdout = the text
```

```yaml
# resources/ocr.yaml
actionId: ocr
name: Extract Text
component:
  name: ocr                        # from the components asset set; a components/ocr/ of your own wins
  with:
    file: /data/scanned-invoice.png  # required
```

```yaml
# resources/respond.yaml
actionId: respond
requires: [ocr]
apiResponse:
  success: true
  response:
    text: "{{ output('ocr').stdout }}"   # the component returns an exec result; the text is .stdout
```

## Inputs

| Input | Default | Description |
|---|---|---|
| `file` | (required) | Path to the image: png, jpg/jpeg, tiff, bmp, gif. Text-layer PDFs go through the [loader](/workflow/resources/loader) instead |
| `language` | `eng` | tesseract `-l` value, e.g. `deu`, `eng+fra`. Needs the matching `tesseract-ocr-<lang>` data pack |
| `psm` | `3` | Page segmentation mode (`--psm`, `0`-`13`); `3` is fully automatic, `6` assumes one uniform block of text |
| `oem` | `3` | Engine mode (`--oem`, `0`-`3`); `3` uses whatever engine is available |

```yaml
component:
  name: ocr
  with:
    file: /data/receipt.jpg
    psm: 6   # a single uniform block of text
```

## Output

The `exec` result of the tesseract run:

```json
{"success": true, "exitCode": 0, "stdout": "kdeps OCR Sample\n", "stderr": "", "command": "tesseract ..."}
```

## The component itself

`kdeps update components/ocr` refreshes it. To change how it runs, copy it into your project's `components/ocr/component.yaml` - a local component with the same name wins:

```yaml
# components/ocr/component.yaml (as shipped, abridged)
apiVersion: kdeps.io/v1
kind: Component
metadata:
  name: ocr
  targetActionId: ocrText
setup:
  osPackages: [tesseract-ocr]   # installed by kdeps build
  provides: [tesseract]         # already on PATH: no package manager runs
interface:
  inputs:
    - {name: file, type: string, required: true}
    - {name: language, type: string, default: eng}
resources:
  - actionId: ocrText
    exec:
      command: tesseract
      args: ["{{ get('ocr.file') }}", stdout, "-l", "{{ get('ocr.language') }}"]
      timeout: 60s
```

## See also

- [Component resource](/agencies/component-resource) - how `component:` calls pass `with:` inputs
- [Transcribe component](/workflow/resources/transcribe) - the equivalent offline pattern for audio (`whisper-cpp` backend)
- [Resources overview](/workflow/resources) - all resource types
