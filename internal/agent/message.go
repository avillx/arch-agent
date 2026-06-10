package agent

import (
	"fmt"
	"slices"
	"strings"
)

type Role string

const (
	UserMessageRole   Role = "user"
	AgentMessageRole  Role = "agent"
	ToolMessageRole   Role = "tool"
	SystemMessageRole Role = "system"
)

type Message interface {
	Role() Role
	Content() []ContentPart
	SetContent([]ContentPart)
	String() string
}

type ContentPart struct {
	Text     string
	ImageURL string
	// InputAudio
	// File
}

type messageBase struct {
	role    Role
	content []ContentPart
}

func newMessageBase[T string | []ContentPart](role Role, content T) messageBase {
	var contentParts []ContentPart
	switch v := any(content).(type) {
	case string:
		contentParts = []ContentPart{{Text: v}}
	case []ContentPart:
		contentParts = v
	}

	return messageBase{
		role:    role,
		content: contentParts,
	}
}

func (b *messageBase) String() string {

	var textParts strings.Builder
	for _, c := range b.content {
		textParts.WriteString(c.Text)
	}

	return fmt.Sprintf("%s:\n%s\n\n", b.role, textParts.String())
}

func (b *messageBase) Role() Role {
	return b.role
}

func (b *messageBase) Content() []ContentPart {
	return b.content
}

func (b *messageBase) SetContent(cp []ContentPart) {
	b.content = cp
}

type UserMessage struct {
	messageBase
}

type AgentMessage struct {
	messageBase
	toolCalls []*ToolCall
}

func (b *AgentMessage) ToolCalls() []*ToolCall {
	return b.toolCalls
}

func (b *AgentMessage) String() string {
	toolCalls := ""
	if len(b.toolCalls) > 0 {
		toolCalls = toolCallsString(b.toolCalls)
	}
	return fmt.Sprintf("%s%s\n\n", b.messageBase.String(), toolCalls)
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

func NewUserMessage[T string | []ContentPart](content T) *UserMessage {
	return &UserMessage{
		messageBase: newMessageBase(UserMessageRole, content),
	}
}

func NewAgentMessage[T string | []ContentPart](content T, tc []*ToolCall) *AgentMessage {
	return &AgentMessage{
		messageBase: newMessageBase(AgentMessageRole, content),
		toolCalls:   tc,
	}
}

func NewToolResultMessage[T string | []ContentPart](id string, content T) *ToolResultMessage {
	return &ToolResultMessage{
		toolCallID:  id,
		messageBase: newMessageBase(ToolMessageRole, content),
	}
}

func NewSystemMessage[T string | []ContentPart](content T) *SystemMessage {
	return &SystemMessage{
		messageBase: newMessageBase(SystemMessageRole, content),
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

func ExcludeToolCallsData(msgs []Message, toolNames []ToolName) []Message {
	eliminationCallsID := map[string]struct{}{}
	toolNameSet := map[ToolName]struct{}{}

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

// agent package
func CloneMessage(m Message) Message {
	switch v := m.(type) {
	case *UserMessage:
		c := *v
		return &c
	case *AgentMessage:
		c := *v
		return &c
	case *ToolResultMessage:
		c := *v
		return &c
	case *SystemMessage:
		c := *v
		return &c
	default:
		return m
	}
}
