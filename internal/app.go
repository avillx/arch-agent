package app

import (
	"arch-agent/internal/a2a"
	"arch-agent/internal/agent"
	"arch-agent/internal/chat"
	"arch-agent/internal/cron"
	"arch-agent/internal/files"
	"arch-agent/internal/openai"
	"arch-agent/internal/runtime"
	"arch-agent/internal/searxng"
	"arch-agent/internal/session"
	"arch-agent/internal/task"
	"arch-agent/internal/telegram"
	"arch-agent/internal/tokenizer"
	"arch-agent/internal/tools"
	"arch-agent/internal/tools/fetch"
	"arch-agent/internal/tools/search"
	"arch-agent/internal/uuid"
	"context"
	"errors"
)

type App struct {
	A2ASvc            *a2a.Service
	ToolSvc           *tools.Service
	runtime           *runtime.AgentRuntime
	TelegramOrchestra *telegram.BotOrchestrator
	TaskSvc           *task.TaskService
	SessionSvc        *session.SessionService
	// TelegramA2AInterceptor *telegram.A2AInterceptor
}

func (a *App) Run(ctx context.Context) {
	// go a.TelegramA2AInterceptor.Run(ctx)
	a.TelegramOrchestra.Run(ctx)
}

func BuildModelsRepo(fs *files.FileSystem) (agent.ModelRepository, error) {

	secretsRepo, err := files.NewSecretsFiles(fs)
	if err != nil {
		return nil, err
	}

	openaiFactory := openai.NewOpenAIFactory(secretsRepo)

	return files.NewModelFiles(fs,
		files.WithFactory(openaiFactory.Type(), openaiFactory.Produce),
	)
}

func BuildTaskService(
	ctx context.Context,
	fs *files.FileSystem,
	sessionSvc *session.SessionService,
	chatSvc *chat.ChatService,
) (*task.TaskService, error) {

	taskRepo, err := files.NewTaskFiles(fs, func(s string) (task.Cron, error) { return cron.NewRobfigCron(s) })
	if err != nil {
		return nil, err
	}

	executor := task.NewTaskExecutor(
		sessionSvc,
		chatSvc,
	)

	return task.NewTaskService(
		ctx,
		taskRepo,
		executor,
	)
}

func BuildApp(ctx context.Context, dataPath, searchHostScheme, searchHost string, groupID int64, botCfgs ...telegram.BotConfig) (*App, error) {

	botOrchestra, err := telegram.NewBotOrchestrator(botCfgs...)
	if err != nil {
		return nil, err
	}

	fs, err := files.NewFS(dataPath)
	if err != nil {
		return nil, err
	}

	modelRepo, err := BuildModelsRepo(fs)
	if err != nil {
		return nil, err
	}
	observerModel, err := modelRepo.Get("observer")
	if err != nil {
		return nil, errors.New("has no observer model")
	}

	tokenizer := tokenizer.NewTokenizer()

	agentRepo := files.NewAgentFiles(fs)

	sessSvc := session.NewSessionService(
		files.NewSessionFiles(fs, tokenizer),
		uuid.NewUUIDGenerator(),
		tokenizer,
	)

	activityRepo := files.NewActivityFiles(fs)
	observer := runtime.NewObserver(observerModel, activityRepo, tokenizer)

	runtime := runtime.NewAgentRuntime(observer)

	toolService := tools.NewService()
	chatSvc := chat.NewChatService(agentRepo, sessSvc, modelRepo, toolService, runtime)

	taskSvc, err := BuildTaskService(
		ctx,
		fs,
		sessSvc,
		chatSvc,
	)
	if err != nil {
		return nil, err
	}

	// a2aFiles, err := files.NewA2AFiles(fs,)
	// if err != nil {
	// 	return nil, err
	// }

	a2aSvc := a2a.NewService(chatSvc, sessSvc)

	searx := searxng.NewSearXSearch(searchHostScheme, searchHost)

	toolService.AddTools(
		// filesystem tools
		tools.NewListDirTool(fs),
		tools.NewReadFileTool(fs),
		tools.NewWriteFileTool(fs),
		tools.NewEditFileTool(fs),
		tools.NewMoveFileTool(fs),
		tools.NewDeleteFileTool(fs),
		tools.NewSearchFilesTool(fs),

		// task scheduling tools
		tools.NewToggleTaskTool(taskSvc),
		tools.NewGetTasksTool(taskSvc),
		tools.NewAddTaskTool(taskSvc, func(s string) (task.Cron, error) { return cron.NewRobfigCron(s) }),

		// a2a tools
		tools.NewCallAgentTool(a2aSvc, agentRepo),
		// tools.NewGetAgentsTool(a2aSvc),

		// telegram tools
		tools.NewSendMessageTool(botOrchestra),
		tools.NewSendStickerTool(botOrchestra),
		tools.NewGetStickersTool(botOrchestra),

		// web tools
		fetch.NewFetchTool(),
		search.NewWebSearchTool(searx),
	)

	botOrchestra.Wire(
		sessSvc,
		chatSvc,
	)

	return &App{
		A2ASvc:            a2aSvc,
		runtime:           runtime,
		TelegramOrchestra: botOrchestra,
		TaskSvc:           taskSvc,
		SessionSvc:        sessSvc,
		// TelegramA2AInterceptor: telegram.NewA2AInterceptor(groupID, botOrchestra, a2aSvc),
	}, nil
}
