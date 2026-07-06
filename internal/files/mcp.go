package files

import (
	"arch-agent/internal/mcp"
	"encoding/json"
	"fmt"
	"sync"
)

const mcpServersFile = "mcp_servers.json"

var _ mcp.ConfigRepo = (*MCPFiles)(nil)

type mcpServerDTO struct {
	Connected bool `json:"connected"`
	mcp.ServerGatewayConfig
}

type MCPFiles struct {
	fs *FileSystem

	mu sync.Mutex
}

func NewMCPFiles(fs *FileSystem) *MCPFiles {
	return &MCPFiles{fs: fs}
}

func (f *MCPFiles) Save(cfg mcp.ServerConfig) error {

	f.mu.Lock()
	defer f.mu.Unlock()

	dtoMap, err := f.loadDTO()
	if err != nil {
		return err
	}

	dtoMap[cfg.ID] = mcpServerDTO{
		Connected:           cfg.Connected,
		ServerGatewayConfig: cfg.ServerGatewayConfig,
	}

	data, err := json.MarshalIndent(dtoMap, "", "	")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", mcpServersFile, err)
	}

	return f.fs.WriteToFile(mcpServersFile, data)
}

func (f *MCPFiles) Load() ([]mcp.ServerConfig, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	dto, err := f.loadDTO()
	if err != nil {
		return nil, err
	}

	configs := make([]mcp.ServerConfig, 0, len(dto))
	for k, v := range dto {
		configs = append(configs, mcp.ServerConfig{
			ID:                  k,
			Connected:           v.Connected,
			ServerGatewayConfig: v.ServerGatewayConfig,
		})
	}

	return configs, nil
}

func (f *MCPFiles) loadDTO() (map[mcp.MCPServerID]mcpServerDTO, error) {
	data, err := f.fs.ReadFile(mcpServersFile)
	if err != nil {
		return nil, err
	}

	var dto map[mcp.MCPServerID]mcpServerDTO
	if err := json.Unmarshal(data, &dto); err != nil {
		return nil, err
	}

	return dto, nil
}
