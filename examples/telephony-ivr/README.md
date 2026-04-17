# telephony-ivr

Programmable IVR (Interactive Voice Response) using `run.telephony` actions.
Mirrors the Adhearsion `CallController` model with `answer`, `menu`, `say`, `dial`,
`record`, `ask`, `hangup`, `reject`, `redirect`, `mute`, and `unmute` actions.

## Flow

```
Inbound call
  -> answer
  -> mainMenu (press 1/2/3)
       1 -> salesGreeting -> dialSales
       2 -> supportGreeting -> dialSupport
       3 -> recordVoicemail -> goodbye -> hangup
       no match/no input -> repeatMenu
```

## Actions reference

| Action     | Description                                      |
|------------|--------------------------------------------------|
| `answer`   | Accept the inbound call                          |
| `say`      | Text-to-speech. Optional `voice` (e.g. `alice`)  |
| `ask`      | Collect DTMF/speech. Use `limit` or `grammar`    |
| `menu`     | Gather + branch dispatch via `matches`           |
| `dial`     | Transfer to SIP URI or PSTN number(s)            |
| `record`   | Record caller audio; `maxDuration`, `interruptible` |
| `hangup`   | Terminate the call                               |
| `reject`   | Reject inbound call; `reason: busy|rejected`     |
| `redirect` | Redirect to another TwiML URL                    |
| `mute`     | Mute the caller leg                              |
| `unmute`   | Unmute the caller leg                            |

## TwiML output

Each resource appends TwiML nodes to the shared call session. The complete
`<Response>` is serialized at the end of the workflow and returned as the
HTTP response body to Twilio.

## Session accessors (in expressions)

```yaml
say: "You pressed {{ telephony.utterance }}."
```

| Key          | Type    | Description                         |
|--------------|---------|-------------------------------------|
| `callId`     | string  | Twilio `CallSid`                    |
| `from`       | string  | Caller number                       |
| `to`         | string  | Called number                       |
| `status`     | string  | `match`, `nomatch`, `noinput`, ...  |
| `utterance`  | string  | Digits pressed or speech recognized |
| `digits`     | string  | Raw DTMF digits                     |
| `speech`     | string  | Raw speech result                   |
| `confidence` | float64 | ASR confidence (0..1)               |
| `match`      | bool    | true when status=match              |
| `twiml`      | string  | Current accumulated TwiML           |

## Prerequisites

- Twilio account with a phone number configured to POST to your workflow URL
- kdeps running with `apiServerMode: true`
- Port 16396 reachable from Twilio (ngrok or public IP)

## Quick start

```bash
kdeps run examples/telephony-ivr
```

Configure your Twilio number's "A call comes in" webhook to:

```
POST https://<your-host>:16396/
```
