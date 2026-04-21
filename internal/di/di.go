package di

import (
	"arch-agent/internal/app/activity"
	"arch-agent/internal/app/knowledge"
	"arch-agent/internal/app/reasoning"
	"arch-agent/internal/app/reflection"
	"arch-agent/internal/app/session"
	"arch-agent/internal/app/summarization"
	"arch-agent/internal/app/usecases/answer"
	"arch-agent/internal/infra/config"
	"arch-agent/internal/infra/llm"
	mcpadapter "arch-agent/internal/infra/mcp"
	openaiadapter "arch-agent/internal/infra/openai"
	activityadapter "arch-agent/internal/infra/storage/activity"
	knowledgeadapter "arch-agent/internal/infra/storage/knowledge"
	sessionadapter "arch-agent/internal/infra/storage/session"
	"arch-agent/internal/infra/storage/transcribtions"
	"arch-agent/internal/infra/tokenizer"
	"arch-agent/internal/infra/tools"
	"log/slog"
	"path/filepath"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

func CreateSummarizationService(cfg *config.LLM) *summarization.Service {
	return summarization.NewService(
		CreateReasoningService(cfg, nil),
		llm.NewSummaryPrompt(),
	)
}

func CreateReflectionService(cfg *config.LLM, personality string) *reflection.Service {
	emoService := reflection.NewEmotionalService()
	serviceTools := tools.BoostEmotion(emoService)
	return reflection.NewService(
		personality,
		llm.NewReflectionPrompt(),
		emoService,
		CreateReasoningService(cfg, llm.NewToolCallRecivier([]llm.Tool{serviceTools})),
	)
}

func CreateReasoningService(cfg *config.LLM, recivier reasoning.ToolCallRecivier) *reasoning.Service {
	return reasoning.NewService(
		10,
		CreateReasonerFromConfig(cfg),
		recivier,
	)
}

func CreateReasonerFromConfig(cfg *config.LLM) *openaiadapter.Reasoner {
	return openaiadapter.NewReasonerFromConfig(openaiadapter.ReasonerConfig{
		Client: openai.NewClient(
			option.WithBaseURL(cfg.OpenAIURL),
			option.WithAPIKey(cfg.APIKey),
		),
		Model:            cfg.Model,
		ReasoningEffort:  cfg.ReasoningEffort,
		ToolChoice:       cfg.ToolChoice,
		TopP:             cfg.TopP,
		FrequencyPenalty: cfg.FrequencyPenalty,
		PresencePenalty:  cfg.PresencePenalty,
		Temperature:      cfg.Temperature,
		Extras:           cfg.Extras,
	})
}

func CreateActivityService(summarizerConfig *config.LLM, absolutePath string) (*activity.Service, error) {
	summarizationService := CreateSummarizationService(summarizerConfig)
	activityfiles, err := activityadapter.NewActivityFiles(absolutePath + "/activities")
	if err != nil {
		return nil, err
	}

	return activity.NewService(
		summarizationService,
		activityfiles,
	), nil
}

func CreateSessionService(activityService *activity.Service, absolutePath string) (*session.Service, error) {

	sessionfiles, err := sessionadapter.NewFileSessionRepository(absolutePath)
	if err != nil {
		return nil, err
	}
	transcribtionfiles, err := transcribtions.NewJSONLTranscriber(absolutePath + "/transciptions")
	if err != nil {
		return nil, err
	}

	return session.NewSessionService(
		sessionfiles,
		transcribtionfiles,
		tokenizer.NewTokenizer(),
		activityService,
	), nil
}

func CreateKnowledgeService(path string) (*knowledge.Service, error) {
	adapter, err := knowledgeadapter.New(path + "/knowledges")
	if err != nil {
		return nil, err
	}
	return knowledge.NewService(adapter), err
}

// Agent repo
type stubAgentRepository struct {
	*config.Agent
}

func NewStubAgentRepo(cfg *config.Agent) *stubAgentRepository {
	return &stubAgentRepository{cfg}
}

func (s *stubAgentRepository) Role() string        { return s.Agent.Role }
func (s *stubAgentRepository) Preferences() string { return s.Agent.Preferences }
func (s *stubAgentRepository) Personality() string { return s.Agent.Personality }
func (s *stubAgentRepository) KeyPhrases() string  { return s.Agent.Keyphrases }
func (s *stubAgentRepository) BannedSlang() string { return s.Agent.BannedSlang }

func NewAnswerUseCase(cfg config.Config, dataPath string, toolBundle []llm.Tool) (*answer.AnswerUseCase, error) {
	absolutePath, _ := filepath.Abs(dataPath)

	knowledgeService, err := CreateKnowledgeService(absolutePath)
	if err != nil {
		return nil, err
	}

	activityService, err := CreateActivityService(cfg.LLMS.Summarization, absolutePath)
	if err != nil {
		return nil, err
	}

	sessionService, err := CreateSessionService(activityService, absolutePath)
	if err != nil {
		return nil, err
	}

	agentRepo := NewStubAgentRepo(cfg.Agent)
	contextAssembler := answer.NewContextAssembler(
		llm.NewAnswerPrompt(),
		CreateReflectionService(cfg.LLMS.Reflection, agentRepo.Personality()),
		agentRepo,
		activityService,
		knowledgeService,
	)

	mcpRecivier := mcpadapter.NewMCPRecivier(cfg.Agent.Name)

	// connect internal mcp server
	answerTools := append(toolBundle, tools.ReadKnowledge(knowledgeService))
	mcpserver := mcpadapter.NewInternalServer("internal")
	mcpserver.AddTools(answerTools)
	if err := mcpRecivier.AddIntenalServer(mcpserver); err != nil {
		return nil, err
	}

	// connect external mcp servers
	for _, s := range cfg.MCP.Servers {
		if err := mcpRecivier.AddHTTPServer(s); err != nil {
			slog.Error("mcp server connection", "error", err)
		}
	}

	return answer.NewAnswerUseCase(
		CreateReasoningService(cfg.LLMS.Reasoning, mcpRecivier),
		sessionService,
		contextAssembler,
	), nil
}
