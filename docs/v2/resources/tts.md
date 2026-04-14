# TTS (Text-to-Speech) Resource

> **Note**: This capability is now provided as an installable component. See the [Components guide](../concepts/components) for how to install and use it.
>
> Install: `kdeps registry install tts`
>
> Usage: `run: { component: { name: tts, with: { text: "...", voice: "alloy", apiKey: "..." } } }`

The TTS component synthesizes speech from text using OpenAI TTS (online) or espeak (offline fallback).
It produces an audio file accessible by downstream resources.

## Component Inputs

| Input | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `text` | string | yes | — | Text to synthesize |
| `voice` | string | no | `alloy` | Voice identifier: `alloy`, `echo`, `fable`, `onyx`, `nova`, `shimmer` (OpenAI voices) |
| `apiKey` | string | no | — | OpenAI API key (required for online synthesis; omit to use local espeak) |

## Using the TTS Component

**Online (OpenAI):**

```yaml
run:
  component:
    name: tts
    with:
      text: "Hello, welcome to KDeps!"
      voice: alloy
      apiKey: "sk-..."
```

**Offline (espeak, no API key required):**

```yaml
run:
  component:
    name: tts
    with:
      text: "Hello, welcome to KDeps!"
```

Access the output audio file path via `output('<callerActionId>')`.

---

## Result Map

| Key | Type | Description |
|-----|------|-------------|
| `success` | bool | `true` if synthesis succeeded. |
| `outputFile` | string | Absolute path to the generated audio file. |

---

## Expression Support

The `text` field supports [KDeps expressions](../concepts/expressions):

<div v-pre>

```yaml
run:
  component:
    name: tts
    with:
      text: "Hello {{ get('name') }}, your score is {{ get('score') }} points."
      voice: nova
      apiKey: "{{ env('OPENAI_API_KEY') }}"
```

</div>

---

## Example: Voice Assistant Response

<div v-pre>

```yaml
# resources/respond.yaml
metadata:
  actionId: respond
run:
  chat:
    model: llama3
    prompt: "{{ input() }}"

# resources/speak.yaml
metadata:
  actionId: speak
  requires: [respond]
run:
  component:
    name: tts
    with:
      text: "{{ output('respond') }}"
      voice: alloy
      apiKey: "{{ env('OPENAI_API_KEY') }}"

# resources/reply.yaml
metadata:
  actionId: reply
  requires: [speak]
run:
  apiResponse:
    success: true
    response:
      audioPath: "{{ output('speak').outputFile }}"
```

</div>

---

## Installation

The `espeak` CLI is required for offline synthesis:

```bash
apt install espeak-ng    # Debian/Ubuntu
brew install espeak      # macOS
```

For online synthesis, an OpenAI API key is sufficient — no local install needed.
