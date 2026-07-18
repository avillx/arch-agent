package model

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/types"
	"errors"
	"fmt"
	"log/slog"
	"path"
	"strings"
	"sync"
)

type APIType string

const APITypeOpenAI APIType = "openai"

type ProviderConfig struct {
	BaseURL      string                         `json:"base_url"`
	KeyReference string                         `json:"key_reference"`
	APIType      APIType                        `json:"api_type"`
	Models       map[string]agent.ModelSettings `json:"models"`
}

type Config map[string]ProviderConfig

type ConfigRepo interface {
	Load() (Config, error)
	Save(Config) error
}

type ModelFactory interface {
	APIType() APIType
	CreateModel(provider ProviderConfig, modelID agent.ModelName, modelSettings agent.ModelSettings) (agent.Model, error)
}

// service
type Service struct {
	configRepo ConfigRepo
	factories  map[APIType]ModelFactory

	// models clinets pool
	models map[agent.ModelName]agent.Model
	mu     sync.RWMutex
}

func NewService(configRepo ConfigRepo, factories ...ModelFactory) (*Service, error) {
	svc := &Service{
		configRepo: configRepo,
		factories:  map[APIType]ModelFactory{},
	}

	// load factories
	for _, f := range factories {
		svc.factories[f.APIType()] = f
	}

	cfg, err := configRepo.Load()
	if err != nil {
		return nil, err
	}

	models, err := loadConfig(svc.factories, cfg)
	if err != nil {
		if wrapedErrs, ok := err.(interface{ Unwrap() []error }); ok {
			for _, e := range wrapedErrs.Unwrap() {
				slog.Error("load models", "error", e)
			}
		}
	}

	svc.models = models

	return svc, nil
}

var ErrShotName = errors.New("short name is not allowed for this operation")

func (s *Service) Get(name agent.ModelName) (agent.Model, error) {

	if isShortName(name) {
		return s.getByShortname(name)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	m, ok := s.models[name]
	if !ok {
		return nil, types.ErrIsNotExist
	}
	return m, nil
}

func (s *Service) Delete(modelName agent.ModelName) error {

	if isShortName(modelName) {
		return ErrShotName
	}

	cfg, err := s.configRepo.Load()
	if err != nil {
		return err
	}

	providerName := strings.Split(string(modelName), "/")[0]
	delete(cfg[providerName].Models, string(modelName))

	if err := s.configRepo.Save(cfg); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.models[modelName]; !ok {
		return types.ErrIsNotExist
	}

	delete(s.models, modelName)

	return nil
}

func (s *Service) Save(modelName agent.ModelName, model agent.Model) error {

	if isShortName(modelName) {
		return ErrShotName
	}

	modelSettings := model.Settings()

	modelNameParts := strings.Split(string(modelName), "/")
	providerName := modelNameParts[0]
	modelShortName := path.Join(modelNameParts[1:]...)

	cfg, err := s.configRepo.Load()
	if err != nil {
		return err
	}
	// build model
	s.mu.RLock()
	factories := s.factories
	s.mu.RUnlock()

	apiType := cfg[providerName].APIType
	f, ok := factories[apiType]
	if !ok {
		return fmt.Errorf("models: unsupported api type: %s", apiType)
	}

	newModel, err := f.CreateModel(cfg[providerName], agent.ModelName(modelShortName), modelSettings)
	if err != nil {
		return err
	}

	// update cfg
	cfg[providerName].Models[modelShortName] = modelSettings
	if err := s.configRepo.Save(cfg); err != nil {
		return err
	}

	// replace. GC eliminate deprecated
	s.mu.Lock()
	defer s.mu.Unlock()
	s.models[modelName] = newModel

	return nil
}

func loadConfig(facrotries map[APIType]ModelFactory, cfg Config) (map[agent.ModelName]agent.Model, error) {

	models := map[agent.ModelName]agent.Model{}
	errs := []error{}
	for providerName, providerCfg := range cfg {
		f, ok := facrotries[providerCfg.APIType]
		if !ok {
			err := fmt.Errorf("models: unsupported api type: %s", providerCfg.APIType)
			errs = append(errs, err)
		}

		providerModels, err := builtProviderModels(providerName, f, providerCfg)
		if err != nil {
			errs = append(errs, err)
		}

		for mName, m := range providerModels {
			models[mName] = m
		}
	}

	return models, errors.Join(errs...)
}

func builtProviderModels(name string, f ModelFactory, cfg ProviderConfig) (map[agent.ModelName]agent.Model, error) {

	models := map[agent.ModelName]agent.Model{}
	errs := []error{}
	for modelShortName, modelCfg := range cfg.Models {

		modelName := agent.ModelName(path.Join(name, modelShortName))

		model, err := f.CreateModel(cfg, agent.ModelName(modelShortName), modelCfg)
		if err != nil {
			errs = append(errs, fmt.Errorf("models: create model client: %w", err))
			continue
		}

		models[modelName] = model
	}

	return models, errors.Join(errs...)
}

func (s *Service) getByShortname(name agent.ModelName) (agent.Model, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for k, v := range s.models {
		if strings.Contains(string(k), string(name)) {
			return v, nil
		}
	}

	return nil, types.ErrIsNotExist
}

func isShortName(name agent.ModelName) bool {
	if len(strings.Split(string(name), "/")) < 3 {
		return true
	}
	return false
}
