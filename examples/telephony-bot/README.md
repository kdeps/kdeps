# Telephony Bot Example

An AI phone IVR built on the native `telephony:` resource. A Twilio-compatible provider POSTs its call webhooks to this workflow; the workflow answers with TwiML: a keypad menu, a static opening-hours line, and an LLM-powered question line where the caller speaks a question and the model answers it as text for the provider glue to speak.

## Call flow

```text
inbound call -> POST /twilio/voice  -> menu: "Press 1 for hours, 2 for assistant"
press 1     -> POST /twilio/hours  -> say: opening hours
press 2     -> POST /twilio/ask    -> gather: spoken question
speech done -> POST /twilio/answer -> chat: LLM answers as text
```

Each webhook is a separate request. The provider drives the flow: it posts the caller's input (`Digits`, `SpeechResult`) to the next route, and each response contains the TwiML for the next call step.

## Prerequisites

- [Ollama](https://ollama.ai) with `ollama pull llama3.2:1b` (for the `/twilio/answer` route)
- A Twilio-compatible telephony provider pointing its voice webhook at this server

## Run

```bash
export KDEPS_API_AUTH_TOKEN=dev-token
kdeps run workflow.yaml --dev
```

Simulate the provider with curl:

```bash
# Welcome menu (no input yet -- returns a <Gather> prompt)
curl -sX POST http://localhost:16395/twilio/voice \
  -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"CallSid":"CA001","From":"+14155551234","To":"+18005559999"}'

# Caller pressed 1
curl -sX POST http://localhost:16395/twilio/hours \
  -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"CallSid":"CA001","Digits":"1"}'

# Caller spoke a question (requires Ollama)
curl -sX POST http://localhost:16395/twilio/answer \
  -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"CallSid":"CA001","SpeechResult":"What are your opening hours?","Confidence":0.92}'
```

Telephony responses contain the TwiML XML under the matching key, e.g.:

```json
{
  "success": true,
  "data": {
    "hours": {
      "twiml": "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<Response>\n  <Say voice=\"alice\">We are open Monday to Friday, nine to five, Central European Time.</Say>\n</Response>"
    }
  }
}
```

The `/twilio/answer` route returns the LLM answer as plain text under `data.answer` -- telephony `say:` fields are not template-interpolated, so the provider glue wraps the text in `<Say>` (or feeds it to TTS) itself.

Your provider integration extracts the `twiml` (or `answer`) field and returns TwiML to the provider as `text/xml`.

## See Also

- [Telephony Resource](../../docs/v2/resources/telephony.md) -- full action and field reference
- [LLM Resource](../../docs/v2/resources/llm.md) -- chat configuration
