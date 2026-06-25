package files

import (
	"arch-agent/internal/mcp"
	"encoding/json"
	"fmt"
	"os"
)

const mcpServersFile = "mcp_servers.json"

type mcpServerDTO struct {
	ID        mcp.MCPServerID `json:"id"`
	URL       string          `json:"url"`
	Connected bool            `json:"connected"`
}

type MCPFiles struct {
	fs *FileSystem
}

func NewMCPFiles(fs *FileSystem) *MCPFiles {
	return &MCPFiles{fs: fs}
}

func (f *MCPFiles) Save(servers []*mcp.MCPServer) error {
	dtos := make([]mcpServerDTO, len(servers))
	for i := range servers {
		dtos[i] = mcpServerDTO{
			ID:        servers[i].ID,
			URL:       servers[i].URL,
			Connected: servers[i].Connected,
		}
	}

	data, err := json.MarshalIndent(dtos, "", "	")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", mcpServersFile, err)
	}

	return f.fs.WriteToFile(mcpServersFile, data)
}

func (f *MCPFiles) Load() ([]*mcp.MCPServer, error) {
	data, err := f.fs.ReadFile(mcpServersFile)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", mcpServersFile, err)
	}

	if len(data) == 0 {
		return nil, nil
	}

	var dtos []mcpServerDTO
	if err := json.Unmarshal(data, &dtos); err != nil {
		return nil, fmt.Errorf("unmarshal %s: %w", mcpServersFile, err)
	}

	servers := make([]*mcp.MCPServer, len(dtos))
	for i, dto := range dtos {

		srv, err := mcp.NewMCPServer(mcp.WithState(dto.ID, dto.URL, dto.Connected))
		if err != nil {
			return nil, err
		}

		servers[i] = srv
	}

	return servers, nil
}
