package api

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/chat"
	"arch-agent/internal/runtime"
	"arch-agent/internal/session"
	"encoding/json"
	"log/slog"
	"net/http"
)

type EventType string

const (
	Error           EventType = "error"
	Complete        EventType = "complete"
	CompleteMistake EventType = "complete_mistake"
	ToolResult      EventType = "tool_result"
	ToolError       EventType = "tool_error"
	Compaction      EventType = "compaction"
	ProvidedCall    EventType = "provided_toolcall"
	LoopExit        EventType = "tool_result"
)

type EventTypeDTO struct {
	Type EventType `json:"type"`
}

type CompletionDTO struct {
	EventTypeDTO
	Done      bool          `json:"done"`
	Content   string        `json:"completion"`
	ToolCalls []ToolCallDTO `json:"tool_calls"`
}

type CompletionMistakeDTO struct {
	EventTypeDTO
	Cause string `json:"error"`
}

type chatHandler struct {
	chatDispatcher   *chat.Dispatcher
	provToolRegister providedToolRegister
}

func (h *chatHandler) Interrupt(w http.ResponseWriter, r *http.Request) Response {
	agentID := agent.ID(r.PathValue("agent"))
	sessionID := session.ID(r.PathValue("session"))
	h.chatDispatcher.Interrupt(sessionID, agentID)
	w.WriteHeader(http.StatusAccepted)
	return NewResponse(http.StatusOK)
}

func (h *chatHandler) Chat(w http.ResponseWriter, r *http.Request) Response {

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
		return NewInvalidRequest(err)
	}

	w.WriteHeader(http.StatusOK)

	toolSevrvers := dtoToServers(chatReqDTO.ProvidedToolServers)
	unregisterProvidedTools := registerProvidedToolServers(
		h.provToolRegister,
		stream,
		toolSevrvers,
	)
	defer unregisterProvidedTools()

	err = h.chatDispatcher.Chat(r.Context(), chat.Request{
		AgentID:             chatReqDTO.AgentID,
		SessionID:           chatReqDTO.SessionID,
		UserMessage:         agent.NewUserMessage(chatReqDTO.UserRequest),
		Logging:             chatReqDTO.Logging,
		ProvidedToolServers: toolSevrvers,
		EventCallbacks:      newEventCallbacks(stream),
	})

	if err != nil {
		return NewInternalError(err)
	}

	return NewResponse(http.StatusOK)
}

func newEventCallbacks(stream *Stream) chat.EventCallbacks {

	type ToolErrorDTO struct {
		EventTypeDTO
		Cause    string              `json:"cause"`
		ToolName agent.ToolName      `json:"tool_name"`
		Args     agent.ToolArguments `json:"args"`
	}

	type CompactionDTO struct {
		EventTypeDTO
		Message string `json:"message"`
		Result  string `json:"result"`
	}

	type ToolResultDTO struct {
		EventTypeDTO
		ID     string              `json:"id"`
		Result []agent.ContentPart `json:"result"`
	}

	type LoopExitDTO struct {
		EventTypeDTO
		Cause string `json:"cause,omitempty"`
	}

	onComplete := func(ev *runtime.CompleteEvent) {
		c := ev.Complete()
		data := CompletionDTO{
			EventTypeDTO: EventTypeDTO{
				Type: Complete,
			},
			Done:      c.Done,
			Content:   c.Content,
			ToolCalls: toolCallsToDTO(c.ToolCalls),
		}
		stream.send(data)
	}

	onCompleteMistake := func(ev *runtime.CompletionMistakeEvent) {
		data := CompletionMistakeDTO{
			EventTypeDTO: EventTypeDTO{
				Type: Complete,
			},
			Cause: ev.Err().Error(),
		}
		stream.send(data)
	}

	onToolResult := func(ev *runtime.ToolResultEvent) {
		tr := ev.Result()
		data := ToolResultDTO{
			EventTypeDTO: EventTypeDTO{
				Type: ToolResult,
			},
			ID:     tr.ID,
			Result: tr.Result,
		}
		stream.send(data)
	}

	onToolErr := func(ev *runtime.ToolCallErrEvent) {
		data := ToolErrorDTO{
			EventTypeDTO: EventTypeDTO{
				Type: CompleteMistake,
			},
			Cause:    ev.Err().Error(),
			ToolName: ev.ToolName(),
			Args:     ev.ToolArgs(),
		}
		stream.send(data)
	}

	onCompaction := func(ev *runtime.CompactionEvent) {
		data := CompactionDTO{
			EventTypeDTO: EventTypeDTO{
				Type: Compaction,
			},
			Message: "compaction has been proceed",
			Result:  ev.Summary(),
		}
		stream.send(data)
	}

	OnLoopExit := func(ev *runtime.LoopExitEvent) {
		data := LoopExitDTO{
			EventTypeDTO: EventTypeDTO{
				Type: LoopExit,
			},
			Cause: ev.Err().Error(),
		}
		stream.send(data)
	}

	return chat.EventCallbacks{
		OnComplete:        onComplete,
		OnCompleteMistake: onCompleteMistake,
		OnToolResult:      onToolResult,
		OnCompaction:      onCompaction,
		OnToolErr:         onToolErr,
		OnLoopExit:        OnLoopExit,
		OnEvent:           func(runtime.Event) {},
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
