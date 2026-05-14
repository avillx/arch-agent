package session

import (
	"arch-agent/internal/domain/types"
	"fmt"
	"slices"
	"strings"
)

const TokenLimit = 20000

type OverflowPolicy struct {
	TokenLimit int
	OnOverflow func(*Session) error
}

type ID string

type TokenCounter interface {
	Calc(string) int
}

type Session struct {
	ID           ID
	Tokens       int
	summaries    string
	messages     []types.Message
	subsessions  map[string]ID
	tokenCounter TokenCounter
}

// TODO: session should not exist without token counter
// implement a DI factrory in repo
func NewSession(id string, tokenCounter TokenCounter) *Session {
	return &Session{
		ID:           ID(id),
		Tokens:       0,
		messages:     []types.Message{},
		subsessions:  map[string]ID{},
		tokenCounter: tokenCounter,
	}
}

func NewRestoredSession(id ID, tokens int, messages []types.Message, tokenCounter TokenCounter, summaries string, subsessions map[string]ID) *Session {
	if subsessions == nil {
		subsessions = map[string]ID{}
	}

	return &Session{
		ID:           id,
		Tokens:       tokens,
		messages:     messages,
		subsessions:  subsessions,
		tokenCounter: tokenCounter,
		summaries:    summaries,
	}
}

func (s *Session) Subsession(key string) ID {
	subsession, ok := s.subsessions[key]
	if !ok {
		return ""
	}
	return subsession
}

func (s *Session) AddSubsession(key string, subsession ID) {
	s.subsessions[key] = subsession
}

func (s *Session) Messages() []types.Message {
	messages := []types.Message{}
	return append(messages, s.messages...)
}

func (s *Session) AddMessages(msgs []types.Message) {
	s.Tokens += messagesTokens(s.tokenCounter, msgs)
	s.messages = slices.Concat(s.messages, msgs)
}

func (s *Session) AddSummary(summary string) {
	content := fmt.Sprintf("%s/n/n", summary)
	s.Tokens += s.tokenCounter.Calc(content)
	s.summaries += content
}

func (s *Session) Summaries() string {
	return s.summaries
}

func (s *Session) OverwriteMessages(new []types.Message) {
	s.messages = new
}

func (s *Session) ProcessOverflow(policy OverflowPolicy) error {
	if policy.TokenLimit < s.Tokens {
		return policy.OnOverflow(s)
	}

	return nil
}

func messagesTokens(counter TokenCounter, msgs []types.Message) int {
	// TODO check complexity
	var heap strings.Builder
	for _, msg := range msgs {
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
