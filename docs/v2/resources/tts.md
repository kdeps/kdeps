# TTS (Text-to-Speech) Resource

The TTS resource synthesizes speech from text, producing an audio file that can be accessed by downstream resources via the `ttsOutput()` expression function. It supports both cloud (online) and local (offline) synthesis engines and can be used as a primary resource or as an [inline resource](../concepts/inline-resources) inside `before` / `after` blocks.

## Basic Usage

```yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: speakResponse
  name: Speak Response

run:
  tts:
    text: "Hello, welcome to KDeps!"
    mode: offline
    offline:
      engine: espeak
```

---

## Configuration Options

| Option | Type | Description |
|--------|------|-------------|
| `text` | string | **Required.** Text to synthesize. Supports [expressions](../concepts/expressions). |
| `mode` | string | **Required.** `online` or `offline`. |
| `language` | string | Optional BCP-47 language code (e.g. `en-US`). |
| `voice` | string | Voice identifier (provider/engine-specific). |
| `speed` | float | Speech rate multiplier. `1.0` is normal speed. |
| `outputFormat` | string | Audio format: `mp3` (default), `wav`, or `ogg`. |
| `outputFile` | string | Explicit output path. If omitted, a path under `/tmp/kdeps-tts/` is auto-generated. |
| `online` | object | Cloud provider config (required when `mode: online`). |
| `offline` | object | Local engine config (required when `mode: offline`). |

### Online Provider Configuration

```yaml
tts:
  mode: online
  online:
    provider: openai-tts     # Required: provider name
    apiKey: sk-...           # API authentication key
    region: us-east-1        # Region (AWS Polly / Azure only)
    subscriptionKey: ...     # Azure Cognitive Services subscription key
```

| Field | Description |
|-------|-------------|
| `provider` | One of: `openai-tts`, `google-tts`, `elevenlabs`, `aws-polly`, `azure-tts`. |
| `apiKey` | Authentication credential for the provider. |
| `region` | AWS region or Azure region (e.g. `eastus`). |
| `subscriptionKey` | Azure-only: Cognitive Services subscription key. |

### Offline Engine Configuration

```yaml
tts:
  mode: offline
  offline:
    engine: piper            # Required: engine name
    model: en_US-lessac-medium  # Model name or path (piper / coqui-tts)
```

| Field | Description |
|-------|-------------|
| `engine` | One of: `piper`, `espeak`, `festival`, `coqui-tts`. |
| `model` | Model name or file path (used by `piper` and `coqui-tts`). |

---

## Online Providers

### OpenAI TTS

