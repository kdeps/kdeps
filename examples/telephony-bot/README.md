# Telephony Bot Example

An AI-powered phone call handler that transcribes caller speech, generates an LLM response, and returns synthesized audio — using Twilio for telephony, Deepgram for speech-to-text, and OpenAI TTS for speech synthesis.

## Features

- ✅ Twilio webhook integration for inbound phone calls
- ✅ Real-time speech transcription via Deepgram
- ✅ LLM response generation with Ollama
- ✅ High-quality speech synthesis with OpenAI TTS
- ✅ Returns transcript, text answer, and audio file path

## Prerequisites

You need:
- A [Twilio](https://twilio.com) account with a phone number
- A [Deepgram](https://deepgram.com) API key (for transcription)
- An [OpenAI](https://platform.openai.com) API key (for TTS)

## Setup

### 1. Configure API Keys

Edit `workflow.yaml` and `resources/tts-response.yaml` with your keys:

```yaml
# workflow.yaml — Deepgram key for transcription
transcriber:
  online:
    apiKey: "dg-your-key-here"

# resources/tts-response.yaml — OpenAI key for TTS
tts:
  online:
    apiKey: "sk-your-key-here"
```

### 2. Run the Server

```bash
# From examples/telephony-bot directory
kdeps run workflow.yaml --dev

# Or from project root
kdeps run examples/telephony-bot/workflow.yaml --dev
```

The server listens on `http://0.0.0.0:16395`.

### 3. Expose to the Internet

Twilio needs a publicly accessible URL. Use a tunnel for local development:

```bash
# ngrok (free tier)
ngrok http 16395

# cloudflared
cloudflared tunnel --url http://localhost:16395
```

### 4. Configure Twilio Webhook

In your [Twilio Console](https://console.twilio.com):

1. Go to **Phone Numbers → Active Numbers**
2. Select your phone number
3. Under **Voice & Fax → A Call Comes In**, set:
   - Webhook: `https://your-tunnel-url.ngrok.io/api/v1/call`
   - HTTP Method: `POST`

## How It Works

### Call Flow

```
Caller → Twilio → Webhook POST /api/v1/call
                           ↓
                   Deepgram Transcription
                           ↓
                     LLM (llama3.2:1b)
                           ↓
                    OpenAI TTS (alloy)
                           ↓
              Response: transcript + answer + audio path
```

When a call comes in, Twilio sends the caller's audio to the workflow's webhook endpoint. KDeps:
1. Transcribes the audio using Deepgram
2. Sends the transcript to the LLM
3. Converts the LLM response to speech using OpenAI TTS
4. Returns a JSON response with the transcript, text answer, and audio file path

### Response

```json
{
  "success": true,
  "data": {
    "transcript": "What is the weather like today?",
    "answer": "I don't have access to real-time weather data, but I can help with other questions!",
    "audio": "/tmp/kdeps-tts/response-abc123.mp3"
  }
}
```

## Structure

```
telephony-bot/
├── workflow.yaml              # Telephony source, Twilio + Deepgram config
└── resources/
    ├── llm-response.yaml      # LLM chat (takes inputTranscript as prompt)
    ├── tts-response.yaml      # OpenAI TTS (converts LLM text to speech)
    └── call-response.yaml     # API response with transcript + answer + audio
```

## Key Expressions

| Expression | Description |
|------------|-------------|
| `inputTranscript` | Caller's speech transcribed to text |
| `get('llmResponse')` | LLM-generated answer |
| `ttsOutput` | Path to synthesized audio file |

## Customization

### Use a Different STT Provider

```yaml
# workflow.yaml
transcriber:
  online:
    provider: assemblyai        # openai-whisper | google-stt | aws-transcribe | deepgram | assemblyai
    apiKey: "YOUR_KEY"
```

### Use ElevenLabs for More Natural TTS

```yaml
# resources/tts-response.yaml
tts:
  mode: online
  voice: "21m00Tcm4TlvDq8ikWAM"   # ElevenLabs voice ID
  online:
    provider: elevenlabs
    apiKey: "xi-your-key"
```

### Use a Stronger LLM

```yaml
# workflow.yaml
agentSettings:
  models:
    - llama3.1:8b

# resources/llm-response.yaml
chat:
  model: llama3.1:8b
```

### Add a System Prompt for a Custom Persona

```yaml
# resources/llm-response.yaml
chat:
  scenario:
    - role: assistant
      prompt: |
        You are Aria, a customer support agent for Acme Corp.
        You help callers with order status, returns, and product questions.
        Always ask for the caller's order number before looking up information.
```

### Use Local SIP Device Instead of Twilio

```yaml
# workflow.yaml
telephony:
  type: local
  device: /dev/ttyUSB0          # USB modem or ATA adapter serial device
```

## See Also

- [Input Sources Documentation](../../docs/v2/concepts/input-sources.md)
- [TTS Resource Documentation](../../docs/v2/resources/tts.md)
- [Voice Assistant Example](../voice-assistant/) — Fully offline, no cloud required
- [Video Analysis Example](../video-analysis/) — Camera surveillance + AI
