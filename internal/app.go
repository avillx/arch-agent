package app

import (
	"arch-agent/internal/a2a"
	"arch-agent/internal/agent"
	"arch-agent/internal/chat"
	"arch-agent/internal/files"
	"arch-agent/internal/llm"
	"arch-agent/internal/openai"
	"arch-agent/internal/session"
	"arch-agent/internal/task"
	"arch-agent/internal/telegram"
	"arch-agent/internal/tokenizer"
	"arch-agent/internal/tools"
	"arch-agent/internal/uuid"
	"context"
)

type App struct {
	A2ASvc                 *a2a.Service
	SessionChatSvc         *session.SessionChatService
	ToolSvc                *tools.Service
	LLMSvc                 *llm.Service
	ChatSvc                *chat.Service
	TelegramOrchestra      *telegram.BotOrchestrator
	TaskSvc                *task.TaskService
	SessionSvc             *session.SessionService
	TelegramA2AInterceptor *telegram.A2AInterceptor
}

func (a *App) Run(ctx context.Context) {
	go a.TelegramA2AInterceptor.Run(ctx)
	a.TelegramOrchestra.Run(ctx)
}

func BuildSessionService(fs *files.FileSystem, uuidGen *uuid.UUIDGenerator) *session.SessionService {
	tokenCounter := tokenizer.NewTokenizer()

	return session.NewSessionService(
		files.NewSessionFiles(fs, tokenCounter),
		uuidGen,
		tokenCounter,
	)
}

func BuildLLMService(fs *files.FileSystem) (*llm.Service, error) {

	secretsRepo, err := files.NewSecretsFiles(fs)
	if err != nil {
		return nil, err
	}

	return llm.NewLLMService(
		files.NewLLMFiles(fs),
		openai.NewOpenAIFactory(secretsRepo),
	)
}

func BuildChatService(fs *files.FileSystem, toolSvc *tools.Service) (*chat.Service, error) {

	llmService, err := BuildLLMService(fs)
	if err != nil {
		return nil, err
	}

	return chat.NewService(
		files.NewAgentFiles(fs),
		llmService,
		toolSvc,
	), nil
}

func BuildSessionChatService(fs *files.FileSystem, sessSvc *session.SessionService, chatSvc *chat.Service) *session.SessionChatService {
	return session.NewSessionChatService(
		chatSvc,
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
	chatService *chat.Service,
	activityRepo agent.ActivityRepo,
	uuidGenerator task.UUIDGenerator,
) (*task.TaskService, error) {

	taskRepo, err := files.NewTaskFiles(fs)
	if err != nil {
		return nil, err
	}

	executor := task.NewTaskExecutor(chatService, activityRepo)

	return task.NewTaskService(
		ctx,
		uuidGenerator,
		taskRepo,
		executor,
	)
}

func BuildApp(ctx context.Context, dataPath string, groupID int64, botCfgs ...telegram.BotConfig) (*App, error) {

	botOrchestra, err := telegram.NewBotOrchestrator(botCfgs...)
	if err != nil {
		return nil, err
	}

	fs, err := files.NewFS(dataPath)
	if err != nil {
		return nil, err
	}

	toolService := tools.NewService()

	uuidGen := uuid.NewUUIDGenerator()

	sessSvc := BuildSessionService(fs, uuidGen)
	chatSvc, err := BuildChatService(fs, toolService)
	if err != nil {
		return nil, err
	}
	activityRepo := files.NewActivityFiles(fs)

	taskSvc, err := BuildTaskService(
		ctx,
		fs,
		chatSvc,
		activityRepo,
		uuidGen,
	)
	if err != nil {
		return nil, err
	}

	// liveChatSvc := BuildLiveSessionChatService(fs, sessSvc, agentSvc, activityRepo)

	sessionChatSvc := session.NewSessionChatService(chatSvc, sessSvc)

	a2aFiles, err := files.NewA2AFiles(fs)
	if err != nil {
		return nil, err
	}

	a2aSvc := a2a.NewService(
		a2aFiles,
		chatSvc,
		sessionChatSvc,
		// liveChatSvc,
	)

	toolService.AddTools(
		tools.NewAddTaskTool(taskSvc),
		tools.NewCallAgentTool(a2aSvc),
		tools.NewGetAgentsTool(a2aSvc),
		tools.NewSendMessageTool(nil),
		tools.NewSendStickerTool(nil),
	)

	botOrchestra.WireSessionService(sessionChatSvc)

	return &App{
		A2ASvc:         a2aSvc,
		SessionChatSvc: sessionChatSvc,
		// LiveChatSvc:    liveChatSvc,
		ChatSvc: chatSvc,
		ToolSvc: toolService,
		// LLMSvc: ,
		TelegramOrchestra:      botOrchestra,
		TaskSvc:                taskSvc,
		SessionSvc:             sessSvc,
		TelegramA2AInterceptor: telegram.NewA2AInterceptor(groupID, botOrchestra, a2aSvc),
	}, nil
}
