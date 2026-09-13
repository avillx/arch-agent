# Getting Started

This page walks you from an empty install to the first message sent to an
agent. It assumes the service is running — see [installation.md](installation.md).

## 1. Make sure the service is up

```sh
curl http://localhost:8080/agent
```

A fresh install has exactly one agent, `default`, created automatically. It is a
minimal placeholder: no model assigned yet, no system prompt, and only the
filesystem tool server attached. List it to confirm:

```sh
curl http://localhost:8080/agent
# -> [ { "model": "", "memory": false, "description": "agent placeholder", "tool_servers": ["filesystem"], "system_prompt": "" } ]
```

## 2. Configure a model and a key

The agent can only use models that are explicitly declared. For an OpenAI-compatible
API you need:

1. an **API key** in `secrets.toml` ([secrets.md](secrets.md)):
   ```toml
   OPENAI_API_KEY = 'sk-...'
   ```
2. a **provider** pointing at the endpoint and the key
   ([models.md](models.md)):
   ```toml
   [open_ai]
   api_type = 'openai'
   base_url = 'https://api.openai.com/v1'
   key_ref = 'OPENAI_API_KEY'
   ```
3. at least one **model** under it:
   ```toml
   [open_ai.models.gpt-4o-mini]
   context_limit = 128000
   ```

Config files are hot-reloaded, so no restart is needed. You can do the same via
the API instead if you prefer (see [api.md](api.md)).

## 3. Assign the model to the agent

Update the `default` agent (or create your own — [agents.md](agents.md)):

```sh
curl -X PUT http://localhost:8080/agent/default \
  -H "Content-Type: application/json" \
  -d '{
    "model": "open_ai/gpt-4o-mini",
    "description": "A helpful assistant",
    "tool_servers": ["filesystem", "web", "todo"]
  }'
```

The save is validated: unknown models or tool servers are rejected.

> If an agent file has no system prompt, the **built-in default prompt** is
> used automatically — an agent with just a model works out of the box.

## 4. Create a session

A chat always happens inside a session ([sessions.md](sessions.md)):

```sh
curl -X POST http://localhost:8080/session/default
# -> { "id": "cvfanjg..." }
```

## 5. Send the first message

```sh
curl -N -X POST http://localhost:8080/chat/default/cvfanjg... \
  -H "Content-Type: application/json" \
  -d '{
    "user_request": [ { "text": "Say hello and list the files you can see." } ]
  }'
```

The response is a stream of events ([chat.md](chat.md)): the agent's answers,
its tool calls and results. A final `data: [DONE]` frame marks the end of the
turn. The conversation stays in the session — send the next message to the same
session id and the agent remembers the context.

## What next

- [agents.md](agents.md) — creating and configuring agents.
- [models.md](models.md) — providers, models, context limits.
- [chat.md](chat.md) — the full chat protocol, events and provided tools.
- [tools.md](tools.md) — what the built-in tools can do and access rules.