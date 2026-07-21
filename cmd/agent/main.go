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
	"strconv"
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
	getENV func(key string) (string, bool),
) error {
	ctx, stop := signal.NotifyContext(
		ctx,
		os.Interrupt,
		syscall.SIGTERM,
	)

	defer stop()

	// cfg
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	dataPath, ok := getENV("DATA_PATH")
	if !ok {
		dataPath = "."
	}

	consolidationModel, ok := getENV("CONSOLIDATION_MODEL")
	if !ok {
		return fmt.Errorf("env 'CONSOLIDATION_MODEL' must be non nil")
	}

	observerModel, ok := getENV("OBSERVER_MODEL")
	if !ok {
		return fmt.Errorf("env 'OBSERVER_MODEL' must be non nil")
	}

	port, ok := getENV("PORT")
	if !ok {
		// default
		port = "8080"
	}

	publicURL, ok := getENV("PUBLIC_URL")
	if !ok {
		// default
		publicURL = fmt.Sprintf("localhost:%s", port)
	}

	logPretty, ok := getENV("LOG_PRETTY")
	if !ok {
		logPretty = "false"
	}

	logLevel, ok := getENV("LOG_LEVEL")
	if !ok {
		// default
		logLevel = "4"
	}

	// logging
	logPrettyBool, err := strconv.ParseBool(logPretty)
	if err != nil {
		return err
	}

	logLevelInt, err := strconv.Atoi(logLevel)
	if err != nil {
		return err
	}

	logging.Set(logPrettyBool, slog.Level(logLevelInt))

	// App composing
	app, err := app.BuildApp(ctx, app.AppConfig{
		DataPath:           dataPath,
		TelegramGroupID:    cfg.Telegram.GroupID,
		BotConfigs:         botConf(cfg),
		ConsolidationModel: consolidationModel,
		ObserverModel:      observerModel,
	})
	if err != nil {
		return err
	}
	go app.Memory.Run(ctx)
	app.TelegramOrchestra.Run(ctx)

	// server
	httpServer := api.NewServer(
		fmt.Sprintf(":%s", port),
		publicURL,
		app.TaskSvc,
		app.ChatSvc,
		app.SessionSvc,
		app.ToolsSvc,
		app.MCPSvc,
		app.MemoryRepo,
		app.MemoryIndexer,
		app.Memory,
		app.ActivityRepo,
		app.AgentRepo,
	)

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
	flag.Parse()

	// run
	ctx := context.Background()
	if err := run(ctx, *configPath, os.LookupEnv); err != nil {
		slog.Error("server run", "error", err)
		os.Exit(1)
	}

	slog.Warn("server shutdown")
}
