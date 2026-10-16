# Transcribe component

The `transcribe` component converts speech in an audio file to text with a Whisper model - OpenAI's API, Groq's API, a self-hosted OpenAI-compatible server, or fully offline through a local `whisper-cli` binary with no API key. It is a kdeps component (YAML, no Go code) from the `components` asset set: call it with a `component:` resource, no install step.

## Where it runs

Both [workflow mode](/workflow/) and [agent mode](/agent/). In agent mode the same job is two built-in tools: `transcribe_audio` (the API backends, a [declared HTTP tool](/agent/tools#declared-http-tools)) and `transcribe_audio_offline` (whisper.cpp, a [declared exec tool](/agent/tools#declared-exec-tools)).

## How it runs

```d2
direction: right
call: "component: transcribe" {shape: oval}
pick: "backend?" {shape: diamond}
api: "httpClient POST\n<base>/audio/transcriptions\n(multipart: file + fields)"
model: "httpClient GET model\nsaveTo models dir\n(skipped when cached)"
cli: "exec whisper-cli\n-m model -f file -l lang"
call -> pick
pick -> api: "openai | groq | local"
pick -> model: "whisper-cpp"
model -> cli
```

Each step is a resource of the component gated by `validations.skip`, so only the chosen backend's steps run.

## Basic usage

```yaml
# resources/transcribe.yaml
actionId: transcribe
name: Transcribe Audio
component:
  name: transcribe           # from the components asset set; a components/transcribe/ of your own wins
  with:
    file: /data/meeting.mp3  # required
    backend: openai          # openai (default) | groq | local | whisper-cpp
```

```yaml
# resources/respond.yaml
actionId: respond
requires: [transcribe]
apiResponse:
  success: true
  response:
    text: "{{ output('transcribe').body }}"   # API backends: the httpClient result's body
    # offline: "{{ output('transcribe').stdout }}" - the whisper-cli exec result
```

The API key comes from `OPENAI_API_KEY` (openai) or `GROQ_API_KEY` (groq); `local` sends none. A non-2xx answer fails the resource with the status and body.

## Offline with whisper.cpp

```yaml
component:
  name: transcribe
  with:
    file: /data/meeting.wav     # whisper-cpp takes flac, mp3, ogg, wav
    backend: whisper-cpp        # no API key, no network after the model is cached
    language: en                # default en
    # modelPath: /models/ggml-small.en.bin   # default: <models dir>/ggml-base.en.bin
```

On first use the component downloads `ggml-base.en.bin` from Hugging Face into `$KDEPS_MODELS_DIR` (default `~/.kdeps/models`); later runs skip that step. `whisper-cli` must be on `PATH` (`brew install whisper-cpp`, or build whisper.cpp).

## Inputs

| Input | Default | Description |
|---|---|---|
| `file` | (required) | Audio/video file. API backends: mp3, mp4, mpeg, mpga, m4a, wav, webm |
| `backend` | `openai` | `openai`, `groq`, `local` or `whisper-cpp` |
| `model` | `whisper-1` | API model (Groq: `whisper-large-v3`) |
| `baseURL` | per backend | Overrides the API base URL (`.../v1`) |
| `language` | auto (`en` offline) | ISO-639-1 code |
| `prompt` | | Spelling hints for names and terms (API) |
| `responseFormat` | `text` | `text`, `json`, `verbose_json`, `srt`, `vtt` (API) |
| `temperature` | | 0-1 (API) |
| `modelPath` | `<models dir>/ggml-base.en.bin` | whisper-cpp GGML model |

## Output

API backends return the `httpClient` result - the transcript is `body` (and `data` when `responseFormat` is JSON). whisper-cpp returns the `exec` result - the transcript is `stdout`.

## See also

- [Component resource](/agencies/component-resource) - how `component:` calls pass `with:` inputs
- [HTTP client resource](/workflow/resources/http-client#file-upload-download-and-failing-on-errors) - `files`, `saveTo`, `failOnError`
- [OCR component](/workflow/resources/ocr) - the same pattern for images
