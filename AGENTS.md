# How to Code

Write simple, robust, human-readable code.
Avoid overengineering, hacks, and tricks.
Maintain high code quality.

## Idiomatic
This project is written in Go — follow Go idioms, do not break them.

## Comments
Avoid writing comments whenever possible — code should be self-documented through meaningful, descriptive names.
> When comments are necessary, keep them short and idiomatic to Go.

Good:
```golang
// Run blocks until ctx is done.
func Run(ctx context.Context)
```

Bad:
```golang
// ---------
// Endpoints
// ---------
```

Comments are allowed only for these 3 purposes:
1. To separate logical blocks of code for better readability.
2. To explain non-obvious behavior that isn't clear from the code itself.
3. As a TODO for something not needed right now but that must be handled later (avoid creating TODOs that never get resolved).

## Errors
All errors must be handled.
Errors are part of business logic and must always be accounted for — never ignore them.

### Ignoring Errors
Bad:
```golang
result, _ := foo()
```
Good:
```golang
result, err := foo()
if err != nil {
    if errors.As(err, &ErrExpected) {
        return fmt.Errorf("minor mistake: %w", err)
    }
    return err
}
```

### Maps
Every map lookup must be checked for a missing key.
Bad:
```golang
result, _ := x["some string"]
```
Good:
```golang
result, ok := x["some string"]
if !ok {
    return fmt.Errorf("has no object")
}
```

### Casts
Every type assertion must be checked.
Bad:
```golang
casted := some.(int)
```
Good:
```golang
casted, ok := some.(int)
if !ok {
    return fmt.Errorf("bad type")
}
```

### Panic
Never use `panic()` or `recover()`. This pattern is reserved for the user only — do not touch it if it's already in use.

## Clean Code

### Edge Cases
Consider edge cases and failure scenarios.
Edge cases should be eliminated by architecture whenever possible — an error that's impossible to occur is the best kind of error.

Bad:
```golang
    if start < 1 || start > total {
        return ErrWrongNum
    } 
    if end < 1 || end > total {
        return ErrWrongNum
    } 
    return arr[start-1:end]
```
Good:
```golang
    start = max(1, min(start, total))
    end = max(start, min(end, total))
    return arr[start-1:end]
```

When that's not possible, close the gap with validation, e.g.:
```golang
func Foo(args any) error {
    typedArgs, ok := args.(string)
    if !ok {
        return fmt.Errorf("unexpected type %T: %w", args, ErrBadType)
    }
    ...
}
```

### Concurrency Safety
If you write something with a lifecycle that's accessed from multiple goroutines, always protect it with a mutex.

### Single Responsibility
Avoid types with multiple responsibilities.

Bad:
```golang
// God struct
type SomeManager struct
func (m *SomeManager) GetOrCreate()
func (m *SomeManager) MayBeStore()
```
Good:
```golang
type Observer interface
type Executor interface
type Repository interface

// thin orchestrator
type Service struct {
    observer   Observer
    executor   Executor
    repository Repository
}
func (s *Service) Get()
func (s *Service) Create()
```

### External Packages
Never add external packages unless the user explicitly mentions them.

### Consistency
Follow the existing code style and patterns already used in the project.

### Global State
Never use a service locator, global variables, or hidden dependencies.
Expose all requirements explicitly as arguments via dependency injection.

### Explicit Construction only situations

explicit construction is used only when developer make this decigion do not chage it
and never make explicit construction by yourself, only when developer say it directly
prefer to use golang constructor function.

Bad:
```golang
obj, err := NewObject()
if err != nil {
    return err
}
Mutate(obj)
```
Good:
```golang
obj, err := NewObject(ObjectParams{
    title:       "as example",
    description: "as example",
})
```

## Tests
The project is under active, dynamic development and has no stable test suite yet — do not write tests unless the user explicitly asks for them.

Tests should live in a separate test package, e.g.:
```files
some_package
- service.go
- service_test.go
```
Go natively supports a separate test package:
```golang
package somepackage_test
```
Always use the same package name as the one being tested, with a `_test` suffix, to keep the package namespace clean.

