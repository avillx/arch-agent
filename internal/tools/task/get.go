package tasktools

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/task"
	"context"
	"fmt"
	"maps"
	"slices"
	"strconv"
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
		return "", mapSvcErrors(err)
	}

	if len(tasks) > 0 {
		return reprTaskRecords(slices.Collect(maps.Values(tasks))), nil
	}

	return "has no tasks", nil
}

func reprTaskRecords(tasks []*task.TaskRecord) string {
	var sb strings.Builder
	sb.WriteString("# Tasks\n")

	for _, t := range tasks {
		fmt.Fprintf(&sb, "%s\n\n", reprTask(t))
	}

	return sb.String()
}

func reprTask(rec *task.TaskRecord) string {

	agentString := []string{}
	for _, r := range rec.Recipients() {
		agentString = append(agentString, string(r))
	}

	state := ""
	if rec.Active {
		state = "active"
	} else {
		state = "inactive"
	}

	return fmt.Sprintf(
		"### %s\n(state: %s) (cron: %s) (recipients: %s) (oneshot: %s) - %s\nrequest:\n%s",
		rec.Name(),
		state,
		rec.Reglament(),
		strings.Join(agentString, " "),
		strconv.FormatBool(rec.Oneshot()),
		rec.Description(),
		rec.Request(),
	)
}
