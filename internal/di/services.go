package di

import (
	service "arch-agent/internal/app"
	"arch-agent/internal/domain/tool"
	"arch-agent/internal/infra/files"
	openaiadapter "arch-agent/internal/infra/openai"
	"arch-agent/internal/infra/tokenizer"
	"arch-agent/internal/infra/uuid"
	"context"
)

func BuildSessionService(fs *files.FileSystem, uuidGen *uuid.UUIDGenerator) *service.SessionService {
	tokenCounter := tokenizer.NewTokenizer()

	return service.NewSessionService(
		files.NewSessionFiles(fs, tokenCounter),
		uuidGen,
		tokenCounter,
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

func BuildAgentService(fs *files.FileSystem, toolSvc *service.ToolService) (*service.AgentService, error) {

	llmService, err := BuildLLMService(fs)
	if err != nil {
		return nil, err
	}

	return service.NewAgentService(
		files.NewAgentFiles(fs),
		llmService,
		toolSvc,
	), nil
}

func BuildSessionChatService(fs *files.FileSystem, sessSvc *service.SessionService, aSvc *service.AgentService) *service.SessionChatService {
	return service.NewSessionChatService(
		aSvc,
		sessSvc,
	)
}

// func BuildLiveSessionChatService(
// 	fs *files.FileSystem,
// 	sessSvc *service.SessionService,
// 	aSvc *service.AgentService,
// 	activityRepo service.ActivityRepo,
// ) *service.LiveChatService {
// 	return service.NewLiveChatService(
// 		sessSvc,
// 		activityRepo,
// 		aSvc,
// 	)
// }

func BuildTaskService(
	ctx context.Context,
	fs *files.FileSystem,
	agentService *service.AgentService,
	activityRepo service.ActivityRepo,
	uuidGenerator service.UUIDGenerator,
) (*service.TaskService, error) {

	taskRepo, err := files.NewTaskFiles(fs)
	if err != nil {
		return nil, err
	}

	executor := service.NewTaskExecutor(agentService, activityRepo)

	return service.NewTaskService(
		ctx,
		uuidGenerator,
		taskRepo,
		executor,
	)
}

func BuildApp(ctx context.Context, dataPath string, tools ...tool.Tool) (*service.App, error) {
	fs, err := files.NewFS(dataPath)
	if err != nil {
		return nil, err
	}

	toolService := service.NewToolService()

	uuidGen := uuid.NewUUIDGenerator()

	sessSvc := BuildSessionService(fs, uuidGen)
	agentSvc, err := BuildAgentService(fs, toolService)
	if err != nil {
		return nil, err
	}
	activityRepo := files.NewActivityFiles(fs)

	taskSvc, err := BuildTaskService(
		ctx,
		fs,
		agentSvc,
		activityRepo,
		uuidGen,
	)
	if err != nil {
		return nil, err
	}

	// liveChatSvc := BuildLiveSessionChatService(fs, sessSvc, agentSvc, activityRepo)

	sessionChatSvc := service.NewSessionChatService(agentSvc, sessSvc)

	a2aFiles, err := files.NewA2AFiles(fs)
	if err != nil {
		return nil, err
	}

	a2aSevice := service.NewA2AService(
		a2aFiles,
		agentSvc,
		sessionChatSvc,
		// liveChatSvc,
	)

	tools = append(tools, service.NewAddTaskTool(taskSvc))
	tools = append(tools, service.NewCallAgentTool(a2aSevice))
	tools = append(tools, service.NewGetAgentsTool(a2aSevice))
	toolService.AddTools(tools...)

	return &service.App{
		A2A:            a2aSevice,
		SessionChatSvc: sessionChatSvc,
		// LiveChatSvc:    liveChatSvc,
		AgentSvc: agentSvc,
	}, nil
}
