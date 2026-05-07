package di

import (
	"arch-agent/internal/infra/files"
)

type StoragesDI struct {
	Root *files.FileSystem
	LLM  *files.LLMFiles
	Secrets *files.SecretsFiles
	Agent   *files.AgentFiles
}

func BuildFileStorage(datapath string) (*StoragesDI, error) {
	root, err := files.NewFS(datapath)
	if err != nil {
		return nil, err
	}

	sf, err := files.NewSecretsFiles(root)
	if err != nil {
		return nil, err
	}

	return &StoragesDI{
		Root:    root,
		LLM:     files.NewLLMFiles(root.Sub("llm")),
		Agent:   files.NewAgentFiles(root.Sub("agents")),
		Secrets: sf,
	}, nil
}
