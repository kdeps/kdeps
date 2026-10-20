# Telephony (twiml component)

The `twiml` component builds a TwiML voice response - the XML a phone provider such as Twilio executes on a call - from a list of verbs. A phone workflow is an ordinary API workflow: the provider POSTs each call step to a route, a resource builds the TwiML with `component: twiml`, and an `apiResponse` returns it as `text/xml`. It is a kdeps component (YAML, no Go code) from the `components` asset set.

This is workflow mode: the API server answers the provider's webhooks.

```d2
direction: right
call: "Caller" {shape: oval}
twilio: "Twilio"
route: "POST /twilio/voice\n(CallSid, From, Digits, SpeechResult)"
twiml: "component: twiml\nverbs -> <Response>"
api: "POST /apis/tools.kdeps.io/v1/\ntools/twiml/call"
resp: "apiResponse\nContent-Type: text/xml"
call -> twilio
twilio -> route
route -> twiml
twiml -> api
api -> resp
resp -> twilio: TwiML
```

## Basic usage

<div v-pre>

```yaml
# workflow.yaml (abridged)
settings:
  apiServer:
    routes:
      - path: /twilio/voice
        methods: [POST]
        public: true            # Twilio sends no bearer token
```

```yaml
# resources/greet.yaml
actionId: greet
name: Greet
component:
  name: twiml
  with:
    verbs:
      - say: {text: "Hello, {{ get('From') }}.", voice: alice}   # Twilio's form fields via get()
      - gather:
          input: dtmf              # dtmf | speech | "dtmf speech"
          numDigits: 1
          action: /twilio/menu     # Twilio posts the key here as Digits
          say: "Press 1 for sales."
      - hangup: true
```

```yaml
# resources/respond.yaml - the workflow's targetActionId
actionId: respond
name: TwiML Response
requires: [greet]
apiResponse:
  success: true
  headers:
    Content-Type: text/xml        # the body is written as-is, not wrapped in JSON
  response: "{{ output('greet').twiml }}"
```

</div>

The response:

```xml
<?xml version="1.0" encoding="UTF-8"?><Response><Say voice="alice">Hello, +15551234.</Say><Gather action="/twilio/menu" input="dtmf" numDigits="1"><Say>Press 1 for sales.</Say></Gather><Hangup /></Response>
```

Text is XML-escaped, so a caller's speech or an LLM answer can go straight into `say`.

## Verbs

Each item of `verbs` is a map with one key, the verb. A verb's value is a map of TwiML attributes (written as given), or the shorthand shown.

| Verb | Shorthand | Map form |
|---|---|---|
| `say` | `say: Hello` | `{text, voice, language, loop}` |
| `play` | `play: https://.../a.mp3` | `{url, loop}` |
| `pause` | `pause: 2` | `{length}` |
| `gather` | - | `{input, numDigits, timeout, speechTimeout, finishOnKey, action, method, hints, language, say, play}`; `say`/`play` nest inside the gather |
| `dial` | `dial: "+15551234"` | `{to, callerId, timeout, record, action}`; `to` is one target or a list; `sip:` targets become `<Sip>`, others `<Number>` |
| `record` | - | `{maxLength, playBeep, finishOnKey, action, transcribe, recordingStatusCallback}` |
| `hangup` | `hangup: true` | - |
| `reject` | `reject: busy` | `{reason}` |
| `redirect` | `redirect: /twilio/voice` | `{url, method}` |

## Call input

Twilio posts each step as a form: read the fields with `get()`.

| Field | Meaning |
|---|---|
| `get('CallSid')` | The call's ID |
| `get('From')`, `get('To')` | Caller and dialed numbers |
| `get('Digits')` | Keys pressed in the last `gather` |
| `get('SpeechResult')` | Transcript of the last speech `gather` |

A menu is a `gather` whose `action` route picks the next step from `Digits`:

<div v-pre>

```yaml
# resources/menu.yaml - route /twilio/menu
component:
  name: twiml
  with:
    verbs:
      - redirect: "{{ get('Digits') == '1' ? '/twilio/sales' : '/twilio/voice' }}"
```

</div>

## Output

`{twiml: "<?xml ...><Response>...</Response>"}`. An unknown verb fails the call with `twiml: unknown verb 'x'`.

## How it is built

The component holds no XML builder. Its one `httpClient` resource POSTs `{verbs}` to the kdeps API's `twiml` tool (`tools/twiml/call`), and `interface.output` unwraps the `{twiml: ...}` reply into the component result. Copy `components/twiml/` into your project to call the same tool with more verbs or to add another provider's XML dialect.

## See also

- [Phone assistant tutorial](/examples/telephony-bot) - a menu, a static answer and an LLM answer
- [API response](/workflow/resources/api-response) - raw `text/xml` bodies
- [Input sources](/workflow/input-sources) - the API server routes
