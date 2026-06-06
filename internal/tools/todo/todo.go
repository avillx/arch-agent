package todo

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/tools"
	"context"
	"fmt"
	"strings"
)

type Status string

const (
	Pending    Status = "pending"
	InProgress Status = "in_progress"
	Done       Status = "done"
	Declined   Status = "declined"
)

var statusBadge = map[Status]string{
	Pending:    "[ ]",
	InProgress: "[>]",
	Done:       "[x]",
	Declined:   "[-]",
}

func renderList(items []TodoItem) string {
	if len(items) == 0 {
		return "_No todos yet._"
	}
	var sb strings.Builder
	for _, item := range items {
		badge := statusBadge[item.Status]
		fmt.Fprintf(&sb, "- %s #%d %s\n", badge, item.ID, item.Title)
	}
	return sb.String()
}

type CreateTodoTool struct{ Store Store }

func (t *CreateTodoTool) Name() agent.ToolName { return "create_todo" }
func (t *CreateTodoTool) Description() string  { return "create one or more todo items" }

func (t *CreateTodoTool) Instruction() string {
	return `Todo management:
- Use create_todo to plan steps before starting a task.
- Keep titles short and action-oriented.`
}

func (t *CreateTodoTool) Schema() []agent.ToolProperty {
	return []agent.ToolProperty{
		{
			Name:        "titles",
			Required:    true,
			IsArray:     true,
			Type:        agent.TypeString,
			Description: "list of todo titles to create",
		},
	}
}

func (t *CreateTodoTool) Call(ctx context.Context, rawArgs agent.ToolArguments) (string, error) {
	args, err := tools.UnwrapArgs[struct {
		Titles []string `json:"titles"`
	}](rawArgs)
	if err != nil {
		return "", err
	}

	sessionID := tools.MustSessionID(ctx)
	agentID := tools.MustAgentID(ctx)

	items := make([]TodoItem, len(args.Titles))
	for i, title := range args.Titles {
		items[i] = TodoItem{Title: title}
	}
	t.Store.Add(sessionID, agentID, items)

	all := t.Store.List(sessionID, agentID)
	return "Todos created:\n" + renderList(all), nil
}

type UpdateTodoTool struct{ Store Store }

func (t *UpdateTodoTool) Name() agent.ToolName { return "update_todo" }
func (t *UpdateTodoTool) Description() string  { return "update the status of a todo by id" }

func (t *UpdateTodoTool) Instruction() string { return "" }

func (t *UpdateTodoTool) Schema() []agent.ToolProperty {
	return []agent.ToolProperty{
		{
			Name:        "id",
			Required:    true,
			Type:        agent.TypeNumber,
			Description: "todo id to update",
		},
		{
			Name:        "status",
			Required:    true,
			Type:        agent.TypeString,
			Description: "new status: pending | in_progress | done | declined",
		},
	}
}

func (t *UpdateTodoTool) Call(ctx context.Context, rawArgs agent.ToolArguments) (string, error) {
	args, err := tools.UnwrapArgs[struct {
		ID     int    `json:"id"`
		Status Status `json:"status"`
	}](rawArgs)
	if err != nil {
		return "", err
	}

	sessionID := tools.MustSessionID(ctx)
	agentID := tools.MustAgentID(ctx)

	if err := t.Store.Update(sessionID, agentID, args.ID, args.Status); err != nil {
		return "", err
	}

	all := t.Store.List(sessionID, agentID)
	return fmt.Sprintf("Todo #%d updated to **%s**:\n%s", args.ID, args.Status, renderList(all)), nil
}

type ListTodoTool struct{ Store Store }

func (t *ListTodoTool) Name() agent.ToolName { return "list_todo" }
func (t *ListTodoTool) Description() string {
	return "list all todos for the current session and agent"
}

func (t *ListTodoTool) Instruction() string { return "" }

func (t *ListTodoTool) Schema() []agent.ToolProperty { return nil }

func (t *ListTodoTool) Call(ctx context.Context, _ agent.ToolArguments) (string, error) {
	sessionID := tools.MustSessionID(ctx)
	agentID := tools.MustAgentID(ctx)

	items := t.Store.List(sessionID, agentID)
	return "**Todo list:**\n" + renderList(items), nil
}
