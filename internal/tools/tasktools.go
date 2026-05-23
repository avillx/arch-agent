package tools

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/task"
	"context"
	"fmt"
	"strings"
)

type ToggleTaskTool struct {
	taskService *task.TaskService
}

func NewToggleTaskTool(s *task.TaskService) *ToggleTaskTool {
	return &ToggleTaskTool{
		taskService: s,
	}
}

func (t *ToggleTaskTool) Name() string {
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
	args, err := unwrapArgs[struct {
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

// get tasks
type GetTasksTool struct {
	taskService *task.TaskService
}

func NewGetTasksTool(s *task.TaskService) *GetTasksTool {
	return &GetTasksTool{
		taskService: s,
	}
}

func (t *GetTasksTool) Name() string {
	return "get_tasks"
}

func (t *GetTasksTool) Description() string {
	return "returns full list of tasks"
}
func (t *GetTasksTool) Schema() []agent.ToolProperty {
	return []agent.ToolProperty{}
}

func (t *GetTasksTool) Call(ctx context.Context, _ agent.ToolArguments) (string, error) {

	tasks, err := t.taskService.All()
	if err != nil {
		return "task is not created", err
	}

	var sb strings.Builder
	sb.WriteString("## Tasks\n")
	for _, t := range tasks {
		state := ""
		if t.Active {
			state = "active"
		} else {
			state = "inactive"
		}

		agentString := []string{}
		for _, r := range t.Recipients {
			agentString = append(agentString, string(r))
		}

		record := fmt.Sprintf(
			"* %s (state: %s) (cron: %s) (recipients: %s) - %s\n",
			t.Name,
			state,
			t.Reglament.Expression(),
			strings.Join(agentString, " "),
			t.Description,
		)
		sb.WriteString(record)
	}

	return sb.String(), nil
}

// add task
type AddTaskTool struct {
	taskService *task.TaskService
	cronFactory func(string) (task.Cron, error)
}

func NewAddTaskTool(s *task.TaskService, cronFactory func(string) (task.Cron, error)) *AddTaskTool {
	return &AddTaskTool{
		taskService: s,
		cronFactory: cronFactory,
	}
}

func (t *AddTaskTool) Name() string {
	return "create_task"
}

func (t *AddTaskTool) Description() string {
	return "creates a task that will be sended for agents by reglament"
}
func (t *AddTaskTool) Schema() []agent.ToolProperty {
	return []agent.ToolProperty{
		{
			Name:        "name",
			Required:    true,
			Type:        agent.TypeString,
			Description: "short unique name for task (e.g. some_task)",
		},
		{
			Name:        "description",
			Required:    true,
			Type:        agent.TypeString,
			Description: "one-line hook description of task",
		},
		{
			Name:        "recipients",
			Required:    true,
			Type:        agent.TypeString,
			Description: "agents ID's who recive task, enumirate them with whitespaces, (e.g. agent1 agent2 agent3 ), ensure agents existance (use lower case)",
		},
		{
			Name:        "reglament",
			Required:    true,
			Type:        agent.TypeString,
			Description: "cron like reglament (e.g. * * * * * )",
		},
		{
			Name:        "request",
			Required:    true,
			Type:        agent.TypeString,
			Description: "This is request for agent, what is need to do, create a detailed explaination of task",
		},
		{
			Name:        "oneshot",
			Required:    true,
			Type:        agent.TypeBoolean,
			Description: "if true - invokes only once, and deactives when done",
		},
	}
}

func (t *AddTaskTool) Call(ctx context.Context, rawArgs agent.ToolArguments) (string, error) {
	args, err := unwrapArgs[struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Recipients  string `json:"recipients"`
		Reglament   string `json:"reglament"`
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

	taskID, err := t.taskService.New(task)
	if err != nil {
		return "task is not created", err
	}

	if err := t.taskService.Start(taskID); err != nil {
		return "task was not runned", err
	}

	return "task has been created", nil
}
