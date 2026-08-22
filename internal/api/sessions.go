package api

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/session"
	"arch-agent/internal/types"
	"errors"
	"net/http"
	"time"
)

type MessageDTO struct {
	Role      string              `json:"role"`
	Content   []agent.ContentPart `json:"content"`
	ToolCalls []ToolCallDTO       `json:"tool_calls,omitempty"`
}

func messagesToDTO(msgs []agent.Message) []MessageDTO {
	dtos := []MessageDTO{}
	for _, m := range msgs {
		dto := MessageDTO{
			Role:    string(m.Role()),
			Content: m.Content(),
		}
		if agentMessage, ok := m.(*agent.AgentMessage); ok {
			dto.ToolCalls = toolCallsToDTO(agentMessage.ToolCalls())
		}
		dtos = append(dtos, dto)
	}
	return dtos
}

type sessionHandler struct {
	sessSvc *session.Service
}

func (h *sessionHandler) Get(w http.ResponseWriter, r *http.Request) Response {

	type SessionDTO struct {
		ID           session.ID     `json:"session_id"`
		Messages     []MessageDTO   `json:"messages"`
		InputTokens  int64          `json:"input_tokens"`
		OutputTokens int64          `json:"output_tokens"`
		CreatedAt    time.Time      `json:"created_at"`
		UpdatedAt    time.Time      `json:"updated_at"`
		Extras       map[string]any `json:"extras,omitempty"`
	}

	agentID := agent.ID(r.PathValue("agent"))
	sessID := session.ID(r.PathValue("session_id"))

	sess, err := h.sessSvc.Get(agentID, sessID)
	if err != nil {
		if errors.Is(err, types.ErrIsNotExist) {
			return NewBadRequest("is not exist")
		}

		return NewInternalError(err)
	}

	dto := SessionDTO{
		ID:           sess.ID(),
		InputTokens:  sess.InputTokens(),
		OutputTokens: sess.OutputTokens(),
		CreatedAt:    sess.CreatedAt(),
		UpdatedAt:    sess.UpdatedAt(),
		Extras:       sess.Extras(),
		Messages:     messagesToDTO(sess.Messages()),
	}

	return NewJSONResponse(http.StatusOK, dto)
}

func (h *sessionHandler) List(w http.ResponseWriter, r *http.Request) Response {
	agentID := agent.ID(r.PathValue("agent"))

	sessions, err := h.sessSvc.List(agentID)
	if err != nil {
		if errors.Is(err, types.ErrIsNotExist) {
			return NewBadRequest("is not exist")
		}
		return NewInternalError(err)
	}

	return NewJSONResponse(http.StatusOK, sessions)
}

func (h *sessionHandler) Create(w http.ResponseWriter, r *http.Request) Response {

	type SessionCreateDTO struct {
		Instructon string `json:"instruction,omitempty"`
	}

	agentID := agent.ID(r.PathValue("agent"))

	requestDTO, err := decode[SessionCreateDTO](r)
	if err != nil {
		return NewInvalidRequest(err)
	}

	sessID, err := h.sessSvc.Create(agentID, requestDTO.Instructon)
	if err != nil {
		if errors.Is(err, types.ErrIsNotExist) {
			return NewBadRequest("is not exist")
		}
		return NewInternalError(err)
	}

	dto := map[string]any{
		"id": sessID,
	}

	return NewJSONResponse(http.StatusOK, dto)
}

func (h *sessionHandler) Delete(w http.ResponseWriter, r *http.Request) Response {
	agentID := agent.ID(r.PathValue("agent"))
	sessID := session.ID(r.PathValue("session_id"))

	if err := h.sessSvc.Delete(agentID, sessID); err != nil {
		return NewInternalError(err)
	}

	return NewResponse(http.StatusOK)
}
