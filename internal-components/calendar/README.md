# calendar

> Generate iCalendar (.ics) events via Python.

The calendar component creates a standard `.ics` calendar file from event details. The generated file can be attached to emails, imported into any calendar application (Google Calendar, Apple Calendar, Outlook), or returned directly in an API response.

## Inputs

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `title` | string | yes | — | Event title / summary |
| `start` | string | yes | — | Event start datetime in ISO 8601 format (e.g. `2024-01-15T10:00:00`) |
| `end` | string | yes | — | Event end datetime in ISO 8601 format |
| `description` | string | no | `""` | Optional event description or notes |
| `outputFile` | string | no | `/tmp/event.ics` | Path to write the `.ics` file |

## Outputs

Accessed via `output('<your-action-id>')`:

| Field | Type | Description |
|-------|------|-------------|
| `ics` | string | Full iCalendar file contents as a UTF-8 string |

The `.ics` file is also written to `outputFile`.

## Usage

Create a basic meeting event:

```yaml
run:
  component:
    name: calendar
    with:
      title: "Team Sync"
      start: "2024-06-15T10:00:00"
      end: "2024-06-15T11:00:00"
      description: "Weekly team check-in"
      outputFile: /tmp/team-sync.ics
  apiResponse:
    success: true
    response:
      icsData: "{{ output('createEvent').ics }}"
```

Using dynamic values from previous workflow steps:

```yaml
run:
  component:
    name: calendar
    with:
      title: "{{ output('bookingStep').eventTitle }}"
      start: "{{ output('bookingStep').startTime }}"
      end: "{{ output('bookingStep').endTime }}"
```

## Requirements

- Python package `icalendar`: automatically available in the kdeps Python environment.

## Notes

- Datetimes must be valid ISO 8601 strings. Time zone info is not added automatically — use UTC offsets if needed (e.g., `2024-06-15T10:00:00+02:00`).
- The `.ics` output conforms to [RFC 5545](https://tools.ietf.org/html/rfc5545) and is compatible with all major calendar applications.
- To create recurring events, use the `ics` output string and post-process it, or extend the workflow with a `run.python:` resource.
- All-day events: use `YYYY-MM-DD` date strings without a time component.
