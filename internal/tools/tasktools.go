package tools

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/task"
	"context"
	"time"
)

type AddTaskTool struct {
	taskService *task.TaskService
}

func NewAddTaskTool(s *task.TaskService) *AddTaskTool {
	return &AddTaskTool{
		taskService: s,
	}
}

func (t *AddTaskTool) Name() string {
	return "create_task"
}

func (t *AddTaskTool) Description() string {
	return "creates a task that will be sended for you after delay expires"
}
func (t *AddTaskTool) Schema() []agent.ToolProperty {
	return []agent.ToolProperty{
		{
			Name:        "name",
			Required:    true,
			Type:        agent.TypeString,
			Description: "short name for task",
		},
		{
			Name:        "description",
			Required:    true,
			Type:        agent.TypeString,
			Description: "one-line hook description of task",
		},
		{
			Name:        "delay",
			Required:    true,
			Type:        agent.TypeNumber,
			Description: "Delay in minutes, you receve this task after delay expires",
		},
		{
			Name:        "request",
			Required:    true,
			Type:        agent.TypeString,
			Description: "This is request for yourself, what is need to do, create a detailed explaination of task",
		},
	}
}

func (t *AddTaskTool) Call(ctx context.Context, rawArgs agent.ToolArguments) (string, error) {
	args, err := unwrapArgs[struct {
		Name        string        `json:"name"`
		Description string        `json:"description"`
		Request     string        `json:"request"`
		Delay       time.Duration `json:"delay"`
	}](rawArgs)
	if err != nil {
		return "", err
	}

	agentID := mustAgentID(ctx)

	taskID, err := t.taskService.New(
		task.NewTask(
			args.Name,
			args.Description,
			[]task.Recipient{
				task.Recipient(agentID),
			},
			args.Request,
			task.Every{
				D: args.Delay * time.Minute,
			},
			true,
		),
	)

	if err != nil {
		return "task is not created", err
	}

	if err := t.taskService.Start(taskID); err != nil {
		return "task was not runned", err
	}

	return "task has been created", nil

}
