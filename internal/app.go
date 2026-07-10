package app

import (
	"arch-agent/internal/a2a"
	"arch-agent/internal/agent"
	"arch-agent/internal/chat"
	"arch-agent/internal/cron"
	"arch-agent/internal/files"
	"arch-agent/internal/hooks"
	"arch-agent/internal/mcp"
	"arch-agent/internal/model"
	"arch-agent/internal/openai"
	"arch-agent/internal/runtime"
	"arch-agent/internal/runtime/memory"
	"arch-agent/internal/session"
	"arch-agent/internal/task"
	"arch-agent/internal/telegram"
	"arch-agent/internal/tools"
	"arch-agent/internal/tools/fetch"
	fstools "arch-agent/internal/tools/fs"
	tasktools "arch-agent/internal/tools/task"
	"arch-agent/internal/tools/todo"
	"arch-agent/internal/uuid"
	"context"
	"errors"
	"fmt"
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
	agentRepo agent.Repo,
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
		taskRepo,
		executor,
		func(s string) (task.Cron, error) { return cron.NewRobfigCron(s) },
		agentRepo,
	)
}

func BuildMemoryConsolidator(
	fs *files.FileSystem,
	rt *runtime.AgentRuntime,
	modelRepo agent.ModelRepository,
	agentRepo agent.Repo,
	additionalTools []agent.Tool,
) (*memory.Memory, error) {

	fsTools := []agent.Tool{
		fstools.NewListDirTool(fs),
		fstools.NewReadFileTool(fs),
		fstools.NewWriteFileTool(fs),
		fstools.NewEditFileTool(fs),
		fstools.NewMoveFileTool(fs),
		fstools.NewDeleteTool(fs),
		fstools.NewSearchFilesTool(fs),
	}

	consolidatorModel, err := modelRepo.Get("consolidator")
	if err != nil {
		return nil, fmt.Errorf("'consolidator' model: %w", err)
	}

	memory, err := memory.NewMemory(
		agentRepo,
		rt,
		append(fsTools, additionalTools...),
		consolidatorModel,
	)
	if err != nil {
		return nil, fmt.Errorf("memory consolidator: %w", err)
	}

	return memory, err
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

	todoStorage := todo.NewInMemoryStore()
	harnessFactory := hooks.ProduceHarnessFactory(fs, todoStorage)

	chatExecutor := chat.NewExecutor(agentRepo, sessSvc, modelRepo, toolSvc, rt, harnessFactory)
	chatSvc := chat.NewService(chatExecutor)

	taskSvc, err := BuildTaskSvc(
		ctx,
		fs,
		sessSvc,
		chatSvc,
		agentRepo,
	)
	if err != nil {
		return nil, err
	}

	a2aSvc := a2a.NewService(chatSvc, sessSvc)

	toolSvc.Connect(
		"filesystem",
		tools.NewBuildInToolServer(
			fstools.WithInstruction(fstools.NewReadFileTool(fs)),
			fstools.NewListDirTool(fs),
			fstools.NewWriteFileTool(fs),
			fstools.NewEditFileTool(fs),
			fstools.NewMoveFileTool(fs),
			fstools.NewDeleteTool(fs),
			fstools.NewSearchFilesTool(fs),
		),
	)

	toolSvc.Connect(
		"tasks",
		tools.NewBuildInToolServer(
			tasktools.NewAddTaskTool(taskSvc),
			tasktools.NewGetTasksTool(taskSvc),
			tasktools.NewEditTaskTool(taskSvc),
			tasktools.NewDeleteTasksTool(taskSvc),
		),
	)

	toolSvc.Connect(
		"web",
		tools.NewBuildInToolServer(
			fetch.NewFetchTool(),
		),
	)

	toolSvc.Connect(
		"agent",
		tools.NewBuildInToolServer(
			tools.NewCallAgentTool(a2aSvc, agentRepo),
		),
	)

	todoTools := tools.NewBuildInToolServer(
		&todo.CreateTodoTool{Store: todoStorage},
		&todo.ListTodoTool{Store: todoStorage},
		&todo.UpdateTodoTool{Store: todoStorage},
	)
	toolSvc.Connect("todo", todoTools)

	// Telegram Bots
	botOrchestra, err := telegram.NewBotOrchestrator(cfg.BotConfigs...)
	if err != nil {
		return nil, err
	}

	memoryConsolidator, err := BuildMemoryConsolidator(fs, rt, modelRepo, agentRepo, todoTools.Tools())
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
