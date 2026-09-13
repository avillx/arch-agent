# Configuration

The agent is configured through files on disk — there is no separate admin
database or configuration service. Everything lives inside a single directory
called the **data directory**, set with the `DATA_PATH` environment variable
(see [environment-variables.md](environment-variables.md)).

## Data directory layout

The data directory contains:

- TOML configuration files listed below;
- one file per agent (`agent.md`, see [agents.md](agents.md));
- conversation history, memory files, activity logs and temporary files,
  managed by the system automatically.

You normally touch only the configuration files listed in this document. Never
edit the internal data files by hand.

## Configuration files

All configuration files use the [TOML](https://toml.io) format and are created
automatically on first start if they do not exist.

| File | Purpose | Documented in |
|------|---------|---------------|
| `models.toml` | LLM providers and the models available to agents | [models.md](models.md) |
| `secrets.toml` | API keys and other secret values | [secrets.md](secrets.md) |
| `mcp.toml` | Connections to external MCP servers | [mcp.md](mcp.md) |
| `memory.toml` | Activity logging and memory consolidation settings | [memory.md](memory.md) |
| `tasks.toml` | Autonomous tasks executed on a cron schedule | [tasks.md](tasks.md) |

Example of a minimal `models.toml`:

```toml
[open_ai]
api_type = 'openai'
key_ref = 'OPENAI_API_KEY'
base_url = 'https://api.openai.com/v1'

[open_ai.models.'openai/gpt-4o']
context_limit = 200000
modalities = ['text', 'image']
```

## Self-documenting files

Every configuration file starts with a comment block that explains the available
settings, with commented-out examples for each option. At first is a reference
for agent.

The header ends with a `# Do not touch this comment!` marker and a reminder to
keep the file consistent. When a configuration is changed through the HTTP API,
the system rewrites the file and restores this header automatically.

## Hot reload

Configuration files are watched while the system is running. When you save a
change, it is applied automatically within a moment — no restart needed. This
applies to all five files above. It also applies to new agents, skills and
skills content, which are picked up as they appear (see
[agents.md](agents.md) and [skills.md](skills.md)).

Rules to keep in mind:

- Bulk editing (for example, replacing a whole file from an editor) is
  supported, but consecutive quick saves are collapsed into one applied change.
- If the changed file contains an error (invalid TOML, unknown field, invalid
  combination of options), the change is **rejected**: the system keeps working
  with the last valid state and writes an error to the log. Nothing breaks, but
  your edit will not take effect until fixed. See
  [operations.md](operations.md) for how to diagnose rejected changes.
- While editing, make sure the whole file stays valid at every save — the
  reload happens on save, not on close.

## Changing configuration

You have two equivalent ways to change configuration:

1. **Edit the file by hand** — the change is hot-reloaded.
2. **Use the HTTP API** — the system updates the file for you, keeps the
   header intact and applies the change immediately. See [api.md](api.md) for
   the endpoints.
3. **Say agent to do this** — any agent with access to configs, can manage it.

All ways change the same files, so you can mix them freely.
