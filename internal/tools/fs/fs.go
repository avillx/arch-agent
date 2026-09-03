package fstools

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/files"
	"arch-agent/internal/prompt"
	"arch-agent/internal/tools"
	"arch-agent/internal/types"
	"errors"
)

type ConsolidationInstuctFS struct {
	agent.ToolServer
	cwd string
}

func NewConsolidationInstuctFS(fs *files.FileSystem, skipPatterns []string) (*ConsolidationInstuctFS, error) {

	ts, err := NewRawFileSystemToolServer(fs, skipPatterns)
	if err != nil {
		return nil, err
	}
	return &ConsolidationInstuctFS{
		ToolServer: ts,
		cwd:        fs.Cwd(),
	}, nil
}

func (r *ConsolidationInstuctFS) AgentInstruction(agt agent.Agent) string {
	return prompt.ConsolidationFSInstruction(r.cwd, agt.ID())
}

type FileSystemToolServer struct {
	*tools.BuildInToolServer
	fs *files.FileSystem
}

func NewFileSystemToolServer(fs *files.FileSystem, skipPatterns []string) (*FileSystemToolServer, error) {

	ts, err := NewRawFileSystemToolServer(fs, skipPatterns)
	if err != nil {
		return nil, err
	}

	return &FileSystemToolServer{
		fs:                fs,
		BuildInToolServer: ts,
	}, nil
}

func NewRawFileSystemToolServer(fs *files.FileSystem, skipPatterns []string) (*tools.BuildInToolServer, error) {

	findTool, err := NewFindTool(fs, skipPatterns)
	if err != nil {
		return nil, err
	}

	readTool, err := NewReadTool(fs, skipPatterns)
	if err != nil {
		return nil, err
	}

	return tools.NewBuildInToolServer(
		&EditTool{fs: fs},
		&MoveTool{fs: fs},
		readTool,
		findTool,
		&WriteTool{fs: fs},
	), nil
}

func (r *FileSystemToolServer) AgentInstruction(agt agent.Agent) string {
	return prompt.FileSystemInstruction(r.fs.Cwd(), agt.ID(), agt.HasMemory())
}

func mapErrs(err error) error {
	if errors.Is(err, types.ErrIsNotExist) {
		return types.NewAgentMistakeError("path is not found, ensure path existence")
	}
	return err
}
