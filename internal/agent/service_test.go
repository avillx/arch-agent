package agent_test

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/types"
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
)

type mockModel struct{}

func (mockModel) Settings() agent.ModelSettings { return nil }
func (mockModel) Complete(context.Context, []agent.Tool, []agent.Message) (*agent.Completion, error) {
	return &agent.Completion{Done: true}, nil
}
func (mockModel) ContextLimit() int64                   { return 0 }
func (mockModel) SupportedModalities() []agent.Modality { return nil }

type mockModelRegistry struct {
	models map[string]agent.Model
}

func newMockModelRegistry(names ...string) *mockModelRegistry {
	registry := &mockModelRegistry{models: map[string]agent.Model{}}
	for _, name := range names {
		registry.models[name] = mockModel{}
	}
	return registry
}

func (r *mockModelRegistry) Get(name string) (agent.Model, error) {
	model, ok := r.models[name]
	if !ok {
		return nil, types.ErrIsNotExist
	}
	return model, nil
}

type mockToolServer struct{}

func (mockToolServer) Tools() []agent.Tool { return nil }

type mockToolRegistry struct {
	servers map[string]agent.ToolServer
}

func newMockToolRegistry(names ...string) *mockToolRegistry {
	registry := &mockToolRegistry{servers: map[string]agent.ToolServer{}}
	for _, name := range names {
		registry.servers[name] = mockToolServer{}
	}
	return registry
}

func (r *mockToolRegistry) ToolServers(names ...string) ([]agent.ToolServer, error) {
	servers := make([]agent.ToolServer, 0, len(names))
	for _, name := range names {
		server, ok := r.servers[name]
		if !ok {
			return nil, types.ErrIsNotExist
		}
		servers = append(servers, server)
	}
	return servers, nil
}

type mockRepo struct {
	mu      sync.Mutex
	agents  map[agent.ID]agent.Agent
	saves   []agent.Agent
	deletes []agent.ID

	allErr    error
	saveErr   error
	deleteErr error
}

func newMockRepo() *mockRepo {
	return &mockRepo{agents: map[agent.ID]agent.Agent{}}
}

func (r *mockRepo) seed(agt agent.Agent) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.agents[agt.ID()] = agt
}

func (r *mockRepo) All() ([]agent.Agent, error) {
	if r.allErr != nil {
		return nil, r.allErr
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	agts := make([]agent.Agent, 0, len(r.agents))
	for _, agt := range r.agents {
		agts = append(agts, agt)
	}
	return agts, nil
}

func (r *mockRepo) Get(id agent.ID) (agent.Agent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	agt, ok := r.agents[id]
	if !ok {
		return nil, types.ErrIsNotExist
	}
	return agt, nil
}

func (r *mockRepo) Save(agt agent.Agent) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.saves = append(r.saves, agt)
	if r.saveErr != nil {
		return r.saveErr
	}

	r.agents[agt.ID()] = agt
	return nil
}

func (r *mockRepo) Delete(id agent.ID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.deletes = append(r.deletes, id)
	if r.deleteErr != nil {
		return r.deleteErr
	}

	delete(r.agents, id)
	return nil
}

func (r *mockRepo) savedCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	return len(r.saves)
}

func (r *mockRepo) savedAgents() []agent.Agent {
	r.mu.Lock()
	defer r.mu.Unlock()

	return append([]agent.Agent(nil), r.saves...)
}

func (r *mockRepo) deletedIDs() []agent.ID {
	r.mu.Lock()
	defer r.mu.Unlock()

	return append([]agent.ID(nil), r.deletes...)
}

type mockSync struct {
	mu    sync.Mutex
	err   error
	calls []agent.ID
}

func (s *mockSync) DeleteAgent(id agent.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.calls = append(s.calls, id)
	return s.err
}

func (s *mockSync) calledWith() []agent.ID {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]agent.ID(nil), s.calls...)
}

func newTestService(
	t *testing.T,
	storage *mockRepo,
	models *mockModelRegistry,
	tools *mockToolRegistry,
	syncs ...agent.AgentSync,
) *agent.Service {
	t.Helper()

	svc, err := agent.NewService(tools, models, storage, syncs)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	return svc
}

func newAgent(id agent.ID) agent.Agent {
	return agent.NewAgent(id, "description", "system prompt", "model-a", []string{"filesystem"}, true)
}

