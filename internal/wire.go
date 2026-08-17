package wire

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/api"
	"arch-agent/internal/chat"
	"arch-agent/internal/cron"
	"arch-agent/internal/files"
	"arch-agent/internal/hooks"
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
	"net/http"
)

type Config struct {
	DataPath           string
	TelegramGroupID    int64
	ConsolidationModel string
	ObserverModel      string
	ShellEnv           []string
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
	model agent.Model,
	agentRepo agent.Repo,
	indexer agent.MemoryIndexer,
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
	)
	if err != nil {
		return nil, err
	}

	return memory, err
}

func BuildServer(ctx context.Context, cfg Config) (*http.ServeMux, error) {

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

	providerSvc, err := model.NewProviderService(modelsSvc, providerFiles)
	if err != nil {
		return nil, err
	}

	agentRepo := files.NewAgentFiles(fs)

	sessSvc := session.NewService(
		files.NewSessionFiles(fs),
		uuid.NewUUIDGenerator(),
	)

	skillFiles := files.NewSkillFiles(fs)
	memoryFiles := files.NewMemoryFiles(fs)
	contextAssembler := chat.NewContextAssembler(skillFiles, memoryFiles)

	toolSvc := tools.NewService()

	mcpRepo := files.NewMCPFiles(fs)
	mcpSvc, err := mcp.NewService(ctx, toolSvc, mcpRepo)
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
	)

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

	toolSvc.Connect("filesystem", fstools.NewFileSystemToolServer(fs))
	toolSvc.Connect("tasks", tasktools.NewTasksToolServer(taskSvc))
	toolSvc.Connect("shell", shell.NewShellToolServer(fs.Cwd()))
	toolSvc.Connect("web", fetch.NewFetchToolServer())
	toolSvc.Connect("todo", todo.NewTodoToolServer(todoStorage))
	subagentSvc := subagent.NewService(chatSvc)
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
	)
	if err != nil {
		return nil, fmt.Errorf("memory consolidator: %w", err)
	}
	go memoryConsolidator.Run(ctx)

	chatDispatcher := chat.NewDispatcher(chatSvc)

	return api.NewServer(
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
