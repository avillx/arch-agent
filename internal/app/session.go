package service

import (
	"arch-agent/internal/domain/agent"
	"arch-agent/internal/domain/session"
	"arch-agent/internal/domain/types"
)

type SessionsRepo interface {
	List(agent.ID) ([]*session.ID, error)
	Session(SessionID session.ID) (*session.Session, error)
	Save(Session *session.Session) error
	Delete(SessionID session.ID) error
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

func (s *SessionService) Get(id session.ID) (*session.Session, error) {
	sess, err := s.repo.Session(id)
	if err != nil {
		return nil, err
	}
	return sess, nil
}

func (s *SessionService) Create() *session.Session {
	return session.NewSession(s.uuid.New())
}

func (s *SessionService) AppendMessages(sess *session.Session, msgs []types.Message) error {
	sess.AddMessages(s.tokenCounter, msgs)
	return s.repo.Save(sess)
}

func (s *SessionService) Save(sess *session.Session) error {
	return s.repo.Save(sess)
}
