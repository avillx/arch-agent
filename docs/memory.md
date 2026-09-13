# Memory

Memory gives an agent a durable record of what it has done and learned. It
consists of two mechanisms that work together: an **activity journal** and
**persistent memory**. Both are configured in `memory.toml`
([configuration.md](configuration.md)), and each is only active for agents that
have memory enabled (`memory: true` in the agent file, see [agents.md](agents.md)).

## Activity journal

While the agent works, conversation transcripts are buffered. Periodically the
system asks a model to turn the buffered segment into a short human-readable
summary — what happened and what was decided — and appends it to the activity
journal.

- Journal entries are stored per day, per agent: `<agent>/activity/YYYY/MM/DD.md`,
  with a time header on each entry.
- The buffer interval and the model used for summarization are configurable.
- The summarization model does not need tool support — a plain chat model works.
- The journal is what the memory consolidation reads from.

### Activity settings

```toml
[activity]
enabled = true
model = 'open_ai/gpt-4o-mini'
interval = 600
```

| Setting | Meaning |
|---------|---------|
| `enabled` | Switch the journal on and off |
| `model` | Model used to summarize journal segments |
| `interval` | How often buffered messages are flushed to the journal, in **seconds** (the template suggests 120–600) |

If the section is absent, activity logging is disabled.

## Persistent memory

Once a day (and on demand through the API) the **consolidation** runs: for every
agent with memory enabled, it launches its own agent loop that reads the
activity journal and writes durable knowledge into the agent's memory files
(`<agent>/memory/`). The consolidation agent has its own toolset — it can read
the journal and write memory files, and it follows the same safety rules
(its files must carry valid memory frontmatter).

A memory file is a markdown file with a `hook` field in its frontmatter:

```markdown
---
hook: Facts about project X the agent should remember
---

Project X uses architecture Y. ...
```

The `hook` value is a short description of what the file contains. Each session's
system message then includes the memory index — the list of memory files with
their hooks — so the agent knows what it knows and where; it reads the actual
files with the filesystem tools when needed. If a memory file loses valid
frontmatter, it is not indexed.

### Consolidation settings

```toml
[consolidation]
enabled = true
model = 'open_ai/gpt-4o'
instruction = """
Save all file paths and note user preferences in a separate file.
"""
```

| Setting | Meaning |
|---------|---------|
| `enabled` | Switch consolidation on and off |
| `model` | Model used for consolidation; it must support tool calls |
| `instruction` | Extra guidance for the consolidation pass (optional) |

Consolidation runs every 24 hours; it can also be triggered on demand through
the API. If the section is absent, consolidation is disabled.

## Enabling memory for an agent

Both mechanisms apply only to agents with `memory: true` in their agent file.
Set it, then make sure models are configured in `memory.toml` and that those
models exist in `models.toml` ([models.md](models.md)).

## Managing memory via the API

The HTTP API lets you read and update both settings sections, list and read
memory files, trigger consolidation immediately and browse the activity journal
by date range. See [api.md](api.md).