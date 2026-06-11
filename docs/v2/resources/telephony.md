# Telephony Resource

The `telephony:` resource models in-call actions (answer, say, ask, menu, dial, record, hangup, and more) for Twilio-compatible telephony providers. The provider POSTs its call webhook to your kdeps API route; the resource reads the webhook fields from the request body and builds a TwiML response that is returned via the standard `apiResponse` mechanism.

## Where it runs

Both [workflow mode](/modes/workflow-mode) and [agent mode](/modes/agent-mode). In workflow mode it executes as a DAG step. In agent mode, the workflow containing this resource runs as a single callable tool.

## How it works

```text
provider webhook POST -> telephony: resource -> apiResponse with TwiML
```

The provider (e.g. Twilio) sends fields like `CallSid`, `From`, `To`, `CallStatus`, `Digits`, `SpeechResult`, and `Confidence` in the webhook body. kdeps populates the call session from them, executes the action, and the resource output contains the TwiML to return. Call state is shared across all telephony resources within the same workflow run.

## Basic Usage

```yaml
# resources/say.yaml
actionId: sayHello
name: Say Hello
validations:
  routes: [/twilio/voice]
  methods: [POST]
telephony:
  action: say
  say: "Hello from kdeps telephony."
  voice: alice
```

```yaml
# resources/respond.yaml
actionId: respond
name: TwiML Response
requires: [sayHello]
apiResponse:
  success: true
  response: "{{ output('sayHello') }}"
```

## Actions

| Action | What it does |
|---|---|
| `answer` | Answer the inbound call |
| `say` | Speak TTS text (or play `audio:`) |
| `ask` | Collect DTMF digits or speech input |
| `menu` | Prompt + match input against options, branch on the result |
| `dial` | Connect the call to SIP URIs or phone numbers |
| `record` | Record the caller |
| `mute` / `unmute` | Mute or unmute the call |
| `hangup` | End the call |
| `reject` | Reject the call (with optional `reason:`) |
| `redirect` | Redirect call control elsewhere |

## Complete reference

<div v-pre>

```yaml
# resources/example.yaml
telephony:
  action: menu            # answer, say, ask, menu, dial, record,
                          # mute, unmute, hangup, reject, redirect

  # --- Output (say / prompt) ---
  say: "Press 1 for sales, press 2 for support."   # TTS text to speak
  voice: alice            # TTS voice name
  audio: "https://example.com/prompt.mp3"          # audio URL/path instead of TTS

  # --- Input collection (ask / menu) ---
  mode: dtmf              # dtmf (default), speech, both
  grammar: ""             # inline GRXML grammar
  grammarUrl: ""          # external grammar URL
  limit: 4                # max digits to collect
  terminator: "#"         # digit that ends input
  timeout: 5s             # no-input timeout
  interDigitTimeout: 2s   # between-digit timeout

  # --- Menu ---
  tries: 3                # retry count (default: 1)
  matches:
    - keys: ["1"]         # DTMF digits or speech phrases to match
      invoke: salesFlow   # component name to invoke on match
    - keys: ["2"]
      expr:               # inline expressions to run on match
        - set('branch', 'support')
  onNoMatch: |            # expression on nomatch
    say("Sorry, that option is not available.")
  onNoInput: |            # expression on noinput
    say("I did not hear anything.")
  onFailure: |            # expression after all tries exhausted
    telephony.action("hangup")

  # --- Dial ---
  to:                     # SIP URIs or tel: numbers
    - sip:agent@pbx.example.com
    - "+15005550001"
  from: "+18005550000"    # caller ID override
  for: 30s                # dial timeout

  # --- Record ---
  maxDuration: 60s        # recording cap
  interruptible: true     # allow keypress to stop recording
  format: wav             # wav (default) or mp3

  # --- Hangup / Reject ---
  reason: busy            # e.g. busy, decline
  headers:                # SIP headers
    X-Custom: value
```

</div>

## IVR Menu Example

<div v-pre>

```yaml
# resources/menu.yaml
actionId: mainMenu
name: Main Menu
validations:
  routes: [/twilio/menu]
  methods: [POST]
telephony:
  action: menu
  say: "Press 1 for sales. Press 2 for support."
  timeout: 8s
  matches:
    - keys: ["1"]
      invoke: salesFlow     # component invoked when caller presses 1
    - keys: ["2"]
      invoke: supportFlow
  onNoMatch: repeatMenu
```

</div>

When the provider posts `Digits: "1"`, the menu resolves to `status: match` with `interpretation: "1"` and invokes the `salesFlow` component. With no digits, the resource returns a TwiML `<Gather>` prompt so the provider collects input.

## Output

The resource output contains the accumulated TwiML and, for `ask`/`menu`, a result object:

| Key | Type | Description |
|---|---|---|
| `twiml` | string | TwiML XML to return to the provider |
| `result.status` | string | `match`, `nomatch`, `noinput`, `hangup`, or `stop` |
| `result.mode` | string | `dtmf` or `speech` |
| `result.utterance` | string | Normalised input (digits or speech text) |
| `result.interpretation` | string | Semantic value extracted from the grammar |
| `result.confidence` | number | Speech confidence 0.0-1.0 (1.0 for DTMF) |
| `result.match` | bool | True when status is `match` |

## Expression Accessors

Usable in `expr` lists and <span v-pre>`{{ }}`</span> blocks:

```yaml
telephony.callId()      # unique call identifier
telephony.from()        # caller number
telephony.to()          # dialed number
telephony.status()      # match | nomatch | noinput | hangup | stop
telephony.utterance()   # DTMF digits or speech text from last ask/menu
telephony.digits()      # raw DTMF string from last gather
telephony.speech()      # speech recognition text from last gather
telephony.confidence()  # speech confidence (0.0-1.0)
telephony.twiml()       # accumulated TwiML XML response
telephony.match()       # true when last ask/menu matched
```

## Webhook Fields

Recognised fields in the inbound webhook body (Twilio format; unknown fields are ignored):

| Field | Maps to |
|---|---|
| `CallSid` | `telephony.callId()` |
| `From` | `telephony.from()` |
| `To` | `telephony.to()` |
| `CallStatus` | call status |
| `Digits` | `telephony.digits()` |
| `SpeechResult` | `telephony.speech()` |
| `Confidence` | `telephony.confidence()` |

## See Also

- [LLM Resource](llm) -- generate spoken responses with an LLM
- [API Response](api-response) -- how the TwiML reaches the provider
- [Components](../concepts/components) -- `invoke:` targets for menu matches
