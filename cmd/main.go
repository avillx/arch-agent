package main

import (
	"arch-agent/internal/app/answer"
	"arch-agent/internal/app/executioncontext"
	"arch-agent/internal/domain/conversation"
	"arch-agent/internal/infra/config"
	"arch-agent/internal/infra/logging"
	openaiadapter "arch-agent/internal/infra/openai"
	"arch-agent/internal/infra/telegram"
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

// tokenizer
type nilTokenizer struct{}

func (t *nilTokenizer) Calc(_ string) int {
	return 0
}

// conversation
type stubConversationRespository struct {
	Messages  []conversation.Message
	Tokenizer *nilTokenizer
}

func NewStubConversationRespository() *stubConversationRespository {
	return &stubConversationRespository{
		Messages:  []conversation.Message{},
		Tokenizer: &nilTokenizer{},
	}
}
func (r *stubConversationRespository) Get() *conversation.Conversation {
	return conversation.NewConversation(r.Tokenizer, r.Messages)
}
func (r *stubConversationRespository) Save(msg []conversation.Message) {
	r.Messages = append(r.Messages, msg...)
}
func (r *stubConversationRespository) Optimize() {}

func (m *stubConversationRespository) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Any("messages", m.Messages),
		slog.Any("tokenizer", m.Tokenizer),
		slog.Any("sample", some{
			Val1: "123",
			Val2: "321",
			Val3: "4123",
			Val4: "4325",
		}),
	)
}

type some struct {
	Val1 string
	Val2 string
	Val3 string
	Val4 string
}

// memory
type stubMemoryProvider struct{}

func (r *stubMemoryProvider) Snapshot(_ context.Context, _ []conversation.Message) executioncontext.Memory {
	return executioncontext.Memory{
		Semantic:         []executioncontext.SemanticData{},
		RecentEpisodes:   []executioncontext.Episode{},
		RelevantEpisodes: []executioncontext.Episode{},
		RunningMemory:    "",
	}
}

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
	// logging
	logger := logging.NewConsoleLogger()
	slog.SetDefault(logger)
	slog.SetLogLoggerLevel(slog.LevelDebug)

	// context
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM)

	defer stop()

	// configQ
	configPath := flag.String("config", "xconfig.toml", "path to config file")
	flag.Parse()

	config, err := config.LoadFile(*configPath)
	if err != nil {
		slog.Error("bad config", "error", err)
		os.Exit(1)
	}

	// root composing
	reflector := openaiadapter.NewReflector(
		openai.NewClient(
			option.WithBaseURL(config.LLM.Reflection.OpenAIURL),
			option.WithAPIKey(config.LLM.Reflection.APIKey),
		),
		config.LLM.Reflection.Model,
		config.LLM.Reflection.Extras,
	)
	contextFactory := executioncontext.NewRequestContextFactory(&stubMemoryProvider{}, reflector)

	reasoner := openaiadapter.NewReasoner(
		openai.NewClient(
			option.WithBaseURL(config.LLM.Reasoning.OpenAIURL),
			option.WithAPIKey(config.LLM.Reasoning.APIKey),
		),
		config.LLM.Reasoning.Model,
		config.LLM.Reasoning.Extras,
	)

	answerUseCase := answer.NewAnswerUseCase(
		reasoner,
		NewStubConversationRespository(),
		&stubAgentRepository{config.Agent},
		contextFactory,
		&logging.AnswerUCLogger{},
	)

	// tg bot
	bot, err := telegram.NewBot(
		answerUseCase,
		telegram.BotConfig{
			APIKey:         config.Telegram.APIKey,
			StickerSetName: config.Telegram.StickerSet,
			Logs:           config.Telegram.Logs,
		},
	)
	if err != nil {
		os.Exit(1)
	}
	go bot.Run(ctx)

	// shutdown await
	<-ctx.Done()

	// graceful shutdown
	slog.Warn("Graceful shutdown")
}
