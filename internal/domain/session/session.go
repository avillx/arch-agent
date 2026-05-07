package session

import (
	"arch-agent/internal/domain/types"
	"slices"
	"strings"
)

const TokenLimit = 20000

type ID string

type Session struct {
	ID          ID
	Tokens      int
	messages    []types.Message
	subsessions map[string]*Session
}

func NewSession(id string) *Session {
	return &Session{
		ID:          ID(id),
		Tokens:      0,
		messages:    []types.Message{},
		subsessions: map[string]*Session{},
	}
}

func NewRestoredSession(tokens int, messages []types.Message, subsessions map[string]*Session) *Session {
	return &Session{
		Tokens:      tokens,
		messages:    messages,
		subsessions: subsessions,
	}
}

func (s *Session) Subsession(key string) *Session {
	subsession, ok := s.subsessions[key]
	if !ok {
		return nil
	}
	return subsession
}

func (s *Session) AddSubsession(key string, subsession *Session) {
	s.subsessions[key] = subsession
}

func (s *Session) Messages() []types.Message { return s.messages }

func (s *Session) AddMessages(counter TokenCounter, msgs []types.Message) {
	s.Tokens += messagesTokens(counter, msgs)
	s.messages = slices.Concat(s.messages, msgs)
}

func (s *Session) OverwriteMessages(new []types.Message) {
	s.messages = new
}

func (s *Session) IsOverflow() bool {
	return s.Tokens >= TokenLimit
}

func messagesTokens(counter TokenCounter, msgs []types.Message) int {
	// TODO check complexity
	var heap strings.Builder
	for _, msg := range msgs {
		heap.WriteString(string(msg.Role()))
		heap.WriteString(msg.Content())

		switch typedMsg := msg.(type) {
		case types.AgentMessage:
			for _, call := range typedMsg.ToolCalls() {
				heap.WriteString(call.ID)
				heap.WriteString(call.ToolName)
				heap.Write(call.Arguments)
			}
		case types.ToolResultMessage:
			heap.WriteString(typedMsg.ToolCallID())
		}
	}

	return counter.Calc(heap.String())
}

type TokenCounter interface {
	Calc(string) int
}
