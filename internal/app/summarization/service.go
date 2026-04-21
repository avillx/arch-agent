package summarization

import (
	"arch-agent/internal/app/reasoning"
	"arch-agent/internal/app/types"
	"context"
	"fmt"
	"strings"
)

type promptRenderer interface {
	Render() (string, error)
}

type Service struct {
	reasoningService *reasoning.Service
	promptRenderer   promptRenderer
}

func NewService(rs *reasoning.Service, pr promptRenderer) *Service {
	return &Service{
		reasoningService: rs,
		promptRenderer:   pr,
	}
}

func (s *Service) Summarize(ctx context.Context, msgs []types.Message) (string, error) {
	prompt, err := s.promptRenderer.Render()
	if err != nil {
		return "", err
	}

	conversation := StringifyConversation(msgs)

	msg := []types.Message{types.NewUserMessage(conversation)}

	result, err := s.reasoningService.Reason(ctx, prompt, nil, msg)
	if err != nil {
		return "", err
	}

	sum := result[len(result)-1].Content()

	return sum, nil
}

// Helpers

func StringifyConversation(messages []types.Message) string {
	var sb strings.Builder

	for _, msg := range messages {
		record := messageToRecord(msg)
		sb.WriteString(record)
	}

	return sb.String()
}

func messageToRecord(msg types.Message) string {

	role := msg.Role()
	content := msg.Content()

	switch v := msg.(type) {
	case *types.AgentMessage:
		if tc := v.ToolCalls(); len(tc) > 0 {
			content += toolCallsRecord(tc)
		}
	}

	return fmt.Sprintf("%s:\n%s\n\n", role, content)
}

func toolCallsRecord(calls []*types.ToolCall) string {
	var sb strings.Builder
	sb.WriteString("<tool-calls>")
	for _, call := range calls {
		callsRecord := toolcallToRecord(call)
		sb.WriteString(callsRecord)
	}
	sb.WriteString("</tool-calls>")
	return sb.String()
}

func toolcallToRecord(tc *types.ToolCall) string {
	return fmt.Sprintf("tool:%s, args:%s", tc.ToolName, tc.Arguments)
}
