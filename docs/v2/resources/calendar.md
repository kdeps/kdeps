# Calendar Resource

The calendar resource supports four actions that cover the full read/write lifecycle of
ICS calendar workflows: **list** (read events), **create** (add a new event), **modify**
(update an existing event), and **delete** (remove an event).  All operations work against
a local `.ics` file — the path is resolved relative to the agent's FSRoot unless an
absolute path is given.

All string fields support [KDeps expressions](../concepts/expressions) such as
<span v-pre>`{{get(...)}}` and `{{env(...)}}`</span>.

---

## Quick Reference

| Action | Description |
|---|---|
| `list` | Read events from an ICS file, optionally filtered by date range or keyword |
| `create` | Append a new calendar event to an ICS file |
| `modify` | Update an existing event identified by its UID |
| `delete` | Remove an event identified by its UID from an ICS file |

The `action` field defaults to `list`, so a bare `calendar:` block with only `filePath`
will return all events.

---

## Basic Usage

```yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: listMeetings
  name: List Meetings

run:
  calendar:
    filePath: /data/team.ics
```

---

## Actions

### List

The `list` action reads events from an ICS file and returns them as a structured array.
Use the filtering fields (`since`, `before`, `limit`, `search`) to narrow the result set.

```yaml
run:
  calendar:
    action: list   # optional — "list" is the default
    filePath: calendars/team.ics
    since: "2025-01-01"
    before: "2025-04-01"
    limit: 50
    search: "standup"
    timeoutDuration: 30s
```

#### List Filtering

| Field | Type | Description |
|---|---|---|
| `since` | string | Return events whose start time is on or after this value. Accepts `YYYY-MM-DD` or RFC3339. |
| `before` | string | Return events whose start time is before this value. Accepts `YYYY-MM-DD` or RFC3339. |
| `limit` | int | Maximum number of events to return (most recent first). Omit for all. |
| `search` | string | Case-insensitive substring match against the event `summary` and `description`. |

All filters are optional and are ANDed together when multiple are specified.

#### List Result Map

```json
{
  "success": true,
  "action": "list",
  "count": 2,
  "events": [
    {
      "uid": "abc123@example.com",
      "summary": "Weekly Standup",
      "description": "Team sync",
      "location": "Zoom",
      "start": "2025-03-10T09:00:00Z",
      "end": "2025-03-10T09:30:00Z",
      "allDay": false,
      "attendees": ["alice@example.com", "bob@example.com"],
      "recurrence": "FREQ=WEEKLY;BYDAY=MO"
    }
  ]
}
```

| Field | Type | Description |
|---|---|---|
| `success` | bool | `true` when the operation completed without error. |
| `action` | string | Always `"list"`. |
| `count` | int | Number of events returned. |
| `events` | []object | Array of event objects (see fields below). |

**Event fields:**

| Field | Type | Description |
|---|---|---|
| `uid` | string | Unique identifier of the event (`UID` property). |
| `summary` | string | Event title (`SUMMARY` property). |
| `description` | string | Long-form description (`DESCRIPTION` property). |
| `location` | string | Location string (`LOCATION` property). |
| `start` | string | Event start time in RFC3339 format. |
| `end` | string | Event end time in RFC3339 format. |
| `allDay` | bool | `true` when the event has a `DATE` (not `DATE-TIME`) value. |
| `attendees` | []string | List of attendee email addresses. |
| `recurrence` | string | RRULE string if the event recurs (e.g. `FREQ=WEEKLY;BYDAY=MO`). |

---

### Create

The `create` action appends a new `VEVENT` component to an existing ICS file (or creates
the file if it does not exist).  Supply at minimum a `summary` and a `start` time.

