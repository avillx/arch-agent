package files

import (
	"arch-agent/internal/model"
	"encoding/json"
	"fmt"
)

const modelSettingsFile = "models.json"

type ModelFiles struct {
	fs *FileSystem
}

func NewModelFiles(fs *FileSystem) (*ModelFiles, error) {
	return &ModelFiles{fs: fs}, nil
}

func (f *ModelFiles) Load() (model.ModelsConfig, error) {
	data, err := f.fs.ReadFile(modelSettingsFile)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", modelSettingsFile, err)
	}

	var config model.ModelsConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("unmarshal %s: %w", modelSettingsFile, err)
	}

	return config, nil
}

func (f *ModelFiles) Save(config model.ModelsConfig) error {
	data, err := json.MarshalIndent(config, "", "	")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", modelSettingsFile, err)
	}

	return f.fs.WriteToFile(modelSettingsFile, data)
}
