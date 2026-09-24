package model_test

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/model"
	"arch-agent/internal/types"
	"context"
	"errors"
	"sync"
	"testing"
)

type mockModel struct {
	settings agent.ModelSettings
}

func (m mockModel) Settings() agent.ModelSettings { return m.settings }
func (m mockModel) Complete(context.Context, []agent.Tool, []agent.Message) (*agent.Completion, error) {
	return &agent.Completion{Done: true}, nil
}
func (m mockModel) ContextLimit() int64                   { return 0 }
func (m mockModel) SupportedModalities() []agent.Modality { return nil }

type factoryCall struct {
	baseURL      string
	keyReference string
	settings     agent.ModelSettings
}

type mockFactory struct {
	apiType model.APIType

	mu    sync.Mutex
	calls []factoryCall
}

func (f *mockFactory) APIType() model.APIType { return f.apiType }

func (f *mockFactory) CreateModel(baseURL, keyReference string, settings agent.ModelSettings) (agent.Model, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.calls = append(f.calls, factoryCall{
		baseURL:      baseURL,
		keyReference: keyReference,
		settings:     settings,
	})

	return mockModel{settings: settings}, nil
}

func (f *mockFactory) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return len(f.calls)
}

type mockRepo struct {
	mu   sync.Mutex
	cfgs map[model.ProviderID]model.ProviderConfig
}

func newMockRepo(cfgs ...model.ProviderConfig) *mockRepo {
	repo := &mockRepo{cfgs: map[model.ProviderID]model.ProviderConfig{}}
	for _, cfg := range cfgs {
		repo.cfgs[cfg.Name] = cfg
	}
	return repo
}

func (r *mockRepo) All() ([]model.ProviderConfig, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	out := make([]model.ProviderConfig, 0, len(r.cfgs))
	for _, cfg := range r.cfgs {
		out = append(out, cfg)
	}
	return out, nil
}

func (r *mockRepo) Get(id model.ProviderID) (model.ProviderConfig, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	cfg, ok := r.cfgs[id]
	if !ok {
		return model.ProviderConfig{}, types.ErrIsNotExist
	}
	return cfg, nil
}

func (r *mockRepo) Save(cfg model.ProviderConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.cfgs[cfg.Name] = cfg
	return nil
}

func (r *mockRepo) Delete(id model.ProviderID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.cfgs, id)
	return nil
}

func providerConfig(name model.ProviderID, models map[string]model.ModelConfig) model.ProviderConfig {
	return model.ProviderConfig{
		Name:         name,
		BaseURL:      "http://example.com",
		KeyReference: "openai-key",
		APIType:      model.APITypeOpenAI,
		Models:       models,
	}
}

func newProviderService(t *testing.T, modelSvc *model.ModelService, repo *mockRepo) *model.ProviderService {
	t.Helper()

	svc, err := model.NewProviderService(modelSvc, repo)
	if err != nil {
		t.Fatalf("NewProviderService() error = %v", err)
	}
	return svc
}

func newModelServiceWithModel(t *testing.T, providerID model.ProviderID, modelName string) *model.ModelService {
	t.Helper()

	modelSvc := model.NewModelService(&mockFactory{apiType: model.APITypeOpenAI})
	repo := newMockRepo(providerConfig(providerID, map[string]model.ModelConfig{}))
	svc := newProviderService(t, modelSvc, repo)

	if err := svc.SetModel(providerID, modelName, model.ModelConfig{}); err != nil {
		t.Fatalf("SetModel() error = %v", err)
	}
	return modelSvc
}

func TestGet_EmptyStringReturnsModel(t *testing.T) {
	modelSvc := newModelServiceWithModel(t, "p", "m")

	got, err := modelSvc.Get("")
	if err != nil {
		t.Fatalf("Get(\"\") error = %v", err)
	}
	if got == nil {
		t.Fatal("expected a model, got nil")
	}
}

func TestGet_SuffixReturnsModel(t *testing.T) {
	modelSvc := newModelServiceWithModel(t, "p", "m")

	got, err := modelSvc.Get("m")
	if err != nil {
		t.Fatalf("Get(\"m\") error = %v", err)
	}
	if got.Settings()["model"] != "m" {
		t.Fatalf("expected settings model name %q, got %v", "m", got.Settings()["model"])
	}
}

func TestGet_UnknownModelReturnsNotExist(t *testing.T) {
	modelSvc := newModelServiceWithModel(t, "p", "m")

	_, err := modelSvc.Get("ghost")
	if !errors.Is(err, types.ErrIsNotExist) {
		t.Fatalf("Get() error = %v, want %v", err, types.ErrIsNotExist)
	}
}

