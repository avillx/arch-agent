package tasktools

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/task"
	"arch-agent/internal/tools"
	"context"
	"fmt"
)

type ToggleTaskTool struct {
	taskService *task.TaskService
}

func NewToggleTaskTool(s *task.TaskService) *ToggleTaskTool {
	return &ToggleTaskTool{
		taskService: s,
	}
}

func (t *ToggleTaskTool) Name() agent.ToolName {
	return "toggle_task"
}

func (t *ToggleTaskTool) Description() string {
	return "turn task, actives - inactive tasks and deactivate - active tasks"
}
func (t *ToggleTaskTool) Schema() []agent.ToolProperty {
	return []agent.ToolProperty{
		{
			Name:        "name",
			Required:    true,
			Type:        agent.TypeString,
			Description: "name of task",
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

	tasks, err := t.taskService.All()
	if err != nil {
		return "", err
	}

	task, ok := tasks[args.Name]
	if !ok {
		return "", fmt.Errorf("task %s does not exist", args.Name)
	}

	switch task.Active {
	case true:
		if err := t.taskService.Stop(args.Name); err != nil {
			return "", err
		}
		return fmt.Sprintf("task %s deactivated", args.Name), nil
	default:
		if err := t.taskService.Start(args.Name); err != nil {
			return "", err
		}
		return fmt.Sprintf("task %s activated", args.Name), nil
	}
}