Actively use helper functions — tests should have their own architecture, where flat test-case functions are just thin checks.
> But keep it simple if possible.

---

# Structure
The project follows clean architecture.
Note: the physical file structure differs from the logical hierarchy below.

File structure:
- Entry point: [main file](./cmd/agent/main.go)
- All application logic lives in [internal packages](./internal)
- Keep the file structure as flat and simple as possible
- [api](./api) holds the OpenAPI specification of the HTTP API

Hierarchy:
- **Domain** — independent packages without app dependencies: `agent`, `prompt`.
  (`session` and `task` are "tier 2" domains: they own both the domain type and its service.)
- **Service** — the business logic layer; works with domains and ports:
  `chat` (with `runtime`), `agent.Service`, `session.Service`, `task.Service`, `model`,
  `tools.Service`, `memory`, `mcp`, `subagent`, `secrets`, `cleanup`.
- **Infra** — port implementations and plumbing:
  `files`, `openai`, `cron`, `logging`, `sentinel`, `hooks`, `api`, `types`, `uuid`.

> [data](./data/) — contains the application's data (a database). Never read it unless you're directly working with data.
> Never read `.env` or `.secrets`.
> No need to read `go.mod` and `go.sum` without a reason.

## [core domain](./internal/agent)
The core domain of the entire app.
It can't import any external packages or packages from other domains — only pure Go and the standard library.

Widely used protection pattern: a public interface and constructor, with a private implementation.

- [activity](./internal/agent/activity.go) — the `ActivityRepo` interface and `ActivityRecord` type, with formatting rules.
- [agent](./internal/agent/agent.go) — the core domain; contains `agent.ID` and the `Agent` interface with a protected implementation.
- [message](./internal/agent/message.go) — the `Message` interface with a per-role implementation, including transcription rules.
- [model](./internal/agent/model.go) — the `Model` interface (the LLM itself) plus `Completion`, `ModelSettings`, and the `ModelRegistry` port. Its implementation lives in `openai`.
- [service](./internal/agent/service.go) — `agent.Service`; thin validation over the agent storage (checks the model and tool servers exist before saving/deleting).
- [tool](./internal/agent/tool.go) — the tool ports used by the agent: `Tool`, `ToolServer`, `ToolRegistry`, and `ToolProperty`.
- [toolcall](./internal/agent/toolcall.go) — part of an agent message; when the agent calls a tool, it should be represented as a `ToolCall`.

## [session domain](./internal/session)
The "tier 2" domain for sessions.
Includes the `Session` interface and its implementation, `SessionHeader`/`ErrBrokenHeaders`, and `session.Service` (with the `SessionsRepo` port) for operating on sessions.

## [tasks](./internal/task)
Tasks for autonomous calls that happen without a direct user request.

- [task](./internal/task/task.go) — the `TaskConfig` domain and the `Cron` port.
- [service](./internal/task/service.go) — `TaskService`; schedules tasks via the cron port.
- [executor](./internal/task/executor.go) — runs a scheduled task by invoking the chat service in a fresh session for each recipient.

## [chat service](./internal/chat)
Orchestrator for the common agent call flow. Launches a completion by `sessionID`, `agentID`, and request.

- [service](./internal/chat/service.go) — `ChatService` (`ChatExecutor` port): resolves agent/model/tools/session, builds the context, runs the agent loop and applies its events to the session.
- [context assembler](./internal/chat/context_assembler.go) — builds the system message from parts (agent system prompt, memory index, skill index, tool instructions, session extras) and caches it per session for 30 minutes.
- [dispatcher](./internal/chat/dispatcher.go) — guarantees a single in-flight completion per session; a new request cancels the previous one, and exposes interruption.

## [agent runtime](./internal/runtime)
Not the app's runtime — the core of the agent's reasoning engine.
[loop](./internal/runtime/loop.go) runs the ReAct loop: completion → hooks → tool calls → compaction → repeat, until the model finishes or a limit is reached.

