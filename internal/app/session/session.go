package session

import "arch-agent/internal/app/message"

const TokenLimit = 10000

type Session struct {
	id       string
	Tokens   int
	messages []message.Message
}

func NewSession(id string, tokens int, messages []message.Message) *Session {
	return &Session{
		id:       id,
		Tokens:   tokens,
		messages: messages,
	}
}

func (s *Session) ID() string                  { return s.id }
func (s *Session) Messages() []message.Message { return s.messages }

func (s *Session) AddToolCallResults(results []message.ToolCallResult) {
	for _, r := range results {
		s.messages = append(s.messages, message.NewToolResultMessage(r.ID, r.Result))
	}
}

func (s *Session) AddToolResultMessage(id string, content string) {
	s.messages = append(s.messages, message.NewToolResultMessage(id, content))
}

func (s *Session) AddUserMessage(content string) {
	s.messages = append(s.messages, message.NewUserMessage(content))
}

func (s *Session) AddAgentMessage(content string, tc []*message.ToolCall) {
	s.messages = append(s.messages, message.NewAgentMessage(content, tc))
}

func (s *Session) Reduce(tail []message.Message) {
	s.messages = tail
}

func (s *Session) IsOverflow() bool {
	return s.Tokens >= TokenLimit
}