func TestNewService_CreatesDefaultAgentWhenEmpty(t *testing.T) {
	storage := newMockRepo()
	svc, err := agent.NewService(
		newMockToolRegistry("filesystem"),
		newMockModelRegistry("model-a"),
		storage,
		nil,
	)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	saved := storage.savedAgents()
	if len(saved) != 1 {
		t.Fatalf("expected 1 saved agent, got %d", len(saved))
	}
	if saved[0].ID() != agent.ID("default") {
		t.Fatalf("expected default agent, got %s", saved[0].ID())
	}

	agts, err := svc.All()
	if err != nil {
		t.Fatalf("All() error = %v", err)
	}
	if len(agts) != 1 || agts[0].ID() != agent.ID("default") {
		t.Fatalf("expected only default agent, got %v", agts)
	}
}

func TestNewService_KeepsExistingAgents(t *testing.T) {
	storage := newMockRepo()
	storage.seed(newAgent("existing"))

	svc, err := agent.NewService(
		newMockToolRegistry("filesystem"),
		newMockModelRegistry("model-a"),
		storage,
		nil,
	)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	if got := storage.savedCount(); got != 0 {
		t.Fatalf("expected no default agent saved, got %d saves", got)
	}

	agts, err := svc.All()
	if err != nil {
		t.Fatalf("All() error = %v", err)
	}
	if len(agts) != 1 || agts[0].ID() != agent.ID("existing") {
		t.Fatalf("expected only existing agent, got %v", agts)
	}
}

func TestNewService_FailsWhenListingAgentsFails(t *testing.T) {
	wantErr := errors.New("list failed")
	storage := newMockRepo()
	storage.allErr = wantErr

	_, err := agent.NewService(
		newMockToolRegistry("filesystem"),
		newMockModelRegistry("model-a"),
		storage,
		nil,
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf("NewService() error = %v, want %v", err, wantErr)
	}
}

func TestNewService_FailsWhenDefaultAgentSaveFails(t *testing.T) {
	wantErr := errors.New("save failed")
	storage := newMockRepo()
	storage.saveErr = wantErr

	_, err := agent.NewService(
		newMockToolRegistry("filesystem"),
		newMockModelRegistry("model-a"),
		storage,
		nil,
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf("NewService() error = %v, want %v", err, wantErr)
	}
}

func TestSave_PersistsValidAgent(t *testing.T) {
	storage := newMockRepo()
	svc := newTestService(t, storage, newMockModelRegistry("model-a"), newMockToolRegistry("filesystem"))

	agt := newAgent("writer")
	if err := svc.Save(agt); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := svc.Get("writer")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.ID() != agt.ID() {
		t.Fatalf("expected agent %s, got %s", agt.ID(), got.ID())
	}
}

func TestSave_RejectsUnknownModel(t *testing.T) {
	storage := newMockRepo()
	storage.seed(newAgent("existing"))
	svc := newTestService(t, storage, newMockModelRegistry("model-a"), newMockToolRegistry("filesystem"))

	agt := agent.NewAgent("writer", "description", "system prompt", "ghost-model", []string{"filesystem"}, false)
	if err := svc.Save(agt); err == nil {
		t.Fatal("expected error for unknown model")
	}
	if got := storage.savedCount(); got != 0 {
		t.Fatalf("expected no save, got %d", got)
	}
}

func TestSave_RejectsUnknownToolServer(t *testing.T) {
	storage := newMockRepo()
	storage.seed(newAgent("existing"))
	svc := newTestService(t, storage, newMockModelRegistry("model-a"), newMockToolRegistry("filesystem"))

	agt := agent.NewAgent("writer", "description", "system prompt", "model-a", []string{"filesystem", "ghost"}, false)
	if err := svc.Save(agt); err == nil {
		t.Fatal("expected error for unknown tool server")
	}
	if got := storage.savedCount(); got != 0 {
		t.Fatalf("expected no save, got %d", got)
	}
}

func TestSave_SameIDOverwritesExistingAgent(t *testing.T) {
	storage := newMockRepo()
	storage.seed(newAgent("writer"))
	svc := newTestService(t, storage, newMockModelRegistry("model-a"), newMockToolRegistry("filesystem"))

	updated := agent.NewAgent("writer", "updated description", "new prompt", "model-a", []string{"filesystem"}, false)
	if err := svc.Save(updated); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := svc.Get("writer")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Description() != "updated description" {
		t.Fatalf("expected updated agent, got %q", got.Description())
	}

	agts, err := svc.All()
	if err != nil {
		t.Fatalf("All() error = %v", err)
	}
	if len(agts) != 1 {
		t.Fatalf("expected a single agent per id, got %d", len(agts))
	}
}

