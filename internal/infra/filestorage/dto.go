package filestorage

import (
	"arch-agent/internal/app/message"
	"arch-agent/internal/app/session"
	"encoding/json"
	"fmt"
)

type basicMessageDTO struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ToolCallDTO struct {
	ID   string `json:"id"`
	Tool string `json:"tool"`
	Args string `json:"args"`
}

type AgentMessageDTO struct {
	basicMessageDTO
	ToolCalls []ToolCallDTO `json:"tool_calls"`
}

type ToolResultDTO struct {
	basicMessageDTO
	CallID string `json:"call_id"`
}

func messageToDTO(internalFromatMessage message.Message) (any, error) {
	switch msg := internalFromatMessage.(type) {
	case *message.AgentMessage:
		agentMsg := AgentMessageDTO{
			basicMessageDTO: basicMessageDTO{
				Role:    "agent",
				Content: msg.Content(),
			},
		}

		for _, tc := range msg.ToolCalls() {
			agentMsg.ToolCalls = append(agentMsg.ToolCalls, ToolCallDTO{
				ID:   tc.ID(),
				Tool: tc.ToolName(),
				Args: string(tc.Arguments()),
			})
		}

		return agentMsg, nil

	case *message.SystemMessage:
		return basicMessageDTO{
			Role:    "system",
			Content: msg.Content(),
		}, nil

	case *message.UserMessage:
		return basicMessageDTO{
			Role:    "user",
			Content: msg.Content(),
		}, nil
	case *message.ToolResultMessage:
		return ToolResultDTO{
			CallID: msg.ToolCallID(),
			basicMessageDTO: basicMessageDTO{
				Role:    "tool",
				Content: msg.Content(),
			},
		}, nil
	default:
		return nil, fmt.Errorf("unknown message type")
	}
}

func dtoToMessage(raw []byte) (message.Message, error) {
	var base basicMessageDTO
	if err := json.Unmarshal(raw, &base); err != nil {
		return nil, fmt.Errorf("unmarshal base: %w", err)
	}

	switch base.Role {
	case "user":
		return message.NewUserMessage(base.Content), nil

	case "system":
		return message.NewSystemMessage(base.Content), nil

	case "agent":
		var dto AgentMessageDTO
		if err := json.Unmarshal(raw, &dto); err != nil {
			return nil, fmt.Errorf("unmarshal agent: %w", err)
		}
		var toolCalls []*message.ToolCall
		for _, tc := range dto.ToolCalls {
			toolCalls = append(toolCalls, message.NewToolCall(tc.ID, tc.Tool, []byte(tc.Args)))
		}
		return message.NewAgentMessage(base.Content, toolCalls), nil

	case "tool":
		var dto ToolResultDTO
		if err := json.Unmarshal(raw, &dto); err != nil {
			return nil, fmt.Errorf("unmarshal tool result: %w", err)
		}
		return message.NewToolResultMessage(dto.CallID, base.Content), nil

	default:
		return nil, fmt.Errorf("unknown role: %s", base.Role)
	}
}

func dtoToMessages(raw []json.RawMessage) ([]message.Message, error) {
	msgs := make([]message.Message, len(raw))
	for i, rawMsg := range raw {
		msg, err := dtoToMessage(rawMsg)
		if err != nil {
			return nil, err
		}
		msgs[i] = msg
	}
	return msgs, nil
}

func UnmarshalSession(raw json.RawMessage) (*session.Session, error) {
	var record struct {
		ID       string            `json:"id"`
		Tokens   int               `json:"tokens"`
		Messages []json.RawMessage `json:"messages"`
	}
	if err := json.Unmarshal(raw, &record); err != nil {
		return nil, err
	}

	messages, err := dtoToMessages(record.Messages)
	if err != nil {
		return nil, err
	}

	return session.NewSession(record.ID, record.Tokens, messages), nil

}

func MarshalSession(s *session.Session) (json.RawMessage, error) {
	var rawMessages []any
	for _, m := range s.Messages() {
		dto, err := messageToDTO(m)
		if err != nil {
			return nil, err
		}
		rawMessages = append(rawMessages, dto)
	}

	return json.Marshal(struct {
		ID       string `json:"id"`
		Tokens   int    `json:"tokens"`
		Messages []any  `json:"messages"`
	}{
		ID:       s.ID(),
		Tokens:   s.Tokens,
		Messages: rawMessages,
	})
}
