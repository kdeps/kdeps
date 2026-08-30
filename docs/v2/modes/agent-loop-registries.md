# Agent Registries

The [agent loop](/modes/agent-loop-mode) maintains three in-memory registries for lifecycle management: tasks, teams, and cron schedules.

## TaskRegistry

Tracks every task created by the agent loop. Each task has a unique ID (`task-N`), status (`created` -> `running` -> `completed`/`failed`/`stopped`), description, prompt, and an append-only output and message transcript. Tasks can be assigned to a team and carry a heartbeat for stall detection.

| Method | Description |
|--------|-------------|
| `Create(prompt, description)` | Create a new task in `created` state |
| `Get(taskID)` | Look up a task by ID |
| `List()` | All tasks, newest first |
| `ListByStatus(status)` | Filter by status |
| `SetStatus(taskID, status)` | Transition to a new status |
| `Stop(taskID)` | Set status to `stopped` |
| `AppendOutput(taskID, text)` | Append to task output |
| `AppendMessage(taskID, msg)` | Append to message transcript |
| `AssignTeam(taskID, teamID)` | Attach a team |
| `UpdateHeartbeat(taskID, alive)` | Record lane aliveness |
| `StalledTasks(stalledAfter)` | Running tasks with stale heartbeats |
| `Delete(taskID)` | Remove a task from the registry |

The LLM manages tasks through these tools:

| Tool | Description |
|------|-------------|
| `task_create` | Create a new task (`prompt`, `description`). Returns the task ID |
| `task_get` | Get a task's status, timestamps, output, and assigned team by `task_id` |
| `task_list` | List all tasks, newest first |
| `task_stop` | Stop a running or created task by `task_id` |
| `task_complete` | Mark a task as completed |
| `task_append_output` | Append text to a task's output log |
| `task_assign_team` | Assign a task to a team for multi-agent coordination |

## TeamRegistry

Groups tasks for multi-agent coordination. Each team has a name, a list of task IDs, and a status (`created` -> `running` -> `completed` -> `deleted`).

| Method | Description |
|--------|-------------|
| `Create(name)` | Create a new team |
| `Get(teamID)` | Look up a team by ID |
| `List()` | All teams |
| `AddTask(teamID, taskID)` | Assign a task to a team |
| `SetStatus(teamID, status)` | Update team status |
| `Delete(teamID)` | Mark as deleted |

The LLM manages teams through these tools:

| Tool | Description |
|------|-------------|
| `team_create` | Create a new team (`name`). Returns the team ID |
| `team_get` | Get a team's task IDs, status, and name by `team_id` |
| `team_list` | List all teams |
| `team_add_task` | Assign an existing task to a team |
| `team_delete` | Delete (mark as deleted) a team |

## CronRegistry

Schedules recurring task creation from the agent loop process. Each cron job stores a cron expression, prompt/description templates, and tracks last/next run times. **Cron jobs fire automatically** — the server starts a background goroutine that calls `Tick()` every 60 seconds and creates tasks for any due jobs.

| CLI tool | Description |
|----------|-------------|
| `cron_create` | Create a new cron job with expression, prompt, and description |
| `cron_list` | List all cron jobs with status, last/next run times |
| `cron_pause` / `cron_resume` | Pause or resume a cron job |
| `cron_delete` | Delete a cron job |

No manual polling or goroutine setup needed. Start `kdeps path/to/agent/` and cron runs in the background.

## See Also

- [Agent Loop Mode](/modes/agent-loop-mode) -- overview and starting the REPL
- [Built-in Tools](/modes/agent-loop-tools) -- the full tool catalog these registries add to
- [Goal-Directed Execution](/modes/agent-loop-goals) -- how individual tasks are driven to completion
