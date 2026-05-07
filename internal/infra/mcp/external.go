package mcprecivier

import (
	"arch-agent/internal/app/types"
	"context"
	"errors"
	"maps"
	"slices"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var ErrNoMCPPrompt = errors.New("mcp server has no prompt")

type ExternalServer struct {
	id          string
	client      *mcp.Client
	session     *mcp.ClientSession
	tools       map[string]types.ToolDefinition
	agentPrompt string
}

func NewExternalServer(ctx context.Context, id, endpoint string) (*ExternalServer, error) {

	transport := &mcp.StreamableClientTransport{Endpoint: endpoint}

	client := mcp.NewClient(&mcp.Implementation{Name: id, Version: "v1.0.0"}, nil)
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return nil, err
	}

	sessionTools, err := pullTools(session)
	if err != nil {
		return nil, err
	}

	agentPrompt, err := extractAgentPrompt(session, "agent")
	if err == nil && !errors.Is(err, ErrNoMCPPrompt) {
		return nil, err
	}

	return &ExternalServer{
		id:          id,
		client:      client,
		session:     session,
		tools:       createToolMap(sessionTools),
		agentPrompt: agentPrompt,
	}, nil
}

func (s *ExternalServer) ID() string {
	return s.id
}

func (s *ExternalServer) PromptForAgent() string {
	return s.agentPrompt
}

func (s *ExternalServer) Tools() []types.ToolDefinition {
	return slices.Collect(maps.Values(s.tools))
}

func (s *ExternalServer) SendCall(ctx context.Context, call *types.ToolCall, agentID string) (string, error) {
	callParams, err := toCallToolParams(call)
	if err != nil {
		return "", err
	}

	callParams.SetMeta(map[string]any{
		"agent": agentID,
	})

	result, err := s.session.CallTool(ctx, callParams)
	if err != nil {
		return "", err
	}

	return resultToString(result), nil
}
