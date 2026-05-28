package session

import "arch-agent/internal/agent"

type SessionsRepo interface {
	List(agent.ID) ([]ID, error)
	Session(agentID agent.ID, SessionID ID) (Session, error)
	Save(agentID agent.ID, Session Session) error
	Delete(agentID agent.ID, SessionID ID) error
}

type UUIDGenerator interface {
	New() string
}

type SessionService struct {
	repo         SessionsRepo
	uuid         UUIDGenerator
	tokenCounter TokenCounter
}

func NewSessionService(repo SessionsRepo, uuid UUIDGenerator, tokenCounter TokenCounter) *SessionService {
	return &SessionService{
		repo:         repo,
		uuid:         uuid,
		tokenCounter: tokenCounter,
	}
}

func (s *SessionService) Get(agentID agent.ID, id ID) (Session, error) {
	sess, err := s.repo.Session(agentID, id)
	if err != nil {
		return nil, err
	}
	return sess, nil
}

func (s *SessionService) Create(agentID agent.ID) (ID, error) {
	newSession := NewSession(ID(s.uuid.New()), s.tokenCounter)
	if err := s.repo.Save(agentID, newSession); err != nil {
		return "", err
	}
	return newSession.ID(), nil
}

func (s *SessionService) Save(agentID agent.ID, sess Session) error {
	return s.repo.Save(agentID, sess)
}

func (s *SessionService) Delete(agentID agent.ID, sessionID ID) error {
	return s.repo.Delete(agentID, sessionID)
}
