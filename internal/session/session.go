package session

import (
	"arch-agent/internal/agent"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"
)

const TokenLimit = 20000

// validate struct asn iface
var _ Session = (*session)(nil)

type TokenCounter interface {
	Calc(string) int
}

type ID string

type Session interface {
	ID() ID
	// Title() string
	// SetTitle(title string)

	Messages() []agent.Message
	GetLastAssistantMessageContent() string
	GetLastUserMessageContent() string
	AddMessages([]agent.Message)

	Subsession(agent.ID) (ID, bool)
	AddSubsession(agent.ID, ID)

	Tokens() int
	MessageTokens() int
	SummaryTokens() int

	AddSummary(string)
	Summary() string

	OverwriteMessages(new []agent.Message)

	CreatedAt() time.Time
	UpdatedAt() time.Time
}

type session struct {
	id            ID
	title         string
	tokens        int
	summaryTokens int
	summaries     string
	messages      []agent.Message
	subsessions   map[agent.ID]ID
	tokenCounter  TokenCounter
	createdAt     time.Time
	updatedAt     time.Time
}

func NewSession(id ID, tokenCounter TokenCounter) *session {
	return &session{
		id:           id,
		tokens:       0,
		messages:     []agent.Message{},
		subsessions:  map[agent.ID]ID{},
		tokenCounter: tokenCounter,
		createdAt:    time.Now(),
	}
}

func NewRestoredSession(
	id ID,
	tokens int,
	summaryTokens int,
	messages []agent.Message,
	tokenCounter TokenCounter,
	summaries string,
	subsessions map[agent.ID]ID,
	createdAt time.Time,
) *session {
	if subsessions == nil {
		subsessions = map[agent.ID]ID{}
	}

	return &session{
		id:            id,
		tokens:        tokens,
		summaryTokens: summaryTokens,
		messages:      messages,
		subsessions:   subsessions,
		tokenCounter:  tokenCounter,
		summaries:     summaries,
		createdAt:     createdAt,
		updatedAt:     time.Now(),
	}
}

func (s *session) ID() ID                { return s.id }
func (s *session) Title() string         { return s.title }
func (s *session) SetTitle(title string) { s.title = title }

func (s *session) Tokens() int        { return s.tokens + s.summaryTokens }
func (s *session) MessageTokens() int { return s.tokens }
func (s *session) SummaryTokens() int { return s.summaryTokens }

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

func (s *session) Subsession(agentID agent.ID) (ID, bool) {
	subsession, ok := s.subsessions[agentID]
	return subsession, ok
}

func (s *session) Messages() []agent.Message {
	messages := []agent.Message{}
	return append(messages, s.messages...)
}

func (s *session) AddMessages(msgs []agent.Message) {
	s.tokens += messagesTokens(s.tokenCounter, msgs)
	s.messages = slices.Concat(s.messages, msgs)
}

func (s *session) AddSummary(summary string) {
	content := fmt.Sprintf("%s/n/n", summary)
	s.summaryTokens += s.tokenCounter.Calc(content)
	s.summaries += content
}

func (s *session) Summary() string {
	return s.summaries
}

func (s *session) OverwriteMessages(new []agent.Message) {
	s.tokens = messagesTokens(s.tokenCounter, new)
	s.tokens += s.tokenCounter.Calc(s.summaries)
	s.messages = new
}

func (s *session) CreatedAt() time.Time {
	return s.createdAt
}
func (s *session) UpdatedAt() time.Time {
	return s.updatedAt
}

func messagesTokens(counter TokenCounter, msgs []agent.Message) int {

	if counter == nil {
		slog.Warn("session without token counters")
		return 0
	}

	// TODO check complexity
	var heap strings.Builder
	for _, msg := range msgs {
		heap.WriteString(msg.Content())

		switch typedMsg := msg.(type) {
		case agent.AgentMessage:
			for _, call := range typedMsg.ToolCalls() {
				heap.WriteString(call.ID)
				heap.WriteString(call.ToolName)
				heap.Write(call.Arguments)
			}
		case agent.ToolResultMessage:
			heap.WriteString(typedMsg.ToolCallID())
		}
	}

	return counter.Calc(heap.String())
}
