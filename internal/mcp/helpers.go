package mcp

import (
	"arch-agent/internal/agent"
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const DefaultMaxAttempts = 3

type ConnectConfig struct {
	MaxAttempts int
}

func tryConnect(ctx context.Context, srv *MCPServer, maxAttempts int) error {
	var errc error

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if err := srv.Connect(ctx); err == nil {
			return nil
		} else {
			errc = errors.Join(errc, fmt.Errorf("connection attempt %w", err))
		}

		if isClientError(errc) {
			break
		}

		time.Sleep(time.Duration(attempt+1) * time.Second)
	}

	return fmt.Errorf("connect %s failed after %d tries: %w", srv.ID, maxAttempts, errc)
}

func isClientError(err error) bool {
	var coder interface{ StatusCode() int }
	if errors.As(err, &coder) {
		return coder.StatusCode() >= http.StatusBadRequest && coder.StatusCode() < http.StatusInternalServerError
	}
	return false
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

func produceSession(ctx context.Context, url string) (*mcpsdk.ClientSession, error) {
	return newClient().Connect(
		ctx,
		&mcpsdk.StreamableClientTransport{
			Endpoint: url,
		},
		&mcp.ClientSessionOptions{})
}

func newClient() *mcpsdk.Client {
	return mcpsdk.NewClient(&mcpsdk.Implementation{
		Name:    "arch-agent",
		Version: "1.0.0",
	}, nil)
}
