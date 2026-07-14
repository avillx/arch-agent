package session

import (
	"arch-agent/internal/agent"
	"reflect"
	"slices"
	"time"
)

const TokenLimit = 20000

// validate struct asn iface
var _ Session = (*session)(nil)

type ID string

type Session interface {
	ID() ID

	Messages() []agent.Message
	GetLastAgentMessage() *agent.AgentMessage
	GetLastUserMessage() *agent.UserMessage

	ApplyCompletion(*agent.Completion)
	AddMessages(...agent.Message)

	InputTokens() int64
	OutputTokens() int64

	OverwriteMessages(int64, []agent.Message)

	CreatedAt() time.Time
	UpdatedAt() time.Time

	SetExtras(map[string]any)
	Extras() map[string]any
}

type session struct {
	id           ID
	title        string
	inputTokens  int64
	outputTokens int64
	messages     []agent.Message
	createdAt    time.Time
	updatedAt    time.Time

	extras map[string]any
}

func NewSession(id ID) *session {
	return &session{
		id:           id,
		inputTokens:  0,
		outputTokens: 0,
		messages:     []agent.Message{},
		createdAt:    time.Now(),
		extras:       map[string]any{},
	}
}

func NewRestoredSession(
	id ID,
	messages []agent.Message,
	inputTokens int64,
	outputTokens int64,
	createdAt time.Time,
	extras map[string]any,
) *session {

	if extras == nil {
		extras = map[string]any{}
	}

	return &session{
		id:        id,
		messages:  messages,
		createdAt: createdAt,
		updatedAt: time.Now(),

		inputTokens:  inputTokens,
		outputTokens: outputTokens,

		extras: extras,
	}
}

func (s *session) ID() ID                { return s.id }
func (s *session) Title() string         { return s.title }
func (s *session) SetTitle(title string) { s.title = title }

func (s *session) InputTokens() int64  { return s.inputTokens }
func (s *session) OutputTokens() int64 { return s.outputTokens }

func (s *session) GetLastAgentMessage() *agent.AgentMessage {
	for _, message := range slices.Backward(s.messages) {
		if typedMessage, ok := message.(*agent.AgentMessage); ok {
			return typedMessage
		}
	}
	return nil
}
func (s *session) GetLastUserMessage() *agent.UserMessage {
	for _, message := range slices.Backward(s.messages) {
		if typedMessage, ok := message.(*agent.UserMessage); ok {
			return typedMessage
		}
	}
	return nil
}

func (s *session) Messages() []agent.Message {
	return s.messages
}
func (s *session) AddMessages(newMessages ...agent.Message) {

	if len(newMessages) == 0 {
		return
	}

	if len(s.messages) > 0 {

		firstNewMessage := newMessages[0]

		messagesLastIdx := len(s.messages) - 1
		lastMessage := s.messages[messagesLastIdx]

		// Unite messages for keep order
		if reflect.TypeOf(firstNewMessage) == reflect.TypeOf(lastMessage) {
			switch typed := firstNewMessage.(type) {
			case *agent.UserMessage:
				unitedContent := slices.Concat(lastMessage.Content(), firstNewMessage.Content())
				s.messages[messagesLastIdx] = agent.NewUserMessage(unitedContent)
				newMessages = newMessages[1:]

			case *agent.AgentMessage:

				content := slices.Concat(firstNewMessage.Content(), lastMessage.Content())
				firstMsgCalls := typed.ToolCalls()

				// this way is guaranteed safe cast
				lastMsgCalls := lastMessage.(*agent.AgentMessage).ToolCalls()

				s.messages[messagesLastIdx] = agent.NewAgentMessage(content, slices.Concat(firstMsgCalls, lastMsgCalls))

				newMessages = newMessages[1:]
			}
		}
	}

	s.messages = append(s.messages, newMessages...)
}

func (s *session) ApplyCompletion(completion *agent.Completion) {
	s.inputTokens = completion.InputTokens
	s.outputTokens = completion.CompletionTokens
	s.messages = append(s.messages, agent.NewAgentMessage(completion.Content, completion.ToolCalls))
}

func (s *session) OverwriteMessages(inputTokens int64, new []agent.Message) {
	s.inputTokens = inputTokens
	s.outputTokens = 0
	s.messages = new
}

func (s *session) CreatedAt() time.Time {
	return s.createdAt
}
func (s *session) UpdatedAt() time.Time {
	return s.updatedAt
}

func (s *session) SetExtras(e map[string]any) { s.extras = e }
func (s *session) Extras() map[string]any     { return s.extras }
