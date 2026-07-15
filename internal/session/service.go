package session

import (
	"arch-agent/internal/agent"
)

type SessionsRepo interface {
	List(agent.ID) ([]ID, error)
	Session(agentID agent.ID, SessionID ID) (Session, error)
	Save(agentID agent.ID, Session Session) error
	Delete(agentID agent.ID, SessionID ID) error
}

type UUIDGenerator interface {
	New() string
}

type Service struct {
	repo SessionsRepo
	uuid UUIDGenerator
}

func NewService(
	repo SessionsRepo,
	uuid UUIDGenerator,
) *Service {
	return &Service{
		repo: repo,
		uuid: uuid,
	}
}

func (s *Service) Create(agentID agent.ID) (ID, error) {
	newSession := NewSession(ID(s.uuid.New()))

	if err := s.repo.Save(agentID, newSession); err != nil {
		return "", err
	}

	return newSession.ID(), nil
}

func (s *Service) Get(agentID agent.ID, id ID) (Session, error) {
	return s.repo.Session(agentID, id)
}

func (s *Service) Save(agentID agent.ID, sess Session) error {
	return s.repo.Save(agentID, sess)
}

func (s *Service) Delete(agentID agent.ID, sessionID ID) error {
	return s.repo.Delete(agentID, sessionID)
}

func (s *Service) List(agentID agent.ID) ([]ID, error) {
	return s.repo.List(agentID)
}
