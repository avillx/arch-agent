package llm

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/types"
	"fmt"
)

type LLMSettings map[string]any

type LLM interface {
	Settings() LLMSettings
	SetSettings(LLMSettings) error
	agent.Reasoner
}

type LLMSettingsRepo interface {
	Load() (map[agent.LLMID]LLMSettings, error)
	Save(agent.LLMID, LLMSettings) error
	Delete(agent.LLMID) error
}

type LLMFactory interface {
	Type() string
	Produce(LLMSettings) (LLM, error)
}

type Service struct {
	repo      LLMSettingsRepo
	llms      map[agent.LLMID]LLM
	factories map[string]LLMFactory
}

func NewLLMService(repo LLMSettingsRepo, factories ...LLMFactory) (*Service, error) {

	llmService := &Service{
		repo:      repo,
		llms:      map[agent.LLMID]LLM{},
		factories: map[string]LLMFactory{},
	}

	for _, factory := range factories {
		llmService.factories[factory.Type()] = factory
	}

	llmSettings, err := repo.Load()
	if err != nil {
		return nil, fmt.Errorf("list llms: %w", err)
	}

	for id, settings := range llmSettings {
		if err := llmService.createLLM(id, settings); err != nil {
			return nil, err
		}
	}

	return llmService, nil
}

func (s *Service) SetLLMSettings(id agent.LLMID, settings LLMSettings) error {

	llm, ok := s.llms[id]
	if !ok {
		return types.ErrIsNotExist
	}

	//TODO type changing edge case
	// exis
	// typeOfExisting, err := extractString(, "type")
	// typeOfNew, _ := extractString(llm.Settings(), "type")
	// if typeOfNew != typeOfExisting {
	// 	return
	// }

	if err := llm.SetSettings(settings); err != nil {
		return err
	}

	return s.repo.Save(id, settings)
}

func (s *Service) GetLLM(id agent.LLMID) (agent.Reasoner, error) {
	llm, ok := s.llms[id]
	if !ok {
		return nil, types.ErrIsNotExist
	}

	return llm, nil
}

func (s *Service) AddNewLLM(id agent.LLMID, settings LLMSettings) error {

	if _, exists := s.llms[id]; exists {
		return fmt.Errorf("llm %q already exists", id)
	}

	if err := s.createLLM(id, settings); err != nil {
		return err
	}

	return s.repo.Save(id, settings)
}

func (s *Service) createLLM(id agent.LLMID, settings LLMSettings) error {

	llmType, err := extractString(settings, "type")
	if err != nil {
		return err
	}

	factory, ok := s.factories[llmType]
	if !ok {
		return fmt.Errorf("unknown llm type %q for id %q", llmType, id)
	}

	llm, err := factory.Produce(settings)
	if err != nil {
		return fmt.Errorf("produce llm %q: %w", id, err)
	}

	s.llms[id] = llm

	return nil
}

func (s *Service) DeleteLLM(id agent.LLMID) error {
	if _, ok := s.llms[id]; !ok {
		return fmt.Errorf("llm %q not found", id)
	}
	delete(s.llms, id)
	return s.repo.Delete(id)
}

func extractString(s LLMSettings, str string) (string, error) {
	v, ok := s[str]
	if !ok {
		return "", fmt.Errorf("settings has no %s", str)
	}

	extracted, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("%s, must be string", str)
	}
	return extracted, nil
}
