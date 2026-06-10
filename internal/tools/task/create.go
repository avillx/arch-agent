package tasktools

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/task"
	"arch-agent/internal/tools"
	"context"
	"strings"
)

// add task
type AddTaskTool struct {
	taskSvc     *task.Service
	cronFactory func(string) (task.Cron, error)
}

func NewAddTaskTool(s *task.Service, cronFactory func(string) (task.Cron, error)) *AddTaskTool {
	return &AddTaskTool{
		taskSvc:     s,
		cronFactory: cronFactory,
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
			Description: "Space-separated agent IDs in lowercase (e.g. agent1 agent2); verify agents exist",
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
		Name        string `json:"name"`
		Description string `json:"description"`
		Recipients  string `json:"recipients"`
		Reglament   string `json:"schedule"`
		Request     string `json:"request"`
		Oneshot     bool   `json:"oneshot"`
	}](rawArgs)
	if err != nil {
		return "", err
	}

	agents := []agent.ID{}
	for _, a := range strings.Fields(args.Recipients) {
		agents = append(agents, agent.ID(a))
	}

	reglament, err := t.cronFactory(args.Reglament)
	if err != nil {
		return "", err
	}

	task := task.NewTask(
		args.Name,
		args.Description,
		agents,
		args.Request,
		reglament,
		args.Oneshot,
	)

	taskID, err := t.taskSvc.New(task)
	if err != nil {
		return "task is not created", err
	}

	if err := t.taskSvc.Start(taskID); err != nil {
		return "task was not runned", err
	}

	return "task has been created", nil
}
