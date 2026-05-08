package service

import (
	"arch-agent/internal/domain/agent"
	"arch-agent/internal/domain/session"
	"arch-agent/internal/domain/types"
)

type SessionsRepo interface {
	List(agent.ID) ([]*session.ID, error)
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

func (s *SessionService) Create() *session.Session {
	return session.NewSession(s.uuid.New())
}

func (s *SessionService) AppendMessages(agentID agent.ID, sess *session.Session, msgs []types.Message) error {
	sess.AddMessages(s.tokenCounter, msgs)
	return s.repo.Save(agentID, sess)
}

func (s *SessionService) Save(agentID agent.ID, sess *session.Session) error {
	return s.repo.Save(agentID, sess)
}
