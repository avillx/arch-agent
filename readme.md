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
id: calude
description: a helpful assistant 
model: sonnet-4.6
#if true activity has been logged and once per day consolidated as memory database
memory: true
# white list of allowed skills, other skill is invisible for agent
skills:
    - reglament
# white list of build in tool bundles and connected mcp servers
tool_servers:
    - filesystem
    - todo
    - tasks
    - web
    - agent
# white list of current tools
tools:
    - read_file 
    - write_file
---

You are helpful assistant
```
---

### Config
Must be runned with `--config` flag and path to config file e.g. `config.toml`
Example in `example.config.toml`

### ENV
Accepts unneccecary vars:
LOG_PRETTY (true/false)
LOG_LEVEL (debug/info/warn/error)

### For compose
Workdir is `/agent`
All memory stores in `/agent/data/...`

---

### TODO
**v1**
- [ ] search_files issue (glob/grep divide?)
- [ ] runtime fallback models pool
- [ ] runtime toolcall loop detection
- [ ] runtime eliminate oobserver to chat service
- [ ] runtime toolcall service as separated responsobility
- [ ] runtime reorganize for simplicity and robustness
- [ ] runtime observability issues
- [ ] better responses and api design
- [ ] agent can't handle on completion harness detection (for avoid loop)
- [ ] advanced error handling and messages
- [ ] eliminate edge case vulnurabilities for every package
- [ ] schemes on tasks, models, mcp_servers
- [ ] add doc's
- [ ] add tests
- [ ] add validations
- [ ] e2e tests 
- [ ] from zero launch (preloaded agent, cli, skills)
- [ ] chache safe summarize (in session request)
- [ ] runtime session chaching

**v1.1**
- [ ] scheduled tasks with skill prefill
- [ ] skill allow tools for agent
- [ ] sub agent call with skill prefill
