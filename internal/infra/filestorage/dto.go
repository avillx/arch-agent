package filestorage

import (
	"arch-agent/internal/app/message"
	"arch-agent/internal/app/session"
	"encoding/json"
	"fmt"
)

type SessionDTO struct {
	ID       string       `json:"id"`
	Tokens   int          `json:"tokens"`
	Messages []MessageDTO `json:"messages"`
}

type MessageDTO struct {
	Role      string        `json:"role"`
	Content   string        `json:"content"`
	ToolCalls []ToolCallDTO `json:"tool_calls,omitempty"`
	CallID    string        `json:"call_id,omitempty"`
}

type ToolCallDTO struct {
	ID   string          `json:"id"`
	Tool string          `json:"tool"`
	Args json.RawMessage `json:"args"`
}

func messageToDTO(msg message.Message) MessageDTO {
	dto := MessageDTO{
		Role:    string(msg.Role()),
		Content: msg.Content(),
	}

	switch typed := msg.(type) {
	case *message.AgentMessage:
		dto.ToolCalls = toolcallsToDTO(typed.ToolCalls())
	case *message.ToolResultMessage:
		dto.CallID = typed.ToolCallID()
	}

	return dto
}

func messagesToDTO(msgs []message.Message) []MessageDTO {
	dtos := make([]MessageDTO, len(msgs))
	for i, msg := range msgs {
		dtos[i] = messageToDTO(msg)
	}
	return dtos
}

func toolcallsToDTO(calls []*message.ToolCall) []ToolCallDTO {
	dto := make([]ToolCallDTO, 0, len(calls))
	for _, tc := range calls {
		dto = append(dto, ToolCallDTO{
			ID:   tc.ID(),
			Tool: tc.ToolName(),
			Args: json.RawMessage(tc.Arguments()),
		})
	}
	return dto
}

func dtoToMessage(dto MessageDTO) (message.Message, error) {
	switch dto.Role {
	case "user":
		return message.NewUserMessage(dto.Content), nil

	case "system":
		return message.NewSystemMessage(dto.Content), nil

	case "agent":

		var toolCalls []*message.ToolCall
		for _, tc := range dto.ToolCalls {
			newToolCall := message.NewToolCall(tc.ID, tc.Tool, message.ToolArguments(tc.Args))
			toolCalls = append(toolCalls, newToolCall)
		}
		return message.NewAgentMessage(dto.Content, toolCalls), nil

	case "tool":

		return message.NewToolResultMessage(dto.CallID, dto.Content), nil

	default:
		return nil, fmt.Errorf("unknown role: %s", dto.Role)
	}
}

func dtoToMessages(dtos []MessageDTO) ([]message.Message, error) {
	msgs := make([]message.Message, len(dtos))
	for i, dto := range dtos {
		msg, err := dtoToMessage(dto)
		if err != nil {
			return nil, err
		}
		msgs[i] = msg
	}
	return msgs, nil
}

func dtoToSession(dto SessionDTO) (*session.Session, error) {
	msgs, err := dtoToMessages(dto.Messages)
	if err != nil {
		return nil, err
	}
	return session.NewSession(dto.ID, dto.Tokens, msgs), nil
}
func sessionToDTO(s *session.Session) SessionDTO {
	return SessionDTO{
		ID:       s.ID(),
		Tokens:   s.Tokens,
		Messages: messagesToDTO(s.Messages()),
	}
}
