# ARCH.agent — Documentation

ARCH.agent is an **agent SDK that runs as a server daemon**: a set of
autonomous AI agents with memory, tools and scheduled tasks, exposed through a
plain HTTP API. You deploy it, integrate it with your system, and the agents
work for you — including fully autonomously, on a cron schedule.

## What it does

- **Agents** — each agent is a configured persona: a model, a system prompt,
  a toolset, optionally memory. Agents can delegate work to each other.
- **Memory** — everything an agent does is journaled as human-readable
  activity logs and consolidated once a day into durable, indexed knowledge
  files that the agent can consult.
- **Tools** — built-in capabilities: filesystem read/write/edit, shell
  execution, web fetching, to-do lists, calling other agents. External tools
  come from MCP servers or from your own API client per request.
- **Scheduled tasks** — cron-driven autonomous requests: an agent works on a
  task without any user present and saves the result.
- **Zero external storage** — all state (configs, agents, sessions, memory,
  secrets) lives in a plain file database on disk; no database server needed.
- **OpenAI-compatible** — works with any OpenAI-compatible endpoint; providers
  and models are simple config entries.


## Reading order

- [getting-started.md](getting-started.md) — from zero to the first message
- [installation.md](installation.md) — running the server (Docker / binary)
- [configuration.md](configuration.md) — data directory, config files, hot reload
- [environment-variables.md](environment-variables.md) — environment reference
- [agents.md](agents.md) — what an agent is, how to configure one
- [models.md](models.md) — providers and models
- [secrets.md](secrets.md) — keys and how they are protected from the agent
- [skills.md](skills.md) — instruction files that guide an agent
- [tools.md](tools.md) — built-in tool servers and access rules
- [mcp.md](mcp.md) — connecting external MCP servers
- [memory.md](memory.md) — activity journal and persistent memory
- [tasks.md](tasks.md) — autonomous scheduled tasks
- [subagents.md](subagents.md) — agents delegating to other agents
- [sessions.md](sessions.md) — conversations and their lifecycle
- [chat.md](chat.md) — streaming chat protocol (SSE), events, provided tools
- [api.md](api.md) — full HTTP API reference
- [operations.md](operations.md) — logs, cleanup, maintenance


## Glossary

Short definitions of the terms used across the documentation.

| Term | Meaning |
|------|---------|
| **Activity** | The journal of what an agent has done — summarized conversation segments, stored per agent and per day. Read by memory consolidation. See [memory.md](memory.md). |
| **Agent** | A named, configured persona: a model, a system prompt, a toolset, optionally memory. The unit everything else works for. See [agents.md](agents.md). |
| **API** | The HTTP interface of the system — how external systems talk to agents, sessions, models, tools and tasks. See [api.md](api.md). |
| **Chat** | Sending a message to an agent's session and receiving the stream of events. See [chat.md](chat.md). |
| **Compaction** | Replacing an over-long conversation with a summary so the model's context limit is not exceeded. |
| **Consolidation** | The periodic pass that turns the activity journal into durable memory files. See [memory.md](memory.md). |
| **Context** | Everything the model sees per turn: system prompt, memory index, skill index, tool instructions, session history. |
| **Hook** | (1) The frontmatter description of a memory file, shown to the agent in the memory index; (2) internal safety checks applied to completions and tool calls. |
| **MCP server** | An external Model Context Protocol server whose tools are exposed to agents. See [mcp.md](mcp.md). |
| **Memory** | The durable knowledge of an agent, stored in files with hooks and indexed into its context. See [memory.md](memory.md). |
| **Model** | A concrete LLM endpoint available to agents, addressed as `provider/model`. See [models.md](models.md). |
| **Provided tool** | A custom tool declared by the API client for a single chat request; the client executes the call itself. See [chat.md](chat.md). |
| **Provider** | A configured LLM API (endpoint, key reference) under which models are registered. See [models.md](models.md). |
| **Secret** | A sensitive value stored encrypted-ish at rest in `secrets.toml` and masked from the agent as `{ env.KEY }`. See [secrets.md](secrets.md). |
| **Session** | A conversation between a user and an agent, with a full history. See [sessions.md](sessions.md). |
| **Skill** | A markdown instruction file (`SKILL.md`) in shared or private skills folders that guides an agent on a theme. See [skills.md](skills.md). |
| **SSE** | Server-Sent Events — the streaming format of chat and consolidation responses. See [chat.md](chat.md). |
| **Subagent** | Another agent invoked via `call_agent` to do a task in a fresh session, with a delegation depth limit. See [subagents.md](subagents.md). |
| **Task** | A scheduled autonomous request sent to one or more agents on a cron schedule. See [tasks.md](tasks.md). |
| **Tool** | A capability an agent can call: filesystem, shell, web, todo, agent, or a tool from an MCP/provided server. See [tools.md](tools.md). |
| **Tool server** | A named group of tools an agent references in its config. See [tools.md](tools.md). |
| **Tool call** | A request from the model to execute a tool; the result is returned to the model as a message. See [chat.md](chat.md). |
