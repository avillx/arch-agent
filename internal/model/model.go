package model

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/types"
	"fmt"
	"strings"
	"sync"
)

var _ agent.ModelRegistry = (*ModelService)(nil)

type ModelService struct {
	factories map[APIType]ModelFactory
	models    map[ModelID]agent.Model
	mu        sync.RWMutex
}

func NewModelService(factories ...TypedModelFactory) *ModelService {
	svc := &ModelService{
		factories: map[APIType]ModelFactory{},
		models:    map[ModelID]agent.Model{},
	}

	for _, f := range factories {
		svc.factories[f.APIType()] = f
	}

	return svc
}

func (m *ModelService) Get(modelName string) (agent.Model, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for k, v := range m.models {
		if strings.HasSuffix(string(k), modelName) {
			return v, nil
		}
	}

	return nil, types.ErrIsNotExist
}

func (m *ModelService) delete(modelName ModelID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, ok := m.models[modelName]
	if !ok {
		return types.ErrIsNotExist
	}

	delete(m.models, modelName)

	return nil
}

func (m *ModelService) add(
	apiType APIType,
	baseURL string,
	keyReference string,
	modelID ModelID,
	modelSettings agent.ModelSettings,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	f, ok := m.factories[apiType]
	if !ok {
		return fmt.Errorf("model api %s: %w", apiType, ErrUnsupportedAPI)
	}

	model, err := f.CreateModel(baseURL, keyReference, modelSettings)
	if err != nil {
		return err
	}

	m.models[modelID] = model

	return nil
}
