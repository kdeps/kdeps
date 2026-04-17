# Telephony Resource

The `run.telephony` resource provides programmable call control that generates TwiML responses for Twilio webhooks. Supported actions: `say`, `ask`, `menu`, `dial`, `record`, `hangup`, `reject`, `redirect`, `mute`, and `unmute`.

## Basic Usage

```yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: greet
  name: Greet Caller

run:
  telephony:
    action: say
    say: "Hello! Welcome to our service."
```

## Workflow Setup

Enable telephony input in the workflow settings:

```yaml
settings:
  input:
    sources: [telephony]
    telephony:
      type: online
      provider: twilio
```

## Actions

### `answer`

Accept the inbound call. Usually the first resource in an IVR workflow.

```yaml
run:
  telephony:
    action: answer
```

### `say`

Speak text to the caller using text-to-speech.

| Option  | Description                                |
|---------|--------------------------------------------|
| `say`   | Text to speak.                             |
| `voice` | TTS voice (e.g. `alice`, `man`, `woman`).  |

```yaml
run:
  telephony:
    action: say
    say: "Your account balance is available."
    voice: alice
```

### `ask`

Collect DTMF digits or speech from the caller.

| Option              | Description                                                |
|---------------------|------------------------------------------------------------|
| `say`               | Prompt text.                                               |
| `mode`              | `dtmf` (default), `speech`, or `both`.                     |
| `limit`             | Max digits to collect.                                     |
| `grammar`           | Inline GRXML grammar string.                               |
| `grammarUrl`        | URL to a GRXML grammar file.                               |
| `timeout`           | Wait time for input (e.g. `5s`).                           |
| `interDigitTimeout` | Timeout between digit presses.                             |
| `terminator`        | Key that ends input early (e.g. `#`).                      |

```yaml
run:
  telephony:
    action: ask
    say: "Please enter your 4-digit PIN."
    limit: 4
    terminator: "#"
    timeout: 10s
```

### `menu`

Gather a single digit and branch based on the caller's selection.

| Option      | Description                                               |
|-------------|-----------------------------------------------------------|
| `say`       | Prompt text.                                              |
| `audio`     | Audio URL to play instead of TTS.                         |
| `matches`   | List of `{keys, invoke}` branch descriptors.              |
| `onNoMatch` | Action ID to invoke when no branch matches.               |
| `onNoInput` | Action ID to invoke on timeout.                           |
| `onFailure` | Action ID to invoke after max retries.                    |
| `tries`     | Number of retry attempts before `onFailure`.              |
| `timeout`   | Input timeout (e.g. `8s`).                                |

```yaml
run:
  telephony:
    action: menu
    say: "Press 1 for sales. Press 2 for support."
    timeout: 8s
    matches:
      - keys: ["1"]
        invoke: salesFlow
      - keys: ["2"]
        invoke: supportFlow
    onNoMatch: repeatMenu
    onNoInput: repeatMenu
```

### `dial`

Transfer the call to one or more SIP URIs or PSTN numbers.

| Option  | Description                                             |
|---------|---------------------------------------------------------|
| `to`    | List of dial targets (SIP URIs or E.164 numbers).       |
| `from`  | Caller ID to present (optional).                        |
| `for`   | Dial timeout (e.g. `30s`).                              |

```yaml
run:
  telephony:
    action: dial
    to:
      - sip:agent@pbx.example.com
      - "+15005550001"
    for: 30s
```

### `record`

Record the caller's audio.

| Option          | Description                                          |
|-----------------|------------------------------------------------------|
| `say`           | Prompt before recording.                             |
| `maxDuration`   | Max recording length (e.g. `60s`).                   |
| `interruptible` | Allow the caller to press a key to stop recording.   |
| `format`        | Audio format (`mp3`, `wav`).                         |

```yaml
run:
  telephony:
    action: record
    say: "Leave your message after the beep."
    maxDuration: 60s
    interruptible: true
```

### `hangup`

Terminate the call.

```yaml
run:
  telephony:
    action: hangup
```

### `reject`

Reject an inbound call before it is answered.

| Option   | Description                               |
|----------|-------------------------------------------|
| `reason` | `busy` or `rejected` (default: `rejected`). |

```yaml
run:
  telephony:
    action: reject
    reason: busy
```

### `redirect`

Redirect the call to another TwiML URL.

| Option | Description                              |
|--------|------------------------------------------|
| `to`   | List with one URL to redirect to.        |

```yaml
run:
  telephony:
    action: redirect
    to:
      - https://example.com/after-hours-ivr
```

### `mute` / `unmute`

Mute or unmute the caller's audio leg.

```yaml
run:
  telephony:
    action: mute
```

## Session Accessors in Expressions

After an `ask` or `menu` action, the session state is available in subsequent resource expressions:

| Key          | Type    | Description                                       |
|--------------|---------|---------------------------------------------------|
| `callId`     | string  | Twilio `CallSid`                                  |
| `from`       | string  | Caller's number                                   |
| `to`         | string  | Called number                                     |
| `status`     | string  | `match`, `nomatch`, `noinput`, `hangup`, or `stop`|
| `utterance`  | string  | Digits pressed or speech recognized               |
| `digits`     | string  | Raw DTMF digits from current request              |
| `speech`     | string  | Raw speech result from current request            |
| `confidence` | float64 | ASR confidence score (0.0 - 1.0)                 |
| `match`      | bool    | `true` when `status == "match"`                   |
| `twiml`      | string  | Accumulated TwiML for the current call            |

```yaml
run:
  telephony:
    action: say
    say: "You pressed {{ telephony.utterance }}. Thank you."
```

## TwiML Accumulation

All `run.telephony` resources in a single workflow execution share one `Session`. Each action appends TwiML nodes to the same `<Response>` document. The complete TwiML is returned to Twilio as the HTTP response body at the end of the workflow.

## Example: Full IVR

See `examples/telephony-ivr/` for a complete IVR workflow with answer, menu, say, dial, record, and hangup.
