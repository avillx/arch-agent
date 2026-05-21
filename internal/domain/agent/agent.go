package agent

import (
	"arch-agent/internal/domain/tool"
	"arch-agent/internal/domain/types"
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
)

type ID string

type Reasoner interface {
	RecallBudget() int
	Reason(context.Context, []tool.Tool, []types.Message) (*ReasonResult, error)
}

type ReasonResult struct {
	ToolCalls []*types.ToolCall
	Content   string
	Done      bool
}

type Agent struct {
	ID           ID
	Description  string
	SystemPrompt string
	Reasoner     Reasoner
	toolKit      map[string]tool.Tool
}

func NewAgent(
	id ID,
	description string,
	systemPrompt string,
	reasoner Reasoner,
	tools []tool.Tool,
) *Agent {
	return &Agent{
		ID:           id,
		Description:  description,
		SystemPrompt: systemPrompt,
		Reasoner:     reasoner,
		toolKit:      toolsToMap(tools),
	}
}

func (a *Agent) systemMessage(additional string) *types.SystemMessage {
	systemPrompt := strings.Join([]string{a.SystemPrompt, additional}, "\n")
	return types.NewSystemMessage(systemPrompt)
}

func (a *Agent) Chat(
	ctx context.Context,
	additionalSystemPrompt string,
	onResult func(result *ReasonResult),
	conversation []types.Message,
) (newMsgs []types.Message, err error) {

	ctx = withID(ctx, a.ID)

	messages := append([]types.Message{a.systemMessage(additionalSystemPrompt)}, conversation...)
	newMessages := []types.Message{}

	for i := 0; i < a.Reasoner.RecallBudget(); i++ {

		// reason request
		result, err := a.Reasoner.Reason(
			ctx,
			slices.Collect(maps.Values(a.toolKit)),
			append(messages, newMessages...),
		)
		if err != nil {
			return newMessages, err
		}

		newMessages = append(newMessages, types.NewAgentMessage(result.Content, result.ToolCalls))

		if onResult != nil {
			onResult(result)
		}

		// process tool calls
		for _, call := range result.ToolCalls {
			var callContent string

			t, exist := a.toolKit[call.ToolName]
			if !exist {
				callContent := fmt.Sprintf("tool %s is not exist", call.ToolName)
				newMessages = append(newMessages, types.NewToolResultMessage(call.ID, callContent))
				continue
			}

			res, err := t.Call(ctx, tool.ToolArguments(call.Arguments))
			if err != nil {
				callContent = res + err.Error()
				newMessages = append(newMessages, types.NewToolResultMessage(call.ID, callContent))
				continue
			}

			newMessages = append(newMessages, types.NewToolResultMessage(call.ID, res))
		}

		// done check
		if result.Done {
			return newMessages, nil
		}

	}

	return newMessages, errors.New("recall budget expires")
}

func toolsToMap(tools []tool.Tool) map[string]tool.Tool {
	toolMap := make(map[string]tool.Tool, len(tools))
	for _, t := range tools {
		toolMap[t.Name()] = t
	}
	return toolMap
}

type ctxKey struct{}

func withID(ctx context.Context, id ID) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}
func IDFromContext(ctx context.Context) (ID, bool) {
	id, ok := ctx.Value(ctxKey{}).(ID)
	return id, ok
}
