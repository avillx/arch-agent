# ARCH.agent

> The project is Agent SDK with memory. written in golang

Key points:
- Project focus on control clean context as possible and present only necceccary accesses
- Agent SDK is designed for work as server daemon, not as desktop app.
- File database is a decigion for minimize infrastructure.
- Agents can works autonomusly by cron scheduling.
- If agent has memory or works autonomusly it logging consolidated activity.
- Has MCP integration. **Stramable http** and **stdio**. support env and token auth for http

# Memory

Divided into 2 parts:

* **Activity Logs** — formatted into time blocks, continuously written, consolidated from raw transcripts.
* **Persistent Memory** — consolidated and indexed knowledge written by another agent based on activity logs.

Stored in agent folders:

`/data/<agent_name>/activity/` — daily activity logs
`/data/<agent_name>/memory/` — persistent knowledge

> Persistent Memory is updated once every 24 hours.

Memory is optional and can be enabled in `agent.md`:

```yaml
memory: true
```

# agent.md

agent discribed in a md file `/data/<agent_name>/agent.md`
> all text after frontmatter is a system prompt for this agent
## example:
```yaml
---
# unique id of agent. id is a agent name and identity. other agents see this name
id: claude
description: a helpful assistant 
model: sonnet-4.6
#if true activity has been logged and once per day consolidated as memory database
memory: true
# white list of build in tool bundles and connected mcp servers
tool_servers:
    - filesystem
    - todo
    - tasks
    - web
    - agent
---

You are helpful assistant
```
---

### ENV
Accepts unneccecary vars:
| Name                | Allow                | Default |
|:--------------------|:---------------------|:-------:|
| `LOG_INDEND`        | true/false           | false   |
| `LOG_SOURCE`        | true/false           | false   |
| `LOG_JSON`          | true/false           | false   |
| `LOG_LEVEL`         | debug/info/warn/error| error   |
| `CLEAN_UP_INTERVAL` | int (hours)          | 12      |
| `SESSION_RETENTION` | int (hours)          | 240     |
| `MAX_LOG_LINES`     | int                  | 1000    |

 
### For compose
Workdir is `/agent`
All memory stores in `/agent/data/...`