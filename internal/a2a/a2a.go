package a2a

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/chat"
	"arch-agent/internal/session"
	"context"
	"fmt"
	"log/slog"
	"strings"
)

type CallType string

const (
	OneCall         CallType = "one_call"
	SubSessionCall  CallType = "sub_session"
	LiveSessionCall CallType = "live_session"
)

type ContactRepo interface {
	Get(agentID agent.ID) ([]*Contact, error)
	Save(agentID agent.ID, newContact *Contact) error
	Delete(agentID agent.ID, contactAgentID agent.ID) error
}

type Contact struct {
	ID        agent.ID
	CallGuide string
	CallType  CallType
}

type Call struct {
	Caller   agent.ID
	Recivier agent.ID
	Request  string
}

type Response struct {
	Call
	Response string
	Err      error
}

type Service struct {
	repo     ContactRepo
	resolver *CallResolver

	callCh chan Call
	respCh chan Response
}

func NewService(
	repo ContactRepo,
	agentService *chat.Service,
	sessionChatService *session.SessionChatService,
	// liveChatService *LiveChatService,
) *Service {
	return &Service{
		repo: repo,
		resolver: &CallResolver{
			agentService:       agentService,
			sessionChatService: sessionChatService,
			// liveChatService:    liveChatService,
		},
		callCh: make(chan Call, 16),
		respCh: make(chan Response, 16),
	}
}

func (s *Service) CallChannel() <-chan Call {
	return s.callCh
}

func (s *Service) ResponseChannel() <-chan Response {
	return s.respCh
}

func (s *Service) AgentContacts(agnetID agent.ID) ([]*Contact, error) {
	return s.repo.Get(agnetID)
}

func (s *Service) getContact(callerAgentID agent.ID, recivierAgent agent.ID) (*Contact, error) {

	contacts, err := s.repo.Get(callerAgentID)
	if err != nil {
		return nil, err
	}

	for _, contact := range contacts {
		if contact.ID == recivierAgent {
			return contact, nil
		}
	}

	return nil, fmt.Errorf("agent %s has no contact of %s", callerAgentID, recivierAgent)

}

func (s *Service) Call(ctx context.Context, callerAgentID, recivierAgentID agent.ID, request string) (string, error) {

	contact, err := s.getContact(callerAgentID, recivierAgentID)
	if err != nil {
		return "", err
	}

	call := Call{
		Caller:   callerAgentID,
		Recivier: recivierAgentID,
		Request:  request,
	}

	select {
	case s.callCh <- call:
	default:
	}

	response, err := s.resolver.Resolve(ctx, callerAgentID, contact, request)

	select {
	case s.respCh <- Response{
		Call:     call,
		Response: response,
		Err:      err}:
	default:
	}

	return response, err
}

type CallResolver struct {
	agentService       *chat.Service
	sessionChatService *session.SessionChatService
	// liveChatService    *LiveChatService
}

func (s *CallResolver) Resolve(ctx context.Context, callerAgentID agent.ID, contact *Contact, request string) (string, error) {

	switch contact.CallType {
	case SubSessionCall:
		return s.subSessionCall(ctx, callerAgentID, contact, request)
	// case LiveSessionCall:
	// 	return s.liveSessionCall(ctx, callerAgentID, contact, request)
	default:
		return s.oneCall(ctx, callerAgentID, contact, request)
	}
}

func (s *CallResolver) oneCall(ctx context.Context, callerAgentID agent.ID, contact *Contact, request string) (string, error) {
	resContent := []string{}
	if _, err := s.agentService.Chat(
		ctx,
		contact.ID,
		"", // TODO: a2a answer prompt
		[]agent.Message{agent.NewUserMessage(
			wrapMessageToPrompt(callerAgentID, request),
		)},
		func(result *agent.ReasonResult) {
			resContent = append(resContent, result.Content)
		},
		nil,
	); err != nil {
		return "", err
	}

	return strings.Join(resContent, "\n"), nil
}

func (s *CallResolver) subSessionCall(ctx context.Context, callerAgentID agent.ID, contact *Contact, request string) (string, error) {

	sessionID, ok := ctx.Value("sessionID").(session.ID)
	if !ok {
		slog.Warn("sub session call without session, had redirected to one call", "caller", callerAgentID, "contact", contact.ID)
		return s.oneCall(ctx, callerAgentID, contact, request)
	}

	// TODO bug, need to create a new session, now is caller session as stub
	resContent := []string{}
	if err := s.sessionChatService.SessionChat(
		ctx,
		contact.ID,
		sessionID,
		"",
		"",
		wrapMessageToPrompt(callerAgentID, request),
		func(result *agent.ReasonResult) {
			resContent = append(resContent, result.Content)
		},
	); err != nil {
		return "", err
	}

	return strings.Join(resContent, "\n"), nil

}

// func (s *CallResolver) liveSessionCall(ctx context.Context, callerAgentID agent.ID, contact *A2AContact, request string) (string, error) {

// 	resContent := []string{}
// 	if err := s.liveChatService.Chat(
// 		ctx,
// 		contact.ID,
// 		wrapMessageToPrompt(callerAgentID, request),
// 		func(result *agent.ReasonResult) {
// 			resContent = append(resContent, result.Content)
// 		},
// 	); err != nil {
// 		return "", err
// 	}

// 	return strings.Join(resContent, "\n"), nil
// }

func wrapMessageToPrompt(caller agent.ID, message string) string {
	var sb strings.Builder

	sb.WriteString("Write answer on agent message, all out put will be sended to caller. Do not use use call_agent for answer back.\n\n")
	sb.WriteString("<AgentMessage>\n")
	sb.WriteString("From:" + string(caller) + "\n")
	sb.WriteString(message)
	sb.WriteString("\n</AgentMessage>")

	return sb.String()
}
