package api

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/chat"
	"arch-agent/internal/session"
	"encoding/json"
	"log/slog"
	"net/http"
)

type EventType string

const (
	Error        EventType = "error"
	Complete     EventType = "complete"
	ToolResult   EventType = "tool_result"
	Compaction   EventType = "compaction"
	ProvidedCall EventType = "provided_toolcall"
)

type Envelope struct {
	Type    EventType `json:"type"`
	Payload any       `json:"payload"`
}

type CompletionDTO struct {
	Done      bool          `json:"done"`
	Content   string        `json:"completion"`
	ToolCalls []ToolCallDTO `json:"tool_calls"`
}

type chatHandler struct {
	chatSvc          *chat.Service
	provToolRegister providedToolRegister
}

func (h *chatHandler) Interrupt(w http.ResponseWriter, r *http.Request) error {
	agentID := agent.ID(r.PathValue("agent"))
	sessionID := session.ID(r.PathValue("session"))
	h.chatSvc.Interrupt(sessionID, agentID)
	w.WriteHeader(http.StatusAccepted)
	return nil
}

func (h *chatHandler) Chat(w http.ResponseWriter, r *http.Request) error {

	type RequestDTO struct {
		AgentID             agent.ID                `json:"agent_id"`
		SessionID           session.ID              `json:"session_id"`
		UserRequest         []agent.ContentPart     `json:"user_request"`
		Logging             bool                    `json:"logging,omitempty"`
		AdditionalPrompt    string                  `json:"additional_prompt,omitempty"`
		ProvidedToolServers []ProvidedToolServerDTO `json:"tool_servers,omitempty"`
	}

	stream := newStream(w)
	defer stream.close()

	chatReqDTO, err := decode[RequestDTO](r)
	if err != nil {
		return badRequest("invalid format")
	}

	w.WriteHeader(http.StatusOK)

	toolSevrvers := dtoToServers(chatReqDTO.ProvidedToolServers)
	unregisterProvidedTools := registerProvidedToolServers(
		h.provToolRegister,
		stream,
		toolSevrvers,
	)
	defer unregisterProvidedTools()

	return h.chatSvc.Chat(r.Context(), chat.Request{
		AgentID:             chatReqDTO.AgentID,
		SessionID:           chatReqDTO.SessionID,
		UserMessage:         agent.NewUserMessage(chatReqDTO.UserRequest),
		Logging:             chatReqDTO.Logging,
		ProvidedToolServers: toolSevrvers,
		EventCallbacks:      newEventCallbacks(stream),
	})
}

func newEventCallbacks(stream *Stream) chat.EventCallbacks {

	type ErrorDTO struct {
		Cause     string     `json:"cause"`
		SessionID session.ID `json:"session"`
		AgentID   agent.ID   `json:"agent"`
	}

	type CompactionDTO struct {
		Message string `json:"message"`
		Result  string `json:"result"`
	}

	type ToolResultDTO struct {
		ID     string              `json:"id"`
		Result []agent.ContentPart `json:"result"`
	}

	onError := func(agentID agent.ID, sessID session.ID, err error) {
		data := Envelope{
			Type: Error,
			Payload: ErrorDTO{
				Cause:     err.Error(),
				SessionID: sessID,
				AgentID:   agentID,
			},
		}
		stream.send(data)
	}
	onComplete := func(agentID agent.ID, sessID session.ID, c *agent.Completion) {
		data := Envelope{
			Type: Complete,
			Payload: CompletionDTO{
				Done:      c.Done,
				Content:   c.Content,
				ToolCalls: toolCallsToDTO(c.ToolCalls),
			},
		}
		stream.send(data)
	}
	onToolResult := func(agentID agent.ID, sessID session.ID, tr *agent.ToolResult) {
		data := Envelope{
			Type: ToolResult,
			Payload: ToolResultDTO{
				ID:     tr.ID,
				Result: tr.Result,
			},
		}
		stream.send(data)
	}
	onCompaction := func(agentID agent.ID, sessID session.ID, summary string) {
		data := Envelope{
			Type: Compaction,
			Payload: CompactionDTO{
				Message: "compaction has been proceed",
				Result:  summary,
			},
		}
		stream.send(data)
	}

	return chat.EventCallbacks{
		OnError:      onError,
		OnComplete:   onComplete,
		OnToolResult: onToolResult,
		OnCompaction: onCompaction,
	}
}

type ToolCallDTO struct {
	ID   string         `json:"id"`
	Name agent.ToolName `json:"tool"`
	Args map[string]any `json:"args"`
}

func toolCallsToDTO(calls []*agent.ToolCall) []ToolCallDTO {
	dtos := []ToolCallDTO{}

	for _, c := range calls {
		var args map[string]any
		if err := json.Unmarshal(c.Arguments, &args); err != nil {
			slog.Error("api: bad completion", "error", err)
		}
		dtos = append(dtos, ToolCallDTO{
			ID:   c.ID,
			Name: c.ToolName,
			Args: args,
		})
	}

	return dtos
}

type providedToolRegister interface {
	registerToolServer(t *ProvidedTool) unregisterFunc
}

func registerProvidedToolServers(r providedToolRegister, s *Stream, toolServers []agent.ToolServer) unregisterFunc {

	unregFuncs := []unregisterFunc{}

	if !(len(toolServers) > 0) {
		return func() {}
	}
	for _, ts := range toolServers {
		for _, t := range ts.Tools() {
			if providedTool, ok := t.(*ProvidedTool); ok {
				providedTool.SetStream(s)
				unreg := r.registerToolServer(providedTool)
				unregFuncs = append(unregFuncs, unreg)
			}
		}
	}
	return func() {
		for _, f := range unregFuncs {
			f()
		}
	}
}
