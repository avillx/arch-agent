package server

import (
	"ard/internal/agent"
	"ard/internal/chat"
	"ard/internal/cleanup"
	"ard/internal/cron"
	"ard/internal/files"
	"ard/internal/hooks"
	"ard/internal/logging"
	"ard/internal/mcp"
	"ard/internal/memory"
	"ard/internal/model"
	"ard/internal/openai"
	"ard/internal/secrets"
	"ard/internal/sentinel"
	"ard/internal/session"
	"ard/internal/subagent"
	"ard/internal/task"
	"ard/internal/tools"
	fstools "ard/internal/tools/fs"
	"ard/internal/tools/shell"
	"ard/internal/tools/todo"
	"ard/internal/uuid"
	"context"
	"fmt"
	"log/slog"
	"path"
	"path/filepath"
	"time"
)

// Simple orchestrator
type Server struct {
	TMPFiles               *files.TemporaryFiles
	SecretSvc              *secrets.Service
	ModelsSvc              *model.ModelService
	ProviderSvc            *model.ProviderService
	ToolSvc                *tools.Service
	AgentSvc               *agent.Service
	SessSvc                *session.Service
	CleanupSvc             *cleanup.CleanUpService
	MCPSvc                 *mcp.Service
	ActivitySvc            *memory.ActivityService
	ChatSvc                *chat.Service
	TaskSvc                *task.Service
	MemoryConsolidationSvc *memory.ConsolidationService
	ChatDispatcherSvc      *chat.Dispatcher
	Sentinels              []*sentinel.Sentinel
	MemoryFiles            *files.MemoryFiles
	ActivityFiles          *files.ActivityFiles
	AgentFiles             *files.AgentFiles
}

