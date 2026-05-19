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
	botConfs := []telegram.BotConfig{}
	for _, acc := range cfg.Telegram.Accs {
		botConfs = append(botConfs, telegram.BotConfig{
			Agent:          acc.Agent,
			APIKey:         acc.APIKey,
			Host:           cfg.Telegram.Host,
			StickerSetName: acc.StickerSet,
		})
	}

	botOrchestra, err := telegram.NewBotOrchestrator(botConfs...)
	if err != nil {
		slog.Error("telegram", "init error", err)
		os.Exit(1)
	}

	// root composing
	app, err := di.BuildApp(
		ctx,
		*dataPath,
		telegram.TelegramTS(botOrchestra),
	)
	if err != nil {
		slog.Error("app", "init error", err)
		os.Exit(1)
	}

	botOrchestra.WireApp(app)

	tgA2AInterceptor := telegram.NewA2AInterceptor(cfg.Telegram.GroupID, botOrchestra, app.A2A)

	// TODO:
	// Remove this shit to diff container
	if cfg.Telegram.Host != "" {
		go http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", cfg.Telegram.Port), nil)
	}

	go botOrchestra.Run(ctx)
	go tgA2AInterceptor.Run(ctx)

	// shutdown await
	<-ctx.Done()

	// graceful shutdown
	slog.Warn("Graceful shutdown")
}
