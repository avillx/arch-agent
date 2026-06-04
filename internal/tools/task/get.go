package tasktools

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/runtime"
	"arch-agent/internal/task"
	"context"
	"fmt"
	"strings"
)

// get tasks
type GetTasksTool struct {
	taskService *task.TaskService
}

func NewGetTasksTool(s *task.TaskService) *GetTasksTool {
	return &GetTasksTool{
		taskService: s,
	}
}

var _ runtime.Instructed = (*GetTasksTool)(nil)

func (t *GetTasksTool) Instruction() string {
	return `Tasks:
- Tasks is a cron-like sheduling yhat invokes you or other agents to process some request
- Tasks use cases:
  'remind somthing regular or only once, greet coworkers, congrat some one with birthday, check some status, etc...'
- When you manage tasks notify user directly like:
  'Okay, i remind you','Understand, i will texting you at this time', 'I won't read logs every... anymore.'.`
}

func (t *GetTasksTool) Name() agent.ToolName {
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