Calls the [OpenAI `/v1/audio/speech`](https://platform.openai.com/docs/api-reference/audio/createSpeech) endpoint.

```yaml
tts:
  text: "Hello from OpenAI TTS"
  mode: online
  voice: alloy         # alloy | echo | fable | onyx | nova | shimmer
  outputFormat: mp3
  online:
    provider: openai-tts
    apiKey: sk-...
```

### Google Cloud Text-to-Speech

Calls the [Google Cloud TTS REST API](https://cloud.google.com/text-to-speech/docs/reference/rest/v1/text/synthesize).

```yaml
tts:
  text: "Hello from Google Cloud"
  mode: online
  language: en-US
  voice: en-US-Standard-A
  speed: 1.0
  online:
    provider: google-tts
    apiKey: AIza...
```

### ElevenLabs

Calls the [ElevenLabs `/v1/text-to-speech`](https://docs.elevenlabs.io/api-reference/text-to-speech) endpoint.

```yaml
tts:
  text: "Hello from ElevenLabs"
  mode: online
  voice: "21m00Tcm4TlvDq8ikWAM"    # ElevenLabs voice ID
  online:
    provider: elevenlabs
    apiKey: xi-...
```

### AWS Polly

> **Note:** AWS Polly requires SigV4 request signing. Use the `exec` resource to call the AWS CLI instead:
>
> ```yaml
> run:
>   exec:
>     command: "aws polly synthesize-speech --output-format mp3 --voice-id Joanna --text 'Hello' /tmp/speech.mp3"
> ```

### Azure Cognitive Services TTS

Calls the [Azure TTS REST API](https://learn.microsoft.com/azure/cognitive-services/speech-service/rest-text-to-speech).

```yaml
tts:
  text: "Hello from Azure"
  mode: online
  language: en-US
  voice: en-US-JennyNeural
  online:
    provider: azure-tts
    region: eastus
    subscriptionKey: ...
```

---

## Offline Engines

Offline engines run locally without any network calls. They must be installed on the host system.

### Piper

High-quality neural TTS. [GitHub: rhasspy/piper](https://github.com/rhasspy/piper)

```bash
# Install
pip install piper-tts
# Or download binary from GitHub releases
```

```yaml
tts:
  text: "Hello from Piper"
  mode: offline
  offline:
    engine: piper
    model: en_US-lessac-medium    # Download from https://huggingface.co/rhasspy/piper-voices
```

### eSpeak-NG

Fast, lightweight speech synthesizer. [eSpeak NG](https://github.com/espeak-ng/espeak-ng)

```bash
# Install
apt install espeak-ng          # Debian/Ubuntu
brew install espeak            # macOS
```

```yaml
tts:
  text: "Hello from eSpeak"
  mode: offline
  voice: en             # eSpeak voice identifier
  speed: 1.2            # Words per minute multiplier
  offline:
    engine: espeak
```

### Festival

Classic speech synthesis system. [Festival Speech Synthesis System](http://www.cstr.ed.ac.uk/projects/festival/)

```bash
# Install
apt install festival            # Debian/Ubuntu
brew install festival           # macOS
```

```yaml
tts:
  text: "Hello from Festival"
  mode: offline
  offline:
    engine: festival
```

### Coqui TTS

Open-source deep-learning TTS. [Coqui TTS](https://github.com/coqui-ai/TTS)

```bash
# Install
pip install TTS
```

```yaml
tts:
  text: "Hello from Coqui"
  mode: offline
  offline:
    engine: coqui-tts
    model: tts_models/en/ljspeech/tacotron2-DDC
```

---

## Accessing the Output File

After the TTS resource runs, the path to the synthesized audio file is available via:

- **`ttsOutput()`** — expression function in any resource evaluated after TTS runs
- **`get("ttsOutput")`** — unified `get()` helper
- **`input("ttsOutput")`** or **`input("tts")`** — `input()` helper

```yaml
resources:
  - metadata:
      actionId: speakThenRespond
    run:
      tts:
        text: "Here is your answer"
        mode: offline
        offline:
          engine: espeak

  - metadata:
      actionId: sendAudioFile
    run:
      apiResponse:
        success: true
        response:
          audioPath: "{{ttsOutput()}}"
```

---

## TTS as an Inline Resource

TTS can run **before** or **after** the main resource action using the `before` / `after` inline blocks:

```yaml
run:
  before:
    - tts:
        text: "Processing your request, please wait…"
        mode: offline
        offline:
          engine: espeak

  chat:
    model: llama3
    prompt: "{{input()}}"

  after:
    - tts:
        text: "Processing complete."
        mode: offline
        offline:
          engine: espeak
```

---

## Using Expressions in `text`

The `text` field supports [KDeps expressions](../concepts/expressions):

```yaml
tts:
  text: "Hello {{get(\"name\")}}, your score is {{get(\"score\")}} points."
  mode: offline
  offline:
    engine: espeak
```

---

## Example: Voice Assistant Response

Full example combining input transcription with TTS output:

```yaml
# workflow.yaml
settings:
  input:
    source: audio
    audio:
      device: hw:0,0
    activation:
      phrase: "hey kdeps"
      mode: offline
      offline:
        engine: faster-whisper
        model: small
    transcriber:
      mode: offline
      output: text
      offline:
        engine: faster-whisper
        model: small

# resources/respond.yaml
run:
  chat:
    model: llama3
    prompt: "{{inputTranscript()}}"

# resources/speak.yaml
run:
  tts:
    text: "{{get(\"respond\")}}"
    mode: offline
    offline:
      engine: piper
      model: en_US-lessac-medium
```
