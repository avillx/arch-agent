package api

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/session"
	"arch-agent/internal/types"
	"errors"
	"net/http"
	"time"
)

type SessionHeaderDTO struct {
	ID           session.ID     `json:"session_id"`
	InputTokens  int64          `json:"input_tokens"`
	OutputTokens int64          `json:"output_tokens"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	Extras       map[string]any `json:"extras,omitempty"`
	Cause        string         `json:"error,omitempty"`
}

type SessionDTO struct {
	SessionHeaderDTO
	Messages []MessageDTO `json:"messages"`
}

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

	agentID := agent.ID(r.PathValue("agent"))
	sessID := session.ID(r.PathValue("session"))

	sess, err := h.sessSvc.Get(agentID, sessID)
	if err != nil {
		if errors.Is(err, types.ErrIsNotExist) {
			return NewBadRequest("session is not exist")
		}
		return NewInternalError(err)
	}

	dto := SessionDTO{
		SessionHeaderDTO: sessHeaderToDTO(sess),
		Messages:         messagesToDTO(sess.Messages()),
	}

	return NewJSONResponse(http.StatusOK, dto)
}

func (h *sessionHandler) Sessions(w http.ResponseWriter, r *http.Request) Response {
	agentID := agent.ID(r.PathValue("agent"))

	dtos := []SessionHeaderDTO{}

	sessions, err := h.sessSvc.Sessions(agentID)
	if err != nil {

		// this route reaches only on agent is not exist
		if errors.Is(err, types.ErrIsNotExist) {
			return NewBadRequest("agent is not exist")
		}

		var errBrokenHeaders *session.ErrBrokenHeaders
		// if is not a broken headers then is some the real internal problem
		if !errors.As(err, &errBrokenHeaders) {
			return NewInternalError(err)
		}

		// packing broken headers
		for _, e := range errBrokenHeaders.Errors {
			dtos = append(dtos, SessionHeaderDTO{
				ID:    e.SessID,
				Cause: e.Error(),
			})
		}
	}

	for _, header := range sessions {
		dtos = append(dtos, sessHeaderToDTO(header))
	}

	return NewJSONResponse(http.StatusOK, dtos)
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
			return NewBadRequest("agent is not exist")
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
	sessID := session.ID(r.PathValue("session"))

	if err := h.sessSvc.Delete(agentID, sessID); err != nil {
		if errors.Is(err, types.ErrIsNotExist) {
			return NewBadRequest("session is not exist")
		}
		return NewInternalError(err)
	}

	return NewResponse(http.StatusOK)
}

func sessHeaderToDTO(header session.SessionHeader) SessionHeaderDTO {
	return SessionHeaderDTO{
		ID:           header.ID(),
		InputTokens:  header.InputTokens(),
		OutputTokens: header.OutputTokens(),
		CreatedAt:    header.CreatedAt(),
		UpdatedAt:    header.UpdatedAt(),
		Extras:       header.Extras(),
	}
}
