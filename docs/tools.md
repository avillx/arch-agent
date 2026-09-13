# Tools

An agent acts through **tools** — small, well-defined operations it can invoke
in the middle of a conversation (read a file, run a command, fetch a page).
Tools are grouped into named **tool servers**; an agent is attached to a tool
server as a whole, not to individual tools. The list of servers attached to an
agent is set in its `tool_servers` setting (see [agents.md](agents.md)).

Tools are declared to the model with their names, parameter schemas and
descriptions; the agent chooses when and how to call them.

## Built-in tool servers

The names below are available out of the box. External MCP servers register
their tools under their own server id (see [mcp.md](mcp.md)), and a client can
provide additional servers for a single request (see [chat.md](chat.md)).

### `filesystem` — file access

Work with files inside the data directory ([configuration.md](configuration.md)):

| Tool | What it does |
|------|--------------|
| `read` | Read a text or image file; read a directory listing (2 levels deep); read a line range of a text file |
| `write` | Write content to a file, creating parent directories; overwrite or append mode |
| `edit` | Replace a unique string in a file; the `old` value must match exactly once |
| `move` | Move or rename a file or a directory |
| `find` | Glob-based path search; with a regex, searches file contents and returns matching lines |

### `shell` — shell commands

Run shell commands inside the data directory:

| Tool | What it does |
|------|--------------|
| `shell` | Execute a command with arguments `command`, `cwd`, `timeout` |

Behaviour details:

- every call runs in a **fresh shell session** — no state persists between
  calls, combine operations with pipes or write files instead;
- default timeout is 30 seconds; longer operations need an explicit `timeout`
  (in seconds, up to 15 minutes);
- secrets are passed as environment variables (see [secrets.md](secrets.md)),
  plus `AGENT_ID` and `AGENT_SESSION_ID`;
- commands return combined stdout/stderr and the exit status;
- a timed-out command is terminated and reported as such.

### `web` — fetching pages

| Tool | What it does |
|------|--------------|
| `fetch` | Fetch a web page by URL; `format` of `markdown` (default, extracts readable text) or `rawHTML` |

Behaviour details:

- only `http`/`https` URLs are allowed;
- only text content types are accepted (`text/*`, `application/json`, XML and
  similar) — binary downloads are rejected;
- the response is truncated to a hard size limit, so a huge page never floods
  the context;
- the fetch has a short default timeout and follows at most 10 redirects.

### `todo` — task planning

A lightweight per-session harness. To-do list the agent uses to plan multi-step work:

| Tool | What it does |
|------|--------------|
| `create_todo` | Create one or more todos |
| `update_todo` | Change a todo's status by id |
| `list_todo` | List todos of the current session |

Statuses: `pending`, `in_progress`, `done`, `declined`. The list is scoped to
the current session and kept in memory — it does not survive restarts.

If agent try to stop without close all tasks it caution agent, about undone points.

### `agent` — delegation

| Tool | What it does |
|------|--------------|
| `call_agent` | Ask a question or delegate a task to another agent by name |

Delegation runs in a fresh session of the target agent; the caller receives the
answer as a tool result. See [subagents.md](subagents.md) for details and
limits.

## Access restrictions

Filesystem access is governed by explicit rules, applied to every file tool
call. Rules are evaluated in order, and the first match wins:

- the agent log file is read-only;
- `secrets.toml` is read-only (and its values are masked anyway);
- an agent's own `agent.md` cannot be accessed;
- session files (`<agent>/sessions/**`) are completely off-limits — the agent
  cannot read or write conversation history;
- the agent's activity folder is read-only;
- everything else is writable.
- paths containing **symlinks** are rejected, so an agent cannot escape the
  data directory through links.

Independent of access rules, session files are hidden from directory listings
and searches, and tool results larger than a fixed size are truncated before
they reach the model.

## Inspecting tools via the API

The HTTP API exposes the connected tool servers and the tools they contain
(see [api.md](api.md)) — useful for checking what an agent can do.