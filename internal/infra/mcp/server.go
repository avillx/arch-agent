package mcpadapter

import (
	"arch-agent/internal/infra/llm"
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type InternalServer struct {
	server *mcp.Server
}

func NewInternalServer(name string) *InternalServer {
	return &InternalServer{
		server: mcp.NewServer(&mcp.Implementation{Name: name, Version: "1.0.0"}, nil),
	}
}

func (s *InternalServer) Connect(ctx context.Context, transport *mcp.InMemoryTransport) (*mcp.ServerSession, error) {
	return s.server.Connect(ctx, transport, nil)
}

func (s *InternalServer) AddTools(internalTools []llm.Tool) {
	for _, t := range internalTools {
		s.AddTool(t)
	}
}

func (s *InternalServer) AddTool(tool llm.Tool) {
	mcpTool := toMCPTool(tool.ToolDefinition)
	s.server.AddTool(&mcpTool, WrapForMCP(tool.CallRsolver))
}
