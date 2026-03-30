package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

type Telegram struct {
	APIKeyEnv  string `toml:"api_key_env"`
	APIKey     string `toml:"-"`
	StickerSet string `toml:"sticker_set"`
	Logs       bool   `toml:"logs"`
	Host       string `toml:"host"`
	Port       int    `toml:"port"`
}

func (c *Telegram) InjectKeys() error {
	if apiKey, ok := os.LookupEnv(c.APIKeyEnv); ok {
		c.APIKey = apiKey
		return nil
	}
	return fmt.Errorf("env var %q is not found for %T config", c.APIKeyEnv, c)
}

type LLM struct {
	OpenAIURL  string         `toml:"openai_api_url"`
	Model      string         `toml:"model"`
	TokenLimit int            `toml:"token_limit"`
	APIKeyEnv  string         `toml:"api_key_env"`
	Extras     map[string]any `toml:"extras"`
	APIKey     string         `toml:"-"`
}

func (c *LLM) InjectKeys() error {
	if apiKey, ok := os.LookupEnv(c.APIKeyEnv); ok {
		c.APIKey = apiKey
		return nil
	}
	return fmt.Errorf("env var %q is not found %T config", c.APIKeyEnv, c)
}

type Agent struct {
	Role        string `toml:"role"`
	Personality string `toml:"personality"`
	Preferences string `toml:"preferences"`
	Keyphrases  string `toml:"keyphrases"`
	BannedSlang string `toml:"banned_slang"`
}

type Config struct {
	LLM struct {
		Reflection *LLM `toml:"reflection"`
		Reasoning  *LLM `toml:"reasoning"`
	} `toml:"llm"`
	Agent    *Agent    `toml:"agent"`
	Telegram *Telegram `toml:"telegram"`
}

func LoadFile(configPath string) (Config, error) {
	var config Config
	if _, err := toml.DecodeFile(configPath, &config); err != nil {
		return Config{}, err
	}

	var errs error
	errs = errors.Join(errs, config.Telegram.InjectKeys())
	errs = errors.Join(errs, config.LLM.Reasoning.InjectKeys())
	errs = errors.Join(errs, config.LLM.Reflection.InjectKeys())
	if errs != nil {
		return Config{}, errs
	}

	return config, nil
}
