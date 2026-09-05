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
	"path/filepath"
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

	fs, err := files.NewFS(cfg.DataPath)
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
	lf := logging.NewLogFile(filepath.Join(fs.Cwd(), "agents.log"))

	agentVisibleLogHandler := logging.WithAgentLog(
		defaultHandler,
		lf,
	)

	logger := slog.New(agentVisibleLogHandler)

	tmpFiles, err := files.NewTemporaryFiles(fs, logger)
	if err != nil {
		return nil, err
	}
	go tmpFiles.Run(ctx)

	secretsRepo, err := files.NewSecretsFiles(fs)
	if err != nil {
		return nil, err
	}

	secretService, err := secrets.New(secretsRepo, logger)
	if err != nil {
		return nil, err
	}

	openaiFactory := openai.NewOpenAIModelFactory(secretService)

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

	idGen := uuid.NewUUIDGenerator()
	sessFiles := files.NewSessionFiles(fs)
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
		cleanup.NewSessionsCleaner(agentRepo, sessFiles, logger),
		lf,
		logger,
	)
	if err != nil {
		return nil, err
	}

	go cleanupSvc.Run(ctx)

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

	memoryRepo, err := files.NewMemoryConfigFile(fs)
	if err != nil {
		return nil, err
	}

	activityRepo := files.NewActivityFiles(fs)
	activityConfigRepo := files.NewActivityRepo(memoryRepo)
	activityService := memory.NewActivityService(
		modelsSvc,
		activityConfigRepo,
		activityRepo,
		logger,
	)

	secretReplacer := secrets.NewReplacer(secretService)

	agentHooks, err := hooks.NewAgentHooks(fs, todoStorage, secretReplacer)
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

	taskRepo, err := files.NewTaskFiles(fs)
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
		agentRepo,
		logger,
	)
	if err != nil {
		return nil, err
	}

	// built in tools

	skipPatterns := []string{
		filepath.Join(fs.Cwd(), "**", "sessions", "**.jsonl"),
	}

	fsToolSrv, err := fstools.NewFileSystemToolServer(fs, skipPatterns)
	if err != nil {
		return nil, err
	}
	toolSvc.Connect("filesystem", fsToolSrv)
	toolSvc.Connect("shell", shell.NewShellToolServer(fs.Cwd(), secretService))
	toolSvc.Connect("web", fetch.NewFetchToolServer())
	toolSvc.Connect("todo", todo.NewTodoToolServer(todoStorage))
	toolSvc.Connect("agent", tools.NewCallAgentToolServer(
		subagent.NewService(chatSvc, sessSvc, logger),
		agentRepo,
	))

	memoryHooksResolver, err := hooks.NewMemoryHooksResolver(fs, memoryFiles)
	if err != nil {
		return nil, fmt.Errorf("harness: %w", err)
	}

	consolidationFsToolSrv, err := fstools.NewConsolidationInstuctFS(fs, skipPatterns)
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
	sent := sentinel.New(fs.Cwd(), logger,
		sentinel.WithWatch(filepath.Join(fs.Cwd(), files.TMPDir), files.NewTMPDetector(tmpFiles)),
		sentinel.WithWatch(filepath.Join(fs.Cwd(), files.MCPConfigFile), files.NewMCPReloader(mcpSvc)),
		sentinel.WithWatch(filepath.Join(fs.Cwd(), files.MemoryConfigFile), files.NewMemoReloader(memoryConsolidator, activityService)),
		sentinel.WithWatch(filepath.Join(fs.Cwd(), files.ModelsConfigFile), files.NewModelsReloader(providerSvc)),
		sentinel.WithWatch(filepath.Join(fs.Cwd(), files.SecretsConfigFile), files.NewSecretsReloader(secretService)),
		sentinel.WithWatch(filepath.Join(fs.Cwd(), files.TaskConfigFile), files.NewTasksReloader(taskSvc)),
	)

	go func() {
		if err := sent.Run(ctx); err != nil {
			if errors.Is(err, sentinel.ErrClosedWatcher) {
				return
			}

			if errors.Is(err, context.Canceled) {
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
		agentRepo,
		providerSvc,
		idGen,
	), nil
}
