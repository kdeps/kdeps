# Build a phone assistant (IVR)

Build an interactive voice response (IVR) phone menu: the caller hears a menu, presses a key or speaks, and the workflow either reads a static answer or has an LLM answer the spoken question. It is an API workflow: Twilio posts each call step to a route, and each route answers with TwiML built by the [`twiml` component](/workflow/resources/telephony).

```d2
direction: right
voice: "/twilio/voice\nmenu (gather 1 key)" {shape: oval}
menu: "/twilio/menu\nDigits?" {shape: diamond}
hours: "/twilio/hours\nsay + hangup"
ask: "/twilio/ask\ngather speech"
answer: "/twilio/answer\nLLM -> say + hangup"
voice -> menu: Digits
menu -> hours: "1"
menu -> ask: "2"
menu -> voice: other
ask -> answer: SpeechResult
```

**Needs:** a Twilio account and phone number for a live call; `curl` is enough to test.

The finished project is [`examples/telephony-bot`](https://github.com/kdeps/kdeps/tree/main/examples/telephony-bot).

## Step 1: define the call routes

```yaml
# workflow.yaml
apiVersion: kdeps.io/v1
kind: Workflow

metadata:
  name: telephony-bot
  description: AI phone IVR - Twilio webhooks answered with TwiML from the twiml component
  version: "1.0.0"
  targetActionId: respond

settings:
  apiServer:
    hostIp: "127.0.0.1"
    portNum: 16395
    routes:
      - path: /twilio/voice    # entry point: welcome menu
        methods: [POST]
        public: true           # Twilio sends no bearer token
      - path: /twilio/menu     # the key the caller pressed
        methods: [POST]
        public: true
      - path: /twilio/hours    # static answer: opening hours
        methods: [POST]
        public: true
      - path: /twilio/ask      # gather a spoken question
        methods: [POST]
        public: true
      - path: /twilio/answer   # LLM answers the spoken question
        methods: [POST]
        public: true

  agentSettings:
    timezone: Etc/UTC
```

## Step 2: the welcome menu

A `gather` collects one key. Twilio posts it to `/twilio/menu` as `Digits`; with no key, the `redirect` replays the menu.

<div v-pre>

```yaml
# resources/welcome.yaml
actionId: welcome
name: Welcome Menu
validations:
  routes: [/twilio/voice]
component:
  name: twiml
  with:
    verbs:
      - gather:
          input: dtmf
          numDigits: 1
          timeout: 8
          action: /twilio/menu      # Twilio posts the pressed key here as Digits
          say: "Welcome to kdeps. Press 1 for opening hours. Press 2 to ask our assistant a question."
      - redirect: /twilio/voice     # no key pressed: play the menu again
```

</div>

## Step 3: route the key

The pressed key picks the next route; anything else goes back to the menu.

<div v-pre>

```yaml
# resources/menu.yaml
actionId: menuChoice
name: Menu Choice
validations:
  routes: [/twilio/menu]
component:
  name: twiml
  with:
    verbs:
      # Digits is the key from the welcome menu's gather.
      - redirect: "{{ get('Digits') == '1' ? '/twilio/hours' : get('Digits') == '2' ? '/twilio/ask' : '/twilio/voice' }}"
```

</div>

## Step 4: a static answer

Speak the hours, then hang up.

<div v-pre>

```yaml
# resources/hours.yaml
actionId: sayHours
name: Say Opening Hours
validations:
  routes: [/twilio/hours]
component:
  name: twiml
  with:
    verbs:
      - say: {text: "We are open Monday to Friday, nine to five, Central European Time.", voice: alice}
      - hangup: true
```

</div>

## Step 5: gather a spoken question

A speech `gather`: Twilio transcribes the caller and posts the text to `/twilio/answer` as `SpeechResult`.

<div v-pre>

```yaml
# resources/ask.yaml
actionId: askQuestion
name: Ask Question
validations:
  routes: [/twilio/ask]
component:
  name: twiml
  with:
    verbs:
      - gather:
          input: speech
          speechTimeout: auto
          action: /twilio/answer    # Twilio posts the transcript here as SpeechResult
          say: "What would you like to know?"
```

</div>

## Step 6: the LLM answer

The model answers the transcript in one sentence.

<div v-pre>

```yaml
# resources/answer.yaml
actionId: answerLLM
name: LLM Answer
validations:
  routes: [/twilio/answer]
  check:
    - get('SpeechResult') != ''   # SpeechResult arrives in Twilio's webhook body
  error:
    code: 400
    message: "SpeechResult is required"
chat:
  model: llama3.2:1b
  role: user
  prompt: |
    A caller asked the following question over the phone:
    "{{ get('SpeechResult') }}"
    Answer in one short, friendly sentence suitable for being read aloud.
  timeout: 60s
```

</div>

## Step 7: speak the answer

The answer goes into `say` - the component XML-escapes it.

<div v-pre>

```yaml
# resources/speak.yaml
actionId: speakAnswer
name: Speak Answer
requires: [answerLLM]
validations:
  routes: [/twilio/answer]
component:
  name: twiml
  with:
    verbs:
      - say: "{{ get('answerLLM').message.content }}"
      - hangup: true
```

</div>

## Step 8: return the TwiML

Each request runs only the resources for its route, so exactly one `twiml` call has output. `Content-Type: text/xml` writes it as the raw body.

<div v-pre>

```yaml
# resources/respond.yaml
actionId: respond
name: TwiML Response
requires: [welcome, menuChoice, sayHours, askQuestion, speakAnswer]
apiResponse:
  success: true
  headers:
    Content-Type: text/xml       # Twilio reads the body as TwiML
  # Exactly one twiml call ran - the one for this request's route.
  response: "{{ (output('welcome') ?? output('menuChoice') ?? output('sayHours') ?? output('askQuestion') ?? output('speakAnswer')).twiml }}"
```

</div>

## Step 9: run and test

```bash
kdeps validate workflow.yaml
export KDEPS_API_AUTH_TOKEN=dev-token   # the /twilio routes are public; others need it
kdeps run workflow.yaml
```

Post what Twilio would post:

```bash
curl -X POST http://localhost:16395/twilio/voice -d CallSid=CA1
curl -X POST http://localhost:16395/twilio/menu -d Digits=1
curl -X POST http://localhost:16395/twilio/answer -d "SpeechResult=what time do you close"
```

```xml
<?xml version="1.0" encoding="UTF-8"?><Response><Gather action="/twilio/menu" input="dtmf" numDigits="1" timeout="8"><Say>Welcome to kdeps. Press 1 for opening hours. Press 2 to ask our assistant a question.</Say></Gather><Redirect>/twilio/voice</Redirect></Response>
<?xml version="1.0" encoding="UTF-8"?><Response><Redirect>/twilio/hours</Redirect></Response>
<?xml version="1.0" encoding="UTF-8"?><Response><Say>We close at five in the evening.</Say><Hangup /></Response>
```

The answer's wording comes from the model and varies. For a live call, set the Twilio number's voice webhook to `https://<your-host>/twilio/voice` (HTTP POST).

## Next steps

- [twiml component](/workflow/resources/telephony) - every verb and attribute
- [Transcribe component](/workflow/resources/transcribe) - speech to text on your own audio
- [Validation and control flow](/workflow/validation) - route scoping
- [Bot replies](/workflow/resources/bot-reply) - chat-platform bots
