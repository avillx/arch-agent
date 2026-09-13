# Secrets

Secrets are named values — API keys, tokens, passwords — that the agent needs to
do its job but must not be able to read, remember or leak.

## Where secrets live

Secrets are declared in `secrets.toml` ([configuration.md](configuration.md)),
one key per line:

```toml
OPENAI_API_KEY = 'sk-...'
MY_MCP_TOKEN = 'abc123'
```

Conventions:

- Key names should use upper snake case (`SOME_VARIABLE`).
- Keys must be unique.
- Files are hot-reloaded: edit and save, the change applies immediately.

## How secrets are used

- **Models** — a provider references its API key by name with `key_ref`
  ([models.md](models.md)).
- **MCP servers** — an HTTP server takes a `token`; a stdio server receives
  secrets as environment variables via `{ env.KEY }` references
  ([mcp.md](mcp.md)).
- **Shell** — every command run through the `shell` tool gets all secrets as
  environment variables. A command can use them as `$OPENAI_API_KEY` etc.
  The shell also exports `AGENT_ID` and `AGENT_SESSION_ID` with the current
  agent and session ids.

## The agent never sees real values

Secret values are masked everywhere the agent can read text — this happens at
the context level, not just in the secrets file:

- if a secret value appears in the context assembled from files (for example, a
  tool returns a config file that contains a key), its occurrences are replaced
  with a placeholder;
- if the model types a secret into a completion, the typed value is replaced
  before the message is stored;
- if the model puts a secret into a tool argument, it is replaced before the
  call is executed.

The placeholder format is `{ env.KEY }`. The agent sees `{ env.OPENAI_API_KEY }`
instead of the key — enough to understand a value is referenced, but not the
value itself.

## Practical limitations

Masking is a text-level replacement of exact values. A few consequences to keep
in mind:

- A secret must never be equal to an empty string — empty values are ignored.
- If a secret is a substring of another secret, the longer one wins, so shorter
  values cannot hide inside longer ones in a way that breaks masking. Still,
  prefer long, unique values.
- Replacement happens on the raw text: if the agent is told a secret value
  through some other channel (a prompt, an MCP result) in an altered form —
  split, encoded, Base64 — masking cannot catch it.
- The model is instructed that secrets belong to the operator. Treat the model's
  memory as untrusted: use a short `SESSION_RETENTION`
  ([environment-variables.md](environment-variables.md)) and avoid repeating
  secret material in prompts and skills.

## Managing secrets via the API

Secrets can be read (placeholders are never returned — only key names and
whether a value exists), added, updated and removed through the HTTP API. See
[api.md](api.md).