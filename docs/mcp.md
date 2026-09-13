# MCP Servers

MCP (Model Context Protocol) lets the agent use tools provided by **external
servers** — file storage, search engines, internal services, anything that
speaks MCP. An external server is described in `mcp.toml`
([configuration.md](configuration.md)); once connected, its tools are registered
in the tool registry under the server's id, and an agent can be attached to it
by name just like a built-in server (see [tools.md](tools.md) and
[agents.md](agents.md)).

## Transport types

An MCP server connects through exactly one of two transports.

### HTTP

The server is reached over HTTP(S):

```toml
[my-mcp]
url = 'https://example.com/mcp'
token = '<bearer token>'
```

| Field | Meaning |
|-------|---------|
| `url` | Endpoint of the MCP server |
| `token` | Optional bearer token sent with every request |

### stdio (process)

The system starts a local process and talks to it over its standard input and
output:

```toml
[local-mcp]
command = 'uvx'
args = ['my-mcp', '--arg', 'value']

[local-mcp.env]
MY_VARIABLE = '1'
TOKEN = '{ env.MY_MCP_TOKEN }'
```

| Field | Meaning |
|-------|---------|
| `command` | Executable to run; must be available in `PATH` |
| `args` | Arguments passed to the command |
| `env` | Extra environment variables for the process |

Environment variables may reference values from `secrets.toml`
([secrets.md](secrets.md)) with the `{ env.KEY }` placeholder, so credentials
stay out of `mcp.toml`.

## Validation rules

The configuration is validated when it is loaded and when it is changed through
the API:

- exactly **one** transport per server — `url` and `command` cannot both be set;
- `args` and `env` belong to the stdio transport only — they are rejected with
  an HTTP server;
- `token` belongs to the HTTP transport only;
- the stdio `command` must exist — a server whose command is missing fails to
  connect and the problem is logged;
- the whole file must stay valid TOML, otherwise the change is rejected
  (see [configuration.md](configuration.md)).

## Lifecycle

- Connected servers are started when the system starts and when `mcp.toml`
  changes (hot reload).
- After connecting, the system lists the server's tools and registers them under
  the server id. If the server provides instructions, they are passed to the
  agent together with the tools.
- The connection is health-checked periodically; if it breaks, the tools are
  unregistered and the event is written to the log.
- Removing a server from `mcp.toml` (or deleting it via the API) shuts it down
  and unregisters its tools.

A server that fails to connect does not stop the rest of the system — it is
reported in the log and simply absent from the tool list until its
configuration is fixed.

## Managing MCP servers via the API

The HTTP API allows listing connected servers with their tools, connecting a
new server (validated and started immediately) and disconnecting one. See
[api.md](api.md).