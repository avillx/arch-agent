# Tasks

A **task** is an autonomous request that the system sends to one or more agents
on a cron schedule — for example, a daily digest, a periodic health check, or a
nightly report. Tasks run without any user interaction: when the schedule fires,
the request is delivered to every recipient agent, and each agent works on it on
its own.

Tasks are declared in `tasks.toml` ([configuration.md](configuration.md)):

```toml
[weekly-report]
description = 'Send a weekly summary of project activity'
recipients = ['analyst', 'default']
schedule = '0 9 * * 1'
active = true
once = false
request = """
Look at the memory and activity of the last week and prepare a short report.
Save the report to ./shared/reports/weekly.md and confirm the result.
"""
```

## Task settings

| Setting | Meaning |
|---------|---------|
| task name | Table name — unique task id |
| `description` | Short one-line description |
| `recipients` | Agent ids that receive the request; at least one is required and each must exist |
| `schedule` | Cron expression defining when the task fires |
| `active` | `true` — scheduled and running; `false` — defined but idle |
| `once` | One-shot task: after the first run it is switched off automatically |
| `request` | The request text the recipients work on; prefer the multiline `"""..."""` form |

## Validation

A task is validated when added through the API and when the file is loaded:

- `name`, `description`, `request` must be non-empty;
- at least one `recipients` entry is required, and every recipient must be an
  existing agent;
- `schedule` must be a valid cron expression — an invalid expression is
  rejected.

## Behaviour

- Each run creates a **fresh session per recipient** and sends the request
  there. Recipients work independently, in parallel.
- The agent works autonomously: it gets the request as its only message and uses
  its tools to complete it. The run is logged like a normal conversation.
- One-shot tasks (`once = true`) are switched off after their first execution.
- Task state is loaded on start and hot-reloaded: new tasks are scheduled,
  changed tasks are restarted with the new settings, removed tasks are stopped.

## Managing tasks via the API

The HTTP API allows listing, adding, updating (partial patch) and deleting
tasks. See [api.md](api.md).