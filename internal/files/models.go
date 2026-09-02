package files

import (
	"arch-agent/internal/model"
	"arch-agent/internal/types"
	"bytes"
	"sync"

	toml "github.com/pelletier/go-toml/v2"
)

const modelsFile = "models.toml"
const modelsConfigDoc = `# Models config file
# LLM models available for agents

# Unique provider ID
# [open_ai]

# Type of API, only 'openai' is supported
# api_type='openai'

# Reference to variable from 'secrets.toml'
# key_ref='API_KEY'

# API URL
# base_url='https://api.openai.com/v1'

# List of provider models
# No need to describe every supported model, only the ones that will be used
# [open_ai.models]

# Current model config, agent can use only the mentioned models
# [open_ai.models.'openai/gpt-4o']

# Supported modalities. Agent receives only supported modalities.
# A model that doesn't support a modality, or shouldn't receive it
# for some other reason, should not be described here
# modalities= ['text', 'image']

# When session reaches ~90% of this limit, the session will be compacted
# context_limit = 200000

# Any other parameters supported by the API
# tool_choice = 'auto'
# reasoning_effort = 'medium'
# extras = { "order" : "some" }
# etc...

# Do not touch this comment!
# After edit, ensure file consistency and comment integrity`

var _ model.ProviderConfigRepo = (*ProviderFiles)(nil)

type modelsConfigDTO map[model.ProviderID]providerDTO

type providerDTO struct {
	BaseURL      string                       `toml:"base_url"`
	KeyReference string                       `toml:"key_ref"`
	APIType      model.APIType                `toml:"api_type"`
	Models       map[string]model.ModelConfig `toml:"models"`
}

type ProviderFiles struct {
	fs *FileSystem

	mu sync.RWMutex
}

func NewProviderFiles(fs *FileSystem) (*ProviderFiles, error) {

	if err := ensureFilePlaceholder(fs, modelsFile, []byte(modelsConfigDoc)); err != nil {
		return nil, err
	}

	return &ProviderFiles{fs: fs}, nil
}

func (f *ProviderFiles) All() ([]model.ProviderConfig, error) {

	f.mu.RLock()
	defer f.mu.RUnlock()

	provConfs := []model.ProviderConfig{}

	cfg, err := f.loadConfig()
	if err != nil {
		return nil, err
	}

	for provName, provDTO := range cfg {
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

	provDTO, ok := cfg[id]
	if !ok {
		return model.ProviderConfig{}, types.ErrIsNotExist
	}

	return model.ProviderConfig{
		Name:         id,
		BaseURL:      provDTO.BaseURL,
		KeyReference: provDTO.KeyReference,
		Models:       provDTO.Models,
		APIType:      provDTO.APIType,
	}, nil
}

func (f *ProviderFiles) Save(cfg model.ProviderConfig) error {

	f.mu.Lock()
	defer f.mu.Unlock()

	cfgDTO, err := f.loadConfig()
	if err != nil {
		return err
	}

	cfgDTO[cfg.Name] = providerDTO{
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

	delete(cfg, id)

	return f.saveConfig(cfg)
}

func (f *ProviderFiles) loadConfig() (modelsConfigDTO, error) {
	var dto modelsConfigDTO

	data, err := f.fs.ReadFile(modelsFile)
	if err != nil {
		// TODO: create place holder if not exist
		return dto, err
	}

	if err := toml.Unmarshal(data, &dto); err != nil {
		return dto, err
	}

	return dto, nil
}

func (f *ProviderFiles) saveConfig(dto modelsConfigDTO) error {

	data, err := toml.Marshal(dto)
	if err != nil {
		return err
	}

	dataWithDoc := bytes.Join(
		[][]byte{[]byte(modelsConfigDoc), data},
		[]byte("\n\n"),
	)

	return f.fs.WriteToFile(modelsFile, dataWithDoc)
}
