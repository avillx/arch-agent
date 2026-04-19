package main

import (
	"arch-agent/internal/di"
	"arch-agent/internal/infra/config"
	"context"
	"flag"
	"log"
	"log/slog"
	"os"
	"path/filepath"
)

func main() {

	// config
	configPath := flag.String("config", "config.toml", "path to config file")
	dataPath := flag.String("datadir", ".", "path to data directory")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("bad config", "error", err)
		os.Exit(1)
	}

	absolutePath, _ := filepath.Abs(*dataPath)

	activityServce, err := di.CreateActivityService(cfg.LLMS.Summarization, absolutePath)
	if err != nil {
		log.Fatalf("activity di %e", err)
		return
	}

	sessionService, err := di.CreateSessionService(activityServce, absolutePath)
	if err != nil {
		log.Fatalf("session di %e", err)
		return
	}

	session, err := sessionService.Session()
	if err != nil {
		log.Fatalf("session load %e", err)
		return
	}
	if err := sessionService.Drop(context.Background(), session); err != nil {
		log.Fatalf("drop error %e", err)
	}

	log.Print("session dropped succeceful")
}
