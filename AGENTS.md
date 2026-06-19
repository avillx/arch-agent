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

### Explicit Construction
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

Hierarchy:
- **Domain** (tier 1 and 2) — independent packages: `agent`, `session`, `prompt`.
- **Service** — `AgentRuntime`, `TaskService`, `SessionService`, `ChatService`, `ModelSerivce`, `ToolService`.
  This is the business logic layer; it works with domains and ports.
- **Infra** — `files`, `cron implementation`, `uuid`, `searxng`, `api`, `config`, `logging`, `mcp`, `openai`, `tools` (implementations, not services), `telegram`.

> [data](./data/) — contains the application's data (a database). Never read it unless you're directly working with data.
> Never read `.env` or `.secrets`.
> No need to read `go.mod` and `go.sum` without a reason.

## [a2a](./internal/a2a)
This is not an implementation of the A2A protocol — it's a service that lets an agent safely call another agent.
Used only by the `call_agent` tool (for agent-to-agent calls).

## [core domain](./internal/agent)
The core domain of the entire app.
It can't import any external packages or packages from other domains — only pure Go, the standard library, and `types` are allowed.

Widely used protection pattern: a public interface and constructor, with a private implementation.

- [activity](./internal/agent/activity.go) — the `ActivityRepo` interface and record type, with formatting rules.
- [agent](./internal/agent/agent.go) — the core domain; contains `agent.ID` and the `Agent` interface with a protected implementation.
- [message](./internal/agent/message.go) — the `Message` interface with a per-role implementation, including transcription rules.
- [model](./internal/agent/model.go) — the `Model` interface, which represents the LLM itself. Its implementation lives in `openai`.
- [skill](./internal/agent/skill.go) — `agent.SkillID`; an agent only has access to its allowed skills.
- [tool](./internal/agent/tool.go) — the tool interface for tools used by the agent.
- [toolcall](./internal/agent/toolcall.go) — part of an agent message; when the agent calls a tool, it should be represented as a `ToolCall`.

## [session domain](./internal/session)
This is the "tier 2" domain.
Includes the `Session` interface and its implementation.
Includes a service for operating on sessions.

## [chat service](./internal/chat)
Orchestrator for the common agent call flow. Launches a completion by `sessionID`, `agentID`, and request.

## [files database](./internal/files)
This is the database for the entire application.
The app uses a file-based database for simplicity.

[locktable](./internal/files/locktable.go) — for concurrency-safe filesystem access.
[Filesystem](./internal/files/filesystem.go) — for path-safe access.

Also includes the following implementations:
- [activity repository](./internal/files/activity.go)
- [agents repository](./internal/files/agents.go)
- [memory repository](./internal/files/memory.go)
- [model repository](./internal/files/model.go)
- [secrets repository](./internal/files/secrets.go)
- [session repository](./internal/files/sessions.go)
- [skill repository](./internal/files/skill.go)
- [task repository](./internal/files/tasks.go)

### Ruled filesystem
[ruled filesystem](./internal/files/rule)
This is a strictly controlled access layer for agent file access — a rule-based wrapper around `Filesystem` with extensive validation.

## [model service](./internal/model)
Service that manages models and their settings. Includes a model settings repository.

## [open ai](./internal/openai)
Implementation of `agent.Model` using the OpenAI API.
Contains many converters: `internal <-> OpenAI`.
Uses the official OpenAI SDK library.
Should work with any OpenAI-compatible endpoint.

## [prompts](./internal/prompt)
The prompt package holds prompts embedded from `.md` templates using Go's templating.
Never read the raw prompt templates in [templates](./internal/prompt/templates).
If you just need to confirm a prompt exists, read [embeds](./internal/prompt/prompt.go) instead.

## [agent runtime](./internal/runtime)
Not the app's runtime — this is the core of the agent's runtime.
Processes a requested session with the agent's settings using the ReAct pattern.

[context assembler](./internal/runtime/context_assembler.go)
Assembles the context for an agent call. Combines the summary, system prompt, and an explanation.
If the agent has memory access, it loads the memory index and the tail of the logs.
If the agent has skills, it loads descriptions of the available skills for skill indexing.
This happens as a pre-context hook — a simple dialogue: user → explanation, agent → agreement.

[compactor](./internal/runtime/compaction.go)
Responsible for session compaction once it reaches a threshold, to avoid context overflow.

[observer](./internal/runtime/observer.go)
Responsible for logging activity. If the agent has memory or works autonomously on tasks, it makes an additional model call to process and log the activity, producing a human-readable explanation of the agent's actions.

[memory](./internal/runtime/memory)
Memory consolidation service, invoked once every 24 hours.
It calls the agent runtime with its own ruled tools and model.
Consolidates the day's logs with the already existing memory.

## [tasks](./internal/task)
Tasks for autonomous calls that happen without a direct user request.
Uses a cron interface for scheduling calls.
Includes the `Task` domain and `TaskService` for managing tasks.

## [telegram](./internal/telegram)
Implementation of the Telegram bot.
This package will be separated out later — that's why its tools and other related code should stay inside this package rather than spreading into others, except for the root DI.

## [tools](./internal/tools)
Tools for the agent. Mainly contains the tool registry implementation, `ToolService`.
Tools are stored by owner — the owner identifies who provides the tool; built-in tools have an empty owner (`""`).
Other tools, like MCP tools, must use their MCP server as the owner.
Contains multiple built-in tool implementations:

- [fetch](./internal/tools/fetch) — tool for fetching web pages.
- [fs](./internal/tools/fs) — tools for giving the agent filesystem access:
  `delete`, `edit_file`, `list_dir`, `move`, `read_file`, `search`, `write_file`.
- [search](./internal/tools/search/search.go) — tool for web search; has a `WebSearch` interface.
- [task](./internal/tools/task) — tools that let the agent manage its own tasks.
- [todo](./internal/tools/todo) — tools for creating and manipulating a `todo` list; a lightweight harness for decomposing tasks.
- [call_agent](./internal/tools/call_agent.go) — tool for calling other agents as subagents; uses `A2AService`.

> First and foremost, every tool is a thin interface providing safe agent usage on top of an `agent.Tool` implementation.
> Any logic not tied to the agent-facing representation should live in the implementation, not in the tool itself.

## [DI](./internal/app.go)
Contains the `App` struct with all services, and `BuildApp` — the root composition point of the application.
Should be very simple, with no logic — composition only.

## Other packages
These packages are not as important.

- [api](./internal/api) — currently empty; intended to become RESTful API endpoints later.
- [config](./internal/config) — thin TOML config parsing and implementation (Telegram bot and search settings).
- [cron](./internal/cron) — implements the `task` package's cron interface using an external library.
- [logging](./internal/logging) — wrappers for logging different things, with 2 logger modes (pretty and plain).
- [mcp](./internal/mcp) — not yet implemented; intended to be a bridge from internal tools to MCP.
- [searxng](./internal/searxng) — implementation of the web search interface using the SearXNG API.
- [types](./internal/types) — contains a single widely-used type; will be eliminated eventually.
- [uuid](./internal/uuid) — simple UUID generator using the `google/uuid` package.