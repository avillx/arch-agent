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
	taskService *task.TaskService
	cronFactory func(string) (task.Cron, error)
}

func NewAddTaskTool(s *task.TaskService, cronFactory func(string) (task.Cron, error)) *AddTaskTool {
	return &AddTaskTool{
		taskService: s,
		cronFactory: cronFactory,
	}
}

func (t *AddTaskTool) Name() agent.ToolName {
	return "create_task"
}

func (t *AddTaskTool) Description() string {
	return "creates a task that will be sended for agents by reglament" +
		"for some reminds or request to do somthing at some time once, use oneshot." +
		"before create a regular task ensure that actually what you should do" +
		"if debt - clarify it"
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
	args, err := tools.UnwrapArgs[struct {
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