func New(
	ctx context.Context,
	datapath string,
	maxLogLines int,
	sessRetention time.Duration,
	cleanUpInterval time.Duration,
	agentShell string,
	agentShellAttributes []string,
	logger *slog.Logger,
) (*Server, error) {

	fileStorage, err := files.NewFileStorage(datapath)
	if err != nil {
		return nil, err
	}

	// wraps common handler to intercept logs and sink in into agent available file
	logFile := logging.NewLogFile(fileStorage)
	fileHandler := logging.WithAgentLog(
		logger.Handler(),
		logFile,
	)

	// Override logger with handler that flush logs in agent readable file
	logger = slog.New(fileHandler)

	tmpFiles, err := files.NewTemporaryFiles(fileStorage, logger)
	if err != nil {
		return nil, err
	}

	secretsRepo, err := files.NewSecretsFiles(fileStorage)
	if err != nil {
		return nil, err
	}

	secretSvc, err := secrets.New(secretsRepo, logger)
	if err != nil {
		return nil, err
	}

	openaiFactory := openai.NewOpenAIModelFactory(secretSvc)

	modelsSvc := model.NewModelService(openaiFactory)

	providerFiles, err := files.NewProviderFiles(fileStorage)
	if err != nil {
		return nil, err
	}

	providerSvc, err := model.NewProviderService(modelsSvc, providerFiles, logger)
	if err != nil {
		return nil, err
	}

	agentRepo := files.NewAgentFiles(fileStorage)
	toolSvc := tools.NewService()

	// TODO: expose syncers
	agentSvc, err := agent.NewService(toolSvc, modelsSvc, agentRepo, []agent.AgentSync{})
	if err != nil {
		return nil, err
	}

	idGen := uuid.NewUUIDGenerator()
	sessFiles := files.NewSessionFiles(fileStorage)
	sessSvc := session.NewService(
		sessFiles,
		idGen,
		logger,
	)

	cleanupSvc, err := cleanup.NewCleanUpService(
		cleanup.CleanUpConfig{
			MaxLogLines:     maxLogLines,
			SessRetention:   sessRetention,
			CleanUpInterval: cleanUpInterval,
		},
		cleanup.NewSessionsCleaner(agentSvc, sessFiles, logger),
		logFile,
		logger,
	)
	if err != nil {
		return nil, err
	}

	// MCP service
	mcpRepo, err := files.NewMCPFiles(fileStorage)
	if err != nil {
		return nil, err
	}
	mcpSvc, err := mcp.NewService(ctx, toolSvc, mcpRepo, logger)
	if err != nil {
		return nil, fmt.Errorf("build mcp service: %w", err)
	}

	// activity service
	memoryRepo, err := files.NewMemoryConfigFile(fileStorage)
	if err != nil {
		return nil, err
	}
	activityRepo := files.NewActivityFiles(fileStorage)
	activityConfigRepo := files.NewActivityRepo(memoryRepo)
	activitySvc := memory.NewActivityService(
		modelsSvc,
		activityConfigRepo,
		activityRepo,
		logger,
	)

	// chat service
	secretReplacer := secrets.NewReplacer(secretSvc)
	todoStorage := todo.NewInMemoryStore()
	agentHooks, err := hooks.NewAgentHooks(todoStorage, secretReplacer)
	if err != nil {
		return nil, err
	}

	skillFiles, err := files.NewSkillFiles(fileStorage, logger)
	if err != nil {
		return nil, err
	}
	memoryFiles := files.NewMemoryFiles(fileStorage, logger)

	chatSvc := chat.NewService(
		agentRepo,
		sessSvc,
		modelsSvc,
		toolSvc,
		chat.NewContextAssembler(skillFiles, memoryFiles),
		activitySvc,
		agentHooks,
		logger,
	)

	// task service
	taskSvc, err := buildTaskService(fileStorage, sessSvc, chatSvc, agentRepo, logger)
	if err != nil {
		return nil, err
	}

	// builtin tools
	skipPatterns := []string{
		path.Join("*", "sessions", "**.jsonl"),
	}

	if err := attachBuildinToolServers(
		datapath, agentShell, agentShellAttributes, toolSvc, secretSvc, fileStorage,
		skipPatterns, todoStorage, chatSvc, sessSvc, logger, agentRepo,
	); err != nil {
		return nil, err
	}

	// memory consolidation
	memoryConsolidationSvc, err := buildMemoryConsolidationService(
		agentRepo, modelsSvc, memoryFiles, memoryRepo, fileStorage, skipPatterns, logger,
	)
	if err != nil {
		return nil, err
	}

	// sentinels
	sentinels, err := buildSentinels(
		datapath, logger, tmpFiles, mcpSvc, memoryConsolidationSvc,
		activitySvc, providerSvc, secretSvc, taskSvc,
	)
	if err != nil {
		return nil, err
	}

	return &Server{
		TMPFiles:               tmpFiles,
		SecretSvc:              secretSvc,
		ModelsSvc:              modelsSvc,
		ProviderSvc:            providerSvc,
		ToolSvc:                toolSvc,
		AgentSvc:               agentSvc,
		SessSvc:                sessSvc,
		CleanupSvc:             cleanupSvc,
		MCPSvc:                 mcpSvc,
		ActivitySvc:            activitySvc,
		TaskSvc:                taskSvc,
		MemoryConsolidationSvc: memoryConsolidationSvc,
		ChatSvc:                chatSvc,
		ChatDispatcherSvc:      chat.NewDispatcher(chatSvc),
		Sentinels:              sentinels,
		MemoryFiles:            memoryFiles,
		ActivityFiles:          activityRepo,
		AgentFiles:             agentRepo,
	}, nil
}

func buildTaskService(
	storage files.FileStorage,
	sessSvc *session.Service,
	chatSvc *chat.Service,
	agentRepo agent.Repo,
	logger *slog.Logger,
) (*task.Service, error) {
	taskRepo, err := files.NewTaskFiles(storage)
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

	return taskSvc, nil
}

