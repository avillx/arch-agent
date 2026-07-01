package tools

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/runtime"
	"arch-agent/internal/session"
	"arch-agent/internal/types"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
)

func UnwrapArgs[T any](raw agent.ToolArguments) (T, error) {
	var args T

	if err := json.Unmarshal(raw, &args); err != nil {
		return args, err
	}
	return args, nil
}

func UnwrapValidArgs[T types.Validator](ctx context.Context, raw agent.ToolArguments) (T, error) {
	var args T

	if err := json.Unmarshal(raw, &args); err != nil {
		return args, types.NewAgentMistakeError(err.Error())
	}
	if err := args.Validate(ctx); err != nil {

		var validationError *types.ValidationError
		if errors.As(err, &validationError) {

			var sb strings.Builder
			for k, v := range validationError.Problems() {
				fmt.Fprintf(&sb, "problem with %s - %s", k, v)
			}

			return args, types.NewAgentMistakeError(sb.String())
		}

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
