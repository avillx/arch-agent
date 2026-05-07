package settingfiles

import (
	oaiadapter "arch-agent/internal/infra/openai"
	"arch-agent/internal/infra/storage/filesystem"
	"os"

	"github.com/segmentio/encoding/json"
)

const settingsFile = "settings.json"

type Storage struct {
	// MCPSettings           *RepoChangeNotifier[mcpadapter.MCPSettings]
	ReflectionSettings    *RepoChangeNotifier[oaiadapter.LLMSettings]
	ReasoningSettings     *RepoChangeNotifier[oaiadapter.LLMSettings]
	SummarizationSettings *RepoChangeNotifier[oaiadapter.LLMSettings]
	DreamingSettings      *RepoChangeNotifier[oaiadapter.LLMSettings]
	fs                    filesystem.FileSystem
}

func New(dir string) (*Storage, error) {
	fs, err := filesystem.New(dir)
	if err != nil {
		return nil, err
	}
	r := &Storage{
		// MCPSettings:           &RepoChangeNotifier[mcpadapter.MCPSettings]{},
		ReflectionSettings:    &RepoChangeNotifier[oaiadapter.LLMSettings]{},
		ReasoningSettings:     &RepoChangeNotifier[oaiadapter.LLMSettings]{},
		SummarizationSettings: &RepoChangeNotifier[oaiadapter.LLMSettings]{},
		DreamingSettings:      &RepoChangeNotifier[oaiadapter.LLMSettings]{},
		fs:                    fs,
	}
	if err := r.load(); err != nil {
		return nil, err
	}
	return r, nil
}

func (sr *Storage) SetSettings(s Settings) error {
	sr.apply(s)
	return sr.save()
}

func (sr *Storage) apply(s Settings) {
	// sr.MCPSettings.SetValue(s.MCP)
	sr.ReflectionSettings.SetValue(s.Reflection)
	sr.ReasoningSettings.SetValue(s.Reasoning)
	sr.SummarizationSettings.SetValue(s.Summarization)
	sr.DreamingSettings.SetValue(s.Dreaming)
}

func (sr *Storage) load() error {
	data, err := sr.fs.ReadFile(settingsFile)
	if err != nil && os.IsNotExist(err) {
		return sr.save()
	}
	if err != nil {
		return err
	}

	var s Settings
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	sr.apply(s)
	return nil
}

func (sr *Storage) save() error {
	s := Settings{
		// MCP:           sr.MCPSettings.Value(),
		Reflection:    sr.ReflectionSettings.Value(),
		Reasoning:     sr.ReasoningSettings.Value(),
		Summarization: sr.SummarizationSettings.Value(),
		Dreaming:      sr.DreamingSettings.Value(),
	}

	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return sr.fs.WriteToFile(settingsFile, data)
}
