package mcp

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/types"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const healthInterval = 30 * time.Second

type MCPServerID string

var _ agent.ToolServer = (MCPServer)(nil)

type ServerGatewayConfig struct {
	HTTPGateway    *HTTPGatewayConfig    `json:"http_gateway,omitempty"`
	CommandGateway *CommandGatewayConfig `json:"command_gateway,omitempty"`
}

type HTTPGatewayConfig struct {
	URL   string `json:"url"`
	Token string `json:"token,omitempty"`
}

type CommandGatewayConfig struct {
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

func validateConfig(cfg ServerGatewayConfig) error {

	problems := map[string]string{}

	if cfg.HTTPGateway != nil &&
		cfg.CommandGateway != nil {

		problems["config"] = "must be at least one gateway (http or command)"
	}

	if cfg.HTTPGateway == nil &&
		cfg.CommandGateway == nil {

		problems["config"] = "must be at least one gateway (http or command)"
	}

	if len(problems) > 0 {
		return types.NewValidationError(problems)
	}

	return nil
}

type gateway interface {
	Type() string
	createSession(context.Context) (*mcp.ClientSession, error)
}

type MCPServer interface {
	ID() MCPServerID
	Gateway() gateway
	Run(ctx context.Context) error
	Shutdown()
	agent.ToolServer
}

type mcpServerInstructed struct {
	*mcpServer
	instruction string
}

func (t *mcpServerInstructed) Instruction() string {
	return fmt.Sprintf("## %s\n%s", t.id, t.instruction)
}

type mcpServer struct {
	id          MCPServerID
	Instruction string

	tools      []agent.Tool
	gateway    gateway
	shutdownCh chan error

	mu sync.Mutex
}

func (s *mcpServer) ID() MCPServerID  { return s.id }
func (s *mcpServer) Gateway() gateway { return s.gateway }

func NewMCPServer(ctx context.Context, cfg ServerGatewayConfig) (MCPServer, error) {

	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	g, err := createGateway(cfg)
	if err != nil {
		return nil, err
	}

	// init
	sess, err := g.createSession(ctx)
	if err != nil {
		return nil, err
	}
	defer sess.Close()

	initResult := sess.InitializeResult()
	if initResult == nil {
		return nil, fmt.Errorf("has no initial result")
	}

	serverInfo := initResult.ServerInfo
	if serverInfo == nil {
		return nil, fmt.Errorf("has no server info")
	}

	if initResult.Instructions != "" {
		return &mcpServerInstructed{
			mcpServer: &mcpServer{
				id:      MCPServerID(serverInfo.Name),
				gateway: g,
			},
			instruction: initResult.Instructions,
		}, nil
	}

	return &mcpServer{
		id:      MCPServerID(serverInfo.Name),
		gateway: g,
	}, nil
}

func (s *mcpServer) Tools() []agent.Tool { return s.tools }

// blocking
func (s *mcpServer) Run(ctx context.Context) error {

	sess, err := s.gateway.createSession(ctx)
	if err != nil {
		return err
	}

	t, err := extractTools(ctx, sess)
	if err != nil {
		return err
	}
	s.tools = t

	// observer
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go monitorSession(ctx, sess)

	// shutdowner
	go func() {
		<-s.shutdownCh
		sess.Close()
	}()

	return sess.Wait()
}

func (s *mcpServer) Shutdown() {
	close(s.shutdownCh)
}

func monitorSession(ctx context.Context, sess *mcp.ClientSession) {
	ticker := time.NewTicker(healthInterval)
	defer ticker.Stop()
	defer sess.Close()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if sess == nil {
				return
			}
			if err := sess.Ping(ctx, &mcpsdk.PingParams{}); err != nil {
				// swallow ping error. cause it always dissconeciton
				return
			}
		}
	}
}

func extractTools(ctx context.Context, session *mcpsdk.ClientSession) ([]agent.Tool, error) {
	toolsResult, err := session.ListTools(ctx, &mcpsdk.ListToolsParams{})
	if err != nil {
		return nil, fmt.Errorf("list tools: %w", err)
	}

	agtTools := make([]agent.Tool, len(toolsResult.Tools))
	for i, t := range toolsResult.Tools {
		agtTools[i] = mcptoolToInternal(t, session)
	}

	return agtTools, nil
}

func createGateway(cfg ServerGatewayConfig) (gateway, error) {
	switch {
	case cfg.HTTPGateway != nil:
		return newHTTPGateway(
			cfg.HTTPGateway.URL,
			cfg.HTTPGateway.Token,
		), nil
	case cfg.CommandGateway != nil:
		return newProcessGateway(
			cfg.CommandGateway.Command,
			cfg.CommandGateway.Args,
			cfg.CommandGateway.Env,
		)
	default:
		return nil, fmt.Errorf("config has no gateway")
	}
}
