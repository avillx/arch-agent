package tools

import (
	"arch-agent/internal/agent"
	"context"
	"encoding/json"
	"log/slog"
)

func unwrapArgs[T any](raw agent.ToolArguments) (T, error) {
	var args T
	if err := json.Unmarshal(raw, &args); err != nil {
		return args, err
	}
	return args, nil
}

func mustAgentID(ctx context.Context) agent.ID {
	agentID, ok := agent.IDFromContext(ctx)
	if !ok {
		slog.Error("Critical context has no agentID")
	}
	return agentID
}
