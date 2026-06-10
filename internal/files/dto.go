package files

import (
	"arch-agent/internal/agent"
	"encoding/json"
	"fmt"
)

type ContentPartDTO struct {
	Text  string `json:"text"`
	Image string `json:"image"`
}

func contentToDTO(parts []agent.ContentPart) []ContentPartDTO {
	res := make([]ContentPartDTO, len(parts))
	for i, p := range parts {
		res[i] = ContentPartDTO{
			Text:  p.Text,
			Image: p.ImageURL,
		}
	}
	return res
}

func dtoToContent(dtos []ContentPartDTO) []agent.ContentPart {
	res := make([]agent.ContentPart, len(dtos))
	for i, dto := range dtos {
		res[i] = agent.ContentPart{
			Text:     dto.Text,
			ImageURL: dto.Image,
		}
	}
	return res
}

type MessageDTO struct {
	Role      string           `json:"role"`
	Content   []ContentPartDTO `json:"content"`
	ToolCalls []ToolCallDTO    `json:"tool_calls,omitempty"`
	CallID    string           `json:"call_id,omitempty"`
}

type ToolCallDTO struct {
	ID   string          `json:"id"`
	Tool agent.ToolName  `json:"tool"`
	Args json.RawMessage `json:"args"`
}

func MessageToDTO(msg agent.Message) MessageDTO {
	dto := MessageDTO{
		Role:    string(msg.Role()),
		Content: contentToDTO(msg.Content()),
	}

	switch typed := msg.(type) {
	case *agent.AgentMessage:
		dto.ToolCalls = ToolcallsToDTO(typed.ToolCalls())
	case *agent.ToolResultMessage:
		dto.CallID = typed.ToolCallID()
	}

	return dto
}

func MessagesToDTO(msgs []agent.Message) []MessageDTO {
	dtos := make([]MessageDTO, len(msgs))
	for i, msg := range msgs {
		dtos[i] = MessageToDTO(msg)
	}
	return dtos
}

func ToolcallsToDTO(calls []*agent.ToolCall) []ToolCallDTO {
	dto := make([]ToolCallDTO, 0, len(calls))
	for _, tc := range calls {
		dto = append(dto, ToolCallDTO{
			ID:   tc.ID,
			Tool: tc.ToolName,
			Args: json.RawMessage(tc.Arguments),
		})
	}
	return dto
}

func DtoToMessage(dto MessageDTO) (agent.Message, error) {

	switch dto.Role {
	case "user":
		return agent.NewUserMessage(dtoToContent(dto.Content)), nil

	case "system":
		return agent.NewSystemMessage(dtoToContent(dto.Content)), nil

	case "agent":
		var toolCalls []*agent.ToolCall
		for _, tc := range dto.ToolCalls {
			newToolCall := agent.NewToolCall(tc.ID, tc.Tool, agent.ToolArguments(tc.Args))
			toolCalls = append(toolCalls, newToolCall)
		}
		return agent.NewAgentMessage(dtoToContent(dto.Content), toolCalls), nil

	case "tool":
		return agent.NewToolResultMessage(dto.CallID, dtoToContent(dto.Content)), nil

	default:
		return nil, fmt.Errorf("unknown role: %s", dto.Role)
	}
}

func DtoToMessages(dtos []MessageDTO) ([]agent.Message, error) {
	msgs := make([]agent.Message, len(dtos))
	for i, dto := range dtos {
		msg, err := DtoToMessage(dto)
		if err != nil {
			return nil, err
		}
		msgs[i] = msg
	}
	return msgs, nil
}
