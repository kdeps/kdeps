# Transcribe components

Two kdeps components turn audio into text. `transcribe_audio` calls a Whisper API (OpenAI, Groq, or a self-hosted OpenAI-compatible server). `transcribe_audio_offline` runs a local `whisper-cli` binary, with no API key or network.

```yaml
# resources/transcribe.yaml
actionId: transcribe
name: Transcribe Audio
component:
  name: transcribe_audio        # from the components asset set; a components/transcribe_audio/ of your own wins
  with:
    file: /data/meeting.mp3     # required
    backend: openai             # openai (default) | groq | local
```

## Read next

- [Transcribe components reference](/reference/resources/transcribe) - every key, option and detail
- [Component resource](/agencies/component-resource) - how `component:` calls pass `with:` inputs
- [OCR component](/workflow/resources/ocr) - the same pattern for images
