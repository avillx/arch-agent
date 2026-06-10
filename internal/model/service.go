package model

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/types"
	"errors"
	"fmt"
	"sync"
)

type Service struct {
	settingsRepo ModelSettingsRepo
	factories    map[APIType]ModelFactory

	mu         sync.RWMutex
	config     ModelsConfig
	modelCache map[agent.ModelID]agent.Model
}

func NewService(settingsRepo ModelSettingsRepo, factories ...ModelFactory) (*Service, error) {
	svc := &Service{
		settingsRepo: settingsRepo,
		factories:    map[APIType]ModelFactory{},
		modelCache:   map[agent.ModelID]agent.Model{},
	}

	for _, f := range factories {
		svc.factories[f.APIType()] = f
	}

	config, err := settingsRepo.Load()
	if err != nil {
		return nil, fmt.Errorf("load model config: %w", err)
	}
	svc.config = config

	return svc, nil
}

func (s *Service) Get(modelID agent.ModelID) (agent.Model, error) {
	s.mu.RLock()
	cached, ok := s.modelCache[modelID]
	s.mu.RUnlock()
	if ok {
		return cached, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	cached, ok = s.modelCache[modelID]
	if ok {
		return cached, nil
	}

	provider, providerSettings, found := s.findProviderModel(modelID)
	if !found {
		return nil, fmt.Errorf("%w: model %s", types.ErrIsNotExist, modelID)
	}

	factory, ok := s.factories[provider.APIType]
	if !ok {
		return nil, fmt.Errorf("unknown api_type %s for model %s", provider.APIType, modelID)
	}

	m, err := factory.CreateModel(provider, modelID, providerSettings)
	if err != nil {
		return nil, fmt.Errorf("create model %s: %w", modelID, err)
	}

	s.modelCache[modelID] = m
	return m, nil
}

func (s *Service) Delete(modelID agent.ModelID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for providerName, provider := range s.config {
		if _, ok := provider.Models[modelID]; ok {
			delete(provider.Models, modelID)
			s.config[providerName] = provider
			delete(s.modelCache, modelID)
			return s.settingsRepo.Save(s.config)
		}
	}

	return errors.New("model not found")
}

func (s *Service) Save(modelID agent.ModelID, model agent.Model) error {
	// In the new design, Save persists a model through settings.
	// The model's Settings() now returns structured data that we store.
	// For now, this is a compatibility stub – the new design doesn't use Save
	// for regular operation; settings are loaded from file.
	return nil
}

func (s *Service) findProviderModel(modelID agent.ModelID) (ProviderDTO, any, bool) {
	for _, provider := range s.config {
		if m, ok := provider.Models[modelID]; ok {
			return provider, m, true
		}
	}
	return ProviderDTO{}, nil, false
}
