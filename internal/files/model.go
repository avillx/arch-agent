package files

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/types"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
)

const modelSettingsFile = "models.json"

var _ agent.ModelRepository = (*ModelFiles)(nil)

type ModelFiles struct {
	fs             *FileSystem
	modelFactories map[string]func(settings agent.ModelSettings) (agent.Model, error)
	models         map[agent.ModelID]agent.Model
	mu             sync.RWMutex
}

func NewModelFiles(fs *FileSystem, opts ...ModelFilesOption) (*ModelFiles, error) {
	mf := &ModelFiles{
		fs:             fs,
		modelFactories: map[string]func(settings agent.ModelSettings) (agent.Model, error){},
		models:         map[agent.ModelID]agent.Model{},
	}
	for _, opt := range opts {
		opt(mf)
	}

	if err := mf.loadModels(); err != nil {
		return nil, err
	}

	return mf, nil
}

func (f *ModelFiles) Get(modelID agent.ModelID) (agent.Model, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	model, ok := f.models[modelID]
	if !ok {
		return nil, types.ErrIsNotExist
	}

	return model, nil
}

func (f *ModelFiles) Delete(modelID agent.ModelID) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	delete(f.models, modelID)

	return f.flushSettings()
}

func (f *ModelFiles) Save(modelID agent.ModelID, model agent.Model) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.models[modelID] = model

	return f.flushSettings()
}

func (f *ModelFiles) loadModels() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	data, err := f.fs.ReadFile(modelSettingsFile)
	if err != nil {
		return err
	}

	var settings map[agent.ModelID]agent.ModelSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return err
	}

	for modelID, modelSettings := range settings {
		model, err := f.produceModel(modelSettings)
		if err != nil {
			slog.Error("can't load model", "model", modelID, "error", err)
			continue
		}
		f.models[modelID] = model
	}

	return nil

}

func (f *ModelFiles) produceModel(settings agent.ModelSettings) (agent.Model, error) {
	modelType, ok := settings["type"]
	if !ok {
		return nil, errors.New("model with unknown type")
	}
	modelTypeStr, ok := modelType.(string)
	if !ok {
		return nil, errors.New("model type is must be string")
	}

	modelFactory, exist := f.modelFactories[modelTypeStr]
	if !exist {
		return nil, fmt.Errorf("can't produce, has no factory of type %s", modelTypeStr)

	}

	model, err := modelFactory(settings)
	if err != nil {
		return nil, err
	}

	return model, nil
}

func (f *ModelFiles) flushSettings() error {
	dto := map[agent.ModelID]agent.ModelSettings{}
	for modelID, model := range f.models {
		dto[modelID] = model.Settings()
	}

	data, err := json.MarshalIndent(dto, "", "	")
	if err != nil {
		return err
	}

	return f.fs.WriteToFile(modelSettingsFile, data)
}

type ModelFilesOption func(f *ModelFiles)

func WithFactory(modelType string, factory func(settings agent.ModelSettings) (agent.Model, error)) ModelFilesOption {
	return func(f *ModelFiles) {
		f.modelFactories[modelType] = factory
	}
}
