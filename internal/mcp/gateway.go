package mcp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/oauth2"
)

var _ gateway = (*httpGateway)(nil)
var _ gateway = (*processGateway)(nil)

type httpGateway struct {
	url       string
	authToken string

	cs *mcp.ClientSession
	mu sync.Mutex
}

func newHTTPGateway(url string, authToken string) *httpGateway {
	return &httpGateway{
		url:       url,
		authToken: authToken,
	}
}

// gateway implementation
func (g *httpGateway) GetSession(ctx context.Context) (*mcp.ClientSession, error) {

	g.mu.Lock()
	defer g.mu.Unlock()

	if g.cs != nil {
		return g.cs, nil
	}

	httpClient := http.DefaultClient

	if g.authToken != "" {
		httpClient = oauth2.NewClient(ctx, oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: g.authToken, TokenType: "Bearer"},
		))
	}

	client := newClient()
	cs, err := client.Connect(
		ctx,
		&mcp.StreamableClientTransport{
			Endpoint:   g.url,
			HTTPClient: httpClient,
		},
		&mcp.ClientSessionOptions{},
	)
	if err != nil {
		return nil, err
	}

	go func() {
		if err := cs.Wait(); err != nil {
			slog.Error("mcp: http gateway", "URL", g.url, "error", err)
		}
		g.mu.Lock()
		defer g.mu.Unlock()

		g.cs = nil
	}()

	g.cs = cs

	return cs, err
}

var ErrNPMRequired = errors.New("npm required")

type processGateway struct {
	command string
	args    []string
	env     map[string]string

	mu sync.Mutex
	cs *mcp.ClientSession
}

func newNPMProcessGateway(pkg string, args []string, env map[string]string) (*processGateway, error) {
	npxPath, err := exec.LookPath("npx")
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return nil, fmt.Errorf("npm gateway: %w", ErrNPMRequired)
		}
		return nil, fmt.Errorf("npm gateway: %w", err)
	}
	return &processGateway{
		command: npxPath,
		args:    append([]string{"-y", pkg}, args...),
		env:     env,
	}, nil
}

func newBinaryProcessGateway(path string, args []string, env map[string]string) (*processGateway, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("binary not found at %s: %w", path, err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("%s is a directory, not an executable", path)
	}
	return &processGateway{
		command: path,
		args:    args,
		env:     env,
	}, nil
}

// gateway implementation
func (g *processGateway) GetSession(ctx context.Context) (*mcp.ClientSession, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.cs != nil {
		return g.cs, nil
	}

	cmd := exec.Command(g.command, g.args...)
	cmd.Env = append(os.Environ(), envToSlice(g.env)...)

	client := newClient()
	cs, err := client.Connect(
		ctx,
		&mcp.CommandTransport{
			Command: cmd,
		},
		&mcp.ClientSessionOptions{},
	)
	if err != nil {
		return nil, err
	}

	go func() {
		if err := cs.Wait(); err != nil {
			slog.Error("mcp: process gateway", "command", g.command, "error", err)
		}
		g.mu.Lock()
		defer g.mu.Unlock()

		if err := killProcessGroup(cmd); err != nil {
			slog.Error("mcp: process gateway", "command", g.command, "error", err)
		}

		g.cs = nil
	}()

	g.cs = cs
	return cs, nil
}

// helpers
func envToSlice(env map[string]string) []string {
	out := make([]string, 0, len(env))
	for k, v := range env {
		out = append(out, k+"="+v)
	}
	return out
}

func newClient() *mcp.Client {
	return mcp.NewClient(&mcp.Implementation{
		Name:    "arch-agent",
		Version: "1.0.0",
	},
		&mcp.ClientOptions{},
	)
}
