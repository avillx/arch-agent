package di

import (
	service "arch-agent/internal/app"
	"arch-agent/internal/app/usecases"
	"arch-agent/internal/infra/files"
	openaiadapter "arch-agent/internal/infra/openai"
	"arch-agent/internal/infra/tokenizer"
	"arch-agent/internal/infra/uuid"
)

func BuildSessionService(fs *files.FileSystem) *service.SessionService {
	return service.NewSessionService(
		files.NewSessionFiles(fs),
		uuid.NewUUIDGenerator(),
		tokenizer.NewTokenizer(),
	)
}

func BuildLLMService(fs *files.FileSystem) (*service.LLMService, error) {

	secretsRepo, err := files.NewSecretsFiles(fs)
	if err != nil {
		return nil, err
	}

	return service.NewLLMService(
		files.NewLLMFiles(fs),
		openaiadapter.NewOpenAIFactory(secretsRepo),
	)
}

func BuildAgentService(fs *files.FileSystem, toolServers ...service.ToolServer) (*service.AgentService, error) {

	llmService, err := BuildLLMService(fs)
	if err != nil {
		return nil, err
	}

	toolService := service.NewToolService()
	for _, tc := range toolServers {
		toolService.Connect(tc)
	}

	return service.NewAgentService(
		files.NewAgentFiles(fs),
		llmService,
		toolService,
	), nil
}

func BuildUseCase(fs *files.FileSystem, toolServers ...service.ToolServer) (*usecases.ChatLoop, error) {

	agentService, err := BuildAgentService(fs, toolServers...)
	if err != nil {
		return nil, err
	}

	return usecases.NewChatLoop(
		agentService,
		BuildSessionService(fs),
	), nil
}
