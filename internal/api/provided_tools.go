package api

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/chat"
	"arch-agent/internal/types"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type ProvidedToolResultDTO struct {
	Result []agent.ContentPart `json:"result"`

	// error message for agent
	ErrMessage string `json:"error_message"`
}

type ProvidedToolDTO struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Schema      map[string]any `json:"schema,omitempty"`
}

type ProvidedToolServerDTO struct {
	ProvidedTools []ProvidedToolDTO `json:"tools"`
	Instruction   string            `json:"instruction,omitempty"`
}

type providedToolsRouter struct {
	waiters map[string]chan ProvidedToolResultDTO
	mu      sync.RWMutex
}

func NewProvidedToolsRouter() *providedToolsRouter {
	return &providedToolsRouter{
		waiters: map[string]chan ProvidedToolResultDTO{},
	}
}

// POST /toolresult/{id}
func (h *providedToolsRouter) ResolveCall(w http.ResponseWriter, r *http.Request) Response {
	id := r.PathValue("id")

	res, err := decode[ProvidedToolResultDTO](r)
	if err != nil {
		return NewBadRequest("invalid json")
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	resCh, ok := h.waiters[id]
	if !ok {
		return NewBadRequest("tool id is not exist")
	}

	resCh <- res
	return NewResponse(http.StatusOK)
}

type unregisterFunc func()

func (h *providedToolsRouter) registerToolServer(t *ProvidedTool) unregisterFunc {
	id := createID()

	t.SetID(id)

	h.mu.Lock()
	defer h.mu.Unlock()

	resCh := t.Chan()
	h.waiters[id] = resCh

	return func() {
		h.mu.Lock()
		defer h.mu.Unlock()

		if r, ok := h.waiters[id]; ok && r == resCh {
			delete(h.waiters, id)
		}
	}
}

func createID() string {
	return fmt.Sprintf("%x%x", time.Now().Nanosecond(), time.Now().UnixMilli())
}

// server
var _ agent.ToolServer = (*ProvidedToolServer)(nil)

type ProvidedToolServer struct {
	tools []agent.Tool
}

func (s *ProvidedToolServer) Tools() []agent.Tool {
	return s.tools
}

// instucted wrapper
var _ chat.ToolInstructer = (*ProvidedToolServerInstructed)(nil)

type ProvidedToolServerInstructed struct {
	ProvidedToolServer
	instuction string
}

func (s *ProvidedToolServerInstructed) Instruction() string {
	return s.instuction
}

func newProvidedToolServer(dto ProvidedToolServerDTO) agent.ToolServer {
	tools := []agent.Tool{}
	for _, toolDTO := range dto.ProvidedTools {
		tools = append(tools, NewProvidedTool(toolDTO))
	}

	if dto.Instruction == "" {
		return &ProvidedToolServer{
			tools: tools,
		}
	}

	return &ProvidedToolServerInstructed{
		ProvidedToolServer: ProvidedToolServer{
			tools: tools,
		},
	}
}

func dtoToServers(dtos []ProvidedToolServerDTO) []agent.ToolServer {

	toolServers := []agent.ToolServer{}
	for _, d := range dtos {
		toolServers = append(toolServers, newProvidedToolServer(d))
	}

	return toolServers
}

// tool
var _ agent.Tool = (*ProvidedTool)(nil)

type ProvidedTool struct {
	name        string
	description string
	scheme      map[string]any
	stream      *Stream

	// URL safe unique id
	id       string
	resultCh chan ProvidedToolResultDTO
}

func NewProvidedTool(dto ProvidedToolDTO) *ProvidedTool {
	return &ProvidedTool{
		name:        dto.Name,
		description: dto.Description,
		scheme:      dto.Schema,
		resultCh:    make(chan ProvidedToolResultDTO),
	}
}

func (t *ProvidedTool) Chan() chan ProvidedToolResultDTO { return t.resultCh }
func (t *ProvidedTool) SetID(id string)                  { t.id = id }
func (t *ProvidedTool) SetStream(s *Stream)              { t.stream = s }

func (t *ProvidedTool) Name() agent.ToolName { return agent.ToolName(t.name) }
func (t *ProvidedTool) Description() string  { return t.description }
func (t *ProvidedTool) Schema() any          { return t.scheme }
func (t *ProvidedTool) Call(ctx context.Context, args agent.ToolArguments) ([]agent.ContentPart, error) {

	type ProvidedToolCallDTO struct {
		EventTypeDTO

		Name string         `json:"tool"`
		Args map[string]any `json:"args,omitempty"`

		// link for await result
		ResultID  string `json:"result_id,omitempty"`
		AgentID   string `json:"agent_id"`
		SessionID string `json:"session_id"`
	}

	var parsedArgs map[string]any

	if args != nil {
		if err := json.Unmarshal(args, &parsedArgs); err != nil {
			return nil, types.NewAgentMistakeError("invalid parameters, broken json")
		}
	}

	// agentID and sessID guaranteed, so no need to check it
	agentID, _ := chat.AgentIDFromContext(ctx)
	sessionID, _ := chat.SessionIDFromContext(ctx)

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if t.stream == nil {
		return nil, fmt.Errorf("bad provided tool, has no stream to send request")
	}

	t.stream.send(ProvidedToolCallDTO{
		EventTypeDTO: EventTypeDTO{
			Type: ProvidedCall,
		},
		Name:      t.name,
		Args:      parsedArgs,
		ResultID:  t.id,
		AgentID:   string(agentID),
		SessionID: string(sessionID),
	})

	// await result

	select {
	case <-ctx.Done():
		msg := "timeout exceed, if tool can work longer than 30s - system anyway drop this toolcall after 30s"
		return nil, types.NewAgentMistakeError(msg)
	case res := <-t.resultCh:

		var agentErrMsg error
		if res.ErrMessage != "" {
			agentErrMsg = types.NewAgentMistakeError(res.ErrMessage)
		}

		return res.Result, agentErrMsg
	}
}
