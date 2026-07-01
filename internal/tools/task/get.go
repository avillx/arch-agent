package tasktools

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/task"
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
)

// get tasks
type GetTasksTool struct {
	taskSvc *task.Service
}

func NewGetTasksTool(s *task.Service) *GetTasksTool {
	return &GetTasksTool{
		taskSvc: s,
	}
}

func (t *GetTasksTool) Name() agent.ToolName {
	return "get_tasks"
}

func (t *GetTasksTool) Description() string {
	return "List all tasks with their current active state"
}
func (t *GetTasksTool) Schema() []agent.ToolProperty {
	return []agent.ToolProperty{}
}

func (t *GetTasksTool) Call(ctx context.Context, _ agent.ToolArguments) (string, error) {

	tasks, err := t.taskSvc.All()
	if err != nil {
		return "task is not created", err
	}

	return reprTaskRecords(slices.Collect(maps.Values(tasks))), nil
}

func reprTaskRecords(tasks []*task.TaskRecord) string {
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
		for _, r := range t.Recipients() {
			agentString = append(agentString, string(r))
		}

		record := fmt.Sprintf(
			"* %s (state: %s) (cron: %s) (recipients: %s) - %s\n",
			t.Name(),
			state,
			t.Reglament(),
			strings.Join(agentString, " "),
			t.Description(),
		)
		sb.WriteString(record)
	}

	return sb.String()
}
