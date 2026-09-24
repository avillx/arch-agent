package memory_test

import (
	"context"
	"errors"
	"log/slog"
	"reflect"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"arch-agent/internal/agent"
	"arch-agent/internal/memory"
	"arch-agent/internal/session"
)

var (
	testAgentID               agent.ID   = "agent-1"
	testSessionID             session.ID = "session-1"
	testSummary                          = "activity summary"
	testActivityRepoModelName            = "activity-model"
)

type stubModel struct {
	content string
}

func (m *stubModel) Settings() agent.ModelSettings {
	return agent.ModelSettings{}
}

func (m *stubModel) Complete(context.Context, []agent.Tool, []agent.Message) (*agent.Completion, error) {
	return &agent.Completion{
		Content: m.content,
		Done:    true,
	}, nil
}

func (m *stubModel) ContextLimit() int64 {
	return 0
}

func (m *stubModel) SupportedModalities() []agent.Modality {
	return nil
}

type stubModelRegistry struct {
	model agent.Model
}

func (r *stubModelRegistry) Get(string) (agent.Model, error) {
	if r.model == nil {
		return nil, errors.New("model is not registered")
	}
	return r.model, nil
}

type stubActivityRepo struct {
	mu      sync.Mutex
	logged  chan agent.ActivityRecord
	records []agent.ActivityRecord
}

func (r *stubActivityRepo) Log(_ agent.ID, record agent.ActivityRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.records = append(r.records, record)
	if r.logged != nil {
		r.logged <- record
	}
	return nil
}

func (r *stubActivityRepo) GetActivity(agent.ID, time.Time) (string, error) {
	return "", nil
}

func (r *stubActivityRepo) GetRange(agent.ID, time.Time, time.Time) ([]agent.ActivityLog, error) {
	return nil, nil
}

type stubActivityConfigRepo struct {
	mu      sync.Mutex
	cfg     memory.ActivityConfig
	loadErr error
}

func (r *stubActivityConfigRepo) Load() (memory.ActivityConfig, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.cfg, r.loadErr
}

func (r *stubActivityConfigRepo) Save(cfg memory.ActivityConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.cfg = cfg
	r.loadErr = nil
	return nil
}

func newTestActivityService(t *testing.T, cfgRepo memory.ActivityConfigRepo, activityRepo agent.ActivityRepo) *memory.ActivityService {
	t.Helper()

	return memory.NewActivityService(
		&stubModelRegistry{model: &stubModel{content: testSummary}},
		cfgRepo,
		activityRepo,
		slog.New(slog.DiscardHandler),
	)
}

func TestActivityService_ReloadErrorKeepsConfig(t *testing.T) {
	cfgRepo := &stubActivityConfigRepo{
		cfg: memory.ActivityConfig{
			Enabled:   true,
			Interval:  30,
			ModelName: testActivityRepoModelName,
		},
	}
	svc := newTestActivityService(t, cfgRepo, &stubActivityRepo{})
	before := svc.Config()

	cfgRepo.loadErr = errors.New("broken config")

	if err := svc.Reload(); err == nil {
		t.Fatal("expected Reload to fail")
	}
	if got := svc.Config(); !reflect.DeepEqual(got, before) {
		t.Fatalf("config changed after failed reload: got %+v, want %+v", got, before)
	}
}

func TestActivityService_SaveConfigAppliesOnlyOnReload(t *testing.T) {
	cfgRepo := &stubActivityConfigRepo{
		cfg: memory.ActivityConfig{
			Enabled:   true,
			Interval:  30,
			ModelName: testActivityRepoModelName,
		},
	}
	svc := newTestActivityService(t, cfgRepo, &stubActivityRepo{})
	before := svc.Config()

	saved := memory.ActivityConfig{
		Enabled:   false,
		Interval:  45,
		ModelName: "another-activity-model",
	}
	if err := svc.SaveConfig(saved); err != nil {
		t.Fatalf("save config: %v", err)
	}

	if got := svc.Config(); !reflect.DeepEqual(got, before) {
		t.Fatalf("SaveConfig changed service config: got %+v, want %+v", got, before)
	}

	stored, err := cfgRepo.Load()
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

func TestActivityService_FlushesOnInterval(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cfgRepo := &stubActivityConfigRepo{
			cfg: memory.ActivityConfig{
				Enabled:   true,
				Interval:  1,
				ModelName: testActivityRepoModelName,
			},
		}
		activityRepo := &stubActivityRepo{
			logged: make(chan agent.ActivityRecord, 1),
		}
		svc := newTestActivityService(t, cfgRepo, activityRepo)

		svc.Commit(testAgentID, testSessionID, []agent.Message{
			agent.NewUserMessage("hello"),
		})

		select {
		case record := <-activityRepo.logged:
			if record.Content != testSummary {
				t.Fatalf("flushed content = %q, want %q", record.Content, testSummary)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("activity was not flushed on interval")
		}
	})
}
