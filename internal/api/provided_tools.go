package api

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/runtime"
	"arch-agent/internal/types"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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
	pubURL  string
	waiters map[string]chan ProvidedToolResultDTO
	mu      sync.RWMutex
}

// POST /toolresult/{id}
func (h *providedToolsRouter) ResolveCall(w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("id")

	res, err := decode[ProvidedToolResultDTO](r)
	if err != nil {
		return badRequest("invalid json")
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	resCh, ok := h.waiters[id]
	if !ok {
		return badRequest("tool id is not exist")
	}

	resCh <- res

	return nil
}

type unregisterFunc func()

const (
	toolResultEndpoint = "toolresult"
)

func (h *providedToolsRouter) registerToolServer(t *ProvidedTool) unregisterFunc {
	id := createID()

	resultURL, _ := url.JoinPath(h.pubURL, apiPrefix, toolResultEndpoint, id)
	t.SetLink(resultURL)

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
var _ runtime.Instructed = (*ProvidedToolServerInstructed)(nil)

type ProvidedToolServerInstructed struct {
	ProvidedToolServer
	instuction string
}

func (s *ProvidedToolServerInstructed) Instruction() string {
	return s.instuction
}

// tool
var _ agent.Tool = (*ProvidedTool)(nil)

type ProvidedTool struct {
	name        string
	description string
	scheme      map[string]any

	stream   *Stream
	link     string
	resultCh chan ProvidedToolResultDTO
}

func NewProvidedTool(s *Stream, dto ProvidedToolDTO) *ProvidedTool {
	return &ProvidedTool{
		name:        dto.Name,
		description: dto.Description,
		scheme:      dto.Schema,
		stream:      s,
		resultCh:    make(chan ProvidedToolResultDTO),
	}
}

func (t *ProvidedTool) Chan() chan ProvidedToolResultDTO { return t.resultCh }
func (t *ProvidedTool) SetLink(l string)                 { t.link = l }

func (t *ProvidedTool) Name() agent.ToolName { return agent.ToolName(t.name) }
func (t *ProvidedTool) Description() string  { return t.description }
func (t *ProvidedTool) Schema() any          { return t.scheme }

func (t *ProvidedTool) Call(ctx context.Context, args agent.ToolArguments) ([]agent.ContentPart, error) {

	type ProvidedToolCall struct {
		Name string         `json:"tool"`
		Args map[string]any `json:"args,omitempty"`

		// link for await result
		ResultLink string `json:"result_link,omitempty"`
		AgentID    string `json:"agent_id"`
		SessionID  string `json:"session_id"`
	}

	var parsedArgs map[string]any

	if args != nil {
		if err := json.Unmarshal(args, &parsedArgs); err != nil {
			return nil, types.NewAgentMistakeError("invalid parameters, broken json")
		}
	}

	agentID, _ := runtime.AgentIDFromContext(ctx)
	sessionID, _ := runtime.SessionIDFromContext(ctx)

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	t.stream.send(Envelope{
		Type: ProvidedCall,
		Payload: ProvidedToolCall{
			Name:       t.name,
			Args:       parsedArgs,
			ResultLink: t.link,
			AgentID:    string(agentID),
			SessionID:  string(sessionID),
		},
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
