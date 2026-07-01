package app

import (
	"arch-agent/internal/a2a"
	"arch-agent/internal/agent"
	"arch-agent/internal/chat"
	"arch-agent/internal/cron"
	"arch-agent/internal/files"
	"arch-agent/internal/mcp"
	"arch-agent/internal/model"
	"arch-agent/internal/openai"
	"arch-agent/internal/runtime"
	"arch-agent/internal/runtime/memory"
	"arch-agent/internal/searxng"
	"arch-agent/internal/session"
	"arch-agent/internal/task"
	"arch-agent/internal/telegram"
	"arch-agent/internal/tools"
	"arch-agent/internal/tools/fetch"
	fstools "arch-agent/internal/tools/fs"
	"arch-agent/internal/tools/search"
	tasktools "arch-agent/internal/tools/task"
	"arch-agent/internal/tools/todo"
	"arch-agent/internal/uuid"
	"context"
	"errors"
	"fmt"
	"slices"
)

// TODO:
// rename package to Wire
// Delete App as datastruct and replace for return server
// this package should assemble all services
// api.NewServer responsobility is assemble a server not services

type AppConfig struct {
	DataPath         string
	SearchHostScheme string
	SearchHost       string
	TelegramGroupID  int64
	BotConfigs       []telegram.BotConfig
}

type App struct {
	runtime           *runtime.AgentRuntime
	TelegramOrchestra *telegram.BotOrchestrator
	TaskSvc           *task.Service
	Memory            *memory.Memory
	MCPSvc            *mcp.Service
}

func BuildModelsRepo(fs *files.FileSystem) (agent.ModelRepository, error) {

	secretsRepo, err := files.NewSecretsFiles(fs)
	if err != nil {
		return nil, err
	}

	modelFiles, err := files.NewModelFiles(fs)
	if err != nil {
		return nil, err
	}

	openaiFactory := openai.NewOpenAIModelFactory(secretsRepo)

	return model.NewService(modelFiles, openaiFactory)
}

func BuildTaskSvc(
	ctx context.Context,
	fs *files.FileSystem,
	sessionSvc *session.Service,
	chatSvc *chat.Service,
) (*task.Service, error) {

	taskRepo, err := files.NewTaskFiles(fs)
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
		func(s string) (task.Cron, error) { return cron.NewRobfigCron(s) },
		executor,
	)
}

func BuildApp(ctx context.Context, cfg AppConfig) (*App, error) {

	fs, err := files.NewFS(cfg.DataPath)
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

	agentRepo := files.NewAgentFiles(fs)

	activityRepo := files.NewActivityFiles(fs)
	sessSvc := session.NewService(
		files.NewSessionFiles(fs),
		uuid.NewUUIDGenerator(),
		activityRepo,
	)

	observer := runtime.NewObserver(observerModel, activityRepo)

	skillFiles := files.NewSkillFiles(fs)
	contextAssembler := runtime.NewContextAssembler(skillFiles, activityRepo, files.NewMemoryFiles(fs))
	rt := runtime.NewAgentRuntime(observer, contextAssembler)

	toolSvc := tools.NewService()

	mcpRepo := files.NewMCPFiles(fs)
	mcpSvc, err := mcp.NewService(ctx, toolSvc, mcpRepo)
	if err != nil {
		return nil, fmt.Errorf("build mcp service: %w", err)
	}

	chatExecutor := chat.NewExecutor(agentRepo, sessSvc, modelRepo, toolSvc, rt)
	chatSvc := chat.NewService(chatExecutor)

	taskSvc, err := BuildTaskSvc(
		ctx,
		fs,
		sessSvc,
		chatSvc,
	)
	if err != nil {
		return nil, err
	}

	a2aSvc := a2a.NewService(chatSvc, sessSvc)

	searx := searxng.NewSearXSearch(cfg.SearchHostScheme, cfg.SearchHost)

	todoStorage := todo.NewInMemoryStore()

	// Tools
	fsTools := []agent.Tool{
		fstools.NewListDirTool(fs, agentRepo),
		fstools.NewReadFileTool(fs, agentRepo),
		fstools.NewWriteFileTool(fs, agentRepo),
		fstools.NewEditFileTool(fs, agentRepo),
		fstools.NewMoveFileTool(fs, agentRepo),
		fstools.NewDeleteTool(fs, agentRepo),
		fstools.NewSearchFilesTool(fs, agentRepo),
	}

	todoTools := []agent.Tool{
		&todo.CreateTodoTool{Store: todoStorage},
		&todo.ListTodoTool{Store: todoStorage},
		&todo.UpdateTodoTool{Store: todoStorage},
	}

	taskControlTools := []agent.Tool{
		tasktools.NewToggleTaskTool(taskSvc),
		tasktools.NewGetTasksTool(taskSvc),
		tasktools.NewAddTaskTool(taskSvc),
	}

	webTools := []agent.Tool{
		fetch.NewFetchTool(),
		search.NewWebSearchTool(searx),
	}

	callAgentTool := tools.NewCallAgentTool(a2aSvc, agentRepo)

	toolSvc.AddTools(
		append(slices.Concat(fsTools, todoTools, taskControlTools, webTools), callAgentTool)...,
	)

	// memory consolidation
	memoryConsolidator := memory.NewMemory(
		agentRepo,
		rt,
		append(fsTools, todoTools...),
	)
	consolidatorModel, err := modelRepo.Get("consolidator")
	if err != nil {
		return nil, err
	}
	memoryConsolidator.SetModel(consolidatorModel)

	// Telegram Bots
	botOrchestra, err := telegram.NewBotOrchestrator(cfg.BotConfigs...)
	if err != nil {
		return nil, err
	}

	botOrchestra.Wire(
		sessSvc,
		chatSvc,
		memoryConsolidator,
		mcpSvc,
	)

	return &App{
		runtime:           rt,
		TelegramOrchestra: botOrchestra,
		TaskSvc:           taskSvc,
		Memory:            memoryConsolidator,
		MCPSvc:            mcpSvc,
	}, nil
}
