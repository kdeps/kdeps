# Multi-Source Input

KDeps supports multiple input sources simultaneously: HTTP API requests, audio hardware (microphones), video hardware (cameras), and telephony devices. Sources are configured in the `settings.input` block of your `workflow.yaml`.

## Overview

| Source | Use Case |
|--------|----------|
| `api` | HTTP API requests (default, REST/JSON) |
| `audio` | Microphone or line-in audio capture |
| `video` | Camera or V4L2 video capture |
| `telephony` | Phone call audio (local SIP device or cloud provider) |

Workflows can combine sources:

```yaml
settings:
  input:
    sources: [audio]            # microphone only
    sources: [audio, video]     # audio + video
    sources: [api, audio]       # API requests + microphone
    sources: [telephony]        # phone/SIP only
```

---

## Source Configuration

### API Source

The default. No additional config needed. The workflow responds to HTTP requests like any standard API.

```yaml
settings:
  input:
    sources: [api]
```

### Audio Source

Captures audio from a hardware device using `arecord` (Linux/ALSA) or `ffmpeg`.

```yaml
settings:
  input:
    sources: [audio]
    audio:
      device: hw:0,0            # ALSA: hw:<card>,<device>
```

**Device identifiers by platform:**

| Platform | Example |
|----------|---------|
| Linux (ALSA) | `hw:0,0`, `default`, `plughw:1,0` |
| macOS | `Built-in Microphone`, `default` |
| Windows | `Microphone (Realtek Audio)` |

List available audio devices:
```bash
# Linux
arecord -l

# macOS / Windows
ffmpeg -list_devices true -f avfoundation -i dummy   # macOS
ffmpeg -list_devices true -f dshow -i dummy          # Windows
```

### Video Source

Captures video from a hardware camera using `ffmpeg` with the platform's native capture driver.

```yaml
settings:
  input:
    sources: [video]
    video:
      device: /dev/video0       # V4L2 device path (Linux)
```

**Device identifiers by platform:**

| Platform | Example |
|----------|---------|
| Linux (V4L2) | `/dev/video0`, `/dev/video1` |
| macOS (AVFoundation) | `FaceTime HD Camera`, `0` |
| Windows (DirectShow) | `USB Video Device`, `0` |

List available video devices:
```bash
# Linux
v4l2-ctl --list-devices

# macOS
ffmpeg -list_devices true -f avfoundation -i dummy

# Windows
ffmpeg -list_devices true -f dshow -i dummy
```

### Telephony Source

Captures audio from a phone or SIP device. Two modes are supported:

**Local** — a hardware telephony device (e.g. USB modem, ATA adapter):

```yaml
settings:
  input:
    sources: [telephony]
    telephony:
      type: local
      device: /dev/ttyUSB0      # Serial device path
```

**Online** — a cloud telephony provider (media arrives via webhook):

```yaml
settings:
  input:
    sources: [telephony]
    telephony:
      type: online
      provider: twilio          # Currently: twilio
```

When using an online provider, configure the provider's webhook to POST audio to your workflow's API endpoint.

---

## Activation (Wake Phrase Detection)

Activation listens continuously for a wake phrase before triggering the main workflow. This is ideal for voice assistants and hands-free operation on edge devices.

```yaml
settings:
  input:
    sources: [audio]
    audio:
      device: hw:0,0
    activation:
      phrase: "hey kdeps"       # Required: the phrase to listen for
      mode: offline             # online | offline
      sensitivity: 0.9          # 0.0–1.0  (1.0 = exact match only)
      chunkSeconds: 3           # Duration of each audio probe (seconds)
      offline:
        engine: faster-whisper
        model: small
```

### How Activation Works

1. The runtime captures `chunkSeconds` of audio in a loop.
2. Each chunk is transcribed using the configured engine.
3. If the transcript matches the wake phrase (within `sensitivity` threshold), the main workflow runs.
4. After the workflow completes, the loop resumes.

### Sensitivity

`sensitivity` controls fuzzy matching: `1.0` requires an exact phrase match, lower values allow approximate matches.

| Value | Behavior |
|-------|----------|
| `1.0` | Exact match only (default) |
| `0.9` | ~90% similarity required |
| `0.5` | Broader matching, more false positives |

### Online Activation

Use a cloud STT provider for the activation loop:

```yaml
activation:
  phrase: "hey kdeps"
  mode: online
  sensitivity: 0.95
  online:
    provider: deepgram
    apiKey: dg-...
```

Supported online providers: `openai-whisper`, `google-stt`, `aws-transcribe`, `deepgram`, `assemblyai`

### Offline Activation

Run entirely on-device with no cloud calls:

```yaml
activation:
  phrase: "hey kdeps"
  mode: offline
  sensitivity: 0.9
  offline:
    engine: faster-whisper     # whisper | faster-whisper | vosk | whisper-cpp
    model: small               # tiny, base, small, medium, large
```

---

## Transcription (Speech-to-Text)

