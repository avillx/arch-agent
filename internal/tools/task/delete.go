package tasktools

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/task"
	"arch-agent/internal/tools"
	"context"
	"fmt"
)

type DeleteTasksTool struct {
	taskSvc *task.Service
}

func NewDeleteTasksTool(s *task.Service) *DeleteTasksTool {
	return &DeleteTasksTool{
		taskSvc: s,
	}
}

func (t *DeleteTasksTool) Name() agent.ToolName {
	return "delete_task"
}

func (t *DeleteTasksTool) Description() string {
	return "Deletes selected task"
}
func (t *DeleteTasksTool) Schema() []agent.ToolProperty {
	return []agent.ToolProperty{
		{
			Name:        "name",
			Required:    true,
			Type:        agent.TypeString,
			Description: "Name of existing task for deletion",
		},
	}
}

func (t *DeleteTasksTool) Call(ctx context.Context, rawArgs agent.ToolArguments) (string, error) {
	args, err := tools.UnwrapArgs[struct {
		Name string `json:"name"`
	}](rawArgs)
	if err != nil {
		return "", err
	}

	if err := t.taskSvc.Delete(args.Name); err != nil {
		return "", mapSvcErrors(err)
	}

	return fmt.Sprintf("task %s deleted succecceful", args.Name), nil
}
