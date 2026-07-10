package agent

import (
	"encoding/base64"
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

type AllowedMIME string

const (
	Png  AllowedMIME = "image/png"
	Jpeg AllowedMIME = "image/jpeg"
	Webp AllowedMIME = "image/webp"
	Bmp  AllowedMIME = "image/bmp"
)

func NewImageContent(mimeType AllowedMIME, data []byte) (ContentPart, error) {

	encoded := base64.StdEncoding.EncodeToString(data)

	// validate mime type
	if !slices.Contains([]AllowedMIME{Png, Jpeg, Webp, Bmp}, mimeType) {
		return ContentPart{}, fmt.Errorf("type %s is not allowed", mimeType)
	}

	return ContentPart{
		ImageURL: fmt.Sprintf("data:%s;base64,%s", mimeType, encoded),
	}, nil
}

func NewContent[T string | []ContentPart](content T) []ContentPart {
	var contentParts []ContentPart
	switch v := any(content).(type) {
	case string:
		contentParts = []ContentPart{{Text: v}}
	case []ContentPart:
		contentParts = v
	}
	return contentParts
}

type messageBase struct {
	role    Role
	content []ContentPart
}

func newMessageBase[T string | []ContentPart](role Role, content T) messageBase {

	return messageBase{
		role:    role,
		content: NewContent(content),
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

func NewToolResultMessage(res *ToolResult) *ToolResultMessage {
	return &ToolResultMessage{
		toolCallID:  res.ID,
		messageBase: newMessageBase(ToolMessageRole, res.Result),
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