- [events](./internal/runtime/events.go) — events emitted by the loop (`CompleteEvent`, `ToolResultEvent`, `CompactionEvent`, `LoopExitEvent`, …).
- [hook](./internal/runtime/hook.go) — generic hook interfaces (`CompletionHook`, `ToolCallHook`, `ToolResultHook`) applied by the loop.
- [compaction](./internal/runtime/compaction.go) — compacts the conversation once it reaches a threshold, to avoid context overflow.
- [error](./internal/runtime/error.go) — runtime and tool-call error types.

## [memory](./internal/memory)
- [activity reporter](./internal/memory/activity_reporter.go) — `ActivityService`: buffers messages, then periodically makes a model call to produce a human-readable activity log.
- [consolidation](./internal/memory/consolidation.go) — `ConsolidationService`, invoked once every 24 hours (and on demand); runs its own agent loop with dedicated tools to consolidate the day's logs into memory files.

## [model service](./internal/model)
- [model](./internal/model/model.go) — `ModelService`, the runtime registry of live `agent.Model` instances keyed by model id.
- [provider](./internal/model/provider.go) — `ProviderService`, API providers and their models; loads models into `ModelService` on config changes.
- [types](./internal/model/types.go) — provider/model config types (`ProviderConfig`, `ModelConfig`, `APIType`, `ProviderConfigRepo`).

## [tools](./internal/tools)
Tools for the agent, plus the tool registry `ToolService`.
Tools are grouped into named [`ToolServer`s](./internal/agent/tool.go); agents reference servers by name.
Built-in servers use fixed names (`filesystem`, `shell`, `web`, `todo`, `agent`); MCP tools register under their MCP server id.

- [service](./internal/tools/service.go) — the tool registry (`Connect`/`Disconnect`/`ToolServers`).
- [helpers](./internal/tools/helpers.go) — shared helpers for built-in tools (`BuildInToolServer`, arg unwrapping, context accessors).
- [call_agent](./internal/tools/call_agent.go) — calls another agent as a subagent; uses [subagent](./internal/subagent).
- [fetch](./internal/tools/fetch) — `fetch` tool for fetching web pages.
- [fs](./internal/tools/fs) — filesystem access: `write`, `read`, `edit`, `move`, `find`.
- [shell](./internal/tools/shell) — `shell` tool for running shell commands.
- [todo](./internal/tools/todo) — `create_todo`, `update_todo`, `list_todo`; a lightweight harness for decomposing tasks.

> Every tool is a thin interface providing safe agent usage on top of an `agent.Tool` implementation.
> Any logic not tied to the agent-facing representation should live in the implementation, not in the tool itself.

## [mcp](./internal/mcp)
MCP (Model Context Protocol) integration: connects external MCP servers (HTTP or process) and registers their tools in the tool registry under the server id.

- [service](./internal/mcp/service.go) — `mcp.Service`; loads servers from config and keeps them connected.
- [gateway](./internal/mcp/gateway.go) — HTTP and process transport gateways.
- [tool](./internal/mcp/tool.go) and [convert](./internal/mcp/convert.go) — adapt MCP tools to `agent.Tool`.

## [secrets](./internal/secrets)
- [service](./internal/secrets/service.go) — in-memory secrets store backed by `secrets.toml`.
- [replacer](./internal/secrets/replacer.go) — replaces secret values with `{ env.KEY }` placeholders in agent-visible text.

## [hooks](./internal/hooks)
Agent safety hooks applied by the runtime loop.

- [file access](./internal/hooks/filesystem.go) — path-based read/write access rules for file tools (replaces the old ruled filesystem).
- [secrets](./internal/hooks/secrets.go) — keep secrets from leaking into completions, tool arguments, or tool results.
- [content size](./internal/hooks/contentsize.go) — truncates oversized tool results.
- [completion](./internal/hooks/completion.go) — completion-safety hooks: empty answer, undone todo, valid memory frontmatter.

