# Chat

Chat is how you talk to an agent: you send a message to one of its sessions and
receive a stream of events — the agent's replies, its tool calls and tool
results — as they happen. Everything else (tools, memory, compaction) is
handled by the system; the stream just reports it.

## Sending a message

```
POST /chat/{agent}/{session}
```

The session must already exist — create it first (see [sessions.md](sessions.md)).

```json
{
  "user_request": [
    { "text": "Review the code in ./src and suggest fixes." }
  ],
  "logging": true
}
```

| Field | Meaning |
|-------|---------|
| `user_request` | Message content, an array of parts: `{"text": "..."}` and, for vision-capable models, `{"image_url": "data:image/png;base64,..."}` |
| `logging` | When `true`, the conversation is written to the agent's activity journal (see [memory.md](memory.md)) |
| `tool_servers` | Declares extra tools for this request only — see "Provided tools" below |

## The response stream

The response is an **SSE stream** (`text/event-stream`). Each event is JSON
sent as a `data:` line; the stream ends with `data: [DONE]`. Errors may arrive
as `error:` frames.

### Events

**`complete`** — a completion from the agent, emitted once per loop turn:

```json
{
  "type": "complete",
  "done": false,
  "completion": "I'll check the files first.",
  "tool_calls": [ { "id": "call_1", "tool": "find", "args": { "pattern": "*.go" } } ]
}
```

- `done: false` — the agent is not finished yet; it issued `tool_calls` and will
  continue after the results arrive. Keep listening.
- `done: true` — final answer; the conversation turn is over.

**`tool_result`** — result of a tool the agent called:

```json
{ "type": "tool_result", "id": "call_1", "result": [ { "text": "..." } ] }
```

**`tool_error`** — a tool call failed:

```json
{ "type": "tool_error", "tool_name": "read", "args": { "path": "/x" }, "cause": "..." }
```

**`compaction`** — the conversation was summarized to fit the model's context.
The agent keeps working; history older than the summary has been replaced.

**`complete_mistake`** — the model produced an unusable completion; the turn
continues.

**`loop_exit`** — the run ended. If the agent loop stopped abnormally, `cause`
carries the reason.

**`provided_toolcall`** — the agent called one of your provided tools — see next.

## Provided tools

You can give the agent custom tools **for a single request** by passing
`tool_servers` with the chat call. Each server has a label (`instruction`) shown
to the agent and a list of tools — `name`, `description`, and an optional JSON
`schema` describing the arguments:

```json
{
  "user_request": [ { "text": "Check the deploy status." } ],
  "tool_servers": [
    {
      "instruction": "Deployment control. Call check when asked about deploys.",
      "tools": [
        { "name": "check_deploy", "description": "Get current deployment status", "schema": {} }
      ]
    }
  ]
}
```

When the agent calls `check_deploy`, you receive a `provided_toolcall` event
instead of a normal `tool_result`:

```json
{
  "type": "provided_toolcall",
  "tool": "check_deploy",
  "args": {},
  "result_id": "abc123",
  "agent_id": "ops",
  "session_id": "sess_1"
}
```

You then perform the call on your side and post the result back:

```
POST /toolresult/{result_id}
```

```json
{ "result": [ { "text": "Deploy 42: healthy" } ] }
```

- Provide `error_message` instead of `result` to report a failure — the agent
  sees it and can recover.
- You have **30 seconds** after the `provided_toolcall` event to post the
  result; otherwise the call times out and the agent is told.
- Provided tools exist only for that chat request and are unregistered
  afterwards.

## Interrupting and concurrency

- Only **one completion may run per session at a time**. Sending a new chat
  request to a session that is still processing **cancels** the previous run and
  starts the new one.
- To stop the agent without sending a new message:
  `POST /chat/{agent}/{session}/interrupt`.
- While a request is running, its SSE stream stays open; the events you have
  already received are final — a cancelled run simply stops producing new ones.

## Errors

Missing agent, missing session, or an empty message is rejected before the run
starts. Failures inside the run are reported as stream events (`tool_error`,
`loop_exit`) rather than HTTP errors.