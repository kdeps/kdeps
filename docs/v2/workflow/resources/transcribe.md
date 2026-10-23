# Transcribe components

Two kdeps components turn audio into text: `transcribe_audio` calls a Whisper API (OpenAI, Groq, or a self-hosted OpenAI-compatible server), and `transcribe_audio_offline` runs a local `whisper-cli` binary with no API key and no network. Both come from the `components` asset set: call them with a `component:` resource, no install step. Each is a thin client over the kdeps API tools of the same name.

## Where it runs

Both [workflow mode](/workflow/) and [agent mode](/agent/). In agent mode the same jobs are the built-in tools `transcribe_audio` (a [declared HTTP tool](/agent/tools#declared-http-tools)) and `transcribe_audio_offline` (a [declared exec tool](/agent/tools#declared-exec-tools)); the components call the same two routes.

## How it runs

```d2
direction: right
apiCall: "component: transcribe_audio" {shape: oval}
pick: "backend?" {shape: diamond}
api: "POST /tools/transcribe_audio/call\nfile + backend + model"
offlineCall: "component: transcribe_audio_offline" {shape: oval}
cli: "POST /tools/transcribe_audio_offline/call\n-> exec whisper-cli\n-m model -f file -l lang"
apiCall -> pick
pick -> api: "openai | groq | local"
offlineCall -> cli
```

Each component is one resource that posts to its tool route. The API tool picks the endpoint from `backend`; the offline tool runs `whisper-cli`.

## Whisper API

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

```yaml
# resources/respond.yaml
actionId: respond
requires: [transcribe]
apiResponse:
  success: true
  response:
    text: "{{ output('transcribe') }}"   # the transcript
```

The API key comes from `OPENAI_API_KEY` (openai) or `GROQ_API_KEY` (groq); `local` sends `OPENAI_API_KEY` when it is set - some self-hosted servers check it - and no key otherwise. A non-2xx answer fails the resource with the status and body.

| Input | Default | Description |
|---|---|---|
| `file` | (required) | Audio/video file: mp3, mp4, mpeg, mpga, m4a, wav, webm |
| `backend` | `openai` | `openai`, `groq`, or `local` |
| `model` | `whisper-1` | API model (Groq: `whisper-large-v3`) |
| `baseURL` | per backend | Overrides the API base URL (`.../v1`) |
| `language` | auto-detect | ISO-639-1 code |
| `prompt` | | Spelling hints for names and terms |
| `responseFormat` | `text` | `text`, `json`, `verbose_json`, `srt`, `vtt` |
| `temperature` | | 0-1 |

## Offline with whisper.cpp

```yaml
component:
  name: transcribe_audio_offline
  with:
    file: /data/meeting.wav     # whisper-cli takes flac, mp3, ogg, wav
    language: en                # default en
    # modelPath: /models/ggml-small.en.bin   # default: <models dir>/ggml-base.en.bin
```

`whisper-cli` must be on `PATH` (`brew install whisper-cpp`, or build whisper.cpp); when it is missing, kdeps installs `whisper-cpp` from the tool asset's `exec.install`. The model must already be at `modelPath`, or at `ggml-base.en.bin` in the models directory (`$KDEPS_MODELS_DIR`, default `~/.kdeps/models`) - download it from [ggerganov/whisper.cpp](https://huggingface.co/ggerganov/whisper.cpp) once.

| Input | Default | Description |
|---|---|---|
| `file` | (required) | Audio file: flac, mp3, ogg, wav |
| `language` | `en` | ISO-639-1 code |
| `modelPath` | `<models dir>/ggml-base.en.bin` | Absolute path to a GGML model file |

## Output

Both components return the transcript as a string:

```json
"the meeting transcript\n"
```

A missing `whisper-cli`, model file, or API key fails the resource with the reason in the error body.

## How it is built

`transcribe_audio` is one resource that posts the inputs to its tool route:

<div v-pre>

```yaml
# assets/components/transcribe_audio/component.yaml (excerpt)
interface:
  inputs:
    - {name: file, type: string, required: true}
    - {name: backend, type: string, default: openai}
    - {name: model, type: string, default: whisper-1}
    # language, baseURL, prompt, responseFormat, temperature
  output: "{{ output('call').body }}"   # the transcript
resources:
  - actionId: call
    httpClient:
      method: POST
      url: http://kdeps.api/apis/tools.kdeps.io/v1/tools/transcribe_audio/call
      headers: {Content-Type: application/json}
      timeout: 300s
      failOnError: true
      data:
        file: "{{ get('transcribe_audio.file') }}"
        backend: "{{ get('transcribe_audio.backend') }}"
```

</div>

`transcribe_audio_offline` has the same shape and posts `{file, language, modelPath}` to `transcribe_audio_offline`. Copy `components/transcribe_audio/` or `components/transcribe_audio_offline/` into your project to change the backend set or the model source.

## See also

- [Component resource](/agencies/component-resource) - how `component:` calls pass `with:` inputs
- [OCR component](/workflow/resources/ocr) - the same pattern for images
- [Resources overview](/workflow/resources) - all resource types
