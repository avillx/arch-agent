package tasktools

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/task"
	"arch-agent/internal/tools"
	"context"
)

// add task
type AddTaskTool struct {
	taskSvc *task.Service
}

func NewAddTaskTool(s *task.Service) *AddTaskTool {
	return &AddTaskTool{
		taskSvc: s,
	}
}

func (t *AddTaskTool) Name() agent.ToolName {
	return "create_task"
}

func (t *AddTaskTool) Description() string {
	return "Create a scheduled task that dispatches a request to agents on a cron schedule; use oneshot for one-time reminders"
}
func (t *AddTaskTool) Schema() []agent.ToolProperty {
	return []agent.ToolProperty{
		{
			Name:        "name",
			Required:    true,
			Type:        agent.TypeString,
			Description: "Short unique task name (e.g. daily_report)",
		},
		{
			Name:        "description",
			Required:    true,
			Type:        agent.TypeString,
			Description: "One-line summary of the task",
		},
		{
			Name:        "recipients",
			Required:    true,
			Type:        agent.TypeString,
			IsArray:     true,
			Description: "agent IDs in lowercase; verify agents exist",
		},
		{
			Name:        "schedule",
			Required:    true,
			Type:        agent.TypeString,
			Description: "Cron expression (e.g. * * * * *)",
		},
		{
			Name:        "request",
			Required:    true,
			Type:        agent.TypeString,
			Description: "Detailed instructions sent to the agent on each invocation",
		},
		{
			Name:        "oneshot",
			Required:    true,
			Type:        agent.TypeBoolean,
			Description: "If true, runs once then deactivates automatically",
		},
	}
}

func (t *AddTaskTool) Call(ctx context.Context, rawArgs agent.ToolArguments) (string, error) {
	args, err := tools.UnwrapArgs[struct {
		Name        string     `json:"name"`
		Description string     `json:"description"`
		Recipients  []agent.ID `json:"recipients"`
		Reglament   string     `json:"schedule"`
		Request     string     `json:"request"`
		Oneshot     bool       `json:"oneshot"`
	}](rawArgs)
	if err != nil {
		return "", err
	}

	newTask, err := task.NewValidTaskConfig(
		args.Name,
		args.Description,
		args.Recipients,
		args.Reglament,
		args.Request,
		args.Oneshot,
	)
	if err != nil {
		return "task is not created", unwrapValidationError(err)
	}

	if err := t.taskSvc.AddTask(newTask); err != nil {
		return "task is not created", mapSvcErrors(err)
	}

	if err := t.taskSvc.Start(args.Name); err != nil {
		return "task was not runned", mapSvcErrors(err)
	}

	return "task created succecceful", nil
}
