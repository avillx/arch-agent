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
	"time"
)

func main() {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM)
	defer stop()

	configPath := flag.String("config", "config.toml", "path to config file")
	dataPath := flag.String("datadir", ".", "path to data directory")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("bad config", "error", err)
		os.Exit(1)
	}

	logging.Set(cfg.Logging.Pretty, cfg.Logging.Level)

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

	// composing
	app, err := app.BuildApp(ctx, app.AppConfig{
		DataPath:         *dataPath,
		SearchHostScheme: cfg.SearchHostScheme,
		SearchHost:       cfg.SearchHost,
		TelegramGroupID:  cfg.Telegram.GroupID,
		BotConfigs:       botCfgs,
	})
	if err != nil {
		slog.Error("app", "init error", err)
		os.Exit(1)
	}

	// webhook server
	var httpSrv *http.Server
	if cfg.Telegram.Host != "" {
		httpSrv = &http.Server{Addr: fmt.Sprintf("0.0.0.0:%d", cfg.Telegram.Port)}
		go func() {
			slog.Info("webhook server started", "addr", httpSrv.Addr)
			if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				slog.Error("webhook server", "error", err)
			}
		}()
	}

	app.Run(ctx)

	// shutdown
	<-ctx.Done()

	if httpSrv != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		httpSrv.Shutdown(shutdownCtx)
	}

	slog.Warn("graceful shutdown")
}