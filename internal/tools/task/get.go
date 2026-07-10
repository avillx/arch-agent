package tasktools

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/task"
	"arch-agent/internal/tools"
	"context"
	"fmt"
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

func (t *GetTasksTool) Call(ctx context.Context, _ agent.ToolArguments) ([]agent.ContentPart, error) {

	tasks, err := t.taskSvc.List()
	if err != nil {
		return nil, mapSvcErrors(err)
	}

	if len(tasks) > 0 {
		return tools.Result(reprTaskRecords(tasks)), nil
	}

	return tools.Result("has no tasks"), nil
}

func reprTaskRecords(tasks []task.TaskConfig) string {
	var sb strings.Builder
	sb.WriteString("# Tasks\n")

	for _, t := range tasks {
		fmt.Fprintf(&sb, "%s\n\n", reprTask(t))
	}

	return sb.String()
}

func reprTask(cfg task.TaskConfig) string {

	agentString := []string{}
	for _, r := range cfg.Recipients {
		agentString = append(agentString, string(r))
	}

	return fmt.Sprintf(
		"### %s\n(active: %s) (cron: %s) (recipients: %s) (oneshot: %s) - %s\nrequest:\n%s",
		cfg.Name,
		strconv.FormatBool(cfg.Active),
		cfg.Reglament,
		strings.Join(agentString, " "),
		strconv.FormatBool(cfg.Oneshot),
		cfg.Description,
		cfg.Request,
	)
}
