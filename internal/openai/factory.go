package openai

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/model"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

type OpenAIModelFactory struct {
	secrets   SecretsRepo
	clients   map[clientKey]*openai.Client
	clientsMu sync.Mutex
}

func NewOpenAIModelFactory(secrets SecretsRepo) *OpenAIModelFactory {
	return &OpenAIModelFactory{
		secrets: secrets,
		clients: map[clientKey]*openai.Client{},
	}
}

func (f *OpenAIModelFactory) APIType() model.APIType {
	return model.APITypeOpenAI
}

func (f *OpenAIModelFactory) CreateModel(
	provider model.ProviderDTO,
	modelID agent.ModelID,
	modelSettings any,
) (agent.Model, error) {
	settings, err := f.parseSettings(modelSettings)
	if err != nil {
		return nil, fmt.Errorf("parse model settings for %s: %w", modelID, err)
	}

	client, err := f.getOrCreateClient(provider)
	if err != nil {
		return nil, fmt.Errorf("get client for provider %s: %w", provider.BaseURL, err)
	}

	return NewOpenAIReasoner(client, settings), nil
}

type clientKey struct {
	Url    string
	KeyRef string
}

func (f *OpenAIModelFactory) getOrCreateClient(provider model.ProviderDTO) (*openai.Client, error) {
	f.clientsMu.Lock()
	defer f.clientsMu.Unlock()

	apiKey, found := f.secrets.Get(provider.KeyReference)
	if !found {
		return nil, fmt.Errorf("api key %s is not found in secrets", provider.KeyReference)
	}

	cacheKey := clientKey{Url: provider.BaseURL, KeyRef: provider.KeyReference}

	if client, ok := f.clients[cacheKey]; ok {
		return client, nil
	}

	client := openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL(provider.BaseURL),
	)

	f.clients[cacheKey] = &client
	return &client, nil
}

func (f *OpenAIModelFactory) parseSettings(raw any) (OpenAIModelSettings, error) {
	switch s := raw.(type) {
	case OpenAIModelSettings:
		return s, nil
	case map[string]any:
		data, err := json.Marshal(s)
		if err != nil {
			return OpenAIModelSettings{}, err
		}
		var settings OpenAIModelSettings
		if err := json.Unmarshal(data, &settings); err != nil {
			return OpenAIModelSettings{}, err
		}
		return settings, nil
	default:
		return OpenAIModelSettings{}, fmt.Errorf("unexpected model settings type: %T", raw)
	}
}
