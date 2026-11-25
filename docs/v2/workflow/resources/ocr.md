# OCR component

The `ocr_image` component extracts text from an image with [tesseract](https://github.com/tesseract-ocr/tesseract). It is fully local, with no API key and no network call.

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

## Read next

- [OCR component reference](/reference/resources/ocr) - basic usage, inputs, output, the component itself
- [Component resource](/agencies/component-resource) - how `component:` calls pass `with:` inputs
- [Transcribe components](/workflow/resources/transcribe) - the equivalent pattern for audio
- [Resources overview](/workflow/resources) - all resource types
