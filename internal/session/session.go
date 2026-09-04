package session

import (
	"arch-agent/internal/agent"
	"fmt"
	"reflect"
	"slices"
	"time"
)

const TokenLimit = 20000

var _ error = (*ErrBrokenHeaders)(nil)

type ErrBrokenHeaders struct {
	Errors []*ErrBrokenHeader
}

func NewErrBrokenHeaders(errs ...*ErrBrokenHeader) *ErrBrokenHeaders {
	return &ErrBrokenHeaders{
		Errors: errs,
	}
}

func (e *ErrBrokenHeaders) Error() string {
	return fmt.Sprintf("occured %d broken session headers", len(e.Errors))
}

var _ error = (*ErrBrokenHeader)(nil)

type ErrBrokenHeader struct {
	SessID  ID
	AgentID agent.ID
	cause   error
}

func NewErrBrokenHeader(
	sessID ID,
	agentID agent.ID,
	cause error,
) *ErrBrokenHeader {
	return &ErrBrokenHeader{
		SessID:  sessID,
		AgentID: agentID,
		cause:   cause,
	}
}

func (e *ErrBrokenHeader) Unwrap() error {
	return e.cause
}

func (e *ErrBrokenHeader) Error() string {
	return fmt.Sprintf("session header is borken. agent: %s, session: %s", e.AgentID, e.SessID)
}

type ID string

var _ SessionHeader = (*sessionHeader)(nil)

type SessionHeader interface {
	ID() ID

	InputTokens() int64
	OutputTokens() int64

	CreatedAt() time.Time
	UpdatedAt() time.Time

	Extras() map[string]any
}

type Session interface {
	SessionHeader

	Messages() []agent.Message
	GetLastAgentMessage() *agent.AgentMessage
	GetLastUserMessage() *agent.UserMessage

	ApplyCompletion(*agent.Completion)
	AddMessages(...agent.Message)

	OverwriteMessages(int64, []agent.Message)

	SetExtras(map[string]any)
}

type sessionHeader struct {
	id           ID
	inputTokens  int64
	outputTokens int64
	createdAt    time.Time
	updatedAt    time.Time

	extras map[string]any
}

func (s *sessionHeader) ID() ID              { return s.id }
func (s *sessionHeader) InputTokens() int64  { return s.inputTokens }
func (s *sessionHeader) OutputTokens() int64 { return s.outputTokens }
func (s *sessionHeader) CreatedAt() time.Time {
	return s.createdAt
}
func (s *sessionHeader) UpdatedAt() time.Time {
	return s.updatedAt
}
func (s *sessionHeader) Extras() map[string]any { return s.extras }

func NewHeader(
	id ID,
	inputTokens int64,
	outputTokens int64,
	createdAt time.Time,
	updatedAt time.Time,
	extras map[string]any,
) *sessionHeader {
	return &sessionHeader{
		id:           id,
		inputTokens:  inputTokens,
		outputTokens: outputTokens,
		createdAt:    createdAt,
		updatedAt:    updatedAt,
		extras:       extras,
	}
}

var _ Session = (*session)(nil)

type session struct {
	*sessionHeader
	messages []agent.Message
}

func NewSession(id ID) *session {
	return &session{
		sessionHeader: NewHeader(id, 0, 0, time.Now(), time.Now(), map[string]any{}),
		messages:      []agent.Message{},
	}
}

func NewRestoredSession(
	h *sessionHeader,
	messages []agent.Message,
) *session {

	return &session{
		sessionHeader: h,
		messages:      messages,
	}
}

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

func (s *session) SetExtras(e map[string]any) { s.extras = e }
