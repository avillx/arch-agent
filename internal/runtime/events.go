package runtime

import (
	"arch-agent/internal/agent"
	"time"
)

type Event interface{}

// base event
type baseEvent struct {
	Timestamp time.Time `json:"timestamp"`
}

func newBaseEvent() baseEvent {
	return baseEvent{
		Timestamp: time.Now(),
	}
}

// compaction event
type CompactionEvent struct {
	baseEvent
	compactedContext []agent.Message
	summary          string
}

func NewCompactionEvent(summary string, compactedContext []agent.Message) *CompactionEvent {
	return &CompactionEvent{
		baseEvent:        newBaseEvent(),
		summary:          summary,
		compactedContext: compactedContext,
	}
}

func (ev *CompactionEvent) Summary() string {
	return ev.summary
}

func (ev *CompactionEvent) CompactedContext() []agent.Message {
	return ev.compactedContext
}

// complete event
type CompleteEvent struct {
	baseEvent
	completion *agent.Completion
}

func NewCompleteEvent(completion *agent.Completion) *CompleteEvent {
	return &CompleteEvent{
		baseEvent:  newBaseEvent(),
		completion: completion,
	}
}

func (e *CompleteEvent) Complete() *agent.Completion {
	return e.completion
}

// tool call result event
type ToolResultEvent struct {
	baseEvent
	result *agent.ToolResult
}

func NewToolCallResultEvent(result *agent.ToolResult) *ToolResultEvent {
	return &ToolResultEvent{
		baseEvent: newBaseEvent(),
		result:    result,
	}
}

func (e *ToolResultEvent) Result() *agent.ToolResult {
	return e.result
}

// tool call error event
type ToolCallErrEvent struct {
	baseEvent
	toolName agent.ToolName
	toolArgs agent.ToolArguments
	err      error
}

func NewErrToolCallEvent(toolName agent.ToolName, args agent.ToolArguments, err error) *ToolCallErrEvent {
	return &ToolCallErrEvent{
		baseEvent: newBaseEvent(),
		toolName:  toolName,
		toolArgs:  args,
		err:       err,
	}
}

func (e *ToolCallErrEvent) ToolName() agent.ToolName {
	return e.toolName
}

func (e *ToolCallErrEvent) ToolArgs() agent.ToolArguments {
	return e.toolArgs
}

func (e *ToolCallErrEvent) Error() error {
	return e.err
}

// loop exit event
type LoopExitEvent struct {
	baseEvent
	err error
}

func NewLoopExitEvent(err error) *LoopExitEvent {
	return &LoopExitEvent{
		baseEvent: newBaseEvent(),
		err:       err,
	}
}

func (e *LoopExitEvent) Err() error {
	return e.err
}
