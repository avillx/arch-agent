package app

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/chat"
	"arch-agent/internal/cron"
	"arch-agent/internal/files"
	"arch-agent/internal/hooks"
	"arch-agent/internal/mcp"
	"arch-agent/internal/memory"
	"arch-agent/internal/model"
	"arch-agent/internal/openai"
	"arch-agent/internal/runtime"
	"arch-agent/internal/session"
	"arch-agent/internal/subagent"
	"arch-agent/internal/task"
	"arch-agent/internal/telegram"
	"arch-agent/internal/tools"
	"arch-agent/internal/tools/fetch"
	fstools "arch-agent/internal/tools/fs"
	"arch-agent/internal/tools/shell"
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
	DataPath           string
	TelegramGroupID    int64
	ConsolidationModel string
	ObserverModel      string
	ShellEnv           []string
	BotConfigs         []telegram.BotConfig
}

type App struct {
	runtime           *runtime.AgentRuntime
	TelegramOrchestra *telegram.BotOrchestrator
	TaskSvc           *task.Service
	Memory            *memory.Memory
	ToolsSvc          *tools.Service
	MemoryRepo        agent.MemoryRepo
	MemoryIndexer     agent.MemoryIndexer
	MemorySvc         *memory.Memory
	MCPSvc            *mcp.Service
	ChatSvc           *chat.Service
	SessionSvc        *session.Service
	ActivityRepo      agent.ActivityRepo
	AgentRepo         agent.Repo
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
	todoStorage todo.Store,
	model agent.Model,
	agentRepo agent.Repo,
	additionalTools []agent.ToolServer,
	indexer agent.MemoryIndexer,
) (*memory.Memory, error) {

	fsToolsSrv := memory.NewInstuctFS(fs.Cwd(), fstools.NewRawFileSystemToolServer(fs))

	memory, err := memory.NewMemory(
		agentRepo,
		rt,
		append(additionalTools, fsToolsSrv),
		model,
		hooks.NewMemoryHarness(fs, todoStorage, indexer),
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

	secretsRepo, err := files.NewSecretsFiles(fs)
	if err != nil {
		return nil, err
	}

	openaiFactory := openai.NewOpenAIModelFactory(secretsRepo)

	modelsSvc := model.NewModelService(openaiFactory)

	providerFiles := files.NewProviderFiles(fs)

	// providerSvc
	if _, err = model.NewProviderService(modelsSvc, providerFiles); err != nil {
		return nil, err
	}

	observerModel, err := modelsSvc.Get(cfg.ObserverModel)
	if err != nil {
		return nil, errors.New("has no observer model")
	}

	agentRepo := files.NewAgentFiles(fs)

	activityRepo := files.NewActivityFiles(fs)
	sessSvc := session.NewService(
		files.NewSessionFiles(fs),
		uuid.NewUUIDGenerator(),
	)

	observer := runtime.NewObserver(observerModel, activityRepo)

	skillFiles := files.NewSkillFiles(fs)
	memoryFiles := files.NewMemoryFiles(fs)
	contextAssembler := runtime.NewContextAssembler(skillFiles, activityRepo, memoryFiles)
	rt := runtime.NewAgentRuntime(observer, contextAssembler)

	toolSvc := tools.NewService()

	mcpRepo := files.NewMCPFiles(fs)
	mcpSvc, err := mcp.NewService(ctx, toolSvc, mcpRepo)
	if err != nil {
		return nil, fmt.Errorf("build mcp service: %w", err)
	}

	todoStorage := todo.NewInMemoryStore()
	agentHarness, err := hooks.NewAgentHarness(fs, todoStorage)
	if err != nil {
		return nil, err
	}

	chatExecutor := chat.NewExecutor(agentRepo, sessSvc, modelsSvc, toolSvc, rt, agentHarness)
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

	subagentSvc := subagent.NewService(agentHarness, rt, toolSvc, agentRepo, modelsSvc)

	toolSvc.Connect("filesystem", fstools.NewFileSystemToolServer(fs))
	toolSvc.Connect("tasks", tasktools.NewTasksToolServer(taskSvc))
	toolSvc.Connect("shell", shell.NewShellToolServer(fs.Cwd()))
	toolSvc.Connect("web", fetch.NewFetchToolServer())
	toolSvc.Connect("agent", tools.NewCallAgentToolServer(subagentSvc, agentRepo))
	todoToolSrv := todo.NewTodoToolServer(todoStorage)
	toolSvc.Connect("todo", todoToolSrv)

	// Telegram Bots
	botOrchestra, err := telegram.NewBotOrchestrator(cfg.BotConfigs...)
	if err != nil {
		return nil, err
	}

	consolidatorModel, err := modelsSvc.Get(cfg.ConsolidationModel)
	if err != nil {
		return nil, fmt.Errorf("'consolidator' model: %w", err)
	}

	memoryConsolidator, err := BuildMemoryConsolidator(
		fs,
		rt,
		todoStorage,
		consolidatorModel,
		agentRepo,
		[]agent.ToolServer{todoToolSrv},
		memoryFiles,
	)
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
		ChatSvc:           chatSvc,
		SessionSvc:        sessSvc,
		ToolsSvc:          toolSvc,
		MemoryRepo:        memoryFiles,
		MemoryIndexer:     memoryFiles,
		MemorySvc:         memoryConsolidator,
		ActivityRepo:      activityRepo,
		AgentRepo:         agentRepo,
	}, nil
}
