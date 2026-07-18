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

// type ProvidedToolDTO struct {
// 	Name string         `json:"name"`
// 	Args map[string]any `json:"args,omitempty"`
// }

// type ProvidedToolServerDTO struct {
// 	ProvidedTools []ProvidedToolDTO `json:"tools"`
// 	Instruction   string            `json:"instruction,omitempty"`
// }

type RequestDTO struct {
	AgentID          agent.ID            `json:"agent_id"`
	SessionID        session.ID          `json:"session_id"`
	UserRequest      []agent.ContentPart `json:"user_request"`
	Logging          bool                `json:"logging,omitempty"`
	AdditionalPrompt string              `json:"additional_prompt,omitempty"`
	// ProvidedToolServers []ProvidedToolServerDTO `json:"tool_servers,omitempty"`
}

type ToolCallDTO struct {
	ToolName string         `json:"tool"`
	Args     map[string]any `json:"args"`
}

type CompletionDTO struct {
	Done      bool          `json:"done"`
	Content   string        `json:"completion"`
	ToolCalls []ToolCallDTO `json:"tool_calls"`
}

type ToolResultDTO struct {
	ID     string              `json:"id"`
	Result []agent.ContentPart `json:"result"`
}

type CompationDTO struct {
	Message string `json:"message"`
	Result  string `json:"result"`
}

type chatHandler struct {
	chatSvc *chat.Service
}

func (h *chatHandler) Chat(w http.ResponseWriter, r *http.Request) error {

	stream := newStream(w)
	defer stream.done()

	chatReqDTO, err := decode[RequestDTO](r)
	if err != nil {
		return badRequest("invalid format")
	}

	w.WriteHeader(http.StatusOK)

	reader := runtime.EventReader{
		OnError: func(agentID agent.ID, sessID session.ID, err error) {
			stream.send(map[string]string{
				"cause":   "problems on completion",
				"agent":   string(agentID),
				"session": string(sessID),
				"msg":     err.Error(),
			})
		},
		OnComplete: func(agentID agent.ID, sessID session.ID, c *agent.Completion) {
			stream.send(CompletionDTO{
				Done:      c.Done,
				Content:   c.Content,
				ToolCalls: toolCallsToDTO(c.ToolCalls),
			})
		},
		OnToolResult: func(agentID agent.ID, sessID session.ID, tr *agent.ToolResult) {
			stream.send(ToolResultDTO{
				ID:     tr.ID,
				Result: tr.Result,
			})
		},
		OnCompaction: func(agentID agent.ID, sessID session.ID, summary string) {
			stream.send(CompationDTO{
				Message: "compaction has been proceed",
				Result:  summary,
			})
		},
	}

	evCh := make(chan runtime.Event)
	go reader.Read(evCh)

	return h.chatSvc.Chat(r.Context(), chat.Request{
		AgentID:     chatReqDTO.AgentID,
		SessionID:   chatReqDTO.SessionID,
		UserMessage: agent.NewUserMessage(chatReqDTO.UserRequest),
		Reader:      reader,
		Logging:     chatReqDTO.Logging,
		Additional:  chatReqDTO.AdditionalPrompt,
	})
}

func toolCallsToDTO(calls []*agent.ToolCall) []ToolCallDTO {
	dtos := []ToolCallDTO{}

	for _, c := range calls {
		var args map[string]any
		if err := json.Unmarshal(c.Arguments, &args); err != nil {
			slog.Error("api: bad comletion", "error", err)
		}
		dtos = append(dtos, ToolCallDTO{
			ToolName: string(c.ToolName),
			Args:     args,
		})
	}

	return dtos
}
