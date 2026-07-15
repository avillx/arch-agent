package tasktools

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/task"
	"arch-agent/internal/tools"
	"context"
)

// add task
type EditTaskTool struct {
	taskSvc *task.Service
}

func (t *EditTaskTool) Name() agent.ToolName {
	return "edit_task"
}

func (t *EditTaskTool) Description() string {
	return `Patch an existing task by 'existed'. 
	Only provided fields change; 
	each one fully replaces its old value (recipients: full new list, not appended).`
}

func (t *EditTaskTool) Schema() any {
	return []agent.ToolProperty{
		{
			Name:        "existed",
			Required:    true,
			Type:        agent.TypeString,
			Description: "Name of the already existing task for editing",
		},
		{
			Name:        "name",
			Required:    false,
			Type:        agent.TypeString,
			Description: "A new short unique task name (e.g. daily_report)",
		},
		{
			Name:        "description",
			Required:    false,
			Type:        agent.TypeString,
			Description: "A new one-line summary of the task",
		},
		{
			Name:        "recipients",
			Required:    false,
			Type:        agent.TypeString,
			IsArray:     true,
			Description: "A new list of agent IDs in lowercase; full overrides current recipients list",
		},
		{
			Name:        "schedule",
			Required:    false,
			Type:        agent.TypeString,
			Description: "A new cron expression (e.g. * * * * *)",
		},
		{
			Name:        "request",
			Required:    false,
			Type:        agent.TypeString,
			Description: "A new detailed instructions sent to the agent on each invocation",
		},
		{
			Name:        "oneshot",
			Required:    false,
			Type:        agent.TypeBoolean,
			Description: "A new after execute order. if true deactivate task after first execution",
		},
		{
			Name:        "active",
			Required:    false,
			Type:        agent.TypeBoolean,
			Description: "A new state - true is enabled, false is disabled",
		},
	}
}

func (t *EditTaskTool) Call(ctx context.Context, rawArgs agent.ToolArguments) ([]agent.ContentPart, error) {
	args, err := tools.UnwrapArgs[struct {
		Existed string `json:"existed"`
		task.TaskPatch
	}](rawArgs)
	if err != nil {
		return nil, err
	}

	if err := t.taskSvc.Patch(args.Existed, args.TaskPatch); err != nil {
		err = mapSvcErrors(err)
		return tools.Result("task is not edited"), unwrapValidationError(err)
	}
	return tools.Result("task edited succecceful"), nil
}
