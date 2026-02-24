# Voice Assistant Example

A fully offline voice assistant that listens for a wake phrase, transcribes speech, generates an LLM response, and speaks the answer aloud — no cloud required.

## Features

- ✅ Wake phrase detection ("hey kdeps") using `whisper` offline
- ✅ Continuous audio capture from a microphone
- ✅ Offline speech-to-text transcription
- ✅ LLM response generation with Ollama (llama3.2:1b)
- ✅ Offline text-to-speech output using Piper TTS
- ✅ Fully air-gapped — no internet connection needed

## Prerequisites

Install required tools:

```bash
# Piper TTS (for speech output)
pip install piper-tts

# whisper (for wake detection + transcription)
pip install whisper

# Ollama (for LLM)
curl -fsSL https://ollama.ai/install.sh | sh
ollama pull llama3.2:1b
```

## Configure Your Microphone

Find your audio device:

```bash
# Linux
arecord -l

# macOS / Windows
ffmpeg -list_devices true -f avfoundation -i dummy   # macOS
ffmpeg -list_devices true -f dshow -i dummy          # Windows
```

Update `workflow.yaml` with your device identifier:

```yaml
audio:
  device: hw:0,0          # Linux example
  # device: "Built-in Microphone"   # macOS
  # device: "Microphone (Realtek Audio)"  # Windows
```

## Run

```bash
# From examples/voice-assistant directory
kdeps run workflow.yaml

# Or from project root
kdeps run examples/voice-assistant/workflow.yaml
```

Once running, say **"hey kdeps"** followed by your question. The assistant will:
1. Detect the wake phrase
2. Capture and transcribe your speech
3. Generate an LLM response
4. Speak the response aloud

## Structure

```
voice-assistant/
├── workflow.yaml              # Input sources, activation, and transcription config
└── resources/
    ├── respond.yaml           # LLM chat resource (takes transcribed speech as input)
    └── speak.yaml             # TTS resource (speaks the LLM response)
```

## How It Works

### Pipeline

```
Microphone → Wake Phrase Detection → Audio Capture → Transcription → LLM → TTS → Speaker
```

### Activation Loop

The runtime captures `chunkSeconds` (3s) of audio in a continuous loop. Each chunk is transcribed with `whisper tiny` and checked against the wake phrase `"hey kdeps"` with 90% similarity threshold. When detected, the full workflow runs.

### Key Expressions

| Expression | Description |
|------------|-------------|
| `inputTranscript` | The transcribed speech text |
| `get('respond')` | The LLM response text |
| `ttsOutput` | Path to the generated audio file |

## Customization

### Use a Different LLM

```yaml
# workflow.yaml
agentSettings:
  models:
    - mistral:7b             # or any Ollama model

# resources/respond.yaml
chat:
  model: mistral:7b
```

### Change the Wake Phrase

```yaml
activation:
  phrase: "okay computer"
  sensitivity: 0.85          # Lower = more flexible matching
```

### Use eSpeak Instead of Piper (Lighter Weight)

```yaml
# resources/speak.yaml
tts:
  mode: offline
  offline:
    engine: espeak
    voice: en
```

### Raspberry Pi / Edge Device Optimizations

For resource-constrained hardware, use the smallest models:

```yaml
activation:
  offline:
    engine: whisper
    model: tiny              # Smallest, fastest

transcriber:
  offline:
    engine: whisper
    model: tiny              # Use tiny on Pi Zero / limited RAM

# resources/respond.yaml
chat:
  model: llama3.2:1b         # Smallest Llama model

# resources/speak.yaml
tts:
  offline:
    engine: espeak           # Lightest TTS engine
```

## See Also

- [Input Sources Documentation](../../docs/v2/concepts/input-sources.md)
- [TTS Resource Documentation](../../docs/v2/resources/tts.md)
- [Telephony Bot Example](../telephony-bot/) — Cloud-based call handler
- [Video Analysis Example](../video-analysis/) — Camera surveillance + AI
