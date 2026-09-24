package memory_test

import (
	"log/slog"
	"reflect"
	"sync"
	"testing"
	"time"

	"arch-agent/internal/agent"
	"arch-agent/internal/memory"
	"arch-agent/internal/types"
)

type consolidationModelRegistry struct {
	models map[string]agent.Model
}

func (r *consolidationModelRegistry) Get(name string) (agent.Model, error) {
	model, ok := r.models[name]
	if !ok {
		return nil, types.ErrIsNotExist
	}
	return model, nil
}

type memoryRepoStub struct {
	mu  sync.Mutex
	cfg memory.ConsolidatorConfig

	loadErr error
	saveErr error
}

func (r *memoryRepoStub) Load() (memory.ConsolidatorConfig, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.cfg, r.loadErr
}

func (r *memoryRepoStub) Save(cfg memory.ConsolidatorConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.saveErr != nil {
		return r.saveErr
	}

	r.cfg = cfg
	r.loadErr = nil
	return nil
}

type consolidationAgentRepo struct{}

func (*consolidationAgentRepo) All() ([]agent.Agent, error) { return nil, nil }
func (*consolidationAgentRepo) Get(agent.ID) (agent.Agent, error) {
	return nil, types.ErrIsNotExist
}
func (*consolidationAgentRepo) Save(agent.Agent) error { return nil }
func (*consolidationAgentRepo) Delete(agent.ID) error  { return nil }

type consolidationToolServer struct{}

func (*consolidationToolServer) Tools() []agent.Tool { return nil }

func newConsolidationService(
	t *testing.T,
	registry agent.ModelRegistry,
	repo memory.MemoryRepo,
) *memory.ConsolidationService {
	t.Helper()

	svc, err := memory.NewConsolidationService(
		&consolidationAgentRepo{},
		[]agent.ToolServer{&consolidationToolServer{}},
		func(agent.ID) []any { return nil },
		registry,
		repo,
		slog.New(slog.DiscardHandler),
	)
	if err != nil {
		t.Fatalf("new consolidation service: %v", err)
	}

	return svc
}

func TestConsolidationService_SaveConfigKeepsOnlyValidModel(t *testing.T) {
	valid := memory.ConsolidatorConfig{
		Enabled:     true,
		Model:       "valid-model",
		Instruction: "consolidate",
	}

	repo := &memoryRepoStub{}
	registry := &consolidationModelRegistry{models: map[string]agent.Model{
		"valid-model": &stubModel{},
	}}
	svc := newConsolidationService(t, registry, repo)

	if err := svc.SaveConfig(valid); err != nil {
		t.Fatalf("save valid config: %v", err)
	}

	stored, err := repo.Load()
	if err != nil {
		t.Fatalf("load saved config: %v", err)
	}
	if !reflect.DeepEqual(stored, valid) {
		t.Fatalf("stored config = %+v, want %+v", stored, valid)
	}

	invalid := memory.ConsolidatorConfig{
		Enabled: true,
		Model:   "missing-model",
	}
	if err := svc.SaveConfig(invalid); err == nil {
		t.Fatal("expected error when saving config with missing model")
	}

	stored, err = repo.Load()
	if err != nil {
		t.Fatalf("load config after invalid save: %v", err)
	}
	if !reflect.DeepEqual(stored, valid) {
		t.Fatalf("invalid config overwrote stored config: got %+v, want %+v", stored, valid)
	}
}

func TestConsolidationService_SaveConfigAppliesOnlyOnReload(t *testing.T) {
	initial := memory.ConsolidatorConfig{
		Enabled:     true,
		Model:       "model-a",
		Instruction: "a",
	}
	repo := &memoryRepoStub{cfg: initial}
	registry := &consolidationModelRegistry{models: map[string]agent.Model{
		"model-a": &stubModel{},
		"model-b": &stubModel{},
	}}
	svc := newConsolidationService(t, registry, repo)

	before := svc.Config()
	if !reflect.DeepEqual(before, initial) {
		t.Fatalf("initial config = %+v, want %+v", before, initial)
	}

	saved := memory.ConsolidatorConfig{
		Enabled:     false,
		Model:       "model-b",
		Instruction: "b",
	}
	if err := svc.SaveConfig(saved); err != nil {
		t.Fatalf("save config: %v", err)
	}

	if got := svc.Config(); !reflect.DeepEqual(got, before) {
		t.Fatalf("SaveConfig changed service config: got %+v, want %+v", got, before)
	}

	stored, err := repo.Load()
	if err != nil {
		t.Fatalf("load saved config: %v", err)
	}
	if !reflect.DeepEqual(stored, saved) {
		t.Fatalf("stored config = %+v, want %+v", stored, saved)
	}

	if err := svc.Reload(); err != nil {
		t.Fatalf("reload saved config: %v", err)
	}
	if got := svc.Config(); !reflect.DeepEqual(got, saved) {
		t.Fatalf("Reload did not apply saved config: got %+v, want %+v", got, saved)
	}
}

func TestNextConsolidation(t *testing.T) {
	loc := time.UTC

	tests := []struct {
		name string
		now  time.Time
		want time.Time
	}{
		{
			name: "before daily run",
			now:  time.Date(2024, 3, 10, 10, 30, 0, 0, loc),
			want: time.Date(2024, 3, 10, 23, 0, 0, 0, loc),
		},
		{
			name: "after daily run",
			now:  time.Date(2024, 3, 10, 23, 30, 0, 0, loc),
			want: time.Date(2024, 3, 11, 23, 0, 0, 0, loc),
		},
		{
			name: "exactly at daily run",
			now:  time.Date(2024, 3, 10, 23, 0, 0, 0, loc),
			want: time.Date(2024, 3, 11, 23, 0, 0, 0, loc),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := memory.NextConsolidation(tt.now); !got.Equal(tt.want) {
				t.Fatalf("NextConsolidation(%v) = %v, want %v", tt.now, got, tt.want)
			}
		})
	}
}

func TestNextConsolidationRunsOncePerDay(t *testing.T) {
	now := time.Date(2024, 3, 10, 10, 0, 0, 0, time.UTC)

	first := memory.NextConsolidation(now)
	second := memory.NextConsolidation(first)

	if diff := second.Sub(first); diff != 24*time.Hour {
		t.Fatalf("NextConsolidation must schedule a run once per day, got interval %v", diff)
	}
}
