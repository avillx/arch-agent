package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

type TelegramAcc struct {
	Agent      string `toml:"agent"`
	APIKeyEnv  string `toml:"api_key_env"`
	APIKey     string `toml:"-"`
	StickerSet string `toml:"sticker_set"`
}

func (c *TelegramAcc) InjectKeys() error {
	if apiKey, ok := os.LookupEnv(c.APIKeyEnv); ok {
		c.APIKey = apiKey
		return nil
	}
	return fmt.Errorf("env var %q is not found for %T config", c.APIKeyEnv, c)
}

type Telegram struct {
	Accs    []*TelegramAcc `toml:"accs"`
	GroupID int64          `toml:"group"`
	Logs    bool           `toml:"logs"`
	Host    string         `toml:"host"`
	Port    int            `toml:"port"`
}

func (c *Telegram) InjectKeys() error {
	for _, acc := range c.Accs {
		if err := acc.InjectKeys(); err != nil {
			return err
		}
	}
	return nil
}

type Config struct {
	Telegram         *Telegram `toml:"telegram"`
	SearchHost       string    `toml:"search_host"`
	SearchHostScheme string    `toml:"search_host_scheme"`
}

func Load(configPath string) (Config, error) {
	config := Config{}
	if _, err := toml.DecodeFile(configPath, &config); err != nil {
		return Config{}, err
	}

	if err := config.Telegram.InjectKeys(); err != nil {
		return Config{}, err
	}

	return config, nil
}
