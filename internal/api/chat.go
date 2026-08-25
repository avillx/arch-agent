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
	Complete        EventType = "complete"
	CompleteMistake EventType = "complete_mistake"
	ToolResult      EventType = "tool_result"
	ToolError       EventType = "tool_error"
	Compaction      EventType = "compaction"
	ProvidedCall    EventType = "provided_toolcall"
	LoopExit        EventType = "loop_exit"
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

// POST /chat/{agent}/{session}
func (h *chatHandler) Chat(w http.ResponseWriter, r *http.Request) Response {

	agentID := agent.ID(r.PathValue("agent"))
	sessionID := session.ID(r.PathValue("session"))

	type RequestDTO struct {
		UserRequest         []agent.ContentPart     `json:"user_request"`
		Logging             bool                    `json:"logging,omitempty"`
		AdditionalPrompt    string                  `json:"additional_prompt,omitempty"`
		ProvidedToolServers []ProvidedToolServerDTO `json:"tool_servers,omitempty"`
	}

	stream := newStream(w)
	defer stream.close()

	// should send error to stream to avoid superflous
	chatReqDTO, err := decode[RequestDTO](r)
	if err != nil {
		stream.sendError(http.StatusBadRequest, err)
		return nil
	}

	toolSevrvers := dtoToServers(chatReqDTO.ProvidedToolServers)
	unregisterProvidedTools := registerProvidedToolServers(
		h.provToolRegister,
		stream,
		toolSevrvers,
	)
	defer unregisterProvidedTools()

	evCh := make(chan runtime.Event, 16)

	go func() {
		defer close(evCh)
		err = h.chatDispatcher.Chat(r.Context(), chat.Request{
			AgentID:             agentID,
			SessionID:           sessionID,
			UserMessage:         agent.NewUserMessage(chatReqDTO.UserRequest),
			Logging:             chatReqDTO.Logging,
			ProvidedToolServers: toolSevrvers,
			Sink:                evCh,
		})
		if err != nil {
			stream.sendError(http.StatusBadRequest, err)
		}
	}()

	forwardEventsToStream(evCh, stream)
	return nil
}

func forwardEventsToStream(evCh chan runtime.Event, stream *Stream) {

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

	for rawEv := range evCh {
		switch ev := rawEv.(type) {

		// loop exit event
		case *runtime.LoopExitEvent:
			cause := ""
			if err := ev.Err(); err != nil {
				cause = err.Error()
			}
			data := LoopExitDTO{
				EventTypeDTO: EventTypeDTO{
					Type: LoopExit,
				},
				Cause: cause,
			}
			stream.send(data)

		// compaction event
		case *runtime.CompactionEvent:
			data := CompactionDTO{
				EventTypeDTO: EventTypeDTO{
					Type: Compaction,
				},
				// TODO: eliminate this shit
				Message: "compaction has been proceed",
				Result:  ev.Summary(),
			}
			stream.send(data)

		// complete event
		case *runtime.CompleteEvent:
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

		// completion mistake
		case *runtime.CompletionMistakeEvent:
			data := CompletionMistakeDTO{
				EventTypeDTO: EventTypeDTO{
					Type: CompleteMistake,
				},
				Cause: ev.Err().Error(),
			}
			stream.send(data)

		// tool result
		case *runtime.ToolResultEvent:
			tr := ev.Result()
			data := ToolResultDTO{
				EventTypeDTO: EventTypeDTO{
					Type: ToolResult,
				},
				ID:     tr.ID,
				Result: tr.Result,
			}
			stream.send(data)

		// tool call error
		case *runtime.ToolCallErrEvent:
			data := ToolErrorDTO{
				EventTypeDTO: EventTypeDTO{
					Type: ToolError,
				},
				Cause:    ev.Err().Error(),
				ToolName: ev.ToolName(),
				Args:     ev.ToolArgs(),
			}
			stream.send(data)
		}
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
	registerTool(t *ProvidedTool) unregisterFunc
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
				unreg := r.registerTool(providedTool)
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
