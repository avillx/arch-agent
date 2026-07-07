package files

import (
	"arch-agent/internal/mcp"
	"encoding/json"
	"fmt"
	"sync"
)

const mcpServersFile = "mcp_servers.json"

var _ mcp.ConfigRepo = (*MCPFiles)(nil)

type MCPFiles struct {
	fs *FileSystem

	mu sync.Mutex
}

func NewMCPFiles(fs *FileSystem) *MCPFiles {
	return &MCPFiles{fs: fs}
}

func (f *MCPFiles) Save(id mcp.MCPServerID, cfg mcp.ServerGatewayConfig) error {

	f.mu.Lock()
	defer f.mu.Unlock()

	dtoMap, err := f.loadDTO()
	if err != nil {
		return err
	}

	dtoMap[id] = cfg

	data, err := json.MarshalIndent(dtoMap, "", "	")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", mcpServersFile, err)
	}

	return f.fs.WriteToFile(mcpServersFile, data)
}

func (f *MCPFiles) Load() ([]mcp.ServerGatewayConfig, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	dto, err := f.loadDTO()
	if err != nil {
		return nil, err
	}

	configs := make([]mcp.ServerGatewayConfig, 0, len(dto))
	for _, v := range dto {
		configs = append(configs, v)
	}

	return configs, nil
}

func (f *MCPFiles) loadDTO() (map[mcp.MCPServerID]mcp.ServerGatewayConfig, error) {
	data, err := f.fs.ReadFile(mcpServersFile)
	if err != nil {
		return nil, err
	}

	var dto map[mcp.MCPServerID]mcp.ServerGatewayConfig
	if err := json.Unmarshal(data, &dto); err != nil {
		return nil, err
	}

	return dto, nil
}
