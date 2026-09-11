package wire

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/api"
	"arch-agent/internal/chat"
	"arch-agent/internal/cleanup"
	"arch-agent/internal/cron"
	"arch-agent/internal/files"
	"arch-agent/internal/hooks"
	"arch-agent/internal/logging"
	"arch-agent/internal/mcp"
	"arch-agent/internal/memory"
	"arch-agent/internal/model"
	"arch-agent/internal/openai"
	"arch-agent/internal/secrets"
	"arch-agent/internal/sentinel"
	"arch-agent/internal/session"
	"arch-agent/internal/subagent"
	"arch-agent/internal/task"
	"arch-agent/internal/tools"
	"arch-agent/internal/tools/fetch"
	fstools "arch-agent/internal/tools/fs"
	"arch-agent/internal/tools/shell"
	"arch-agent/internal/tools/todo"
	"arch-agent/internal/uuid"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path"
	"time"
)

type Config struct {
	DataPath string
	ShellEnv []string

	LogLevel  slog.Level
	AddSource bool
	Indented  bool
	JSON      bool

	MaxLogLines     int
	SessRetention   time.Duration
	CleanUpInterval time.Duration
}

func BuildServer(ctx context.Context, cfg Config) (*api.HTTPServer, error) {

	fileStorage, err := files.NewFileStorage(cfg.DataPath)
	if err != nil {
		return nil, err
	}

	defaultHandler := logging.NewHandler(logging.LoggerConfig{
		Level:     cfg.LogLevel,
		AddSource: cfg.AddSource,
		Indented:  cfg.Indented,
	})

	// writes json in stdio and never write logs in log file for agents
	agentUnreachibleLogger := slog.New(defaultHandler)
	slog.SetDefault(agentUnreachibleLogger)

	// write logs to agents visible log file with simplified format
	// and in stdio in json format
	// must be used for common logic
	lf := logging.NewLogFile(fileStorage)

	agentVisibleLogHandler := logging.WithAgentLog(
		defaultHandler,
		lf,
	)

	logger := slog.New(agentVisibleLogHandler)

	tmpFiles, err := files.NewTemporaryFiles(fileStorage, logger)
	if err != nil {
		return nil, err
	}
	go tmpFiles.Run(ctx)

	secretsRepo, err := files.NewSecretsFiles(fileStorage)
	if err != nil {
		return nil, err
	}

	secretService, err := secrets.New(secretsRepo, logger)
	if err != nil {
		return nil, err
	}

	openaiFactory := openai.NewOpenAIModelFactory(secretService)

	modelsSvc := model.NewModelService(openaiFactory)

	providerFiles, err := files.NewProviderFiles(fileStorage)
	if err != nil {
		return nil, err
	}

	providerSvc, err := model.NewProviderService(modelsSvc, providerFiles)
	if err != nil {
		return nil, err
	}
	toolSvc := tools.NewService()
	agentRepo := files.NewAgentFiles(fileStorage)

	// TODO: expose syncers
	agentSvc := agent.NewService(toolSvc, modelsSvc, agentRepo, []agent.AgentSync{})
	idGen := uuid.NewUUIDGenerator()
	sessFiles := files.NewSessionFiles(fileStorage)
	sessSvc := session.NewService(
		sessFiles,
		idGen,
		logger,
	)

	cleanupSvc, err := cleanup.NewCleanUpService(
		cleanup.CleanUpConfig{
			MaxLogLines:     cfg.MaxLogLines,
			SessRetention:   cfg.SessRetention,
			CleanUpInterval: cfg.CleanUpInterval,
		},
		cleanup.NewSessionsCleaner(agentSvc, sessFiles, logger),
		lf,
		logger,
	)
	if err != nil {
		return nil, err
	}

	go cleanupSvc.Run(ctx)

	skillFiles := files.NewSkillFiles(fileStorage, logger)
	memoryFiles := files.NewMemoryFiles(fileStorage, logger)
	contextAssembler := chat.NewContextAssembler(skillFiles, memoryFiles)

	mcpRepo, err := files.NewMCPFiles(fileStorage)
	if err != nil {
		return nil, err
	}
	mcpSvc, err := mcp.NewService(ctx, toolSvc, mcpRepo, logger)
	if err != nil {
		return nil, fmt.Errorf("build mcp service: %w", err)
	}

	todoStorage := todo.NewInMemoryStore()

	memoryRepo, err := files.NewMemoryConfigFile(fileStorage)
	if err != nil {
		return nil, err
	}

	activityRepo := files.NewActivityFiles(fileStorage)
	activityConfigRepo := files.NewActivityRepo(memoryRepo)
	activityService := memory.NewActivityService(
		modelsSvc,
		activityConfigRepo,
		activityRepo,
		logger,
	)

	secretReplacer := secrets.NewReplacer(secretService)

	agentHooks, err := hooks.NewAgentHooks(todoStorage, secretReplacer)
	if err != nil {
		return nil, err
	}

	chatSvc := chat.NewService(
		agentRepo,
		sessSvc,
		modelsSvc,
		toolSvc,
		contextAssembler,
		activityService,
		agentHooks,
		logger,
	)

	taskRepo, err := files.NewTaskFiles(fileStorage)
	if err != nil {
		return nil, err
	}

	executor := task.NewExecutor(
		sessSvc,
		chatSvc,
		logger,
	)

	taskSvc, err := task.NewService(
		taskRepo,
		executor,
		func(s string) (task.Cron, error) { return cron.NewRobfigCron(s) },
		agentSvc,
		logger,
	)
	if err != nil {
		return nil, err
	}

	// built in tools

	skipPatterns := []string{
		path.Join("*", "sessions", "**.jsonl"),
	}

	fsToolSrv, err := fstools.NewFileSystemToolServer(fileStorage, skipPatterns)
	if err != nil {
		return nil, err
	}
	toolSvc.Connect("filesystem", fsToolSrv)
	toolSvc.Connect("shell", shell.NewShellToolServer(cfg.DataPath, secretService))
	toolSvc.Connect("web", fetch.NewFetchToolServer())
	toolSvc.Connect("todo", todo.NewTodoToolServer(todoStorage))
	toolSvc.Connect("agent", tools.NewCallAgentToolServer(
		subagent.NewService(chatSvc, sessSvc, logger),
		agentRepo,
	))

	memoryHooksResolver, err := hooks.NewMemoryHooksResolver(memoryFiles)
	if err != nil {
		return nil, fmt.Errorf("harness: %w", err)
	}

	consolidationFsToolSrv, err := fstools.NewConsolidationInstuctFS(fileStorage, skipPatterns)
	if err != nil {
		return nil, err
	}

	memoryConsolidator, err := memory.NewConsolidationService(
		agentRepo,
		[]agent.ToolServer{consolidationFsToolSrv},
		memoryHooksResolver,
		modelsSvc,
		files.NewConsolidatorRepo(memoryRepo),
		logger,
	)
	if err != nil {
		return nil, err
	}
	go memoryConsolidator.Run(ctx)

	chatDispatcher := chat.NewDispatcher(chatSvc)

	// all sentinels
	sent := sentinel.New(cfg.DataPath, logger,
		sentinel.WithWatch(files.TMPDir, files.NewTMPDetector(tmpFiles)),
		sentinel.WithWatch(files.MCPConfigFile, files.NewMCPReloader(mcpSvc)),
		sentinel.WithWatch(files.MemoryConfigFile, files.NewMemoReloader(memoryConsolidator, activityService)),
		sentinel.WithWatch(files.ModelsConfigFile, files.NewModelsReloader(providerSvc)),
		sentinel.WithWatch(files.SecretsConfigFile, files.NewSecretsReloader(secretService)),
		sentinel.WithWatch(files.TaskConfigFile, files.NewTasksReloader(taskSvc)),
	)

	go func() {
		if err := sent.Run(ctx); err != nil {
			if errors.Is(err, sentinel.ErrClosedWatcher) ||
				errors.Is(err, context.Canceled) {
				return
			}

			agentUnreachibleLogger.Error("file_sentinel", "error", err)
		}
	}()

	return api.NewHTTPServer(
		agentUnreachibleLogger,
		taskSvc,
		chatDispatcher,
		sessSvc,
		toolSvc,
		mcpSvc,
		memoryFiles,
		memoryFiles,
		memoryConsolidator,
		activityRepo,
		agentSvc,
		providerSvc,
		idGen,
		activityService,
	), nil
}
