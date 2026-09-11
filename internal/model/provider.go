package model

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/types"
	"errors"
	"fmt"
	"log/slog"
	"path"
	"sync"
)

type ProviderService struct {
	modelSvc *ModelService
	repo     ProviderConfigRepo
	logger   *slog.Logger

	// technically service is already concurrent safe on memory level
	// mutex here for guarantee concurrent safe on business logic level
	// mutex must lock service on every operation even if is include I/O
	mu sync.RWMutex
}

func NewProviderService(modelSvc *ModelService, repo ProviderConfigRepo) (*ProviderService, error) {
	svc := &ProviderService{
		modelSvc: modelSvc,
		repo:     repo,
	}

	if err := svc.Reload(); err != nil {
		return nil, err
	}

	return svc, nil
}

func (s *ProviderService) Reload() error {
	providers, err := s.repo.All()
	if err != nil {
		return err
	}

	for _, p := range providers {
		for _, e := range s.loadModels(p) {
			s.logger.Error("model not loaded",
				"provider", e.Provider,
				"model", e.ModelName,
				"error", e.Unwrap(),
			)
		}
	}

	return nil
}

func (s *ProviderService) GetAll() ([]ProviderConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.repo.All()
}

func (s *ProviderService) GetProvider(id ProviderID) (ProviderConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.repo.Get(id)
}

func (s *ProviderService) AddProvider(cfg ProviderConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.repo.Get(cfg.Name)
	if err == nil {
		return types.ErrAlreadyExist
	}
	if !errors.Is(err, types.ErrIsNotExist) {
		return err
	}

	return s.repo.Save(cfg)
}

func (s *ProviderService) UpdateProvider(id ProviderID, patch ProviderConfigPatch) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cfg, err := s.repo.Get(id)
	if err != nil {
		return err
	}

	for modelName := range cfg.Models {
		s.modelSvc.delete(resolveModelID(id, modelName))
	}

	// patching
	if patch.APIType != nil {
		cfg.APIType = *patch.APIType
	}

	if patch.BaseURL != nil {
		cfg.BaseURL = *patch.BaseURL
	}

	if patch.KeyReference != nil {
		cfg.KeyReference = *patch.KeyReference
	}

	if patch.Name != nil && *patch.Name != id {
		cfg.Name = *patch.Name

		if err := s.repo.Delete(id); err != nil {
			return err
		}
	}

	return s.repo.Save(cfg)
}

func (s *ProviderService) DeleteProvider(id ProviderID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	providerConf, err := s.repo.Get(id)
	if err != nil {
		return err
	}

	for modelName := range providerConf.Models {
		s.modelSvc.delete(resolveModelID(id, modelName))
	}

	return s.repo.Delete(id)
}

func (s *ProviderService) DeleteModel(providerID ProviderID, modelName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	providerConf, err := s.repo.Get(providerID)
	if err != nil {
		return err
	}

	delete(providerConf.Models, modelName)

	if err := s.repo.Save(providerConf); err != nil {
		return err
	}

	s.modelSvc.delete(resolveModelID(providerID, modelName))

	return nil
}

func (s *ProviderService) SetModel(providerID ProviderID, modelName string, modelCfg ModelConfig) error {

	// base validation
	if modelName == "" {
		return ErrEmptyModelName
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	providerConf, err := s.repo.Get(providerID)
	if err != nil {
		return err
	}

	if err := s.modelSvc.add(
		providerConf.APIType,
		providerConf.BaseURL,
		providerConf.KeyReference,
		resolveModelID(providerID, modelName),
		newModelSettings(modelName, modelCfg),
	); err != nil {
		return err
	}

	providerConf.Models[modelName] = modelCfg

	return s.repo.Save(providerConf)
}

type ErrModelLoad struct {
	Provider  string
	ModelName string
	err       error
}

func (e *ErrModelLoad) Unwrap() error {
	return e.err
}

func (e *ErrModelLoad) Error() string {
	return fmt.Sprintf(
		"model '%s' of '%s' provider is not loaded",
		e.ModelName,
		e.Provider,
	)
}

func (s *ProviderService) loadModels(cfg ProviderConfig) []*ErrModelLoad {
	loadErrs := []*ErrModelLoad{}
	for modelName, modelCfg := range cfg.Models {
		if err := s.modelSvc.add(
			cfg.APIType,
			cfg.BaseURL,
			cfg.KeyReference,
			resolveModelID(cfg.Name, modelName),
			newModelSettings(modelName, modelCfg),
		); err != nil {
			loadErrs = append(loadErrs, &ErrModelLoad{
				Provider:  string(cfg.Name),
				ModelName: modelName,
				err:       err,
			})
		}
	}
	return loadErrs
}

func resolveModelID(providerName ProviderID, modelName string) ModelID {
	return ModelID(path.Join(string(providerName), modelName))
}

func newModelSettings(modelName string, cfg ModelConfig) agent.ModelSettings {
	modelSettings := agent.ModelSettings(cfg)
	modelSettings["model"] = modelName

	return modelSettings
}
