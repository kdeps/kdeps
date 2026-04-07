# tts

> Text-to-speech synthesis — offline via eSpeak or online via OpenAI TTS API.

The tts component converts text to speech. It provides two modes: an offline resource using the `espeak` system command (no API key needed), and an online resource using the OpenAI TTS API (requires an API key and network access).

## Inputs

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `text` | string | yes | — | Text to convert to speech |
| `voice` | string | no | `alloy` | Voice name for OpenAI TTS (`alloy`, `echo`, `fable`, `onyx`, `nova`, `shimmer`) |
| `apiKey` | string | no | — | OpenAI API key (required for `speak-online` resource) |

## Resources

The component exposes two independently callable resources:

| Resource | ActionId | Description |
|----------|----------|-------------|
| Speak Offline | `speak-offline` | Uses `espeak` — works without any API key or internet |
| Speak Online | `speak-online` | Uses OpenAI TTS API — requires `apiKey` |

## Outputs

### speak-offline

Accessed via `output('<your-action-id>')`:

```json
{
  "command": "espeak 'Hello world'",
  "exitCode": 0,
  "result": "",
  "stdout": "",
  "stderr": "",
  "success": true,
  "timedOut": false
}
```

Audio is played directly via `espeak` — there is no output file. The result indicates success/failure of the command.

### speak-online

The output is the raw OpenAI API response. The audio binary is returned in the HTTP response body — save it or stream it as needed.

## Usage

Offline TTS (no API key):

```yaml
run:
  component:
    name: tts
    with:
      text: "Hello! Your order has been confirmed."
  apiResponse:
    success: true
    response:
      ok: "{{ output('speakOffline').success }}"
```

Online TTS with OpenAI:

```yaml
run:
  component:
    name: tts
    with:
      text: "Welcome back! Here is your daily briefing."
      voice: nova
      apiKey: "{{ get('env.OPENAI_API_KEY') }}"
  apiResponse:
    success: true
    response:
      audio: "{{ output('speakOnline') }}"
```

Choosing a voice:

```yaml
run:
  component:
    name: tts
    with:
      text: "{{ output('summaryStep').text }}"
      voice: shimmer
      apiKey: "{{ get('env.OPENAI_API_KEY') }}"
```

## Requirements

### Offline (`speak-offline`)
- `espeak` must be installed on the system:
  - macOS: `brew install espeak`
  - Ubuntu/Debian: `apt-get install espeak`
  - CentOS/RHEL: `yum install espeak`

### Online (`speak-online`)
- An [OpenAI API key](https://platform.openai.com/api-keys) with access to the `tts-1` model.
- Network access to `api.openai.com`.

## Available Voices (OpenAI)

| Voice | Characteristics |
|-------|----------------|
| `alloy` | Neutral, balanced (default) |
| `echo` | Male, clear |
| `fable` | Expressive, warm |
| `onyx` | Deep, authoritative |
| `nova` | Female, friendly |
| `shimmer` | Female, soft |

## Notes

- Offline mode plays audio immediately via the system's audio output. It does not produce a file.
- Online mode returns a binary MP3 audio stream. To save it to a file, pass the response through a `run.exec:` step (e.g. with `curl -o output.mp3`).
- Store `apiKey` in an environment variable — never hardcode it in workflow YAML.
- The OpenAI TTS model used is `tts-1` (standard quality). For high-definition audio use `tts-1-hd` — switch by using `run.httpClient:` directly.
