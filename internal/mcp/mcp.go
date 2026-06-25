package mcp

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/tools"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type MCPServerID string

var _ tools.ToolServer = (*MCPServer)(nil)

type MCPServer struct {
	ID        MCPServerID
	URL       string
	Connected bool

	session      *mcpsdk.ClientSession
	tools        []agent.Tool
	onChanged    func() error
	onDisconnect func()
}

type MCPRepo interface {
	Save([]*MCPServer) error
	Load() ([]*MCPServer, error)
}

func NewMCPServer(opts ...MCPServerOption) (*MCPServer, error) {

	s := &MCPServer{}
	for _, opt := range opts {
		if err := opt(s); err != nil {
			return nil, err
		}
	}

	return s, nil
}

func (s *MCPServer) Name() string { return string(s.ID) }

func (s *MCPServer) Tools() []agent.Tool { return s.tools }

// invoke callback on tools changed, operation not override other callbacks
// append - like behaviour
func (s *MCPServer) OnToolsChanged(fn func() error) {
	if s.onChanged == nil {
		s.onChanged = fn
		return
	}

	unwrappedOnChanged := s.onChanged
	s.onChanged = func() error {
		return errors.Join(fn(), unwrappedOnChanged())
	}
}

// invoke callback on server disconnection, operation not override other callbacks
// append - like behaviour
func (s *MCPServer) OnDisconnect(fn func()) {

	if s.onDisconnect == nil {
		s.onDisconnect = fn
		return
	}

	unwrappedOnDisconnect := s.onDisconnect
	s.onDisconnect = func() {
		fn()
		unwrappedOnDisconnect()
	}
}

func (s *MCPServer) Connect(ctx context.Context) error {

	// session
	sess, err := produceSession(ctx, s.URL)
	if err != nil {
		return err
	}
	if s.session != nil {
		s.session.Close()
	}
	s.session = sess

	// tools
	agtTools, err := extractTools(ctx, sess)
	if err != nil {
		return err
	}
	s.tools = agtTools

	go s.safeMontior()

	s.Connected = true

	return nil
}

func (s *MCPServer) Disconnect() {
	s.tools = nil
	s.onChanged = nil
	s.Connected = false

	if s.onDisconnect != nil {
		s.onDisconnect()
		s.onDisconnect = nil
	}

	if s.session != nil {
		s.session.Close()
		s.session = nil
	}
}

func (s *MCPServer) safeMontior() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s.OnDisconnect(cancel)

	if err := s.monitor(ctx); err != nil {
		slog.Error("mcp: health monitor", "error", err)
	}
}

// blocking
func (s *MCPServer) monitor(ctx context.Context) error {
	ticker := time.NewTicker(healthInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := s.session.Ping(ctx, &mcpsdk.PingParams{}); err != nil {
				s.Disconnect()
				return err
			}
		}
	}
}

type MCPServerOption func(*MCPServer) error

func WithInit(ctx context.Context, url string) MCPServerOption {
	return func(s *MCPServer) error {
		s.URL = url
		sess, err := produceSession(ctx, url)
		if err != nil {
			return err
		}

		initResult := sess.InitializeResult()
		if initResult == nil {
			return fmt.Errorf("has no initial result")
		}

		serverInfo := initResult.ServerInfo
		if serverInfo == nil {
			return fmt.Errorf("has no server info")
		}

		s.ID = MCPServerID(serverInfo.Name)
		return nil
	}
}

func WithState(id MCPServerID, url string, connected bool) MCPServerOption {
	return func(s *MCPServer) error {
		s.ID = id
		s.URL = url
		s.Connected = connected

		return nil
	}
}
