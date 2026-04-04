package main

import (
	"arch-agent/internal/app/answer"
	"arch-agent/internal/app/executioncontext"
	"arch-agent/internal/app/memory"
	"arch-agent/internal/app/session"
	"arch-agent/internal/infra/config"
	"arch-agent/internal/infra/filestorage"
	"arch-agent/internal/infra/logging"
	openaiadapter "arch-agent/internal/infra/openai"
	"arch-agent/internal/infra/telegram"
	"arch-agent/internal/infra/tokenizer"
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

// Agent repo
type stubAgentRepository struct {
	*config.Agent
}

func (r *stubAgentRepository) Get() executioncontext.AgentConfig {
	return executioncontext.AgentConfig{
		Role:        r.Role,
		Personality: r.Personality,
		Preferences: r.Preferences,
		Keyphrases:  r.Keyphrases,
		BannedSlang: r.BannedSlang,
	}
}

func main() {
	// context
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM)

	defer stop()

	// configQ
	configPath := flag.String("config", "config.toml", "path to config file")
	flag.Parse()

	config, err := config.Load(*configPath)
	if err != nil {
		slog.Error("bad config", "error", err)
		os.Exit(1)
	}

	// logging
	logging.Set(config.Logging.Pretty, config.Logging.Level)

	// root composing
	reflector := openaiadapter.NewReflector(
		openai.NewClient(
			option.WithBaseURL(config.LLM.Reflection.OpenAIURL),
			option.WithAPIKey(config.LLM.Reflection.APIKey),
		),
		config.LLM.Reflection.Model,
		config.LLM.Reflection.Extras,
	)

	reasoner := openaiadapter.NewReasoner(
		openai.NewClient(
			option.WithBaseURL(config.LLM.Reasoning.OpenAIURL),
			option.WithAPIKey(config.LLM.Reasoning.APIKey),
		),
		config.LLM.Reasoning.Model,
		config.LLM.Reasoning.Extras,
	)

	summarizer := openaiadapter.NewSummarizer(
		openai.NewClient(
			option.WithBaseURL(config.LLM.Reasoning.OpenAIURL),
			option.WithAPIKey(config.LLM.Reasoning.APIKey),
		),
		config.LLM.Reasoning.Model,
		config.LLM.Reasoning.Extras,
	)

	absolutePath, _ := filepath.Abs(".")

	dailyActivityLogger := filestorage.NewMDDailyActivityLogger(absolutePath + "/data/memory/daily_logs")

	sessionService := session.NewSessionService(
		filestorage.NewFileSessionRepository(absolutePath+"/data/memory"),
		filestorage.NewJSONLTranscriber(absolutePath+"/data/memory/transciptions"),
		tokenizer.NewTokenizer(),
		summarizer,
		dailyActivityLogger,
	)

	executionContextFactory := executioncontext.NewRequestContextFactory(reflector)
	dailyActivityProvider := filestorage.NewDailyActivityProvider(dailyActivityLogger)

	answerUseCase := answer.NewAnswerUseCase(
		reasoner,
		sessionService,
		memory.NewMemoryService(dailyActivityProvider),
		&stubAgentRepository{config.Agent},
		executionContextFactory,
		&logging.AnswerUCLogger{},
	)

	// tg bot
	bot, err := telegram.NewBot(
		answerUseCase,
		telegram.BotConfig{
			APIKey:         config.Telegram.APIKey,
			StickerSetName: config.Telegram.StickerSet,
			Logs:           config.Telegram.Logs,
			Host:           config.Telegram.Host,
		},
	)
	if err != nil {
		slog.Error("telegram", "init error", err)
		os.Exit(1)
	}

	if config.Telegram.Host != "" {
		go http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", config.Telegram.Port), nil)
	}

	go bot.Run(ctx)

	// shutdown await
	<-ctx.Done()

	// graceful shutdown
	slog.Warn("Graceful shutdown")
}
