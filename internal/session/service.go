package session

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/prompt"
	"fmt"
	"time"
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
	repo         SessionsRepo
	uuid         UUIDGenerator
	activityRepo agent.ActivityRepo
}

func NewService(
	repo SessionsRepo,
	uuid UUIDGenerator,
	activityRepo agent.ActivityRepo,
) *Service {
	return &Service{
		repo:         repo,
		uuid:         uuid,
		activityRepo: activityRepo,
	}
}

func (s *Service) Get(agentID agent.ID, id ID) (Session, error) {
	sess, err := s.repo.Session(agentID, id)
	if err != nil {
		return nil, err
	}
	return sess, nil
}

func (s *Service) Create(agentID agent.ID, withLastActivity bool) (ID, error) {
	newSession := NewSession(ID(s.uuid.New()))

	if withLastActivity {
		activity, err := s.activityRepo.GetActivity(agentID, time.Now())
		if err != nil {
			return newSession.ID(), fmt.Errorf("reading activity %w", err)
		}
		newSession.AddSummary(prompt.ActivityExplanation(activity))
	}

	if err := s.repo.Save(agentID, newSession); err != nil {
		return "", err
	}

	return newSession.ID(), nil
}

func (s *Service) Save(agentID agent.ID, sess Session) error {
	return s.repo.Save(agentID, sess)
}

func (s *Service) Delete(agentID agent.ID, sessionID ID) error {
	return s.repo.Delete(agentID, sessionID)
}