```yaml
run:
  calendar:
    action: create
    filePath: calendars/team.ics
    uid: "sprint-review-2025-03-28@acme.com"
    summary: "Sprint Review"
    description: "End-of-sprint demo and retrospective"
    location: "Conference Room A"
    start: "2025-03-28T14:00:00Z"
    end: "2025-03-28T15:30:00Z"
    allDay: false
    attendees:
      - alice@example.com
      - bob@example.com
      - carol@example.com
    recurrence: "FREQ=WEEKLY;INTERVAL=2;BYDAY=FR"
    timeoutDuration: 30s
```

#### Create Result Map

```json
{
  "success": true,
  "action": "create",
  "uid": "sprint-review-2025-03-28@acme.com"
}
```

| Field | Type | Description |
|---|---|---|
| `success` | bool | `true` when the event was written successfully. |
| `action` | string | Always `"create"`. |
| `uid` | string | The UID of the newly created event. Auto-generated if not supplied. |

Access this result from downstream resources:

```yaml
# In a later resource:
run:
  apiResponse:
    success: true
    response:
      created_uid: "{{get('createMeeting.uid')}}"
```

---

### Modify

The `modify` action locates an existing event by its `uid` and updates any supplied
fields.  Fields that are not specified are left unchanged.

```yaml
run:
  calendar:
    action: modify
    filePath: calendars/team.ics
    uid: "sprint-review-2025-03-28@acme.com"
    summary: "Sprint Review (rescheduled)"
    start: "2025-03-28T15:00:00Z"
    end: "2025-03-28T16:30:00Z"
    location: "Conference Room B"
    timeoutDuration: 30s
```

#### Modify Result Map

```json
{
  "success": true,
  "action": "modify",
  "uid": "sprint-review-2025-03-28@acme.com"
}
```

| Field | Type | Description |
|---|---|---|
| `success` | bool | `true` when the event was updated successfully. |
| `action` | string | Always `"modify"`. |
| `uid` | string | The UID of the modified event. |

---

### Delete

The `delete` action removes the event with the given `uid` from the ICS file.

```yaml
run:
  calendar:
    action: delete
    filePath: calendars/team.ics
    uid: "sprint-review-2025-03-28@acme.com"
    timeoutDuration: 30s
```

#### Delete Result Map

```json
{
  "success": true,
  "action": "delete",
  "uid": "sprint-review-2025-03-28@acme.com"
}
```

| Field | Type | Description |
|---|---|---|
| `success` | bool | `true` when the event was removed successfully. |
| `action` | string | Always `"delete"`. |
| `uid` | string | The UID of the deleted event. |

---

## Filtering Reference

The following fields are only evaluated when `action: list`:

| Field | Type | Default | Description |
|---|---|---|---|
| `since` | string | — | Include events with `start >= since`. Accepts `YYYY-MM-DD` or RFC3339. |
| `before` | string | — | Include events with `start < before`. Accepts `YYYY-MM-DD` or RFC3339. |
| `limit` | int | — | Cap the number of returned events. |
| `search` | string | — | Substring filter on `summary` and `description` (case-insensitive). |

---

## Expression Support

All string fields and per-item list entries across every action are evaluated using
the KDeps expression engine.  This lets you inject runtime values from previous steps,
environment variables, or request parameters:

```yaml
calendar:
  action: create
  filePath: "{{env('CALENDAR_PATH')}}"
  uid: "meeting-{{get('booking_id')}}@example.com"
  summary: "{{get('meeting_title')}}"
  description: "{{get('meeting_notes')}}"
  location: "{{get('meeting_room')}}"
  start: "{{get('meeting_start')}}"
  end: "{{get('meeting_end')}}"
  attendees:
    - "{{get('organizer_email')}}"
    - "{{get('attendee_email')}}"
```

List filtering fields support expressions equally:

```yaml
calendar:
  action: list
  filePath: "{{env('CALENDAR_PATH')}}"
  since: "{{get('range_start')}}"
  before: "{{get('range_end')}}"
  search: "{{get('keyword')}}"
```

---

## Calendar as an Inline Resource

