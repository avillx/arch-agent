package session

import (
	"arch-agent/internal/agent"
	"fmt"
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
	GetLastAssistantMessageContent() string
	GetLastUserMessageContent() string

	ApplyCompletion(*agent.Completion)
	AddMessages([]agent.Message)

	Subsessions() map[agent.ID]ID
	AddSubsession(agent.ID, ID)

	InputTokens() int64
	OutputTokens() int64

	AddSummary(string)
	Summary() string

	OverwriteMessages(int64, []agent.Message)

	CreatedAt() time.Time
	UpdatedAt() time.Time
}

type session struct {
	id           ID
	title        string
	inputTokens  int64
	outputTokens int64
	summaries    string
	messages     []agent.Message
	subsessions  map[agent.ID]ID
	createdAt    time.Time
	updatedAt    time.Time
}

func NewSession(id ID) *session {
	return &session{
		id:           id,
		inputTokens:  0,
		outputTokens: 0,
		messages:     []agent.Message{},
		subsessions:  map[agent.ID]ID{},
		createdAt:    time.Now(),
	}
}

func NewRestoredSession(
	id ID,
	messages []agent.Message,
	inputTokens int64,
	outputTokens int64,
	summaries string,
	subsessions map[agent.ID]ID,
	createdAt time.Time,
) *session {
	if subsessions == nil {
		subsessions = map[agent.ID]ID{}
	}
	return &session{
		id:          id,
		messages:    messages,
		subsessions: subsessions,
		summaries:   summaries,
		createdAt:   createdAt,
		updatedAt:   time.Now(),

		inputTokens:  inputTokens,
		outputTokens: outputTokens,
	}
}

func (s *session) ID() ID                { return s.id }
func (s *session) Title() string         { return s.title }
func (s *session) SetTitle(title string) { s.title = title }

func (s *session) InputTokens() int64  { return s.inputTokens }
func (s *session) OutputTokens() int64 { return s.outputTokens }

func (s *session) GetLastAssistantMessageContent() string {
	for _, message := range slices.Backward(s.messages) {
		if typedMessage, ok := message.(*agent.AgentMessage); ok {
			return typedMessage.Content()
		}
	}
	return ""
}
func (s *session) GetLastUserMessageContent() string {
	for _, message := range slices.Backward(s.messages) {
		if typedMessage, ok := message.(*agent.UserMessage); ok {
			return typedMessage.Content()
		}
	}
	return ""
}

func (s *session) AddSubsession(agentID agent.ID, subsessionID ID) {
	s.subsessions[agentID] = subsessionID
}

func (s *session) Subsessions() map[agent.ID]ID {
	return s.subsessions
}

func (s *session) Messages() []agent.Message {
	messages := []agent.Message{}
	return append(messages, s.messages...)
}
func (s *session) AddMessages(msgs []agent.Message) { s.messages = append(s.messages, msgs...) }

func (s *session) ApplyCompletion(completion *agent.Completion) {
	s.inputTokens = completion.InputTokens
	s.outputTokens = completion.CompletionTokens
	s.messages = append(s.messages, agent.NewAgentMessage(completion.Content, completion.ToolCalls))
}

func (s *session) AddSummary(summary string) {
	content := fmt.Sprintf("%s/n/n", summary)
	s.summaries += content
}

func (s *session) Summary() string {
	return s.summaries
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
