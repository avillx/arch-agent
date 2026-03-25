package conversation

type Role string

const (
	User   Role = "user"
	Agent  Role = "agent"
	Tool   Role = "tool"
	System Role = "system"
)

type messageBase struct {
	role    Role
	content string
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
	toolCalls []ToolCall
}

func (m *AgentMessage) ToolCalls() []ToolCall {
	return m.toolCalls
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

func NewAgentMessage(content string, tc []ToolCall) *AgentMessage {
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
