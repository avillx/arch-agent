package main

import (
	app "arch-agent/internal"
	"arch-agent/internal/api"
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

func botConf(cfg config.Config) []telegram.BotConfig {
	// bot configs
	botCfgs := make([]telegram.BotConfig, 0, len(cfg.Telegram.Accs))
	for _, acc := range cfg.Telegram.Accs {
		botCfgs = append(botCfgs, telegram.BotConfig{
			Agent:          acc.Agent,
			APIKey:         acc.APIKey,
			Host:           cfg.Telegram.Host,
			StickerSetName: acc.StickerSet,
		})
	}
	return botCfgs
}

func run(ctx context.Context,
	configPath string,
	dataPath string,
	logLevel slog.Level,
	logPretty bool,

) error {
	ctx, stop := signal.NotifyContext(
		ctx,
		os.Interrupt,
		syscall.SIGTERM,
	)

	defer stop()

	// logging
	logging.Set(logPretty, slog.Level(logLevel))

	// cfg
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	// App composing
	app, err := app.BuildApp(ctx, app.AppConfig{
		DataPath:           dataPath,
		SearchHostScheme:   cfg.SearchHostScheme,
		SearchHost:         cfg.SearchHost,
		TelegramGroupID:    cfg.Telegram.GroupID,
		BotConfigs:         botConf(cfg),
		ShellEnv:           cfg.ShellEnv,
		ConsolidationModel: cfg.ConsolidationModel,
		ObserverModel:      cfg.ObserverModel,
	})
	if err != nil {
		return err
	}
	go app.Memory.Run(ctx)
	app.TelegramOrchestra.Run(ctx)

	// server
	svc := api.NewServer(
		app.TaskSvc,
		app.ChatSvc,
		app.SessionSvc,
		app.ToolsSvc,
		app.MCPSvc,
		app.MemoryRepo,
		app.MemoryIndexer,
		app.Memory,
	)

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: svc,
	}

	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}

	// await
	<-ctx.Done()
	if err := httpServer.Shutdown(context.Background()); err != nil {
		return err
	}

	return nil
}

func main() {

	// flags
	configPath := flag.String("config", "config.toml", "path to config file")
	dataPath := flag.String("datadir", ".", "path to data directory")
	logLevel := flag.Int("log-level", int(slog.LevelWarn), "path to data directory")
	logPretty := flag.Bool("log-pretty", false, "path to data directory")
	flag.Parse()

	// run
	ctx := context.Background()
	if err := run(ctx, *configPath, *dataPath, slog.Level(*logLevel), *logPretty); err != nil {
		slog.Error("server run", "error", err)
		os.Exit(1)
	}

	slog.Warn("server shutdown")
}
