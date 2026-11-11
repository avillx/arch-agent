---
name: ard-tasks-managing
description: |
  How to gracefully manage tasks. Read this skill if you need to create or
  manage scheduled tasks. Do not read it if you are not ready to manage
  tasks — when you just need to know about existing tasks, read the config
  instead of this skill.
  It fits when the user says something like "do this" or "stop doing this",
  implying it should be done continuously, repeatedly, or after some time.
version: 1.0.0
author: avillx
---

# Schedule Manager

You know how to create and manage scheduled tasks.


## Task

A task is a autonomus request for an agent (you or another) invoked by the 
cron planner. 


## Available agents

A task can invoke multiple agents specified as recipients.

To see all existing agents available for tasks, check `CWD`; it contains
configs and folders, except for technical folders (`skills`, `shared`).
Each folder corresponds to an agent, e.g. the folder `./example-agent`
means the `example-agent` exists and is available as a recipient for a
task.


## Config

Config is a `toml` file in your `CWD`: [Config]({your_cwd}/tasks.toml).
It contains all tasks for all agents in the system.

Fields:
- `name` a unique identifier in snake_case for the task in the config
  (a toml `object` key).
- `description` a short, one-line hook description.
- `recipients` an array of agent ids. All agents in this array receive the
  request on schedule.
- `schedule` a cron expression that plans the scheduled invocation.
- `active` true/false task state. If the task is inactive, it is never
  invoked.
- `once` true/false. If a task has `once=true`, it is disabled after its
  first execution and `active` is set to false. Use it for one-shot
  invocations, reminders, etc.
- `request` the text for the recipient.

> all fields if required

Example:
```toml
[server_health]
description="Daily check server health"
recipients=["beta"]
schedule="0 17 * * *"
active=true
once=false
request="""
# Check server health

- Check the logs from the last 24h; use a sub-agent to explore the logs.
- Check free space. It should be greater than 100gb.
- Run RKhunter.

> NEVER try to clean up space yourself!

If something goes wrong or you detect problems, contact the user using
  [notify](../notify/SKILL.md).

If you can't contact anyone, create a README.md file describing the issues
about the server and the problems reaching the user.

1. Run a sub-agent with the list of logs to check and have it summarize the results.
2. If there are issues, investigate them yourself.
3. Send the user an alert about the problems.
"""
```


## Request

A request is an exhaustive prompt that gives the agent full context to
successfully complete the task while working autonomously.

Guides:
- Prefer the multiline `"""..."""` format over the single-line `'...'`
  format.
- Agents receive this request with a clean context. You should describe
  all details exhaustively.
- Use English in the prompt. If the agent should use another language,
  mention it explicitly, e.g. `use Spanish in docs`. Do not specify a
  language if the agent only performs a silent automatic process and
  shouldn't contact the user or work with documents in a non-English
  language.
- When it fits, tell the recipient to use a skill (e.g.
  `use [code](.../SKILL.md)`).
- Agents in these invocations work autonomously and, without special
  tools and skills, cannot contact anyone else. Do not require the agent
  to contact the user before you're sure it can do so.
- Do not demand something from the agent if you're not sure the agent can
  reach that goal.
- If the task is repetitive and too complex (5+ tool calls), consider
  creating a separate skill, and make the request a thin hint to use that
  skill, with only the necessary details in the request.
- Think about the recipient's capabilities. It runs on the same system as
  you and has access to `mcp`, `memory`, `logs`, `models`, `secrets`, and
  `tasks`. Feel free to instruct it to use its full capabilities.

Struct:
- `task` - what the agent should do.
- `details` - details the agent should know about the task.
- `failure mode` - how the agent should report if it has issues: to the
  user, or by creating a report file.
- `verification` - how the agent can verify the task is completed or has
  failed.
- `steps` *optional* - if the task is complex, create a multi-step
  workflow.
- `guard rails` *optional* - agents sometimes try to find a hacky way to
  do something, or do things nobody asked for. If the task domain implies
  the agent might attempt more than required, caution the agent to do
  only what it should.
- `skill` *optional* - you can mention several skills to quickly give the
  agent context.


## Verify

If a skill implies contacting the user, make sure this is actually
possible for the agent. Being in a session with the user right now
doesn't mean an agent woken up autonomously (e.g. by the scheduler) has
the same capability. The only way to contact the user is through MCP
servers with that capability, or special tools/skills that let the agent
create a communication channel with the user by itself.
If you not sure, call agent as sub agent and ask it directly.

After you edit the config, the system automatically picks up the
changes. Check the logs for issues — if `sys.log` contains errors in
the tasks group, that means you made a mistake.


## Workflow

1. Read the config.
2. Edit the config.
3. Verify the tasks activated successfully.
