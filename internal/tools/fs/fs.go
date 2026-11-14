package fstools

import (
	"ard/internal/agent"
	"ard/internal/files"
	"ard/internal/prompt"
	"ard/internal/tools"
	"ard/internal/types"
	"errors"
)

type ConsolidationInstuctFS struct {
	agent.ToolServer

	cwd string
}

func NewConsolidationInstuctFS(storage files.FileStorage, skipPatterns []string) (*ConsolidationInstuctFS, error) {
	ts, err := NewRawFileSystemToolServer(storage, skipPatterns)
	if err != nil {
		return nil, err
	}
	return &ConsolidationInstuctFS{
		ToolServer: ts,
		cwd:        "", // TODO : eliminate
	}, nil
}

func (r *ConsolidationInstuctFS) AgentInstruction(agt agent.Agent) string {
	return prompt.ConsolidationFSInstruction(r.cwd, agt.ID())
}

type FileSystemToolServer struct {
	*tools.BuildInToolServer

	storage files.FileStorage
}

func NewFileSystemToolServer(storage files.FileStorage, skipPatterns []string) (*FileSystemToolServer, error) {
	ts, err := NewRawFileSystemToolServer(storage, skipPatterns)
	if err != nil {
		return nil, err
	}

	return &FileSystemToolServer{
		storage:           storage,
		BuildInToolServer: ts,
	}, nil
}

func NewRawFileSystemToolServer(storage files.FileStorage, skipPatterns []string) (*tools.BuildInToolServer, error) {
	findTool, err := NewFindTool(storage, skipPatterns)
	if err != nil {
		return nil, err
	}

	readTool, err := NewReadTool(storage, skipPatterns)
	if err != nil {
		return nil, err
	}

	return tools.NewBuildInToolServer(
		&EditTool{storage: storage},
		&MoveTool{storage: storage},
		readTool,
		findTool,
		&WriteTool{storage: storage},
	), nil
}

func (r *FileSystemToolServer) AgentInstruction(agt agent.Agent) string {
	return prompt.FileSystemInstruction("", agt.ID(), agt.HasMemory())
}

func mapErrs(err error) error {
	if errors.Is(err, types.ErrIsNotExist) {
		return types.NewAgentMistakeError("path is not found, ensure path existence")
	}
	return err
}
