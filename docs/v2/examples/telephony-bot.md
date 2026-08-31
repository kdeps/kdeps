# Build a phone assistant (IVR)

*Applies to workflow mode.*

## Overview

In this tutorial you build an interactive voice response (IVR) phone menu: the
caller hears a menu, presses a key or speaks, and the workflow either reads a
static answer or has an LLM answer a spoken question. It uses the `telephony:`
resource with a provider such as Twilio.

This tutorial is for developers who have completed the
[quickstart](/getting-started/quickstart). It assumes you know:

- Basic YAML
- How webhook-based telephony providers work (a call triggers HTTP POSTs)

By the end you will be able to:

- Present a keypad menu with `telephony: action: menu`
- Read a static message with `telephony: action: say`
- Gather spoken input with `telephony: action: ask`
- Route the spoken question to an LLM

## Background

A telephony provider turns a phone call into HTTP webhooks. Each menu choice
maps to a route; the provider posts to that route when the caller acts. The
`telephony:` resource returns instructions the provider glue renders as call
control (TwiML for Twilio). A spoken transcription arrives in the webhook body
as `SpeechResult`.

## Before you start

- kdeps installed (`kdeps --version`).
- A telephony provider account (Twilio) and a phone number, for a live test.
- A working directory for the project.

## Step 1: create the project

```bash
mkdir phone-bot
cd phone-bot
mkdir resources
```

## Step 2: define the call routes

Create `workflow.yaml`:

```yaml
# workflow.yaml
apiVersion: kdeps.io/v1
kind: Workflow

metadata:
  name: phone-bot
  version: "1.0.0"
  targetActionId: respond

settings:
  apiServer:
    hostIp: "127.0.0.1"
    portNum: 16395
    routes:
      - path: /twilio/voice     # entry: welcome menu
        methods: [POST]
      - path: /twilio/hours     # static answer
        methods: [POST]
      - path: /twilio/ask       # gather a spoken question
        methods: [POST]
      - path: /twilio/answer    # LLM answers it
        methods: [POST]
```

## Step 3: the welcome menu

Create `resources/menu.yaml`:

```yaml
# resources/menu.yaml
actionId: welcomeMenu
name: Welcome menu
validations:
  routes: [/twilio/voice]
  methods: [POST]
telephony:
  action: menu
  say: "Welcome. Press 1 for opening hours. Press 2 to ask a question."
  voice: alice
  timeout: 8s
  matches:
    - keys: ["1"]              # provider then posts to /twilio/hours
    - keys: ["2"]              # provider then posts to /twilio/ask
```

## Step 4: the static answer

Create `resources/hours.yaml`:

```yaml
# resources/hours.yaml
actionId: sayHours
name: Say opening hours
validations:
  routes: [/twilio/hours]
  methods: [POST]
telephony:
  action: say
  say: "We are open Monday to Friday, nine to five."
  voice: alice
```

## Step 5: gather a spoken question

Create `resources/ask.yaml`:

```yaml
# resources/ask.yaml
actionId: askQuestion
name: Ask question
validations:
  routes: [/twilio/ask]
  methods: [POST]
telephony:
  action: ask
  say: "What would you like to know? Speak after the tone."
  mode: both                  # speech or keypad
  limit: 32
  timeout: 10s
```

## Step 6: let the LLM answer

Create `resources/answer.yaml`:

<div v-pre>

```yaml
# resources/answer.yaml
actionId: answerLLM
name: LLM answer
validations:
  routes: [/twilio/answer]
  methods: [POST]
  check:
    - get('SpeechResult') != ''   # the transcription from the provider
  error:
    code: 400
    message: "SpeechResult is required"
chat:
  model: llama3.2:1b
  role: user
  prompt: |
    A caller asked over the phone: "{{ get('SpeechResult') }}"
    Answer in one short, friendly sentence to be read aloud.
  timeout: 60s
```

</div>

## Step 7: the target resource

Create `resources/respond.yaml`:

<div v-pre>

```yaml
# resources/respond.yaml
actionId: respond
name: Response
requires: [welcomeMenu, sayHours, askQuestion, answerLLM]
before:
  - set('answerText', safe(safe(get('answerLLM'), 'message'), 'content'))
apiResponse:
  success: true
  response:
    menu:   "{{ output('welcomeMenu') }}"
    hours:  "{{ output('sayHours') }}"
    ask:    "{{ output('askQuestion') }}"
    answer: "{{ get('answerText') }}"
```

</div>

Only the resource whose route matches the current webhook runs; the rest are
skipped.

## Step 8: validate and test

```bash
kdeps validate .
export KDEPS_API_AUTH_TOKEN=dev-token
kdeps run .
```

Simulate the "ask a question" webhook:

```bash
curl -X POST http://localhost:16395/twilio/answer \
  -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"SpeechResult": "what time do you close"}'
```

For a live call, point your provider's voice webhook at
`https://<your-host>/twilio/voice`.

## Summary

You built an IVR that:

- Presents a keypad menu with `telephony: action: menu`
- Reads a fixed answer with `action: say`
- Gathers speech with `action: ask`
- Sends the transcription (`SpeechResult`) to an LLM

## Next steps

- [Telephony resource](/resources/messaging/telephony) - all actions, voices, providers
- [Transcribe resource](/resources/media/transcribe) - speech to text on your own audio
- [Validation and control flow](/concepts/validation-and-control) - route scoping
- [Bot reply resource](/resources/messaging/bot-reply) - chat-platform bots
