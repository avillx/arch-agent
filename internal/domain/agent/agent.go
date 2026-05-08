package agent

import (
	"arch-agent/internal/domain/types"
	"context"
	"errors"
	"strings"
)

type ID string

type ReasonResult struct {
	ToolCalls []*types.ToolCall
	Content   string
	Done      bool
}

type Agent struct {
	ID             ID
	Description    string
	SystemPrompt   string
	Reasoner       Reasoner
	toolKit        ToolKit
	onContent      func(content string)
	contentChannel chan string
}

func NewAgent(
	id ID,
	description string,
	systemPrompt string,
	reasoner Reasoner,
	toolKit ToolKit,
) *Agent {
	return &Agent{
		ID:           id,
		Description:  description,
		SystemPrompt: systemPrompt,
		Reasoner:     reasoner,
		toolKit:      toolKit,
	}
}

func (a *Agent) systemMessage() *types.SystemMessage {
	systemPrompt := strings.Join([]string{a.SystemPrompt, a.toolKit.ToolGuides()}, "\n")
	return types.NewSystemMessage(systemPrompt)
}

func (a *Agent) OnContent(fn func(string)) {
	a.onContent = fn
}

func (a *Agent) Chat(ctx context.Context, conversation []types.Message) (newMsgs []types.Message, err error) {
	messages := append([]types.Message{a.systemMessage()}, conversation...)
	newMessages := []types.Message{}

	for i := 0; i < a.Reasoner.RecallBudget(); i++ {

		// reason request
		result, err := a.Reasoner.Reason(ctx, a.toolKit.Tools(), append(messages, newMessages...))
		if err != nil {
			return newMessages, err
		}

		// process content
		if result.Content != "" {
			newMessages = append(newMessages, types.NewAgentMessage(result.Content, result.ToolCalls))

			// TODO
			// Remove on content channel and create a ReasonResult channel
			if a.contentChannel != nil {
				select {
				case a.contentChannel <- result.Content:
				default:
				}
			}
		}

		// process tool calls
		for _, call := range result.ToolCalls {
			newMessages = append(newMessages, a.toolKit.SendCall(ctx, call))
		}

		// done check
		if result.Done {
			return newMessages, nil
		}

	}

	return newMessages, errors.New("recall budget expires")
}

type Reasoner interface {
	RecallBudget() int
	Reason(context.Context, []types.ToolDefinition, []types.Message) (*ReasonResult, error)
}

type ToolKit interface {
	Tools() []types.ToolDefinition
	ToolGuides() string
	SendCall(ctx context.Context, call *types.ToolCall) types.Message
}
