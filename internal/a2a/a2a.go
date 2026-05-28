package a2a

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/session"
	"errors"

	"arch-agent/internal/runtime"

	"context"
	"fmt"
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
	repo          ContactRepo
	agentRepo     agent.Repo
	sessionSevice *session.SessionService
	agentRuntime  *runtime.AgentRuntime
	modelRepo     agent.ModelRepository

	callCh chan Call
	respCh chan Response
}

func NewService(
	repo ContactRepo,
	agentRepo agent.Repo,
	agentRuntime *runtime.AgentRuntime,
	modelRepo agent.ModelRepository,
	sessionSvc *session.SessionService,
) *Service {
	return &Service{
		repo:          repo,
		agentRepo:     agentRepo,
		modelRepo:     modelRepo,
		sessionSevice: sessionSvc,
		agentRuntime:  agentRuntime,

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

func (s *Service) Call(
	ctx context.Context,
	callerAgentID,
	recivierAgentID agent.ID,
	sessionID session.ID,
	request string,
) (string, error) {

	// contact, err := s.getContact(callerAgentID, recivierAgentID)
	// if err != nil {
	// 	return "", err
	// }

	agt, err := s.agentRepo.Get(recivierAgentID)
	if err != nil {
		return "", err
	}

	sess, err := s.sessionSevice.Get(callerAgentID, sessionID)
	if err != nil {
		return "", err
	}

	subSessionID, ok := sess.Subsession(recivierAgentID)
	if !ok {
		subSessionID, err = s.sessionSevice.Create(recivierAgentID)
		if err != nil {
			return "", err
		}
		sess.AddSubsession(recivierAgentID, subSessionID)
	}

	subSession, err := s.sessionSevice.Get(recivierAgentID, subSessionID)
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

	model, err := s.modelRepo.Get(agt.Model())
	if err != nil {
		return "", err
	}

	sink := s.agentRuntime.RunStream(ctx, model, agt, agt.Tools(), subSession)

	var errc error
	evReader := runtime.EventReader{
		OnError: func(i1 agent.ID, i2 session.ID, err error) {
			errc = errors.Join(errc, err)
		},
	}
	evReader.Read(sink)

	response := subSession.GetLastAssistantMessageContent()

	select {
	case s.respCh <- Response{
		Call:     call,
		Response: response,
		Err:      errc}:
	default:
	}

	if err := s.sessionSevice.Save(call.Recivier, subSession); err != nil {
		errc = errors.Join(errc, err)
	}

	return response, errc
}

func wrapMessageToPrompt(caller agent.ID, message string) string {
	var sb strings.Builder

	sb.WriteString("Write answer on agent message, all out put will be sended to caller. Do not use use call_agent for answer back.\n\n")
	sb.WriteString("<AgentMessage>\n")
	fmt.Fprintf(&sb, "From: %s \n", string(caller))
	sb.WriteString(message)
	sb.WriteString("\n</AgentMessage>")

	return sb.String()
}
