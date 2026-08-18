package main

import (
	"arch-agent/internal/model"
	"os"

	"github.com/pelletier/go-toml/v2"
)

type ProviderConfig struct {
	BaseURL      string                       `toml:"base_url,commented"`
	KeyReference string                       `toml:"key_ref,commented"`
	APIType      model.APIType                `toml:"api_type,commented"`
	Models       map[string]model.ModelConfig `toml:"models,commented"`
}

func main() {
	dto := map[model.ProviderID]ProviderConfig{
		"openRouter": ProviderConfig{
			Models: map[string]model.ModelConfig{
				"some/shit-10": model.ModelConfig{
					"context_limit": 1000,
				},
			},
		},
	}

	data, err := toml.Marshal(dto)
	if err != nil {
		return
	}

	os.WriteFile("E:/repos/arch-agent/data/providers.toml", data, os.ModeAppend)
}