Create or delete a calendar event **before** or **after** the main resource action using
the `before` / `after` inline blocks.  This is useful for booking confirmations or
cleanup without making the calendar operation the primary step:

```yaml
run:
  after:
    - calendar:
        action: create
        filePath: "{{env('CALENDAR_PATH')}}"
        summary: "Follow-up: {{get('candidate_name')}}"
        start: "{{get('interview_date')}}"
        end: "{{get('interview_end')}}"
        attendees:
          - "{{get('recruiter_email')}}"
          - "{{get('candidate_email')}}"

  llm:
    prompt: "Summarise the interview notes for {{get('candidate_name')}}."
```

---

## Full Example: Meeting Scheduler Pipeline

This pipeline creates a team meeting, lists upcoming meetings to confirm the slot, then
deletes a previously cancelled event.

```yaml
# Step 1: Create the team meeting
- apiVersion: kdeps.io/v1
  kind: Resource

  metadata:
    actionId: create-team-meeting
    name: Create Team Meeting

  run:
    calendar:
      action: create
      filePath: /data/calendars/team.ics
      uid: "team-meeting-{{get('week_id')}}@acme.com"
      summary: "Weekly Team Sync"
      description: "{{get('agenda')}}"
      location: "https://meet.acme.com/team"
      start: "{{get('meeting_start')}}"
      end: "{{get('meeting_end')}}"
      attendees:
        - alice@acme.com
        - bob@acme.com
        - carol@acme.com
      recurrence: "FREQ=WEEKLY;BYDAY=MO"
      timeoutDuration: 30s

# Step 2: List upcoming meetings to confirm the slot is correctly booked
- apiVersion: kdeps.io/v1
  kind: Resource

  metadata:
    actionId: list-upcoming
    name: List Upcoming Meetings
    requires:
      - create-team-meeting

  run:
    calendar:
      action: list
      filePath: /data/calendars/team.ics
      since: "{{get('meeting_start')}}"
      limit: 5
      timeoutDuration: 30s

# Step 3: Delete the cancelled event from last sprint
- apiVersion: kdeps.io/v1
  kind: Resource

  metadata:
    actionId: delete-cancelled
    name: Delete Cancelled Event
    requires:
      - list-upcoming

  run:
    calendar:
      action: delete
      filePath: /data/calendars/team.ics
      uid: "{{get('cancelled_event_uid')}}"
      timeoutDuration: 30s
```

---

## Full Example: Daily Briefing Pipeline

This pipeline lists today's events from a calendar file and passes them to an LLM for
a natural-language morning briefing.

```yaml
# Step 1: List today's events
- apiVersion: kdeps.io/v1
  kind: Resource

  metadata:
    actionId: todays-events
    name: Fetch Today's Calendar

  run:
    calendar:
      action: list
      filePath: "{{env('PERSONAL_CALENDAR')}}"
      since: "{{get('today_start')}}"   # e.g. 2025-03-16T00:00:00Z
      before: "{{get('today_end')}}"    # e.g. 2025-03-16T23:59:59Z
      timeoutDuration: 30s

# Step 2: LLM generates the briefing
- apiVersion: kdeps.io/v1
  kind: Resource

  metadata:
    actionId: morning-briefing
    name: Morning Briefing
    requires:
      - todays-events

  run:
    llm:
      prompt: |
        You are a helpful assistant. Here are the user's calendar events for today:

        {{get('todays-events.events')}}

        Write a concise, friendly morning briefing that summarises the day ahead,
        highlights any back-to-back meetings, and suggests preparation notes for
        the most important event.

# Step 3: Return the briefing as an API response
- apiVersion: kdeps.io/v1
  kind: Resource

  metadata:
    actionId: briefing-response
    name: Briefing Response
    requires:
      - morning-briefing

  run:
    apiResponse:
      success: true
      response:
        briefing: "{{get('morning-briefing')}}"
        event_count: "{{get('todays-events.count')}}"
```
