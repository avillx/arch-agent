# Subagents

An agent can delegate work to **another agent** with the `call_agent` tool
(a part of the `agent` tool server, see [tools.md](tools.md)). This is useful
when the task fits another agent's capabilities better — a different model, a
different toolset — or simply to keep the current context clean for long,
multi-step work.

## How a delegation works

- The calling agent invokes `call_agent` with the target agent's name and a
  request text.
- The system starts a **fresh session** for the target agent and sends the
  request there. The target agent is stateless: it sees only the request, not
  the caller's conversation.
- The target agent works with its own tools and returns its final answer, which
  the caller receives as the tool result.
- Delegated runs do not produce activity journal entries — they are private
  between the caller and the target.

## Depth limit

Delegation is limited to **3 levels deep**:

```
agent A → agent B → agent C   (ok, 3 levels)
agent A → agent B → agent C → agent D   (refused)
```

The limit exists to prevent runaway chains of agents calling agents. When a
delegation would exceed the limit, the call is refused and the caller gets a
message explaining that the delegation depth was reached — it can then finish
the work itself or break it into smaller requests.

For the same reason, the request should be **exhaustive**: include the task,
the relevant context and the expected result, because the target cannot ask
follow-up questions about your conversation. If the target asks for
clarification, resend the full request with the clarification added.

## When to delegate

- The task needs capabilities of another agent (its model, its tool servers).
- The task would take many tool calls and clutter the current context —
  delegating to a fresh instance keeps the parent conversation lean.
- The task is a natural sub-job with a clear input and output.

If the target agent does not exist, the call fails with an error the calling
agent sees and can handle.