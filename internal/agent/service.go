package agent

import (
	"errors"
)

var _ Repo = (*Service)(nil)

type AgentSync interface {
	DeleteAgent(ID) error
}

type Service struct {
	toolReg   ToolRegistry
	modelRepo ModelRegistry
	storage   Repo
	syncs     []AgentSync
}

func NewService(
	toolReg ToolRegistry,
	modelRepo ModelRegistry,
	storage Repo,
	syncs []AgentSync,
) *Service {
	return &Service{
		toolReg:   toolReg,
		modelRepo: modelRepo,
		storage:   storage,
		syncs:     syncs,
	}
}

func (s *Service) All() ([]Agent, error) {
	return s.storage.All()
}

func (s *Service) Get(agentID ID) (Agent, error) {
	return s.storage.Get(agentID)
}

func (s *Service) Save(agt Agent) error {

	// validate model existence
	if _, err := s.modelRepo.Get(agt.Model()); err != nil {
		return err
	}

	// validate tool servers existence
	if _, err := s.toolReg.ToolServers(agt.ToolServers()...); err != nil {
		return err
	}

	return s.storage.Save(agt)
}

func (s *Service) Delete(agentID ID) error {

	// validate agent existence
	if _, err := s.storage.Get(agentID); err != nil {
		return err
	}

	if err := s.storage.Delete(agentID); err != nil {
		return err
	}

	var errs []error
	for _, syncr := range s.syncs {
		err := syncr.DeleteAgent(agentID)
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}
