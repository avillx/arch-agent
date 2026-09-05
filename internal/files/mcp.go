package files

import (
	"arch-agent/internal/mcp"
	"arch-agent/internal/sentinel"
	"arch-agent/internal/types"
	"bytes"
	"context"
	"fmt"
	"sync"

	"github.com/fsnotify/fsnotify"
	toml "github.com/pelletier/go-toml/v2"
)

const MCPConfigFile = "mcp.toml"
const mcpConfigDoc = `# MCP servers connections config
# All MCP servers described here will be connected

# Unique MCP server id
# [my-mcp]

# Address for HTTP transport
# url = 'https://example.com/mcp'

# Auth token for server
# token = '<bearer token>'

# Unique MCP server id
# [other-mcp]

# Shell command for stdio transport
# command = 'uvx'

# Args for command
# args = ['my-mcp', '--arg', 'arg']

# Environment variables for stdio process
# [my-command-mcp.env]
# MY_ENVIRONMENT_VARIABLE = '1'
# ANOTHER_ENV = '{ env.SECRET_REFERENCES }'

# Do not touch this comment!
# After edit, ensure file consistency and comment integrity`

var _ mcp.ConfigRepo = (*MCPFiles)(nil)

type MCPFiles struct {
	fs *FileSystem

	mu sync.Mutex
}

func NewMCPFiles(fs *FileSystem) (*MCPFiles, error) {

	if err := ensureFilePlaceholder(fs, MCPConfigFile, []byte(mcpConfigDoc)); err != nil {
		return nil, err
	}

	return &MCPFiles{fs: fs}, nil
}

func (f *MCPFiles) Save(id mcp.MCPServerID, cfg mcp.ServerGatewayConfig) error {

	f.mu.Lock()
	defer f.mu.Unlock()

	dtoMap, err := f.LoadDTO()
	if err != nil {
		return err
	}

	dtoMap[id] = cfg

	data, err := MarshalMCPConfig(dtoMap)
	if err != nil {
		return fmt.Errorf("marshal %s: %w", MCPConfigFile, err)
	}

	return f.fs.WriteToFile(MCPConfigFile, data)
}

func (f *MCPFiles) Load() (map[mcp.MCPServerID]mcp.ServerGatewayConfig, error) {

	f.mu.Lock()
	defer f.mu.Unlock()

	dto, err := f.LoadDTO()
	if err != nil {
		return nil, err
	}

	return dto, nil
}

func (f *MCPFiles) LoadDTO() (map[mcp.MCPServerID]mcp.ServerGatewayConfig, error) {
	data, err := f.fs.ReadFile(MCPConfigFile)
	if err != nil {
		return nil, err
	}

	return UnmarshalMCPConfig(data)
}

type MCPServerConfigDTO struct {
	URL     string            `json:"url" toml:"url,omitempty"`
	Token   string            `json:"token,omitempty" toml:"token,omitempty"`
	Command string            `json:"command" toml:"command,omitempty"`
	Args    []string          `json:"args,omitempty" toml:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty" toml:"env,omitempty"`
}

func ValidateMCPServerConfigDTO(data []byte) error {

	var cfg map[mcp.MCPServerID]MCPServerConfigDTO
	dec := toml.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()

	if err := dec.Decode(&cfg); err != nil {
		return err
	}

	problems := map[string]string{}
	for k, dto := range cfg {

		if dto.Command != "" && dto.URL != "" {
			p := "allowed only one transport 'url' for http or 'command' for stdio"
			problems[string(k)] = p
			continue
		}

		if dto.URL != "" {
			if dto.Args != nil {
				p := "field 'args' not allowed for http transport"
				problems[string(k)] = p
			}
			if dto.Env != nil {
				p := "table 'env' not allowed for http transport"
				problems[string(k)] = p
			}
		}

		if dto.Command != "" && dto.Token != "" {
			p := "field 'token' allowed only for 'http' transport, but used with 'command'"
			problems[string(k)] = p
		}

	}

	if len(problems) > 0 {
		return types.NewValidationError(problems)
	}

	return nil
}

func MarshalMCPConfig(config map[mcp.MCPServerID]mcp.ServerGatewayConfig) ([]byte, error) {

	newMCPConfig := map[mcp.MCPServerID]MCPServerConfigDTO{}
	for k, v := range config {

		if v.CommandGateway == nil {
			v.CommandGateway = &mcp.CommandGatewayConfig{}
		}

		if v.HTTPGateway == nil {
			v.HTTPGateway = &mcp.HTTPGatewayConfig{}
		}

		newMCPConfig[k] = MCPServerConfigDTO{
			URL:     v.HTTPGateway.URL,
			Token:   v.HTTPGateway.Token,
			Command: v.CommandGateway.Command,
			Args:    v.CommandGateway.Args,
			Env:     v.CommandGateway.Env,
		}
	}

	data, err := toml.Marshal(newMCPConfig)
	if err != nil {
		return nil, err
	}

	return bytes.Join(
		[][]byte{[]byte(mcpConfigDoc), data},
		[]byte("\n\n"),
	), nil
}

func UnmarshalMCPConfig(data []byte) (map[mcp.MCPServerID]mcp.ServerGatewayConfig, error) {

	var dto map[mcp.MCPServerID]MCPServerConfigDTO
	if err := toml.Unmarshal(data, &dto); err != nil {
		return nil, err
	}

	config := map[mcp.MCPServerID]mcp.ServerGatewayConfig{}
	for k, v := range dto {

		var commandGateway *mcp.CommandGatewayConfig
		if v.Command != "" {
			commandGateway = &mcp.CommandGatewayConfig{
				Command: v.Command,
				Args:    v.Args,
				Env:     v.Env,
			}
		}

		var httpGateway *mcp.HTTPGatewayConfig
		if v.URL != "" {
			httpGateway = &mcp.HTTPGatewayConfig{
				URL:   v.URL,
				Token: v.Token,
			}
		}

		config[k] = mcp.ServerGatewayConfig{
			HTTPGateway:    httpGateway,
			CommandGateway: commandGateway,
		}
	}

	return config, nil
}

// mcpSent
func NewMCPReloader(mcpSvc *mcp.Service) sentinel.Action {
	return func(ctx context.Context, ev fsnotify.Event) error {
		return mcpSvc.Reload(ctx)
	}
}
