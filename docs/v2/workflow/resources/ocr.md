# OCR component

The `ocr_image` component extracts text from an image with [tesseract](https://github.com/tesseract-ocr/tesseract). It is fully local, with no API key and no network call.

```text
image -> tesseract (local) -> text
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

- [OCR component reference](/reference/resources/ocr) - every key, option and detail
- [Component resource](/agencies/component-resource) - how `component:` calls pass `with:` inputs
- [Transcribe components](/workflow/resources/transcribe) - the equivalent pattern for audio
