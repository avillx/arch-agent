package types

import (
	"fmt"
	"slices"
	"strings"
)

type Role string

const (
	User   Role = "user"
	Agent  Role = "agent"
	Tool   Role = "tool"
	System Role = "system"
)

type Message interface {
	Role() Role
	Content() string
	String() string
}

type messageBase struct {
	role    Role
	content string
}

func (b messageBase) String() string {
	return fmt.Sprintf("%s:\n%s\n\n", b.role, b.content)
}

func (b messageBase) Role() Role {
	return b.role
}

func (b messageBase) Content() string {
	return b.content
}

type UserMessage struct {
	messageBase
}

type AgentMessage struct {
	messageBase
	toolCalls []*ToolCall
}

func (b AgentMessage) ToolCalls() []*ToolCall {
	return b.toolCalls
}

func (b AgentMessage) String() string {
	return fmt.Sprintf("%s\n%s\n\n", b.messageBase.String(), toolCallsString(b.toolCalls))
}

type ToolResultMessage struct {
	toolCallID string
	messageBase
}

func (m *ToolResultMessage) ToolCallID() string {
	return m.toolCallID
}

type SystemMessage struct {
	messageBase
}

func NewUserMessage(content string) *UserMessage {
	return &UserMessage{
		messageBase: messageBase{
			role:    User,
			content: content,
		},
	}
}

func NewAgentMessage(content string, tc []*ToolCall) *AgentMessage {
	return &AgentMessage{
		messageBase: messageBase{
			role:    Agent,
			content: content,
		},
		toolCalls: tc,
	}
}

func NewToolResultMessage(id string, content string) *ToolResultMessage {
	return &ToolResultMessage{
		toolCallID: id,
		messageBase: messageBase{
			role:    Tool,
			content: content,
		},
	}
}

func NewSystemMessage(content string) *SystemMessage {
	return &SystemMessage{
		messageBase: messageBase{
			role:    System,
			content: content,
		},
	}
}

func toolCallsString(calls []*ToolCall) string {
	var sb strings.Builder
	sb.WriteString("<tool-calls>")
	for _, call := range calls {
		sb.WriteString(call.String())
	}
	sb.WriteString("</tool-calls>")
	return sb.String()
}

func StringifyConversation(messages []Message) string {
	var sb strings.Builder
	for _, msg := range messages {
		sb.WriteString(msg.String())
	}
	return sb.String()
}

func ExcludeToolCallsData(msgs []Message, toolNames []string) []Message {
	eliminationCallsID := map[string]struct{}{}
	toolNameSet := map[string]struct{}{}

	for _, name := range toolNames {
		toolNameSet[name] = struct{}{}
	}

	for _, m := range msgs {
		agentMessage, ok := m.(*AgentMessage)
		if !ok {
			continue
		}
		for _, tc := range agentMessage.ToolCalls() {
			if _, found := toolNameSet[tc.ToolName]; found {
				eliminationCallsID[tc.ID] = struct{}{}
			}
		}
	}

	return slices.DeleteFunc(msgs, func(msg Message) bool {
		if toolMessage, ok := msg.(*ToolResultMessage); ok {
			_, found := eliminationCallsID[toolMessage.ToolCallID()]
			return found
		}
		return false
	})
}
