package session

import (
	"arch-agent/internal/app/types"
	"slices"
)

const TokenLimit = 10000

type Session struct {
	Tokens   int
	messages []types.Message
}

func NewSession() *Session {
	return &Session{
		Tokens:   0,
		messages: []types.Message{},
	}
}

func NewRestoredSession(tokens int, messages []types.Message) *Session {
	return &Session{
		Tokens:   tokens,
		messages: messages,
	}
}

func (s *Session) Messages() []types.Message { return s.messages }

func (s *Session) AddToolCallResults(results []types.ToolCallResult) {
	for _, r := range results {
		s.messages = append(s.messages, types.NewToolResultMessage(r.ID, r.Result))
	}
}

func (s *Session) AddToolResultMessage(id string, content string) {
	s.messages = append(s.messages, types.NewToolResultMessage(id, content))
}

func (s *Session) AddUserMessage(content string) {
	s.messages = append(s.messages, types.NewUserMessage(content))
}

func (s *Session) AddAgentMessage(content string, tc []*types.ToolCall) {
	s.messages = append(s.messages, types.NewAgentMessage(content, tc))
}

func (s *Session) AddMessages(msgs []types.Message) {
	s.messages = slices.Concat(s.messages, msgs)
}

func (s *Session) OverwriteMessages(new []types.Message) {
	s.messages = new
}

func (s *Session) IsOverflow() bool {
	return s.Tokens >= TokenLimit
}
