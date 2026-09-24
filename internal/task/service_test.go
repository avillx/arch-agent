package task_test

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"sync"
	"testing"
	"time"

	"arch-agent/internal/agent"
	"arch-agent/internal/task"
	"arch-agent/internal/types"
)

func strPtr(s string) *string { return &s }

func validTask(name string) task.TaskConfig {
	return task.TaskConfig{
		Name:        name,
		Description: "description",
		Recipients:  []agent.ID{"agent-a"},
		Reglament:   "* * * * *",
		Request:     "do something",
	}
}

type stubCron struct {
	nextFn func() time.Duration
	next   time.Duration
	expr   string
}

func (c *stubCron) NextTime() time.Duration {
	if c.nextFn != nil {
		return c.nextFn()
	}
	return c.next
}

func (c *stubCron) Expression() string { return c.expr }

func okCronFactory(d time.Duration) func(string) (task.Cron, error) {
	return func(expr string) (task.Cron, error) {
		return &stubCron{next: d, expr: expr}, nil
	}
}

type taskRepoStub struct {
	mu      sync.Mutex
	tasks   map[string]task.TaskConfig
	saves   []task.TaskConfig
	deletes []string

	allErr    error
	getErr    error
	saveErr   error
	deleteErr error
}

func newTaskRepoStub() *taskRepoStub {
	return &taskRepoStub{tasks: map[string]task.TaskConfig{}}
}

func (r *taskRepoStub) seed(cfg task.TaskConfig) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.tasks[cfg.Name] = cfg
}

func (r *taskRepoStub) All() (map[string]task.TaskConfig, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.allErr != nil {
		return nil, r.allErr
	}

	out := make(map[string]task.TaskConfig, len(r.tasks))
	for name, cfg := range r.tasks {
		out[name] = cfg
	}
	return out, nil
}

func (r *taskRepoStub) Get(name string) (task.TaskConfig, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.getErr != nil {
		return task.TaskConfig{}, r.getErr
	}

	cfg, ok := r.tasks[name]
	if !ok {
		return task.TaskConfig{}, types.ErrIsNotExist
	}
	return cfg, nil
}

func (r *taskRepoStub) Save(cfg task.TaskConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.saves = append(r.saves, cfg)
	if r.saveErr != nil {
		return r.saveErr
	}

	r.tasks[cfg.Name] = cfg
	return nil
}

func (r *taskRepoStub) Delete(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.deletes = append(r.deletes, name)
	if r.deleteErr != nil {
		return r.deleteErr
	}

	delete(r.tasks, name)
	return nil
}

func (r *taskRepoStub) saved() []task.TaskConfig {
	r.mu.Lock()
	defer r.mu.Unlock()

	return slices.Clone(r.saves)
}

func (r *taskRepoStub) deleted() []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return slices.Clone(r.deletes)
}

type agentRepoStub struct {
	mu     sync.Mutex
	agents map[agent.ID]agent.Agent
}

func newAgentRepoStub(ids ...agent.ID) *agentRepoStub {
	repo := &agentRepoStub{agents: map[agent.ID]agent.Agent{}}
	for _, id := range ids {
		repo.agents[id] = agent.NewAgent(id, "description", "system prompt", "model", nil, false)
	}
	return repo
}

func (r *agentRepoStub) All() ([]agent.Agent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	out := make([]agent.Agent, 0, len(r.agents))
	for _, agt := range r.agents {
		out = append(out, agt)
	}
	return out, nil
}

func (r *agentRepoStub) Get(id agent.ID) (agent.Agent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	agt, ok := r.agents[id]
	if !ok {
		return nil, types.ErrIsNotExist
	}
	return agt, nil
}

func (r *agentRepoStub) Save(agent.Agent) error { return nil }
func (r *agentRepoStub) Delete(agent.ID) error  { return nil }

type executorStub struct {
	mu      sync.Mutex
	calls   []task.TaskConfig
	lastCtx context.Context
	callCh  chan task.TaskConfig
}

func newExecutorStub() *executorStub {
	return &executorStub{callCh: make(chan task.TaskConfig, 16)}
}

func (e *executorStub) Execute(ctx context.Context, cfg task.TaskConfig) {
	e.mu.Lock()
	e.calls = append(e.calls, cfg)
	e.lastCtx = ctx
	e.mu.Unlock()

	select {
	case e.callCh <- cfg:
	default:
	}
}

