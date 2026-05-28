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

type EventReader struct {
	OnError      func(agent.ID, session.ID, error)
	OnComplete   func(agent.ID, session.ID, *agent.Completion)
	OnToolResult func(agent.ID, session.ID, agent.ToolCallResult)
}

func (r EventReader) Read(ch <-chan Event) {
	for ev := range ch {
		switch typedEv := ev.(type) {
		case ErrEvent:
			if r.OnError != nil {
				r.OnError(typedEv.Agent(), typedEv.Session(), typedEv.Err())
			}
		case CompleteEvent:
			if r.OnComplete != nil {
				r.OnComplete(typedEv.Agent(), typedEv.Session(), typedEv.Complete())
			}
		case ToolCallResultEvent:
			if r.OnToolResult != nil {
				r.OnToolResult(typedEv.Agent(), typedEv.Session(), agent.ToolCallResult{Result: typedEv.Result()})
			}
		}
	}
}

type Event interface {
	Agent() agent.ID
	Session() session.ID
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
	Result() string
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
	result string
}

func NewToolCallResultEvent(agentID agent.ID, sessionID session.ID, result string) toolCallResultEvent {
	return toolCallResultEvent{
		baseEvent: baseEvent{
			agentID:   agentID,
			sessionID: sessionID,
		},
		result: result,
	}
}

func (e toolCallResultEvent) Result() string {
	return e.result
}

//

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
