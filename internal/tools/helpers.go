package tools

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/runtime"
	"arch-agent/internal/session"
	"context"
	"encoding/json"
	"log/slog"
)

func UnwrapArgs[T any](raw agent.ToolArguments) (T, error) {
	var args T
	if err := json.Unmarshal(raw, &args); err != nil {
		return args, err
	}
	return args, nil
}

func MustAgentID(ctx context.Context) agent.ID {
	agentID, ok := runtime.AgentIDFromContext(ctx)
	if !ok {
		slog.Error("critical error context has no agentID")
	}
	return agentID
}

func MustSessionID(ctx context.Context) session.ID {
	sessionID, ok := runtime.SessionIDFromContext(ctx)
	if !ok {
		slog.Error("critical error context has no sessionID")
	}
	return sessionID
}
