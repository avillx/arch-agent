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
	addr    string
	chatSvc *chat.Service
}

func (h *chatHandler) Chat(w http.ResponseWriter, r *http.Request) error {

	type ErrorDTO struct {
		Cause     string     `json:"cause"`
		SessionID session.ID `json:"session"`
		AgentID   agent.ID   `json:"agent"`
	}

	type RequestDTO struct {
		AgentID             agent.ID                `json:"agent_id"`
		SessionID           session.ID              `json:"session_id"`
		UserRequest         []agent.ContentPart     `json:"user_request"`
		Logging             bool                    `json:"logging,omitempty"`
		AdditionalPrompt    string                  `json:"additional_prompt,omitempty"`
		ProvidedToolServers []ProvidedToolServerDTO `json:"tool_servers,omitempty"`
	}

	type ToolResultDTO struct {
		ID     string              `json:"id"`
		Result []agent.ContentPart `json:"result"`
	}

	type CompactionDTO struct {
		Message string `json:"message"`
		Result  string `json:"result"`
	}

	stream := newStream(w)
	defer stream.done()

	chatReqDTO, err := decode[RequestDTO](r)
	if err != nil {
		return badRequest("invalid format")
	}

	w.WriteHeader(http.StatusOK)

	reader := runtime.EventReader{
		OnError: func(agentID agent.ID, sessID session.ID, err error) {
			stream.send(Envelope{
				Type: Error,
				Payload: ErrorDTO{
					Cause:     err.Error(),
					SessionID: sessID,
					AgentID:   agentID,
				},
			})
		},
		OnComplete: func(agentID agent.ID, sessID session.ID, c *agent.Completion) {
			stream.send(Envelope{
				Type: Complete,
				Payload: CompletionDTO{
					Done:      c.Done,
					Content:   c.Content,
					ToolCalls: toolCallsToDTO(c.ToolCalls),
				},
			})
		},
		OnToolResult: func(agentID agent.ID, sessID session.ID, tr *agent.ToolResult) {
			stream.send(Envelope{
				Type: ToolResult,
				Payload: ToolResultDTO{
					ID:     tr.ID,
					Result: tr.Result,
				},
			})
		},
		OnCompaction: func(agentID agent.ID, sessID session.ID, summary string) {
			stream.send(Envelope{
				Type: Compaction,
				Payload: CompactionDTO{
					Message: "compaction has been proceed",
					Result:  summary,
				},
			})
		},
	}

	evCh := make(chan runtime.Event)
	go reader.Read(evCh)

	return h.chatSvc.Chat(r.Context(), chat.Request{
		AgentID:             chatReqDTO.AgentID,
		SessionID:           chatReqDTO.SessionID,
		UserMessage:         agent.NewUserMessage(chatReqDTO.UserRequest),
		Reader:              reader,
		Logging:             chatReqDTO.Logging,
		Additional:          chatReqDTO.AdditionalPrompt,
		ProvidedToolServers: extractProvidedToolServers(h.addr, stream, chatReqDTO.ProvidedToolServers),
	})
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

func extractProvidedToolServers(addr string, s *Stream, dtos []ProvidedToolServerDTO) []agent.ToolServer {

	providedToolServers := []agent.ToolServer{}

	if !(len(dtos) > 0) {
		return providedToolServers
	}

	for _, dto := range dtos {
		tools := []agent.Tool{}
		for _, dtoTool := range dto.ProvidedTools {
			tools = append(tools, NewProvidedTool(addr, s, dtoTool))
		}

		if dto.Instruction != "" {
			providedToolServers = append(providedToolServers, &ProvidedToolServerInstructed{
				ProvidedToolServer: ProvidedToolServer{
					tools: tools,
				},
				instuction: dto.Instruction,
			})
			continue
		}

		providedToolServers = append(providedToolServers, &ProvidedToolServer{
			tools: tools,
		})
	}
	return providedToolServers
}
