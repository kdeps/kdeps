# Media resources

Two resources that pull text out of a media file. Each has its own reference
page. OCR and offline transcription need no API key and no network.

Works in workflow and agent mode.

| Resource | Extracts text from | Reference |
| :--- | :--- | :--- |
| `component: transcribe_audio` | Audio (speech to text via a Whisper API) | [Transcribe](/workflow/resources/transcribe) |
| `component: transcribe_audio_offline` | Audio (speech to text offline via whisper.cpp) | [Transcribe](/workflow/resources/transcribe) |
| `component: ocr_image` | Images (printed or handwritten text via tesseract) | [OCR](/workflow/resources/ocr) |

## See also

- [LLM resource](/workflow/resources/llm) - vision models can read images directly in a prompt
- [Image analysis tutorial](/examples/vision) - `files:` on a chat prompt instead of OCR
- [RAG resources](/workflow/resources/rag) - index the extracted text for search
- [Resources overview](/workflow/resources) - all resource types