## [cleanup](./internal/cleanup)
Periodic maintenance service.

- [service](./internal/cleanup/service.go) — `CleanUpService`; runs on an interval.
- [session](./internal/cleanup/session.go) — `SessionsCleaner`; removes broken and expired sessions, and trims the agent log file.

## [sentinel](./internal/sentinel)
Watches config files with `fsnotify` and triggers debounced reload actions (`.toml` hot reload).

## [files database](./internal/files)
The file-based database for the entire application (kept simple on purpose).

[locktable](./internal/files/locktable.go) — path-keyed locks for concurrency-safe filesystem access.
[filesystem](./internal/files/filesystem.go) — path-safe filesystem access.
[format](./internal/files/format.go) and [dto](./internal/files/dto.go) — shared formatting and DTO helpers.

Repositories and config files:
- [agents](./internal/files/agents.go) — agent definitions (`agent.md`).
- [activity](./internal/files/activity.go) — activity logs (the `activity` folder).
- [models](./internal/files/models.go) — `models.toml`; providers and models, hot-reloaded.
- [memory config](./internal/files/memory_config.go) — `memory.toml`; consolidation config, hot-reloaded.
- [memory index](./internal/files/memory_index.go) — per-agent memory file index (the `memory` folder).
- [mcp](./internal/files/mcp.go) — `mcp.toml`; MCP server connections, hot-reloaded.
- [secrets](./internal/files/secrets.go) — `secrets.toml` storage.
- [sessions](./internal/files/sessions.go) — conversation files (the `sessions` folder).
- [skill](./internal/files/skill.go) — discovers agent skills (`skills` folder, `SKILL.md`).
- [tasks](./internal/files/tasks.go) — `tasks.toml`; cron tasks, hot-reloaded.
- [temporary](./internal/files/temporary.go) — the `tmp` folder with automatic cleanup.

## [open ai](./internal/openai)
Implementation of `agent.Model` using the OpenAI API, built on the `openai-go` SDK. Works with any OpenAI-compatible endpoint.

- [factory](./internal/openai/factory.go) — `OpenAIModelFactory`; builds `agent.Model` instances for the model service.
- [openai](./internal/openai/openai.go) — converters `internal <-> OpenAI`.
- [reasoner](./internal/openai/reasoner.go) — `OpenAIReasoner`, the `agent.Model` implementation.
- [settings](./internal/openai/settings.go) — model settings parsing.

## [prompts](./internal/prompt)
The prompt package holds prompts embedded from `.md` templates using Go's templating.
Never read the raw prompt templates in [templates](./internal/prompt/templates).
If you just need to confirm a prompt exists, read [embeds](./internal/prompt/prompt.go) instead.

## [api](./internal/api)
The HTTP (REST) API server that all services are exposed through.
Routes for agents, sessions, chat, tools, mcp, memory, tasks, providers, and activity are registered in [server.go](./internal/api/server.go).
Its OpenAPI spec lives in [api](./api).

## [subagent](./internal/subagent)
Lets an agent safely call another agent as a subagent (used only by the `call_agent` tool).
Routes the call through the chat service and creates a fresh session for each subagent call.
Tracks the delegation depth in context to prevent infinite chains (max depth 3).

## [DI](./internal/wire.go)
Root composition point of the application.
[BuildServer](./internal/wire.go) wires all services together and returns the `*api.HTTPServer`.
Should be simple, with no logic — composition only.

## Other packages
These packages are not as important.

- [cron](./internal/cron) — implements the `task` package's `Cron` port using `robfig/cron`.
- [logging](./internal/logging) — `slog` handlers: JSON console output, plain-text output for the agent-visible log file, and a multi-handler for both.
- [types](./internal/types) — shared types: sentinel errors (`ErrIsNotExist`, `ErrAlreadyExist`), `AgentMistakeError`, and validation helpers.
- [uuid](./internal/uuid) — ID generator using the `rs/xid` package.
