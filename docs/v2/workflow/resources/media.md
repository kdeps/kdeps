# Media resources

Two resources that pull text out of a media file. Each has its own reference
page. Both can run fully offline with no API key.

*Applies to both workflow mode and agent mode.*

| Resource | Extracts text from | Reference |
| :--- | :--- | :--- |
| `transcribe:` | Audio (speech to text via Whisper) | [Transcribe](/workflow/resources/transcribe) |
| `ocr:` | Images (printed or handwritten text via tesseract) | [OCR](/workflow/resources/ocr) |

## See also

- [LLM resource](/workflow/resources/llm) - vision models can read images directly in a prompt
- [Image analysis tutorial](/examples/vision) - `files:` on a chat prompt instead of OCR
- [RAG resources](/workflow/resources/rag) - index the extracted text for search
- [Resources overview](/workflow/resources) - all resource types
