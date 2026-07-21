package model

import (
	"arch-agent/internal/agent"
	"errors"
)

type APIType string
type ModelID string

const APITypeOpenAI APIType = "openai"

var (
	ErrShortName      = errors.New("short name is not allowed for this operation")
	ErrUnsupportedAPI = errors.New("this api is not supported")
)

type TypedModelFactory interface {
	APIType() APIType
	ModelFactory
}

type ModelFactory interface {
	CreateModel(
		BaseURL string,
		KeyReference string,
		modelSettings agent.ModelSettings,
	) (agent.Model, error)
}

type ProviderID string

type ModelConfig map[string]any

type ProviderConfig struct {
	Name         ProviderID
	BaseURL      string
	KeyReference string
	APIType      APIType
	Models       map[string]ModelConfig
}

type ProviderConfigRepo interface {
	All() ([]ProviderConfig, error)
	Get(ProviderID) (ProviderConfig, error)
	Save(ProviderConfig) error
	Delete(ProviderID) error
}

type ProviderConfigPatch struct {
	Name         *ProviderID `json:"name"`
	BaseURL      *string     `json:"base_url"`
	KeyReference *string     `json:"key_ref"`
	APIType      *APIType    `json:"api_type"`
}
