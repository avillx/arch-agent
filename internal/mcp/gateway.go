package mcp

import (
	"arch-agent/internal/types"
	"context"
	"errors"
	"fmt"
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

	mu sync.Mutex
}

func newHTTPGateway(url string, authToken string) *httpGateway {
	return &httpGateway{
		url:       url,
		authToken: authToken,
	}
}

// gateway implementation
func (g *httpGateway) createSession(ctx context.Context) (*mcp.ClientSession, error) {

	g.mu.Lock()
	defer g.mu.Unlock()

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

	return cs, nil
}

var ErrNPMRequired = errors.New("npm required")

type processGateway struct {
	command string
	args    []string
	env     map[string]string

	mu sync.Mutex
}

func newBinaryProcessGateway(command string, args []string, env map[string]string) (*processGateway, error) {

	_, err := exec.LookPath(command)
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return nil, fmt.Errorf("command %s: %w", command, types.ErrIsNotExist)
		}
		return nil, fmt.Errorf("%s: %w", command, err)
	}

	return &processGateway{
		command: command,
		args:    args,
		env:     env,
	}, nil
}

// gateway implementation
func (g *processGateway) createSession(ctx context.Context) (*mcp.ClientSession, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

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
