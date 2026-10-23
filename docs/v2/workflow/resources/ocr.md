# OCR component

The `ocr_image` component extracts text from an image using [tesseract](https://github.com/tesseract-ocr/tesseract) - fully local, no API key, no network call. It is a kdeps component from the `components` asset set: call it with a `component:` resource, no install step. The component is a thin client: the tesseract call itself is declared in the `ocr_image` tool asset and runs in the kdeps API.

## Where it runs

Both [workflow mode](/workflow/) and [agent mode](/agent/). In agent mode the same tool is the `ocr_image` built-in, a [declared exec tool](/agent/tools#declared-exec-tools); the component and the agent both reach it through the kdeps API.

## Requirements

The `tesseract` binary on `PATH` (`apt install tesseract-ocr`, `brew install tesseract`). When it is missing, kdeps installs `tesseract-ocr` from the tool asset's `exec.install` on the next run, and `kdeps build` bakes it into the image.

## Basic usage

```text
component: ocr_image -> POST http://kdeps.api/apis/tools.kdeps.io/v1/tools/ocr_image/call
                    -> tools/ocr_image.yaml exec: tesseract <file> stdout -l <language> --psm <psm> --oem <oem>
                    -> .stdout = the text
```

```yaml
# resources/ocr.yaml
actionId: ocr
name: Extract Text
component:
  name: ocr_image                  # from the components asset set; a components/ocr_image/ of your own wins
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
    text: "{{ output('ocr') }}"   # the component output is the extracted text
```

## Inputs

| Input | Default | Description |
|---|---|---|
| `file` | (required) | Path to the image: png, jpg/jpeg, tiff, bmp, gif. Text-layer PDFs go through the [load_document component](/workflow/resources/loader) instead |
| `language` | `eng` | tesseract `-l` value, e.g. `deu`, `eng+fra`. Needs the matching `tesseract-ocr-<lang>` data pack |
| `psm` | `3` | Page segmentation mode (`--psm`, `0`-`13`); `3` is fully automatic, `6` assumes one uniform block of text |
| `oem` | `3` | Engine mode (`--oem`, `0`-`3`); `3` uses whatever engine is available |

```yaml
component:
  name: ocr_image
  with:
    file: /data/receipt.jpg
    psm: 6   # a single uniform block of text
```

## Output

The extracted text, as a string:

```json
"kdeps OCR Sample\n"
```

A missing `tesseract` binary fails the call: the API returns `400`, and the component's `failOnError: true` stops the workflow.

## The component itself

`kdeps update components/ocr_image` refreshes it. To change how it runs, copy it into your project's `components/ocr_image/component.yaml` - a local component with the same name wins:

```yaml
# components/ocr_image/component.yaml (as shipped, abridged)
apiVersion: kdeps.io/v1
kind: Component
metadata:
  name: ocr_image
interface:
  inputs:
    - {name: file, type: string, required: true}
    - {name: language, type: string, default: eng}
    - {name: psm, type: integer, default: 3}
    - {name: oem, type: integer, default: 3}
  output: "{{ output('call').body }}"   # the tool returns the text
resources:
  - actionId: call
    httpClient:
      method: POST
      url: http://kdeps.api/apis/tools.kdeps.io/v1/tools/ocr_image/call
      headers: {Content-Type: application/json}
      failOnError: true
      timeout: 60s
      data:
        file: "{{ get('ocr_image.file') }}"
        language: "{{ get('ocr_image.language') }}"
        psm: "{{ get('ocr_image.psm') }}"
        oem: "{{ get('ocr_image.oem') }}"
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
- [Transcribe components](/workflow/resources/transcribe) - the equivalent pattern for audio
- [Resources overview](/workflow/resources) - all resource types
