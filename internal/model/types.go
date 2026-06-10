package model

import "arch-agent/internal/agent"

type APIType string

const APITypeOpenAI APIType = "openai"

type ProviderDTO struct {
	BaseURL      string                    `json:"base_url"`
	KeyReference string                    `json:"key_reference"`
	APIType      APIType                   `json:"api_type"`
	Models       map[agent.ModelID]any     `json:"models"`
}

type ModelsConfig map[string]ProviderDTO
