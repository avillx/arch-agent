package tasktools

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/task"
	"arch-agent/internal/tools"
	"context"
	"fmt"
)

type ToggleTaskTool struct {
	taskSvc *task.Service
}

func NewToggleTaskTool(s *task.Service) *ToggleTaskTool {
	return &ToggleTaskTool{
		taskSvc: s,
	}
}

func (t *ToggleTaskTool) Name() agent.ToolName {
	return "toggle_task"
}

func (t *ToggleTaskTool) Description() string {
	return "Toggle a task's active state: activates inactive tasks and deactivates active ones"
}
func (t *ToggleTaskTool) Schema() []agent.ToolProperty {
	return []agent.ToolProperty{
		{
			Name:        "name",
			Required:    true,
			Type:        agent.TypeString,
			Description: "Task name",
		},
	}
}

func (t *ToggleTaskTool) Call(ctx context.Context, rawArgs agent.ToolArguments) (string, error) {
	args, err := tools.UnwrapArgs[struct {
		Name string `json:"name"`
	}](rawArgs)
	if err != nil {
		return "", err
	}

	tasks, err := t.taskSvc.All()
	if err != nil {
		return "", err
	}

	task, ok := tasks[args.Name]
	if !ok {
		return "", fmt.Errorf("task %s does not exist", args.Name)
	}

	switch task.Active {
	case true:
		if err := t.taskSvc.Stop(args.Name); err != nil {
			return "", err
		}
		return fmt.Sprintf("task %s deactivated", args.Name), nil
	default:
		if err := t.taskSvc.Start(args.Name); err != nil {
			return "", err
		}
		return fmt.Sprintf("task %s activated", args.Name), nil
	}
}
