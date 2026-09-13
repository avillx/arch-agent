# Sessions

A **session** is a single conversation between a user and an agent. It is bound
to one agent, keeps the whole message history, and is the unit you address when
sending messages (see [chat.md](chat.md)). Sessions are also created
automatically: every scheduled task run and every subagent call opens a fresh
session.

## Lifecycle

- **Create** — an empty session is created with a generated id, bound to the
  agent you specify. You can pass an optional extra instruction that is added to
  that session's context.
- **Chat** — messages are sent to an existing session (`agent/session` id); the
  conversation continues accumulating in it.
- **List / get** — the API returns session headers (creation and last-update
  time, token usage) or the full conversation.
- **Delete** — a session and its history are removed for good.

## Where sessions are stored

Session history lives in the data directory under
`<agent>/sessions/<session_id>.jsonl` ([configuration.md](configuration.md)) —
one line of metadata, then one line per message.

The agent **cannot access session files**: file-tool access to the sessions
folder is denied outright, and session files are hidden from directory listings
and searches. An agent sees only the current session's context, assembled for
it — the rest of the history is off-limits by design.

## Retention and cleanup

Routine cleanup (see [operations.md](operations.md)) removes:

- **expired sessions** — sessions whose last update is older than
  `SESSION_RETENTION` hours (default 240, i.e. 10 days,
  [environment-variables.md](environment-variables.md));
- **broken sessions** — files whose metadata cannot be read, which can happen
  if a file was edited by hand or corrupted.

Deleted sessions are gone permanently — there is no trash or recovery.

## Managing sessions via the API

The HTTP API provides create, list, get, chat and delete operations. See
[api.md](api.md) and [chat.md](chat.md).