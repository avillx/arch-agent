package files

import (
	"arch-agent/internal/model"
	"arch-agent/internal/types"
	"encoding/json"
	"sync"
)

const providersFile = "providers.json"

var _ model.ProviderConfigRepo = (*ProviderFiles)(nil)

type ProviderFiles struct {
	fs *FileSystem

	mu sync.RWMutex
}

func NewProviderFiles(fs *FileSystem) *ProviderFiles {
	return &ProviderFiles{fs: fs}
}

func (f *ProviderFiles) All() ([]model.ProviderConfig, error) {

	f.mu.RLock()
	defer f.mu.RUnlock()

	provConfs := []model.ProviderConfig{}

	cfg, err := f.loadConfig()
	if err != nil {
		return nil, err
	}

	for provName, provDTO := range cfg.Providers {
		provConfs = append(provConfs, model.ProviderConfig{
			Name:         provName,
			BaseURL:      provDTO.BaseURL,
			KeyReference: provDTO.KeyReference,
			APIType:      provDTO.APIType,
			Models:       provDTO.Models,
		})
	}

	return provConfs, nil
}

func (f *ProviderFiles) Get(id model.ProviderID) (model.ProviderConfig, error) {

	f.mu.RLock()
	defer f.mu.RUnlock()

	cfg, err := f.loadConfig()
	if err != nil {
		return model.ProviderConfig{}, err
	}

	provDTO, ok := cfg.Providers[id]
	if !ok {
		return model.ProviderConfig{}, types.ErrIsNotExist
	}

	return model.ProviderConfig{
		Name:         id,
		BaseURL:      provDTO.BaseURL,
		KeyReference: provDTO.KeyReference,
		Models:       provDTO.Models,
	}, nil
}

func (f *ProviderFiles) Save(cfg model.ProviderConfig) error {

	f.mu.Lock()
	defer f.mu.Unlock()

	cfgDTO, err := f.loadConfig()
	if err != nil {
		return err
	}

	cfgDTO.Providers[cfg.Name] = providerDTO{
		BaseURL:      cfg.BaseURL,
		KeyReference: cfg.KeyReference,
		APIType:      cfg.APIType,
		Models:       cfg.Models,
	}

	return f.saveConfig(cfgDTO)
}

func (f *ProviderFiles) Delete(id model.ProviderID) error {

	f.mu.Lock()
	defer f.mu.Unlock()

	cfg, err := f.loadConfig()
	if err != nil {
		return err
	}

	delete(cfg.Providers, id)

	return f.saveConfig(cfg)
}

type configDTO struct {
	Providers map[model.ProviderID]providerDTO `json:"providers"`
}

type providerDTO struct {
	BaseURL      string                       `json:"base_url"`
	KeyReference string                       `json:"key_ref"`
	APIType      model.APIType                `json:"api_type"`
	Models       map[string]model.ModelConfig `json:"models"`
}

func (f *ProviderFiles) loadConfig() (configDTO, error) {
	var dto configDTO

	data, err := f.fs.ReadFile(providersFile)
	if err != nil {
		return dto, err
	}

	if err := json.Unmarshal(data, &dto); err != nil {
		return dto, err
	}

	return dto, nil
}

func (f *ProviderFiles) saveConfig(dto configDTO) error {

	data, err := json.Marshal(dto)
	if err != nil {
		return err
	}

	return f.fs.WriteToFile(providersFile, data)
}
