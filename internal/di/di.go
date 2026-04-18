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
	openaiadapter "arch-agent/internal/infra/openai"
	activityadapter "arch-agent/internal/infra/storage/activity"
	knowledgeadapter "arch-agent/internal/infra/storage/knowledge"
	sessionadapter "arch-agent/internal/infra/storage/session"
	"arch-agent/internal/infra/storage/transcribtions"
	"arch-agent/internal/infra/tokenizer"
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
	return reflection.NewService(
		personality,
		llm.NewReflectionPrompt(),
		CreateReasoningService(cfg, nil),
	)
}

func CreateReasoningService(cfg *config.LLM, tools []llm.Tool) *reasoning.Service {
	return reasoning.NewService(
		20,
		CreateReasonerFromConfig(cfg),
		llm.NewToolCallRecivier(tools),
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

func CreateActivityService(summarizerConfig *config.LLM, absolutePath string) *activity.Service {
	summarizationService := CreateSummarizationService(summarizerConfig)
	return activity.NewService(
		summarizationService,
		activityadapter.NewActivityFiles(absolutePath+"/activities"),
	)
}

func CreateSessionService(activityService *activity.Service, absolutePath string) *session.Service {

	return session.NewSessionService(
		// data dir
		sessionadapter.NewFileSessionRepository(absolutePath),
		transcribtions.NewJSONLTranscriber(absolutePath+"/transciptions"),
		tokenizer.NewTokenizer(),
		activityService,
	)
}

func CreateKnowledgeService(path string) *knowledge.Service {
	adapter := knowledgeadapter.New(path + "/knowledges")
	return knowledge.NewService(adapter)
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

func NewAnswerUseCase(cfg config.Config, dataPath string, tools []llm.Tool) *answer.AnswerUseCase {
	absolutePath, _ := filepath.Abs(dataPath)

	knowledgeService := CreateKnowledgeService(absolutePath)
	activityService := CreateActivityService(cfg.LLMS.Summarization, absolutePath)
	sessionService := CreateSessionService(activityService, absolutePath)

	agentRepo := NewStubAgentRepo(cfg.Agent)
	contextAssembler := answer.NewContextAssembler(
		llm.NewAnswerPrompt(),
		CreateReflectionService(cfg.LLMS.Reflection, agentRepo.Personality()),
		agentRepo,
		activityService,
		knowledgeService,
	)

	return answer.NewAnswerUseCase(
		CreateReasoningService(cfg.LLMS.Reasoning, tools), // append knowledgeService.ReadTool()
		sessionService,
		contextAssembler,
	)
}
