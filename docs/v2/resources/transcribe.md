# Transcribe resource

The `transcribe:` resource converts speech in an audio file to text using a Whisper model - OpenAI's API, Groq's API, a self-hosted OpenAI-compatible server, or fully offline via a local `whisper-cli` binary with no API key at all.

## Where it runs

Both [workflow mode](/modes/workflow-mode) and [agent mode](/modes/agent-loop-mode). In agent mode, the same executor is available as the `transcribe_audio` built-in tool.

## Basic usage

```yaml
# resources/transcribe.yaml
actionId: transcribe
name: Transcribe Audio
transcribe:
  file: /data/meeting.mp3
  backend: openai
```

```yaml
# resources/respond.yaml
actionId: respond
requires: [transcribe]
apiResponse:
  success: true
  response:
    text: "{{ output('transcribe') }}"
```

## Backends

| `backend` | Requires | Formats | Notes |
|---|---|---|---|
| `openai` (default) | `OPENAI_API_KEY` | mp3, mp4, mpeg, mpga, m4a, wav, webm | Whisper API, model default `whisper-1` |
| `groq` | `GROQ_API_KEY` | mp3, mp4, mpeg, mpga, m4a, wav, webm | Faster/cheaper Whisper-compatible API, e.g. `whisper-large-v3` |
| `local` | - | mp3, mp4, mpeg, mpga, m4a, wav, webm | Self-hosted OpenAI-compatible Whisper HTTP server, `baseURL` points at it |
| `whisper-cpp` | local `whisper-cli` binary on PATH | flac, mp3, ogg, wav only | Fully offline - no API key, no network after the model is cached |

## Offline transcription (`whisper-cpp`)

```yaml
transcribe:
  file: /data/meeting.mp3
  backend: whisper-cpp
```

No config beyond `file` and `backend` is required. On first use, a default English model (`ggml-base.en.bin`, ~140MB) auto-downloads to `~/.kdeps/models/` - the same cache directory used by `chat:`'s llamafile models - and every later call reuses the cached file. Requires the `whisper-cli` binary from [whisper.cpp](https://github.com/ggerganov/whisper.cpp) to be installed and on `PATH`.

Use `modelPath` to point at a different GGML model (a multilingual or larger model for better accuracy):

```yaml
transcribe:
  file: /data/interview.wav
  backend: whisper-cpp
  modelPath: /models/ggml-medium.bin
  language: en
```

## Configuration options

| Option | Applies to | Description |
|---|---|---|
| `file` | all | Path to the audio file (required) |
| `backend` | all | `openai` (default), `groq`, `local`, or `whisper-cpp` |
| `model` | openai, groq | Model name. Default `whisper-1`; Groq: `whisper-large-v3` |
| `baseURL` | local | Base URL of the self-hosted server. Ignored for `whisper-cpp` |
| `modelPath` | whisper-cpp | Path to a GGML model file. Default: auto-downloaded `ggml-base.en.bin` |
| `language` | all | ISO-639-1 language hint, e.g. `en` |
| `prompt` | openai, groq, local | Optional context prompt to guide transcription style/vocabulary |
| `responseFormat` | openai, groq, local | `text` (default), `json`, `srt`, `verbose_json`, `vtt` |
| `temperature` | openai, groq, local | Sampling temperature, `0`-`1` |
| `timestampGranularities` | openai, groq, local | `["segment"]` and/or `["word"]`, only with `responseFormat: verbose_json` |

## Output

The transcribed text, as a plain string:

```json
"This is a kdeps transcription test."
```

## See also

- [OCR Resource](ocr) - the equivalent offline, no-API-key pattern for images
- [Resources Overview](overview) - all resource types
