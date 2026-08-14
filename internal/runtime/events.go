package runtime

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/session"
)

var (
	_ ErrEvent            = errEvent{}
	_ CompleteEvent       = completeEvent{}
	_ ToolCallResultEvent = toolCallResultEvent{}
)

type Event interface {
	Agent() agent.ID
	Session() session.ID
}

type CompactionEvent interface {
	Summary() string
}

type compactionEvent struct {
	baseEvent
	summary string
}

func NewCompactionEvent(agentID agent.ID, sessionID session.ID, summary string) compactionEvent {
	return compactionEvent{
		baseEvent: baseEvent{
			agentID:   agentID,
			sessionID: sessionID,
		},
		summary: summary,
	}
}

func (ev compactionEvent) Summary() string {
	return ev.summary
}

type ErrEvent interface {
	Event
	Err() error
}

type CompleteEvent interface {
	Event
	Complete() *agent.Completion
}

type ToolCallResultEvent interface {
	Event
	Result() *agent.ToolResult
}

type baseEvent struct {
	agentID   agent.ID
	sessionID session.ID
}

func (e baseEvent) Agent() agent.ID {
	return e.agentID
}

func (e baseEvent) Session() session.ID {
	return e.sessionID
}

type errEvent struct {
	baseEvent
	err error
}

func NewErrEvent(agentID agent.ID, sessionID session.ID, err error) errEvent {
	return errEvent{
		baseEvent: baseEvent{
			agentID:   agentID,
			sessionID: sessionID,
		},
		err: err,
	}
}

func (e errEvent) Err() error {
	return e.err
}

type completeEvent struct {
	baseEvent
	completion *agent.Completion
}

func NewCompleteEvent(agentID agent.ID, sessionID session.ID, completion *agent.Completion) completeEvent {
	return completeEvent{
		baseEvent: baseEvent{
			agentID:   agentID,
			sessionID: sessionID,
		},
		completion: completion,
	}
}

func (e completeEvent) Complete() *agent.Completion {
	return e.completion
}

type toolCallResultEvent struct {
	baseEvent
	result *agent.ToolResult
}

func NewToolCallResultEvent(agentID agent.ID, sessionID session.ID, result *agent.ToolResult) toolCallResultEvent {
	return toolCallResultEvent{
		baseEvent: baseEvent{
			agentID:   agentID,
			sessionID: sessionID,
		},
		result: result,
	}
}

func (e toolCallResultEvent) Result() *agent.ToolResult {
	return e.result
}

type ErrToolCallEvent struct {
	baseEvent
	err error
}

func NewErrToolCallEvent(agentID agent.ID, sessionID session.ID, err error) ErrToolCallEvent {
	return ErrToolCallEvent{
		baseEvent: baseEvent{
			agentID:   agentID,
			sessionID: sessionID,
		},
		err: err,
	}
}

func (e ErrToolCallEvent) Error() error {
	return e.err
}
