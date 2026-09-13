# API

The agent is controlled through a REST HTTP API. The server listens on `PORT`
([environment-variables.md](environment-variables.md)); all endpoints below are
relative to `http://<host>:<port>`.

- JSON bodies and responses, unless stated otherwise.
- Errors: validation problems → `400` with `{"problems": {...}}`; not found →
  `404`; internal failures → `500` (the cause is logged, not returned to the
  client).
- A machine-readable OpenAPI specification is included in the project
  repository — it is the source of truth for exact schemas.

## Agents

Manage agent definitions — see [agents.md](agents.md).

| Method & path | Description |
|---------------|-------------|
| `GET /agent` | List all agents |
| `GET /agent/{id}` | Read one agent |
| `POST /agent/{id}` | Create an agent (`model` is required) |
| `PUT /agent/{id}` | Update an existing agent |
| `DELETE /agent/{id}` | Delete an agent |

Agent body: `model`, `description`, `system_prompt`, `tool_servers` (array of
server names) and `memory` (bool).

## Sessions

| Method & path | Description |
|---------------|-------------|
| `POST /session/{agent}` | Create a session; optional body `{"instruction": "..."}` adds a standing instruction to the session context |
| `GET /session/{agent}` | List session headers (timestamps, token usage) |
| `GET /session/{agent}/{session}` | Read the full conversation |
| `DELETE /session/{agent}/{session}` | Delete a session |

See [sessions.md](sessions.md) and [chat.md](chat.md).

## Chat

| Method & path | Description |
|---------------|-------------|
| `POST /chat/{agent}/{session}` | Send a message; responds with an SSE event stream |
| `POST /chat/{agent}/{session}/interrupt` | Stop the running completion |

The chat endpoint and its events, including provided tools and the
`/toolresult` round-trip, are documented in [chat.md](chat.md).

## Tools

| Method & path | Description |
|---------------|-------------|
| `GET /tools` | List all tool servers with their tools and descriptions |

See [tools.md](tools.md).

## MCP servers

| Method & path | Description |
|---------------|-------------|
| `GET /mcp` | List connected servers, their configs and tools |
| `POST /mcp/{mcp}` | Connect or replace a server; body is the server config |
| `DELETE /mcp/{mcp}` | Disconnect and remove a server |

See [mcp.md](mcp.md).

## Memory and activity

| Method & path | Description |
|---------------|-------------|
| `GET /memory/config` | Read consolidation settings |
| `POST /memory/config` | Update consolidation settings |
| `GET /memory/{agent}` | List the memory index (file → hook) |
| `GET /memory/{agent}/{memory_name}` | Read a memory file |
| `POST /memory/{agent}/consolidate` | Run consolidation now; responds with an SSE stream of the consolidation work; fails with `400` if memory is disabled for the agent |
| `GET /activity` | Read the activity journal; body `{"agent": "...", "from": "...", "to": "..."}` (ISO timestamps) |
| `GET /activity/config` | Read activity settings |
| `POST /activity/config` | Update activity settings |

See [memory.md](memory.md).

## Models and providers

| Method & path | Description |
|---------------|-------------|
| `GET /providers` | List providers and their models |
| `GET /providers/{name}` | Read one provider |
| `POST /providers` | Add a provider |
| `PATCH /providers/{name}` | Update a provider |
| `DELETE /providers/{name}` | Delete a provider |
| `POST /providers/{name}/models/{model}` | Add or replace a model |
| `DELETE /providers/{name}/models/{model}` | Delete a model |

The model name is **base64url-encoded** in the path (it can contain slashes).
See [models.md](models.md).

## Tasks

| Method & path | Description |
|---------------|-------------|
| `GET /task` | List tasks |
| `POST /task/{name}` | Create a task; body is the task config |
| `PATCH /task/{name}` | Partial update of a task |
| `DELETE /task/{name}` | Delete a task |

See [tasks.md](tasks.md).

## Streaming responses

Two endpoints stream (`/chat/...` and `/memory/.../consolidate`). They use the
SSE format: JSON events as `data:` lines, an optional `error:` frame for
failures, and a closing `data: [DONE]`. See [chat.md](chat.md) for the event
shapes.