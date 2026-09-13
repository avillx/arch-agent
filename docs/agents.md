# Agents

An **agent** is a configured instance of an assistant. It has an identity, a
system prompt, a model, and a set of tool servers it is allowed to use. You talk
to an agent through a [session](sessions.md); when a message arrives, the agent
runs its model in a loop, calling tools as needed until it produces an answer.

## What defines an agent

| Setting | Meaning |
|---------|---------|
| `id` | Unique name of the agent, used in the API and in session addresses |
| `description` | Short human-readable description (optional) |
| `model` | Model the agent uses, e.g. `openai/gpt-4o` (see below) |
| `tool_servers` | List of tool servers attached to the agent (see [tools.md](tools.md)) |
| `memory` | `true`/`false` — whether the agent can use long-term memory (see [memory.md](memory.md)) |
| system prompt | Free-form instructions that lead the agent's behaviour |

## Where agents live

Every agent is a directory `<id>/agent.md` inside the data directory
([configuration.md](configuration.md)). You can create agents by writing these
files by hand or through the HTTP API — both are equivalent, and new files are
picked up automatically without a restart.

The file starts with a YAML settings block between two `---` lines, followed by
the system prompt:

```markdown
---
id: support-bot
description: Handles support requests
model: openai/gpt-4o
tool_servers:
  - filesystem
  - web
memory: true
---

You are a support assistant. Be brief and precise. When you need up-to-date
information, use the web tool.
```

The system prompt is everything after the closing `---`. It is added to the
beginning of every session context as part of the system message.

## Choosing a model

The `model` setting must reference a model declared in `models.toml`
([models.md](models.md)). Use the full form `provider/model` — the same id the
model is declared under, for example `openai/gpt-4o`.

An agent cannot be saved with a model that is not declared, and the model cannot
be used in a conversation if it fails to load on start (the error is written to
the log, see [operations.md](operations.md)).

## Validation on save

Saving an agent is validated: both the model and every listed tool server must
exist at that moment. If you reference a tool server that is not connected
(see [tools.md](tools.md)) or a model that is not declared, the save is rejected.

## Reserved names

The following names cannot be used as an agent id — they are taken by the
system's own files and folders: `models.toml`, `secrets.toml`, `mcp.toml`,
`memory.toml`, `tasks.toml`, `tmp`, `skills`, `shared`.

## The default agent

On a fresh install an agent named `default` is created automatically. It is a
minimal placeholder: no model, no system prompt, and only the `filesystem` tool
server attached. To use it, give it a model and whatever prompts and tools you
need — either by editing its file or through the API.

## Managing agents via the API

Agents can be fully managed through the HTTP API: list, get, create, update and
delete. See [api.md](api.md) for the endpoints. Deleting an agent removes its
directory (and with it its skills and other per-agent files).