func TestSetModel_UnsupportedAPI(t *testing.T) {
	factory := &mockFactory{apiType: model.APITypeOpenAI}
	modelSvc := model.NewModelService(factory)
	repo := newMockRepo(model.ProviderConfig{
		Name:    "p",
		BaseURL: "http://example.com",
		APIType: "unsupported",
		Models:  map[string]model.ModelConfig{},
	})
	svc := newProviderService(t, modelSvc, repo)

	err := svc.SetModel("p", "m", model.ModelConfig{})
	if !errors.Is(err, model.ErrUnsupportedAPI) {
		t.Fatalf("SetModel() error = %v, want %v", err, model.ErrUnsupportedAPI)
	}
	if factory.callCount() != 0 {
		t.Fatalf("expected no factory calls, got %d", factory.callCount())
	}
}

func TestSetModel_RegistersWithProviderPrefixAndPlainSettings(t *testing.T) {
	factory := &mockFactory{apiType: model.APITypeOpenAI}
	modelSvc := model.NewModelService(factory)
	repo := newMockRepo(providerConfig("provider-a", map[string]model.ModelConfig{}))
	svc := newProviderService(t, modelSvc, repo)

	if err := svc.SetModel("provider-a", "model-a", model.ModelConfig{"temperature": float64(0.5)}); err != nil {
		t.Fatalf("SetModel() error = %v", err)
	}

	got, err := modelSvc.Get("provider-a/model-a")
	if err != nil {
		t.Fatalf("Get(%q) error = %v", "provider-a/model-a", err)
	}

	settings := got.Settings()
	if settings["model"] != "model-a" {
		t.Fatalf("expected settings model name %q, got %v", "model-a", settings["model"])
	}
	if settings["temperature"] != float64(0.5) {
		t.Fatalf("expected settings temperature %v, got %v", float64(0.5), settings["temperature"])
	}
}

func TestReload_LoadsModelsFromRepo(t *testing.T) {
	modelSvc := model.NewModelService(&mockFactory{apiType: model.APITypeOpenAI})
	repo := newMockRepo(providerConfig("p", map[string]model.ModelConfig{
		"a": {},
		"b": {},
	}))

	newProviderService(t, modelSvc, repo)

	for _, name := range []string{"p/a", "p/b"} {
		if _, err := modelSvc.Get(name); err != nil {
			t.Fatalf("expected model %s to be loaded, got error %v", name, err)
		}
	}
}

func TestSetModel_AddsModelToRegistry(t *testing.T) {
	modelSvc := model.NewModelService(&mockFactory{apiType: model.APITypeOpenAI})
	repo := newMockRepo(providerConfig("p", map[string]model.ModelConfig{}))
	svc := newProviderService(t, modelSvc, repo)

	if err := svc.SetModel("p", "a", model.ModelConfig{}); err != nil {
		t.Fatalf("SetModel() error = %v", err)
	}

	if _, err := modelSvc.Get("p/a"); err != nil {
		t.Fatalf("expected model p/a in registry, got error %v", err)
	}
}

func TestDeleteModel_RemovesModelFromRegistry(t *testing.T) {
	modelSvc := model.NewModelService(&mockFactory{apiType: model.APITypeOpenAI})
	repo := newMockRepo(providerConfig("p", map[string]model.ModelConfig{
		"a": {},
	}))
	svc := newProviderService(t, modelSvc, repo)

	if err := svc.DeleteModel("p", "a"); err != nil {
		t.Fatalf("DeleteModel() error = %v", err)
	}

	if _, err := modelSvc.Get("p/a"); !errors.Is(err, types.ErrIsNotExist) {
		t.Fatalf("expected model p/a removed, got error %v", err)
	}
}

func TestDeleteProvider_RemovesAllProviderModels(t *testing.T) {
	modelSvc := model.NewModelService(&mockFactory{apiType: model.APITypeOpenAI})
	repo := newMockRepo(providerConfig("p", map[string]model.ModelConfig{
		"a": {},
		"b": {},
	}))
	svc := newProviderService(t, modelSvc, repo)

	if err := svc.DeleteProvider("p"); err != nil {
		t.Fatalf("DeleteProvider() error = %v", err)
	}

	for _, name := range []string{"p/a", "p/b"} {
		if _, err := modelSvc.Get(name); !errors.Is(err, types.ErrIsNotExist) {
			t.Fatalf("expected model %s removed, got error %v", name, err)
		}
	}
}

func TestUpdateProvider_RemovesProviderModels(t *testing.T) {
	modelSvc := model.NewModelService(&mockFactory{apiType: model.APITypeOpenAI})
	repo := newMockRepo(providerConfig("p", map[string]model.ModelConfig{
		"a": {},
	}))
	svc := newProviderService(t, modelSvc, repo)

	baseURL := "http://new.example.com"
	if err := svc.UpdateProvider("p", model.ProviderConfigPatch{BaseURL: &baseURL}); err != nil {
		t.Fatalf("UpdateProvider() error = %v", err)
	}

	if _, err := modelSvc.Get("p/a"); !errors.Is(err, types.ErrIsNotExist) {
		t.Fatalf("expected model p/a removed after update, got error %v", err)
	}
}
