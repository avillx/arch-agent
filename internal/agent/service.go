package agent

import (
	"errors"
	"fmt"
)

var _ Repo = (*Service)(nil)

var defaultAgent = NewAgent(
	"default",
	"agent placeholder",
	"",
	"",
	nil,
	false,
)

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
) (*Service, error) {
	svc := &Service{
		toolReg:   toolReg,
		modelRepo: modelRepo,
		storage:   storage,
		syncs:     syncs,
	}

	agts, err := svc.All()
	if err != nil {
		return nil, err
	}

	if len(agts) <= 0 {
		if err := svc.Save(defaultAgent); err != nil {
			return nil, fmt.Errorf("failed to create default agent, %w", err)
		}
	}

	return svc, nil
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
