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
	tiktoken "arch-agent/internal/tokenizer"
	"arch-agent/internal/tools"
	"arch-agent/internal/tools/fetch"
	fstools "arch-agent/internal/tools/fs"
	"arch-agent/internal/tools/search"
	tasktools "arch-agent/internal/tools/task"
	"arch-agent/internal/tools/todo"
	"arch-agent/internal/uuid"
	"context"
	"errors"
)

type App struct {
	A2ASvc            *a2a.Service
	ToolSvc           *tools.Service
	runtime           *runtime.AgentRuntime
	TelegramOrchestra *telegram.BotOrchestrator
	TaskSvc           *task.Service
	SessionSvc        *session.Service
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

func BuildTaskSvc(
	ctx context.Context,
	fs *files.FileSystem,
	sessionSvc *session.Service,
	chatSvc *chat.Service,
) (*task.Service, error) {

	taskRepo, err := files.NewTaskFiles(fs, func(s string) (task.Cron, error) { return cron.NewRobfigCron(s) })
	if err != nil {
		return nil, err
	}

	executor := task.NewExecutor(
		sessionSvc,
		chatSvc,
	)

	return task.NewService(
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

	tt := tiktoken.New()

	agentRepo := files.NewAgentFiles(fs)

	sessSvc := session.NewService(
		files.NewSessionFiles(fs, tt),
		uuid.NewUUIDGenerator(),
		tt,
	)

	activityRepo := files.NewActivityFiles(fs)
	observer := runtime.NewObserver(observerModel, activityRepo, tt)

	skillFiles := files.NewSkillFiles(fs)
	contextAssembler := runtime.NewContextAssembler(skillFiles)
	rt := runtime.NewAgentRuntime(observer, contextAssembler, runtime.NewCompactor(tt))

	toolSvc := tools.NewService()
	chatSvc := chat.NewService(agentRepo, sessSvc, modelRepo, toolSvc, rt)

	taskSvc, err := BuildTaskSvc(
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

	todoStorage := todo.NewInMemoryStore()

	toolSvc.AddTools(
		// filesystem tools
		fstools.NewListDirTool(fs),
		fstools.NewReadFileTool(fs),
		fstools.NewWriteFileTool(fs),
		fstools.NewEditFileTool(fs),
		fstools.NewMoveFileTool(fs),
		fstools.NewDeleteTool(fs),
		fstools.NewSearchFilesTool(fs),

		// task scheduling tools
		tasktools.NewToggleTaskTool(taskSvc),
		tasktools.NewGetTasksTool(taskSvc),
		tasktools.NewAddTaskTool(taskSvc, func(s string) (task.Cron, error) { return cron.NewRobfigCron(s) }),

		// callagent tools
		tools.NewCallAgentTool(a2aSvc, agentRepo),

		// // telegram tools
		// tgtools.NewSendMessageTool(botOrchestra),
		// tgtools.NewSendStickerTool(botOrchestra),

		// web tools
		fetch.NewFetchTool(),
		search.NewWebSearchTool(searx),

		// todo tools
		&todo.CreateTodoTool{Store: todoStorage},
		&todo.ListTodoTool{Store: todoStorage},
		&todo.UpdateTodoTool{Store: todoStorage},
	)

	botOrchestra.Wire(
		sessSvc,
		chatSvc,
	)

	return &App{
		A2ASvc:            a2aSvc,
		runtime:           rt,
		TelegramOrchestra: botOrchestra,
		TaskSvc:           taskSvc,
		SessionSvc:        sessSvc,
		// TelegramA2AInterceptor: telegram.NewA2AInterceptor(groupID, botOrchestra, a2aSvc),
	}, nil
}
