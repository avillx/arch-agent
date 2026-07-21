package model

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/types"
	"context"
	"errors"
)

type APIType string
type ModelID string

const APITypeOpenAI APIType = "openai"

var (
	ErrUnsupportedAPI = errors.New("this api is not supported")
	ErrEmptyModelName = errors.New("model name is required")
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
	Name         ProviderID             `json:"name"`
	BaseURL      string                 `json:"base_url"`
	KeyReference string                 `json:"key_ref"`
	APIType      APIType                `json:"api_type"`
	Models       map[string]ModelConfig `json:"models"`
}

func (c ProviderConfig) Validate(_ context.Context) error {
	problems := map[string]string{}
	if c.APIType != APITypeOpenAI {
		problems["api_type"] = ErrUnsupportedAPI.Error()
	}

	if c.Name == "" {
		problems["name"] = "must be not empty"
	}

	if c.BaseURL == "" {
		problems["base_url"] = "empty field"
	}

	if len(problems) > 0 {
		return types.NewValidationError(problems)
	}

	return nil
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
