package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"

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

type Logging struct {
	Pretty bool
	Level  slog.Level
}

func LoadLogging() Logging {
	l := Logging{
		Pretty: false,
		Level:  slog.LevelInfo,
	}

	if logPretty, ok := os.LookupEnv("LOG_PRETTY"); ok {
		logPrettyBool, _ := strconv.ParseBool(logPretty)
		l.Pretty = logPrettyBool
	}

	if LogLevel, ok := os.LookupEnv("LOG_LEVEL"); ok {
		l.Level = toLogLevel(LogLevel)
	}

	return l
}

type Config struct {
	Telegram         *Telegram `toml:"telegram"`
	SearchHost       string    `toml:"search_host"`
	SearchHostScheme string    `toml:"search_host_scheme"`
	Logging          Logging
}

func Load(configPath string) (Config, error) {
	config := Config{}
	if _, err := toml.DecodeFile(configPath, &config); err != nil {
		return Config{}, err
	}

	if err := config.Telegram.InjectKeys(); err != nil {
		return Config{}, err
	}

	config.Logging = LoadLogging()

	return config, nil
}

func toLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError

	default:
		slog.Error("logging", "log level", level, "levels", "debug/info/warn/error")
		return slog.LevelError
	}
}
