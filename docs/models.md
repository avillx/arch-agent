# Models

An agent talks to language models through **providers**. A provider is a
declaration of an OpenAI-compatible API endpoint; a provider declares a list of
**models** that agents may use. Anything that speaks the OpenAI protocol works:
OpenAI, local gateways (for example vLLM or Ollama-compatible servers), or any
other compatible endpoint.

Models are declared in `models.toml` ([configuration.md](configuration.md)).

## Providers

A provider block looks like this:

```toml
[open_ai]
api_type = 'openai'              # only 'openai' is supported
key_ref = 'OPENAI_API_KEY'       # key name from secrets.toml
base_url = 'https://api.openai.com/v1'
```

| Setting | Meaning |
|---------|---------|
| provider id | Table name — unique name, used as part of the model id |
| `api_type` | Type of API. Only `openai` (OpenAI-compatible) is supported |
| `key_ref` | Name of the secret holding the API key (see [secrets.md](secrets.md)) |
| `base_url` | Base URL of the API, e.g. `https://api.openai.com/v1` |

Providers without a key still load — the key is resolved when the model is
actually used, so a provider can be configured before any secret exists.

## Models

Only models explicitly listed under a provider are available to agents.
Declaring a model gives it an id of `provider/model`, for example
`open_ai/openai/gpt-4o` is **not** the format — the model id is
`<provider-id>/<model-key>`, e.g.:

```toml
[open_ai.models.'gpt-4o']
```

The model can then be referenced as `open_ai/gpt-4o` in an agent's settings
([agents.md](agents.md)).

### Model settings

Every model entry may carry the following settings:

| Setting | Meaning |
|---------|---------|
| `context_limit` | Context window in tokens. When a conversation reaches ~90% of this limit it is compacted — the older part is summarized by the model and the summary replaces it, so long conversations continue without losing the system prompt and the most recent messages. When not set, a default of 100,000 tokens is assumed |
| `modalities` | What the agent may receive from this model: `text`, `image`, or both. If a model is not meant to receive images, leave it out |
| `max_turns` | Maximum number of model/tool round trips in one request (default 100). A loop stops with an error when exceeded |
| `max_output_tokens`, `max_completion_tokens` | Output token limits |
| `temperature`, `top_p`, `frequency_penalty`, `presence_penalty` | Standard sampling parameters |
| `tool_choice`, `reasoning_effort` | Model-behaviour parameters |
| `recall_budget` | Model-specific budget parameter |
| `extras` | Any additional parameters, passed through to the API as-is |

Example:

```toml
[open_ai.models.'gpt-4o']
context_limit = 200000
modalities = ['text', 'image']
max_turns = 50
temperature = 0.7
extras = { order = 'some' }
```

## Restart and reload

On start and on every change of `models.toml` (hot reload, see
[configuration.md](configuration.md)) the system loads declared models into the
runtime. A model that fails to load (bad endpoint, missing key, unsupported
settings) is skipped and the problem is written to the log — the rest of the
configuration keeps working.

## Managing providers and models via the API

Providers and their models can be managed through the HTTP API: list providers,
add/update/delete a provider, add/delete a model within a provider. See
[api.md](api.md) for the endpoints.
