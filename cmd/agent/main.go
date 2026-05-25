package main

import (
	app "arch-agent/internal"
	"arch-agent/internal/config"
	"arch-agent/internal/logging"
	"arch-agent/internal/telegram"
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// context
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM)

	defer stop()

	// config
	configPath := flag.String("config", "config.toml", "path to config file")
	dataPath := flag.String("datadir", ".", "path to data directory")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("bad config", "error", err)
		os.Exit(1)
	}

	// logging
	logging.Set(cfg.Logging.Pretty, cfg.Logging.Level)

	// tg bot
	botConfs := []telegram.BotConfig{}
	for _, acc := range cfg.Telegram.Accs {
		botConfs = append(botConfs, telegram.BotConfig{
			Agent:          acc.Agent,
			APIKey:         acc.APIKey,
			Host:           cfg.Telegram.Host,
			StickerSetName: acc.StickerSet,
		})
	}

	// root composing
	app, err := app.BuildApp(
		ctx,
		*dataPath,
		cfg.SearchHostScheme,
		cfg.SearchHost,
		cfg.Telegram.GroupID,
		botConfs...,
	)
	if err != nil {
		slog.Error("app", "init error", err)
		os.Exit(1)
	}

	// TODO:
	// Remove this shit to diff container
	if cfg.Telegram.Host != "" {
		go http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", cfg.Telegram.Port), nil)
	}

	app.Run(ctx)

	// shutdown await
	<-ctx.Done()

	// graceful shutdown
	slog.Warn("Graceful shutdown")
}
