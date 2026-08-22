package wire

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/api"
	"arch-agent/internal/chat"
	"arch-agent/internal/cron"
	"arch-agent/internal/files"
	"arch-agent/internal/hooks"
	"arch-agent/internal/logging"
	"arch-agent/internal/mcp"
	"arch-agent/internal/memory"
	"arch-agent/internal/model"
	"arch-agent/internal/openai"
	"arch-agent/internal/session"
	"arch-agent/internal/subagent"
	"arch-agent/internal/task"
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
	"log/slog"
	"path"
)

type Config struct {
	DataPath           string
	TelegramGroupID    int64
	ConsolidationModel string
	ObserverModel      string
	ShellEnv           []string

	LogLevel  slog.Level
	AddSource bool
	Indented  bool
	JSON      bool
}

func BuildMemoryConsolidator(
	fs *files.FileSystem,
	model agent.Model,
	agentRepo agent.Repo,
	indexer agent.MemoryIndexer,
	logger *slog.Logger,
) (*memory.Memory, error) {

	// TODO: consolidation prompt is unreachible for memory consolidator
	fsToolsSrv := memory.NewInstuctFS(fs.Cwd(), fstools.NewRawFileSystemToolServer(fs))

	memoryHooksResolver, err := hooks.NewMemoryHooksResolver(fs, indexer)
	if err != nil {
		return nil, fmt.Errorf("harness: %w", err)
	}

	memory, err := memory.NewMemory(
		agentRepo,
		[]agent.ToolServer{fsToolsSrv},
		model,
		memoryHooksResolver,
		logger,
	)
	if err != nil {
		return nil, err
	}

	return memory, err
}

func BuildServer(ctx context.Context, cfg Config) (*api.HTTPServer, error) {

	fs, err := files.NewFS(cfg.DataPath)
	if err != nil {
		return nil, err
	}

	logger := logging.NewLogger(logging.LoggerConfig{
		Level:        cfg.LogLevel,
		AddSource:    cfg.AddSource,
		Indented:     cfg.Indented,
		JSON:         cfg.JSON,
		AgentLogFile: path.Join(fs.Cwd(), ".log"),
	})

	slog.SetDefault(logger)

	secretsRepo, err := files.NewSecretsFiles(fs)
	if err != nil {
		return nil, err
	}

	openaiFactory := openai.NewOpenAIModelFactory(secretsRepo)

	modelsSvc := model.NewModelService(openaiFactory)

	providerFiles, err := files.NewProviderFiles(fs)
	if err != nil {
		return nil, err
	}

	providerSvc, err := model.NewProviderService(modelsSvc, providerFiles)
	if err != nil {
		return nil, err
	}

	agentRepo := files.NewAgentFiles(fs)

	sessSvc := session.NewService(
		files.NewSessionFiles(fs),
		uuid.NewUUIDGenerator(),
		logger,
	)

	skillFiles := files.NewSkillFiles(fs, logger)
	memoryFiles := files.NewMemoryFiles(fs, logger)
	contextAssembler := chat.NewContextAssembler(skillFiles, memoryFiles)

	toolSvc := tools.NewService()

	mcpRepo, err := files.NewMCPFiles(fs)
	if err != nil {
		return nil, err
	}
	mcpSvc, err := mcp.NewService(ctx, toolSvc, mcpRepo, logger)
	if err != nil {
		return nil, fmt.Errorf("build mcp service: %w", err)
	}

	todoStorage := todo.NewInMemoryStore()

	// chat service
	observerModel, err := modelsSvc.Get(cfg.ObserverModel)
	if err != nil {
		return nil, errors.New("has no observer model")
	}

	activityRepo := files.NewActivityFiles(fs)
	observer := memory.NewObserver(
		memory.NewActivityReporter(observerModel),
		activityRepo,
		logger,
	)

	agentHooks, err := hooks.NewAgentHooks(fs, todoStorage)
	if err != nil {
		return nil, err
	}

	chatSvc := chat.NewService(
		agentRepo,
		sessSvc,
		modelsSvc,
		toolSvc,
		contextAssembler,
		observer,
		agentHooks,
		logger,
	)

	taskRepo, err := files.NewTaskFiles(fs)
	if err != nil {
		return nil, err
	}

	executor := task.NewExecutor(
		sessSvc,
		chatSvc,
	)

	taskSvc, err := task.NewService(
		taskRepo,
		executor,
		func(s string) (task.Cron, error) { return cron.NewRobfigCron(s) },
		agentRepo,
		logger,
	)
	if err != nil {
		return nil, err
	}

	toolSvc.Connect("filesystem", fstools.NewFileSystemToolServer(fs))
	toolSvc.Connect("tasks", tasktools.NewTasksToolServer(taskSvc))
	toolSvc.Connect("shell", shell.NewShellToolServer(fs.Cwd()))
	toolSvc.Connect("web", fetch.NewFetchToolServer())
	toolSvc.Connect("todo", todo.NewTodoToolServer(todoStorage))
	subagentSvc := subagent.NewService(chatSvc, sessSvc, logger)
	toolSvc.Connect("agent", tools.NewCallAgentToolServer(subagentSvc, agentRepo))

	consolidatorModel, err := modelsSvc.Get(cfg.ConsolidationModel)
	if err != nil {
		return nil, fmt.Errorf("'consolidator' model: %w", err)
	}

	memoryConsolidator, err := BuildMemoryConsolidator(
		fs,
		consolidatorModel,
		agentRepo,
		memoryFiles,
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("memory consolidator: %w", err)
	}
	go memoryConsolidator.Run(ctx)

	chatDispatcher := chat.NewDispatcher(chatSvc)

	return api.NewHTTPServer(
		logger,
		taskSvc,
		chatDispatcher,
		sessSvc,
		toolSvc,
		mcpSvc,
		memoryFiles,
		memoryFiles,
		memoryConsolidator,
		activityRepo,
		agentRepo,
		providerSvc,
	), nil
}
