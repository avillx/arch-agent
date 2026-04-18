package storage

import (
	"arch-agent/internal/app/types"
	"encoding/json"
	"fmt"
)

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

func MessageToDTO(msg types.Message) MessageDTO {
	dto := MessageDTO{
		Role:    string(msg.Role()),
		Content: msg.Content(),
	}

	switch typed := msg.(type) {
	case *types.AgentMessage:
		dto.ToolCalls = ToolcallsToDTO(typed.ToolCalls())
	case *types.ToolResultMessage:
		dto.CallID = typed.ToolCallID()
	}

	return dto
}

func MessagesToDTO(msgs []types.Message) []MessageDTO {
	dtos := make([]MessageDTO, len(msgs))
	for i, msg := range msgs {
		dtos[i] = MessageToDTO(msg)
	}
	return dtos
}

func ToolcallsToDTO(calls []*types.ToolCall) []ToolCallDTO {
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

func DtoToMessage(dto MessageDTO) (types.Message, error) {
	switch dto.Role {
	case "user":
		return types.NewUserMessage(dto.Content), nil

	case "system":
		return types.NewSystemMessage(dto.Content), nil

	case "agent":

		var toolCalls []*types.ToolCall
		for _, tc := range dto.ToolCalls {
			newToolCall := types.NewToolCall(tc.ID, tc.Tool, types.ToolArguments(tc.Args))
			toolCalls = append(toolCalls, newToolCall)
		}
		return types.NewAgentMessage(dto.Content, toolCalls), nil

	case "tool":

		return types.NewToolResultMessage(dto.CallID, dto.Content), nil

	default:
		return nil, fmt.Errorf("unknown role: %s", dto.Role)
	}
}

func DtoToMessages(dtos []MessageDTO) ([]types.Message, error) {
	msgs := make([]types.Message, len(dtos))
	for i, dto := range dtos {
		msg, err := DtoToMessage(dto)
		if err != nil {
			return nil, err
		}
		msgs[i] = msg
	}
	return msgs, nil
}
