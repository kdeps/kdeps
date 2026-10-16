# Telephony Bot Example

An AI phone IVR as a plain API workflow. Twilio POSTs each call step to a route; each route answers with TwiML built by the `twiml` component and returned as `text/xml`: a keypad menu, a static opening-hours line, and a question line where the caller speaks and a local LLM answers out loud.

## Call flow

```text
inbound call -> POST /twilio/voice  -> gather 1 key: "Press 1 for hours, 2 for assistant"
key pressed  -> POST /twilio/menu   -> redirect by Digits: 1 -> /hours, 2 -> /ask, else -> /voice
             -> POST /twilio/hours  -> say opening hours, hang up
             -> POST /twilio/ask    -> gather speech
speech done  -> POST /twilio/answer -> LLM answers SpeechResult -> say it, hang up
```

Twilio drives the flow: it posts the caller's input (`Digits`, `SpeechResult`) to the route named in the last `gather` or `redirect`.

## Prerequisites

- No LLM install needed - `llama3.2:1b` runs as a local llamafile, auto-downloaded on first run (~1.1 GB; used by `/twilio/answer`)
- uv (the `twiml` component is a python resource)
- For a live call: a Twilio number whose voice webhook is `https://<host>/twilio/voice`

## Run

```bash
export KDEPS_API_AUTH_TOKEN=dev-token   # the /twilio routes are public; other routes need it
kdeps run workflow.yaml
```

Simulate Twilio with curl (form posts, no token):

```bash
curl -sX POST http://localhost:16395/twilio/voice -d CallSid=CA001 -d From=+14155551234
curl -sX POST http://localhost:16395/twilio/menu -d Digits=1
curl -sX POST http://localhost:16395/twilio/hours
curl -sX POST http://localhost:16395/twilio/answer -d "SpeechResult=What are your opening hours?"
```

Each returns TwiML, e.g. `/twilio/hours`:

```xml
<?xml version="1.0" encoding="UTF-8"?><Response><Say voice="alice">We are open Monday to Friday, nine to five, Central European Time.</Say><Hangup /></Response>
```

## See Also

- [twiml component](https://kdeps.com/workflow/resources/telephony) - verbs and attributes
- [Phone assistant tutorial](https://kdeps.com/examples/telephony-bot)
