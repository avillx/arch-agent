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
		fmt.Fprintf(&sb, "%s\n", item.String())
	}
	return sb.String()
}

type CreateTodoTool struct{ Store Store }

func (t *CreateTodoTool) Name() agent.ToolName { return "create_todo" }
func (t *CreateTodoTool) Description() string  { return "Create one or more todos" }

func (t *CreateTodoTool) Instruction() string {
	return `## Todo managment:
When your task is too complex create a plan with todo list and follow it step by step
Use create_todo to plan steps before starting a multistep task.
Keep titles short and action-oriented.`
}

func (t *CreateTodoTool) Schema() any {
	return []agent.ToolProperty{
		{
			Name:        "titles",
			Required:    true,
			IsArray:     true,
			Type:        agent.TypeString,
			Description: "Todo titles to create",
		},
	}
}

func (t *CreateTodoTool) Call(ctx context.Context, rawArgs agent.ToolArguments) ([]agent.ContentPart, error) {
	args, err := tools.UnwrapArgs[struct {
		Titles []string `json:"titles"`
	}](rawArgs)
	if err != nil {
		return nil, err
	}

	sessionID := tools.MustSessionID(ctx)
	agentID := tools.MustAgentID(ctx)

	items := make([]TodoItem, len(args.Titles))
	for i, title := range args.Titles {
		items[i] = TodoItem{Title: title}
	}
	t.Store.Add(sessionID, agentID, items)

	all := t.Store.List(sessionID, agentID)
	return tools.Result("Todos created:\n" + renderList(all)), nil
}

type UpdateTodoTool struct{ Store Store }

func (t *UpdateTodoTool) Name() agent.ToolName { return "update_todo" }
func (t *UpdateTodoTool) Description() string  { return "Update a todo's status by ID" }

func (t *UpdateTodoTool) Schema() any {
	return []agent.ToolProperty{
		{
			Name:        "id",
			Required:    true,
			Type:        agent.TypeNumber,
			Description: "Todo ID",
		},
		{
			Name:        "status",
			Required:    true,
			Type:        agent.TypeString,
			Description: "New status value",
			Enum:        []string{"pending", "in_progress", "done", "declined"},
		},
	}
}

func (t *UpdateTodoTool) Call(ctx context.Context, rawArgs agent.ToolArguments) ([]agent.ContentPart, error) {
	args, err := tools.UnwrapArgs[struct {
		ID     int    `json:"id"`
		Status Status `json:"status"`
	}](rawArgs)
	if err != nil {
		return nil, err
	}

	sessionID := tools.MustSessionID(ctx)
	agentID := tools.MustAgentID(ctx)

	if err := t.Store.Update(sessionID, agentID, args.ID, args.Status); err != nil {
		return nil, err
	}

	all := t.Store.List(sessionID, agentID)

	resulMessage := fmt.Sprintf("Todo #%d updated to **%s**:\n%s", args.ID, args.Status, renderList(all))
	return tools.Result(resulMessage), nil
}

type ListTodoTool struct{ Store Store }

func (t *ListTodoTool) Name() agent.ToolName { return "list_todo" }
func (t *ListTodoTool) Description() string {
	return "List all todos in the current session"
}

func (t *ListTodoTool) Schema() any { return nil }

func (t *ListTodoTool) Call(ctx context.Context, _ agent.ToolArguments) ([]agent.ContentPart, error) {
	sessionID := tools.MustSessionID(ctx)
	agentID := tools.MustAgentID(ctx)

	items := t.Store.List(sessionID, agentID)
	return tools.Result("**Todo list:**\n" + renderList(items)), nil
}
