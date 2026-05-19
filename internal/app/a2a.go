package service

import (
	"arch-agent/internal/domain/agent"
	"arch-agent/internal/domain/session"
	"arch-agent/internal/domain/types"
	"context"
	"fmt"
	"log/slog"
	"strings"
)

type A2ACallType string

const (
	OneCall         A2ACallType = "one_call"
	SubSessionCall  A2ACallType = "sub_session"
	LiveSessionCall A2ACallType = "live_session"
)

type A2AContactRepo interface {
	Get(agentID agent.ID) ([]*A2AContact, error)
	Save(agentID agent.ID, newContact *A2AContact) error
	Delete(agentID agent.ID, contactAgentID agent.ID) error
}

type A2AContact struct {
	ID        agent.ID
	CallGuide string
	CallType  A2ACallType
}

type A2ACall struct {
	Caller   agent.ID
	Recivier agent.ID
	Request  string
}

type A2AResponse struct {
	A2ACall
	Response string
	Err      error
}

type A2AService struct {
	repo     A2AContactRepo
	resolver *A2ACallResolver

	callCh chan A2ACall
	respCh chan A2AResponse
}

func NewA2AService(
	repo A2AContactRepo,
	agentService *AgentService,
	sessionChatService *SessionChatService,
	liveChatService *LiveChatService,
) *A2AService {
	return &A2AService{
		repo: repo,
		resolver: &A2ACallResolver{
			agentService:       agentService,
			sessionChatService: sessionChatService,
			liveChatService:    liveChatService,
		},
		callCh: make(chan A2ACall, 16),
		respCh: make(chan A2AResponse, 16),
	}
}

func (s *A2AService) CallChannel() <-chan A2ACall {
	return s.callCh
}

func (s *A2AService) ResponseChannel() <-chan A2AResponse {
	return s.respCh
}

func (s *A2AService) AgentContacts(agnetID agent.ID) ([]*A2AContact, error) {
	return s.repo.Get(agnetID)
}

func (s *A2AService) getContact(callerAgentID agent.ID, recivierAgent agent.ID) (*A2AContact, error) {

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

func (s *A2AService) Call(ctx context.Context, callerAgentID, recivierAgentID agent.ID, request string) (string, error) {

	contact, err := s.getContact(callerAgentID, recivierAgentID)
	if err != nil {
		return "", err
	}

	call := A2ACall{
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
	case s.respCh <- A2AResponse{
		A2ACall:  call,
		Response: response,
		Err:      err}:
	default:
	}

	return response, err
}

type A2ACallResolver struct {
	agentService       *AgentService
	sessionChatService *SessionChatService
	liveChatService    *LiveChatService
}

func (s *A2ACallResolver) Resolve(ctx context.Context, callerAgentID agent.ID, contact *A2AContact, request string) (string, error) {

	switch contact.CallType {
	case SubSessionCall:
		return s.subSessionCall(ctx, callerAgentID, contact, request)
	case LiveSessionCall:
		return s.liveSessionCall(ctx, callerAgentID, contact, request)
	default:
		return s.oneCall(ctx, callerAgentID, contact, request)
	}
}

func (s *A2ACallResolver) oneCall(ctx context.Context, callerAgentID agent.ID, contact *A2AContact, request string) (string, error) {
	resContent := []string{}
	if _, err := s.agentService.Chat(
		ctx,
		contact.ID,
		"",
		nil,
		nil,
		[]types.Message{types.NewUserMessage(
			wrapMessageToPrompt(callerAgentID, request),
		)},
		func(result *agent.ReasonResult) {
			resContent = append(resContent, result.Content)
		},
	); err != nil {
		return "", err
	}

	return strings.Join(resContent, "\n"), nil
}

func (s *A2ACallResolver) subSessionCall(ctx context.Context, callerAgentID agent.ID, contact *A2AContact, request string) (string, error) {

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
func (s *A2ACallResolver) liveSessionCall(ctx context.Context, callerAgentID agent.ID, contact *A2AContact, request string) (string, error) {

	resContent := []string{}
	if err := s.liveChatService.Chat(
		ctx,
		contact.ID,
		wrapMessageToPrompt(callerAgentID, request),
		func(result *agent.ReasonResult) {
			resContent = append(resContent, result.Content)
		},
	); err != nil {
		return "", err
	}

	return strings.Join(resContent, "\n"), nil
}

func wrapMessageToPrompt(caller agent.ID, message string) string {
	var sb strings.Builder

	sb.WriteString("Write answer on agent message, all out put will be sended to caller. Do not use use call_agent for answer back.\n\n")
	sb.WriteString("<AgentMessage>\n")
	sb.WriteString("From:" + string(caller) + "\n")
	sb.WriteString(message)
	sb.WriteString("\n</AgentMessage>")

	return sb.String()
}