After audio capture (and optional activation), the transcriber converts the media signal into text that your workflow resources can use.

```yaml
settings:
  input:
    sources: [audio]
    audio:
      device: hw:0,0
    transcriber:
      mode: offline             # online | offline
      output: text              # text | media
      language: en-US           # Optional BCP-47 language code
      offline:
        engine: faster-whisper
        model: small
```

### Output Modes

| `output` | Description |
|----------|-------------|
| `text` | Transcribed string (default) |
| `media` | Raw media file path (skips transcription) |

### Accessing Transcription Results

In any resource that runs after transcription:

```yaml
run:
  chat:
    prompt: "{{ inputTranscript() }}"    # expression function
```

Equivalent accessors:
- `inputTranscript()` — expression function
- `inputMedia()` — path to the raw media file
- `get("inputTranscript")` — unified API

### Online Transcription Providers

| Provider | `provider` value |
|----------|-----------------|
| OpenAI Whisper API | `openai-whisper` |
| Google Cloud STT | `google-stt` |
| AWS Transcribe | `aws-transcribe` |
| Deepgram | `deepgram` |
| AssemblyAI | `assemblyai` |

```yaml
transcriber:
  mode: online
  output: text
  language: en-US
  online:
    provider: deepgram
    apiKey: dg-...
```

### Offline Transcription Engines

All engines run locally — no network calls, no data leaving the device.

| Engine | `engine` value | Notes |
|--------|---------------|-------|
| OpenAI Whisper | `whisper` | Requires Python + `openai-whisper` |
| Faster Whisper | `faster-whisper` | CTranslate2 backend, faster + lower RAM |
| Vosk | `vosk` | Lightweight, great for embedded devices |
| Whisper.cpp | `whisper-cpp` | C++ port, runs on CPU without Python |

```yaml
transcriber:
  mode: offline
  output: text
  offline:
    engine: faster-whisper
    model: small              # tiny | base | small | medium | large
```

---

## Combined Examples

### Offline Voice Assistant (Raspberry Pi / Jetson)

Fully offline voice assistant — no cloud required:

```yaml
settings:
  input:
    sources: [audio]
    audio:
      device: hw:0,0
    activation:
      phrase: "hey kdeps"
      mode: offline
      sensitivity: 0.9
      offline:
        engine: faster-whisper
        model: tiny             # Use tiny model for fast response on edge hardware
    transcriber:
      mode: offline
      output: text
      offline:
        engine: faster-whisper
        model: small
```

Resource that processes the spoken input:

```yaml
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: voiceChat
run:
  chat:
    model: llama3.2:1b
    prompt: "{{ inputTranscript() }}"
  tts:
    text: "{{ get('voiceChat') }}"
    mode: offline
    offline:
      engine: piper
      model: en_US-lessac-medium
```

### Video Surveillance + AI Analysis

```yaml
settings:
  input:
    sources: [video]
    video:
      device: /dev/video0
    transcriber:
      mode: offline
      output: media             # Keep raw video, no transcription
      offline:
        engine: faster-whisper
        model: base
```

Resource that analyzes video frames:

```yaml
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: analyzeFrame
run:
  chat:
    model: llama3.2-vision
    prompt: "Describe what you see in this video frame."
    images:
      - "{{ inputMedia() }}"
```

### Telephony Call Handler

```yaml
settings:
  input:
    sources: [telephony]
    telephony:
      type: online
      provider: twilio
    transcriber:
      mode: online
      output: text
      online:
        provider: deepgram
        apiKey: dg-...
```

### Multi-Source: API + Audio

Accept both HTTP requests and microphone input in the same workflow:

```yaml
settings:
  input:
    sources: [api, audio]
    audio:
      device: hw:0,0
    transcriber:
      mode: offline
      output: text
      offline:
        engine: faster-whisper
        model: small
```

---

## Edge Device Notes

KDeps is designed to run on resource-constrained hardware. Recommendations for edge deployments:

| Device | Recommended Config |
|--------|--------------------|
| Raspberry Pi 4 | `faster-whisper` with `tiny` or `base` model, `espeak` TTS |
| NVIDIA Jetson Nano | `faster-whisper` with `small` model, `piper` TTS |
| x86 mini-PC (no GPU) | `whisper-cpp` with `base` model |
| Online-only edge | Use `deepgram` or `openai-whisper` for STT |

For fully offline/air-gapped deployments, set `offlineMode: true` in `agentSettings` and use only offline engines:

```yaml
settings:
  agentSettings:
    offlineMode: true
    models:
      - llama3.2:1b
  input:
    sources: [audio]
    audio:
      device: hw:0,0
    transcriber:
      mode: offline
      offline:
        engine: faster-whisper
        model: small
```

---

## See Also

- [Workflow Configuration](../configuration/workflow) — Full `settings.input` reference
- [TTS Resource](../resources/tts) — Speech output
- [LLM Resource](../resources/llm) — Language model integration
- [Docker Deployment](../deployment/docker) — Package for edge deployment
