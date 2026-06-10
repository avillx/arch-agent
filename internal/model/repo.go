package model

import "arch-agent/internal/agent"

type ModelSettingsRepo interface {
	Load() (ModelsConfig, error)
	Save(ModelsConfig) error
}

type ModelFactory interface {
	APIType() APIType
	CreateModel(provider ProviderDTO, modelID agent.ModelID, modelSettings any) (agent.Model, error)
}
