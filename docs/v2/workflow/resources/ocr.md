# OCR component

The `ocr` component extracts text from an image using [tesseract](https://github.com/tesseract-ocr/tesseract) - fully local, no API key, no network call. It is a kdeps component from the `components` asset set: call it with a `component:` resource, no install step. The component is a thin client: the tesseract call itself is declared in the `ocr_image` tool asset and runs in the kdeps API.

## Where it runs

Both [workflow mode](/workflow/) and [agent mode](/agent/). In agent mode the same tool is the `ocr_image` built-in, a [declared exec tool](/agent/tools#declared-exec-tools); the component and the agent both reach it through the kdeps API.

## Requirements

The `tesseract` binary on `PATH` (`apt install tesseract-ocr`, `brew install tesseract`). `kdeps build` installs `tesseract-ocr` in the image for you: the component's `setup.osPackages` lists it.

## Basic usage

```text
component: ocr -> POST http://kdeps.api/apis/tools.kdeps.io/v1/tools/ocr_image/call
               -> tools/ocr_image.yaml exec: tesseract <file> stdout -l <language> --psm <psm> --oem <oem>
               -> .stdout = the text
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

The `exec` result shape the component always returned, built from the tool's text:

```json
{"success": true, "exitCode": 0, "stdout": "kdeps OCR Sample\n", "stderr": ""}
```

A missing `tesseract` binary fails the call: the API returns `400`, and the component's `failOnError: true` stops the workflow.

## The component itself

`kdeps update components/ocr` refreshes it. To change how it runs, copy it into your project's `components/ocr/component.yaml` - a local component with the same name wins:

```yaml
# components/ocr/component.yaml (as shipped, abridged)
apiVersion: kdeps.io/v1
kind: Component
metadata:
  name: ocr
interface:
  inputs:
    - {name: file, type: string, required: true}
    - {name: language, type: string, default: eng}
    - {name: psm, type: integer, default: 3}
    - {name: oem, type: integer, default: 3}
  output:
    stdout: "{{ output('call').body }}"   # the tool returns the text
    stderr: ""
    exitCode: 0
    success: true
setup:
  osPackages: [tesseract-ocr]   # installed by kdeps build
  provides: [tesseract]         # already on PATH: no package manager runs
resources:
  - actionId: call
    httpClient:
      method: POST
      url: http://kdeps.api/apis/tools.kdeps.io/v1/tools/ocr_image/call
      headers: {Content-Type: application/json}
      failOnError: true
      timeout: 60s
      data:
        file: "{{ get('ocr.file') }}"
        language: "{{ get('ocr.language') }}"
        psm: "{{ get('ocr.psm') }}"
        oem: "{{ get('ocr.oem') }}"
```

The tesseract command line lives in `tools/ocr_image.yaml`, the same tool the agent loop uses:

```yaml
# assets/tools/ocr_image.yaml (as shipped, abridged)
parameters:
  file: {type: string, required: true}
  language: {type: string, default: eng}
  psm: {type: integer, default: 3}
  oem: {type: integer, default: 3}
exec:
  command: tesseract
  install: tesseract-ocr
  paths: [file]
  args:
    - "{{.args.file}}"
    - stdout
    - "-l"
    - "{{.args.language}}"
    - "--psm"
    - "{{.args.psm}}"
    - "--oem"
    - "{{.args.oem}}"
```

## See also

- [Component resource](/agencies/component-resource) - how `component:` calls pass `with:` inputs
- [Transcribe component](/workflow/resources/transcribe) - the equivalent offline pattern for audio (`whisper-cpp` backend)
- [Resources overview](/workflow/resources) - all resource types
