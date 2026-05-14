package service

import (
	"arch-agent/internal/domain/agent"
	"arch-agent/internal/domain/session"
)

const SessionContextKey = "session"
const AgentContextKey = "agent"

type SessionsRepo interface {
	List(agent.ID) ([]session.ID, error)
	Session(agentID agent.ID, SessionID session.ID) (*session.Session, error)
	Save(agentID agent.ID, Session *session.Session) error
	Delete(agentID agent.ID, SessionID session.ID) error
}

type UUIDGenerator interface {
	New() string
}

type SessionService struct {
	repo         SessionsRepo
	uuid         UUIDGenerator
	tokenCounter session.TokenCounter
}

func NewSessionService(repo SessionsRepo, uuid UUIDGenerator, tokenCounter session.TokenCounter) *SessionService {
	return &SessionService{
		repo:         repo,
		uuid:         uuid,
		tokenCounter: tokenCounter,
	}
}

func (s *SessionService) Get(agentID agent.ID, id session.ID) (*session.Session, error) {
	sess, err := s.repo.Session(agentID, id)
	if err != nil {
		return nil, err
	}
	return sess, nil
}

func (s *SessionService) Create(agentID agent.ID) (session.ID, error) {
	newSession := session.NewSession(s.uuid.New(), s.tokenCounter)
	if err := s.repo.Save(agentID, newSession); err != nil {
		return "", err
	}
	return newSession.ID, nil
}

func (s *SessionService) Save(agentID agent.ID, sess *session.Session) error {
	return s.repo.Save(agentID, sess)
}

func (s *SessionService) Delete(agentID agent.ID, sessionID session.ID) error {
	return s.repo.Delete(agentID, sessionID)
}
