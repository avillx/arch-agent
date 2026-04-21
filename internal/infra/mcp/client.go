package mcpadapter

import (
	"arch-agent/internal/app/types"
	"context"
	"fmt"
	"log/slog"
	"maps"
	"slices"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type WiredToolDefinition struct {
	types.ToolDefinition
	session *mcp.ClientSession
}

type Recivier struct {
	agnetName string
	toolDefs  map[string]WiredToolDefinition
}

func NewMCPRecivier(agentName string) *Recivier {
	return &Recivier{
		agnetName: agentName,
		toolDefs:  map[string]WiredToolDefinition{},
	}
}
func (r *Recivier) AddHTTPServer(endpoint string) error {
	return r.connectServer(&mcp.StreamableClientTransport{Endpoint: endpoint})
}

func (r *Recivier) AddIntenalServer(server *InternalServer) error {
	t1, t2 := mcp.NewInMemoryTransports()
	_, err := server.Connect(context.Background(), t2)
	if err != nil {
		return err
	}
	return r.connectServer(t1)
}

func (r *Recivier) Tools() ([]types.ToolDefinition, error) {
	wiredTools := slices.Collect(maps.Values(r.toolDefs))
	availableTools := []types.ToolDefinition{}
	for _, t := range wiredTools {
		availableTools = append(availableTools, t.ToolDefinition)
	}
	return availableTools, nil
}

func (r *Recivier) ReciveCall(ctx context.Context, call *types.ToolCall) (string, error) {
	if t, ok := r.toolDefs[call.ToolName]; ok {
		callParams, err := toCallToolParams(call)
		if err != nil {
			return "", err
		}

		callParams.SetMeta(map[string]any{
			"agent": r.agnetName,
		})

		result, err := t.session.CallTool(ctx, callParams)
		if err != nil {
			return "", err
		}

		return resultToString(result), nil
	}
	return "", fmt.Errorf("tool %s is not found", call.ToolName)
}

// helpers
func (r *Recivier) connectServer(transport mcp.Transport) error {
	session, err := r.createSession(transport)
	if err != nil {
		return err
	}

	return r.registerSession(session)
}

func (r *Recivier) onSessionClosed(session *mcp.ClientSession) {
	servername := session.InitializeResult().ServerInfo.Name
	go func() {
		err := session.Wait()
		r.cleanUpSessionTools(session)

		if err != nil {
			slog.Error("mcp server has been disconnected", "server", servername, "error", err)
			return
		}
		slog.Info("mcp server has been disconnected", "server", servername)
	}()
}

func (r *Recivier) cleanUpSessionTools(session *mcp.ClientSession) {
	maps.DeleteFunc(r.toolDefs, func(k string, v WiredToolDefinition) bool {
		return v.session == session
	})
}

func (r *Recivier) createSession(transport mcp.Transport) (*mcp.ClientSession, error) {
	client := mcp.NewClient(&mcp.Implementation{Name: "mcp-client", Version: "v1.0.0"}, nil)
	return client.Connect(context.Background(), transport, nil)
}

func (r *Recivier) registerSession(session *mcp.ClientSession) error {
	serverTools, err := pullTools(session)
	if err != nil {
		return err
	}

	r.registerTools(serverTools, session)
	return nil
}

func (r *Recivier) registerTools(list []types.ToolDefinition, session *mcp.ClientSession) {
	for _, def := range list {
		r.toolDefs[def.Name] = WiredToolDefinition{
			ToolDefinition: def,
			session:        session,
		}
	}
}

// TODO: ToolListChanged Hander
// client := mcp.NewClient(impl, &mcp.ClientOptions{
//     ToolListChangedHandler: func(ctx context.Context, req *mcp.ToolListChangedRequest) {
//         c.registerSession(ctx, req.GetSession().(*mcp.ClientSession))
//     },
// })
