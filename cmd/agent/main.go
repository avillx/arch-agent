package main

import (
	"arch-agent/internal/di"
	"arch-agent/internal/infra/config"
	"arch-agent/internal/infra/logging"
	"arch-agent/internal/infra/telegram"
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
	bot, err := telegram.NewBot(
		telegram.BotConfig{
			APIKey:         cfg.Telegram.APIKey,
			StickerSetName: cfg.Telegram.StickerSet,
			Logs:           cfg.Telegram.Logs,
			Host:           cfg.Telegram.Host,
		},
	)
	if err != nil {
		slog.Error("telegram", "init error", err)
		os.Exit(1)
	}

	// root composing
	uc, err := di.NewAnswerUseCase(cfg, *dataPath, bot.Tools())
	if err != nil {
		slog.Error("bad di", "error", err)
		os.Exit(1)
	}

	bot.WireAnswerUC(uc)

	// TODO:
	// Remove this shit to diff container
	if cfg.Telegram.Host != "" {
		go http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", cfg.Telegram.Port), nil)
	}

	go bot.Run(ctx)

	// shutdown await
	<-ctx.Done()

	// graceful shutdown
	slog.Warn("Graceful shutdown")
}
