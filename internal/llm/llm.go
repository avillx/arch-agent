package llm

import (
	"arch-agent/internal/agent"
	"fmt"
)

type LLMID string
type LLMSettings map[string]any

type LLM interface {
	Settings() LLMSettings
	SetSettings(LLMSettings) error
	agent.Reasoner
}

type LLMSettingsRepo interface {
	Load() (map[LLMID]LLMSettings, error)
	Save(LLMID, LLMSettings) error
	Delete(LLMID) error
}

type LLMFactory interface {
	Type() string
	Produce(LLMSettings) (LLM, error)
}

type Service struct {
	repo      LLMSettingsRepo
	llms      map[LLMID]LLM
	factories map[string]LLMFactory
}

func NewLLMService(repo LLMSettingsRepo, factories ...LLMFactory) (*Service, error) {

	llmService := &Service{
		repo:      repo,
		llms:      map[LLMID]LLM{},
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

func (s *Service) SetLLMSettings(id LLMID, settings LLMSettings) error {

	llm, err := s.GetLLM(id)
	if err != nil {
		return err
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

func (s *Service) GetLLM(id LLMID) (LLM, error) {
	llm, ok := s.llms[id]
	if !ok {
		return nil, fmt.Errorf("llm %s is not exist", id)
	}

	return llm, nil
}

func (s *Service) AddNewLLM(id LLMID, settings LLMSettings) error {

	if _, exists := s.llms[id]; exists {
		return fmt.Errorf("llm %q already exists", id)
	}

	if err := s.createLLM(id, settings); err != nil {
		return err
	}

	return s.repo.Save(id, settings)
}

func (s *Service) createLLM(id LLMID, settings LLMSettings) error {

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

func (s *Service) DeleteLLM(id LLMID) error {
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
