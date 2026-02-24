# Video Analysis Example

Continuous AI-powered camera surveillance that captures video frames, analyzes them with a vision LLM, and logs activity detections — all running locally with Ollama.

## Features

- ✅ Continuous video capture from a camera device
- ✅ AI frame analysis with `llava:7b` vision model
- ✅ Structured JSON output: people count, vehicles, activity description, alert flag
- ✅ Activity log written to `/tmp/kdeps-surveillance/analysis.log`
- ✅ Runs entirely on-device — no cloud required

## Prerequisites

Install Ollama and the vision model:

```bash
curl -fsSL https://ollama.ai/install.sh | sh
ollama pull llava:7b
```

Install `ffmpeg` (for video capture):

```bash
# Linux
apt install ffmpeg

# macOS
brew install ffmpeg

# Windows
# Download from https://ffmpeg.org/download.html
```

## Configure Your Camera

Find your video device:

```bash
# Linux
v4l2-ctl --list-devices

# macOS / Windows
ffmpeg -list_devices true -f avfoundation -i dummy   # macOS
ffmpeg -list_devices true -f dshow -i dummy          # Windows
```

Update `workflow.yaml` with your device path:

```yaml
video:
  device: /dev/video0             # Linux
  # device: "FaceTime HD Camera"  # macOS
  # device: "USB Video Device"    # Windows
```

## Run

```bash
# From examples/video-analysis directory
kdeps run workflow.yaml

# Or from project root
kdeps run examples/video-analysis/workflow.yaml
```

The workflow runs continuously, capturing and analyzing frames in a loop.

## View the Log

```bash
tail -f /tmp/kdeps-surveillance/analysis.log
```

Each line contains an ISO 8601 timestamp and the JSON analysis:

```
2025-03-15T14:22:01Z {"description":"Empty corridor with overhead lighting","people_count":0,"vehicles":[],"activity":"No movement detected","alert":false}
2025-03-15T14:22:45Z {"description":"Person walking through lobby","people_count":1,"vehicles":[],"activity":"Individual moving left to right","alert":false}
2025-03-15T14:23:12Z {"description":"Two people near entrance","people_count":2,"vehicles":[],"activity":"Individuals appear to be entering the building","alert":false}
```

## Structure

```
video-analysis/
├── workflow.yaml              # Video source and capture config
└── resources/
    ├── analyze-frame.yaml     # Vision LLM analysis (llava:7b)
    └── log-result.yaml        # Writes analysis to log file
```

## How It Works

### Pipeline

```
Camera → Frame Capture (ffmpeg) → inputMedia() → Vision LLM → JSON Analysis → Log File
```

Setting `transcriber.output: media` skips text transcription and delivers the raw video/image file path directly to resources via `inputMedia()`. The vision LLM receives the frame as an image and returns structured analysis.

### Key Expressions

| Expression | Description |
|------------|-------------|
| `inputMedia()` | Path to the captured video/image frame |
| `get('analyzeFrame')` | JSON analysis from the vision LLM |

## Customization

### Change the Analysis Prompt

```yaml
# resources/analyze-frame.yaml
chat:
  prompt: "Is there any fire or smoke visible in this image? Respond with yes or no and a brief description."
  jsonResponseKeys:
    - fire_detected
    - smoke_detected
    - description
    - confidence
```

### Use a Smaller Model on Limited Hardware

```yaml
# workflow.yaml
agentSettings:
  models:
    - moondream:1.8b             # Very small vision model

# resources/analyze-frame.yaml
chat:
  model: moondream:1.8b
```

### Send Alerts via HTTP

Replace the exec log step with an HTTP POST to send alerts:

```yaml
# resources/alert-webhook.yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: alertWebhook
  name: Send Alert
  requires:
    - analyzeFrame

run:
  http:
    url: "https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK"
    method: POST
    headers:
      Content-Type: application/json
    body: |
      {"text": "Camera alert: {{ get('analyzeFrame') }}"}
```

### Combine Video with API Source

Add an HTTP endpoint to query the latest analysis result alongside continuous video monitoring:

```yaml
# workflow.yaml
input:
  sources: [api, video]
  video:
    device: /dev/video0
```

## See Also

- [Input Sources Documentation](../../docs/v2/concepts/input-sources.md)
- [Vision Example](../vision/) — Image upload and analysis via HTTP API
- [Voice Assistant Example](../voice-assistant/) — Audio capture with TTS output
- [Telephony Bot Example](../telephony-bot/) — Cloud-based call handler
