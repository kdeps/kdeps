# Calendar Resource

> **Note**: This capability is now provided as an installable component. See the [Components guide](../concepts/components) for how to install and use it.
>
> Install: `kdeps component install calendar`
>
> Usage: `run: { component: { name: calendar, with: { title: "...", start: "...", end: "...", outputFile: "/tmp/event.ics" } } }`

The Calendar component generates iCalendar (`.ics`) event files from structured inputs using Python.

> **Note**: The component creates ICS event files. For reading, modifying, or deleting existing ICS events, use a Python resource with the `icalendar` library directly.

## Component Inputs

| Input | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `title` | string | yes | — | Event title / summary |
| `start` | string | yes | — | Event start time in ISO 8601 format (e.g. `2024-01-15T10:00:00`) |
| `end` | string | yes | — | Event end time in ISO 8601 format |
| `description` | string | no | — | Optional event description |
| `outputFile` | string | no | `/tmp/event.ics` | Path to write the generated `.ics` file |

## Using the Calendar Component

```yaml
run:
  component:
    name: calendar
    with:
      title: "Team Standup"
      start: "2025-01-15T09:00:00"
      end: "2025-01-15T09:30:00"
      description: "Daily sync"
      outputFile: "/tmp/standup.ics"
```

Access the result via `output('<callerActionId>')`.

---

## Result Map

| Key | Type | Description |
|-----|------|-------------|
| `success` | bool | `true` when the ICS file was written successfully. |
| `outputFile` | string | Absolute path to the generated `.ics` file. |

---

## Expression Support

All fields support [KDeps expressions](../concepts/expressions):

<div v-pre>

```yaml
run:
  component:
    name: calendar
    with:
      title: "{{ get('meeting_title') }}"
      start: "{{ get('meeting_start') }}"
      end: "{{ get('meeting_end') }}"
      description: "{{ get('agenda') }}"
      outputFile: /tmp/meeting.ics
```

</div>

---

## Full Example: Meeting Scheduler Pipeline

<div v-pre>

```yaml
# Step 1: LLM parses user request into structured event data
- apiVersion: kdeps.io/v1
  kind: Resource

  metadata:
    actionId: parse-request
    name: Parse Meeting Request

  run:
    chat:
      model: gpt-4o
      prompt: |
        Extract meeting details from this request as JSON with keys:
        title, start (ISO 8601), end (ISO 8601), description.

        Request: {{ get('user_request') }}

# Step 2: Create the ICS event file
- apiVersion: kdeps.io/v1
  kind: Resource

  metadata:
    actionId: create-event
    name: Create Calendar Event
    requires:
      - parse-request

  run:
    component:
      name: calendar
      with:
        title: "{{ output('parse-request').title }}"
        start: "{{ output('parse-request').start }}"
        end: "{{ output('parse-request').end }}"
        description: "{{ output('parse-request').description }}"
        outputFile: /tmp/event.ics

# Step 3: Email the ICS file
- apiVersion: kdeps.io/v1
  kind: Resource

  metadata:
    actionId: send-invite
    name: Send Calendar Invite
    requires:
      - create-event

  run:
    component:
      name: email
      with:
        to: "{{ get('attendee_email') }}"
        subject: "Invite: {{ output('parse-request').title }}"
        body: "Please find your calendar event attached."
        smtpHost: "{{ env('SMTP_HOST') }}"
        smtpUser: "{{ env('SMTP_USER') }}"
        smtpPass: "{{ env('SMTP_PASS') }}"

# Step 4: Return confirmation
- apiVersion: kdeps.io/v1
  kind: Resource

  metadata:
    actionId: confirm
    name: Return Confirmation
    requires:
      - send-invite

  run:
    apiResponse:
      success: true
      response:
        icsFile: "{{ output('create-event').outputFile }}"
        title: "{{ output('parse-request').title }}"
```

</div>
