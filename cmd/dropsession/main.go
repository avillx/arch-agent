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

	activityServce := di.CreateActivityService(cfg.LLMS.Summarization, absolutePath)
	sessionService := di.CreateSessionService(activityServce, absolutePath)

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