func attachBuildinToolServers(
	datapath string,
	agentShell string,
	agentShellAttributes []string,
	toolSvc *tools.Service,
	secretSvc *secrets.Service,
	fileStorage files.FileStorage,
	skipPatterns []string,
	todoStorage todo.Store,
	chatExecutor chat.ChatExecutor,
	sessSvc *session.Service,
	logger *slog.Logger,
	agentRepo agent.Repo,
) error {
	fsToolSrv, err := fstools.NewFileSystemToolServer(fileStorage, skipPatterns)
	if err != nil {
		return err
	}
	toolSvc.Connect("filesystem", fsToolSrv)
	toolSvc.Connect("todo", todo.NewTodoToolServer(todoStorage))
	toolSvc.Connect("agent", tools.NewCallAgentToolServer(
		subagent.NewService(chatExecutor, sessSvc, logger),
		agentRepo,
	))
	toolSvc.Connect("shell", shell.NewShellToolServer(
		datapath,
		secretSvc,
		agentShell,
		agentShellAttributes,
	))

	return nil
}

func buildMemoryConsolidationService(
	agentRepo agent.Repo,
	modelsSvc *model.ModelService,
	memoryFiles *files.MemoryFiles,
	memoryConfigFiles *files.MemoryFile,
	fileStorage files.FileStorage,
	skipPatterns []string,
	logger *slog.Logger,
) (*memory.ConsolidationService, error) {
	memoryHooksResolver, err := hooks.NewMemoryHooksResolver(memoryFiles)
	if err != nil {
		return nil, fmt.Errorf("harness: %w", err)
	}

	consolidationFsToolSrv, err := fstools.NewConsolidationInstuctFS(fileStorage, skipPatterns)
	if err != nil {
		return nil, err
	}

	return memory.NewConsolidationService(
		agentRepo,
		[]agent.ToolServer{consolidationFsToolSrv},
		memoryHooksResolver,
		modelsSvc,
		files.NewConsolidatorRepo(memoryConfigFiles),
		logger,
	)
}

func buildSentinels(
	datapath string,
	logger *slog.Logger,
	tmpFiles *files.TemporaryFiles,
	mcpSvc *mcp.Service,
	memoryConsolidationSvc *memory.ConsolidationService,
	activitySvc *memory.ActivityService,
	providerSvc *model.ProviderService,
	secretSvc *secrets.Service,
	taskSvc *task.Service,
) ([]*sentinel.Sentinel, error) {
	tmpSent, err := sentinel.New(datapath, logger,
		sentinel.WithWatch(files.TMPDir, files.NewTMPDetector(tmpFiles)),
	)
	if err != nil {
		return nil, fmt.Errorf("tmp sentinel: %w", err)
	}

	cfgsDirPath := filepath.Join(datapath, files.ConfigDir)
	cfgSent, err := sentinel.New(cfgsDirPath, logger,
		sentinel.WithWatch(files.MCPConfigFile, files.NewMCPReloader(mcpSvc)),
		sentinel.WithWatch(files.MemoryConfigFile, files.NewMemoReloader(memoryConsolidationSvc, activitySvc)),
		sentinel.WithWatch(files.ModelsConfigFile, files.NewModelsReloader(providerSvc)),
		sentinel.WithWatch(files.SecretsConfigFile, files.NewSecretsReloader(secretSvc)),
		sentinel.WithWatch(files.TaskConfigFile, files.NewTasksReloader(taskSvc)),
	)
	if err != nil {
		return nil, fmt.Errorf("config sentinel: %w", err)
	}

	return []*sentinel.Sentinel{cfgSent, tmpSent}, nil
}

func (s *Server) Run(ctx context.Context) {
	go s.TMPFiles.Run(ctx)
	go s.CleanupSvc.Run(ctx)
	go s.MemoryConsolidationSvc.Run(ctx)
	for _, sent := range s.Sentinels {
		go sent.Run(ctx)
	}
}
