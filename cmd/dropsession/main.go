package main

import (
	"arch-agent/internal/app/session"
	"arch-agent/internal/infra/config"
	"arch-agent/internal/infra/filestorage"
	openaiadapter "arch-agent/internal/infra/openai"
	"arch-agent/internal/infra/tokenizer"
	"context"
	"flag"
	"log"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

func main() {

	// configQ
	configPath := flag.String("config", "config.toml", "path to config file")
	dataPath := flag.String("datadir", ".", "path to data directory")
	flag.Parse()

	config, err := config.Load(*configPath)
	if err != nil {
		slog.Error("bad config", "error", err)
		os.Exit(1)
	}

	summarizer := openaiadapter.NewSummarizer(
		openai.NewClient(
			option.WithBaseURL(config.LLM.Reasoning.OpenAIURL),
			option.WithAPIKey(config.LLM.Reasoning.APIKey),
		),
		config.LLM.Reasoning.Model,
		config.LLM.Reasoning.Extras,
	)

	absolutePath, _ := filepath.Abs(*dataPath)

	dailyActivityStore := filestorage.NewMDDailyActivityStore(absolutePath + "/memory/daily_logs")

	sessionService := session.NewSessionService(
		filestorage.NewFileSessionRepository(absolutePath+"/"),
		filestorage.NewJSONLTranscriber(absolutePath+"/memory/transciptions"),
		tokenizer.NewTokenizer(),
		summarizer,
		dailyActivityStore,
	)

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