func (e *executorStub) count() int {
	e.mu.Lock()
	defer e.mu.Unlock()

	return len(e.calls)
}

func (e *executorStub) context() context.Context {
	e.mu.Lock()
	defer e.mu.Unlock()

	return e.lastCtx
}

func waitExecution(t *testing.T, exec *executorStub) task.TaskConfig {
	t.Helper()

	select {
	case cfg := <-exec.callCh:
		return cfg
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for task execution")
		return task.TaskConfig{}
	}
}

func newService(
	t *testing.T,
	repo task.TaskRepo,
	agentRepo agent.Repo,
	exec task.TaskExecutor,
	cronFactory func(string) (task.Cron, error),
) *task.Service {
	t.Helper()

	svc, err := task.NewService(repo, exec, cronFactory, agentRepo, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	return svc
}

func validationProblem(t *testing.T, err error, key string) string {
	t.Helper()

	problems := types.ResovleValidationProblems(err)
	if problems == nil {
		t.Fatalf("expected validation error, got %v", err)
	}

	value, ok := problems[key]
	if !ok {
		t.Fatalf("expected validation problem for %q, got %v", key, problems)
	}
	return value
}

func TestValidate_NilRecipients(t *testing.T) {
	repo := newTaskRepoStub()
	svc := newService(t, repo, newAgentRepoStub("agent-a"), newExecutorStub(), okCronFactory(time.Second))

	cfg := validTask("task")
	cfg.Recipients = nil

	err := svc.Add(cfg)
	if got := validationProblem(t, err, "recipients"); got != task.ErrNoRecipients.Error() {
		t.Fatalf("recipients problem = %q, want %q", got, task.ErrNoRecipients.Error())
	}
	if got := len(repo.saved()); got != 0 {
		t.Fatalf("expected no saves, got %d", got)
	}
}

func TestValidate_UnknownRecipient(t *testing.T) {
	repo := newTaskRepoStub()
	svc := newService(t, repo, newAgentRepoStub("agent-a"), newExecutorStub(), okCronFactory(time.Second))

	cfg := validTask("task")
	cfg.Recipients = []agent.ID{"missing"}

	err := svc.Add(cfg)
	if got := validationProblem(t, err, "missing"); got != "is not exist" {
		t.Fatalf("recipient problem = %q, want %q", got, "is not exist")
	}
	if got := len(repo.saved()); got != 0 {
		t.Fatalf("expected no saves, got %d", got)
	}
}

func TestValidate_BadCronOnAdd(t *testing.T) {
	repo := newTaskRepoStub()
	badCronFactory := func(string) (task.Cron, error) { return nil, task.ErrCron }
	svc := newService(t, repo, newAgentRepoStub("agent-a"), newExecutorStub(), badCronFactory)

	err := svc.Add(validTask("task"))
	if got := validationProblem(t, err, "schedule"); got != "invalid cron" {
		t.Fatalf("schedule problem = %q, want %q", got, "invalid cron")
	}
	if got := len(repo.saved()); got != 0 {
		t.Fatalf("expected no saves, got %d", got)
	}
}

func TestValidate_BadCronOnPatch(t *testing.T) {
	repo := newTaskRepoStub()
	repo.seed(validTask("task"))
	badCronFactory := func(string) (task.Cron, error) { return nil, task.ErrCron }
	svc := newService(t, repo, newAgentRepoStub("agent-a"), newExecutorStub(), badCronFactory)

	err := svc.Patch("task", task.TaskPatch{Reglament: strPtr("bad")})
	if got := validationProblem(t, err, "schedule"); got != "invalid cron" {
		t.Fatalf("schedule problem = %q, want %q", got, "invalid cron")
	}
	if got := len(repo.deleted()); got != 0 {
		t.Fatalf("expected no deletes, got %v", got)
	}
	if got := len(repo.saved()); got != 0 {
		t.Fatalf("expected no saves, got %d", got)
	}
}

func TestValidate_EmptyRequest(t *testing.T) {
	repo := newTaskRepoStub()
	svc := newService(t, repo, newAgentRepoStub("agent-a"), newExecutorStub(), okCronFactory(time.Second))

	cfg := validTask("task")
	cfg.Request = ""

	err := svc.Add(cfg)
	if got := validationProblem(t, err, "request"); got != "must be not empty" {
		t.Fatalf("request problem = %q, want %q", got, "must be not empty")
	}
	if got := len(repo.saved()); got != 0 {
		t.Fatalf("expected no saves, got %d", got)
	}
}

func TestAdd_AllowsOnlyUniqueNames(t *testing.T) {
	repo := newTaskRepoStub()
	svc := newService(t, repo, newAgentRepoStub("agent-a"), newExecutorStub(), okCronFactory(time.Second))

	if err := svc.Add(validTask("dup")); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	err := svc.Add(validTask("dup"))
	if got := validationProblem(t, err, "dup"); got != "already exist" {
		t.Fatalf("identity problem = %q, want %q", got, "already exist")
	}
	if got := len(repo.saved()); got != 1 {
		t.Fatalf("expected one save, got %d", got)
	}
}

func TestPatch_NewNameDeletesPreviousName(t *testing.T) {
	repo := newTaskRepoStub()
	svc := newService(t, repo, newAgentRepoStub("agent-a"), newExecutorStub(), okCronFactory(time.Second))

	if err := svc.Add(validTask("old")); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	if err := svc.Patch("old", task.TaskPatch{Name: strPtr("new")}); err != nil {
		t.Fatalf("Patch() error = %v", err)
	}

	if _, err := repo.Get("new"); err != nil {
		t.Fatalf("Get(new) error = %v", err)
	}
	if _, err := repo.Get("old"); !errors.Is(err, types.ErrIsNotExist) {
		t.Fatalf("Get(old) error = %v, want ErrIsNotExist", err)
	}
	if got := repo.deleted(); !slices.Equal(got, []string{"old"}) {
		t.Fatalf("deleted names = %v, want [old]", got)
	}
}

func TestOnceTask_DisabledAfterExecution(t *testing.T) {
	repo := newTaskRepoStub()
	cfg := validTask("once")
	cfg.Active = true
	cfg.Oneshot = true
	repo.seed(cfg)

	exec := newExecutorStub()
	newService(t, repo, newAgentRepoStub("agent-a"), exec, okCronFactory(5*time.Millisecond))

	waitExecution(t, exec)

	deadline := time.Now().Add(2 * time.Second)
	for {
		stored, err := repo.Get("once")
		if err == nil && !stored.Active {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("expected task to be disabled after oneshot execution")
		}
		time.Sleep(5 * time.Millisecond)
	}

	if got := exec.count(); got != 1 {
		t.Fatalf("expected one execution, got %d", got)
	}
}

func TestPlannedTask_RunsAndStaysActive(t *testing.T) {
	repo := newTaskRepoStub()
	cfg := validTask("planned")
	cfg.Active = true
	repo.seed(cfg)

	exec := newExecutorStub()
	factory := func(expr string) (task.Cron, error) {
		calls := 0
		return &stubCron{
			nextFn: func() time.Duration {
				if calls == 0 {
					calls++
					return 5 * time.Millisecond
				}
				return time.Hour
			},
			expr: expr,
		}, nil
	}
	svc := newService(t, repo, newAgentRepoStub("agent-a"), exec, factory)

	got := waitExecution(t, exec)
	if !got.Equals(cfg) {
		t.Fatalf("executed config = %+v, want %+v", got, cfg)
	}

	if got := exec.count(); got != 1 {
		t.Fatalf("expected one execution, got %d", got)
	}
	if got := len(repo.saved()); got != 0 {
		t.Fatalf("expected task to stay active without saves, got %d", got)
	}

	if err := svc.Delete("planned"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
}

func TestDelete_RemovesTaskFromRepoAndReloadStopsRuntime(t *testing.T) {
	repo := newTaskRepoStub()
	cfg := validTask("periodic")
	cfg.Active = true
	repo.seed(cfg)

	exec := newExecutorStub()
	svc := newService(t, repo, newAgentRepoStub("agent-a"), exec, okCronFactory(5*time.Millisecond))

	waitExecution(t, exec)
	ctx := exec.context()
	if ctx == nil {
		t.Fatal("expected executor to capture run context")
	}

	if err := svc.Delete("periodic"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	if got := repo.deleted(); !slices.Equal(got, []string{"periodic"}) {
		t.Fatalf("deleted names = %v, want [periodic]", got)
	}
	if err := ctx.Err(); err != nil {
		t.Fatalf("runtime context error = %v, want still running after Delete", err)
	}

	if err := svc.Reload(context.Background()); err != nil {
		t.Fatalf("Reload() error = %v", err)
	}
	if err := ctx.Err(); !errors.Is(err, context.Canceled) {
		t.Fatalf("runtime context error = %v, want context.Canceled after Reload", err)
	}
}
