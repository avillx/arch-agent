package runtime

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/session"
	"errors"
	"log/slog"
)

var ErrContextOverflow = errors.New("context is overflow")

type RuntimeError struct {
	sessionID session.ID
	agt       agent.ID
	err       error
}

func NewRuntimeError(
	sessionID session.ID,
	agt agent.ID,
	err error,
) error {
	return &RuntimeError{
		sessionID: sessionID,
		agt:       agt,
		err:       err,
	}
}

func (e *RuntimeError) Session() session.ID { return e.sessionID }
func (e *RuntimeError) Agent() agent.ID     { return e.agt }
func (e *RuntimeError) Error() string       { return "runtime error" }
func (e *RuntimeError) Unwrap() error       { return e.err }

type ToolCallError struct {
	call *agent.ToolCall
	err  error
}

func NewToolCallError(call *agent.ToolCall, err error) error {
	return &ToolCallError{
		call: call,
		err:  err,
	}
}

func (e *ToolCallError) Call() *agent.ToolCall { return e.call }
func (e *ToolCallError) Error() string         { return "tool call processing error" }
func (e *ToolCallError) Unwrap() error         { return e.err }

func (e *ToolCallError) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("tool", string(e.call.ToolName)),
		slog.String("args", string(e.call.Arguments)),
		slog.String("cause", e.err.Error()),
	)
}