func TestDelete_DeletesFromStorageAndSyncs(t *testing.T) {
	storage := newMockRepo()
	storage.seed(newAgent("writer"))
	sync1 := &mockSync{}
	sync2 := &mockSync{}
	svc := newTestService(t, storage, newMockModelRegistry("model-a"), newMockToolRegistry("filesystem"), sync1, sync2)

	if err := svc.Delete("writer"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	if got := storage.deletedIDs(); !slices.Equal(got, []agent.ID{"writer"}) {
		t.Fatalf("expected storage delete of writer, got %v", got)
	}
	if got := sync1.calledWith(); !slices.Equal(got, []agent.ID{"writer"}) {
		t.Fatalf("expected sync1 delete of writer, got %v", got)
	}
	if got := sync2.calledWith(); !slices.Equal(got, []agent.ID{"writer"}) {
		t.Fatalf("expected sync2 delete of writer, got %v", got)
	}
}

func TestDelete_RejectsUnknownAgent(t *testing.T) {
	storage := newMockRepo()
	sync := &mockSync{}
	svc := newTestService(t, storage, newMockModelRegistry("model-a"), newMockToolRegistry("filesystem"), sync)

	if err := svc.Delete("missing"); err == nil {
		t.Fatal("expected error for unknown agent")
	}
	if got := storage.deletedIDs(); len(got) != 0 {
		t.Fatalf("expected no storage delete, got %v", got)
	}
	if got := sync.calledWith(); len(got) != 0 {
		t.Fatalf("expected no sync calls, got %v", got)
	}
}

func TestDelete_StorageErrorSkipsSyncs(t *testing.T) {
	wantErr := errors.New("delete failed")
	storage := newMockRepo()
	storage.seed(newAgent("writer"))
	storage.deleteErr = wantErr
	sync := &mockSync{}
	svc := newTestService(t, storage, newMockModelRegistry("model-a"), newMockToolRegistry("filesystem"), sync)

	err := svc.Delete("writer")
	if !errors.Is(err, wantErr) {
		t.Fatalf("Delete() error = %v, want %v", err, wantErr)
	}
	if got := sync.calledWith(); len(got) != 0 {
		t.Fatalf("expected no sync calls, got %v", got)
	}
}

func TestDelete_ContinuesSyncsAfterError(t *testing.T) {
	syncErr := errors.New("sync failed")
	storage := newMockRepo()
	storage.seed(newAgent("writer"))
	sync1 := &mockSync{err: syncErr}
	sync2 := &mockSync{}
	svc := newTestService(t, storage, newMockModelRegistry("model-a"), newMockToolRegistry("filesystem"), sync1, sync2)

	err := svc.Delete("writer")
	if !errors.Is(err, syncErr) {
		t.Fatalf("Delete() error = %v, want sync error", err)
	}
	if got := sync1.calledWith(); len(got) != 1 {
		t.Fatalf("expected sync1 to be called once, got %v", got)
	}
	if got := sync2.calledWith(); !slices.Equal(got, []agent.ID{"writer"}) {
		t.Fatalf("expected sync2 to be called despite sync1 error, got %v", got)
	}
}

func TestDelete_JoinsSyncErrors(t *testing.T) {
	err1 := errors.New("sync 1 failed")
	err2 := errors.New("sync 2 failed")
	storage := newMockRepo()
	storage.seed(newAgent("writer"))
	svc := newTestService(
		t,
		storage,
		newMockModelRegistry("model-a"),
		newMockToolRegistry("filesystem"),
		&mockSync{err: err1},
		&mockSync{err: err2},
	)

	err := svc.Delete("writer")
	if !errors.Is(err, err1) || !errors.Is(err, err2) {
		t.Fatalf("Delete() error = %v, want both sync errors joined", err)
	}
}

func TestDelete_WithoutSyncs(t *testing.T) {
	storage := newMockRepo()
	storage.seed(newAgent("writer"))
	svc := newTestService(t, storage, newMockModelRegistry("model-a"), newMockToolRegistry("filesystem"))

	if err := svc.Delete("writer"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if got := storage.deletedIDs(); !slices.Equal(got, []agent.ID{"writer"}) {
		t.Fatalf("expected storage delete of writer, got %v", got)
	}
}