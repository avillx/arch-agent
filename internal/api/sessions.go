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

func (h *sessionHandler) Get(w http.ResponseWriter, r *http.Request) error {

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
		return err
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

	return respond(w, http.StatusOK, dto)
}

func (h *sessionHandler) List(w http.ResponseWriter, r *http.Request) error {
	agentID := r.PathValue("agent")

	sessions, err := h.sessSvc.List(agent.ID(agentID))
	if err != nil {
		if errors.Is(err, types.ErrIsNotExist) {
			return badRequest("is not exist")
		}
		return err
	}

	return respond(w, http.StatusOK, map[string]any{
		"sessions": sessions,
	})
}

func (h *sessionHandler) Create(w http.ResponseWriter, r *http.Request) error {
	agentID := r.PathValue("agent")

	sessID, err := h.sessSvc.Create(agent.ID(agentID))
	if err != nil {
		if errors.Is(err, types.ErrIsNotExist) {
			return badRequest("is not exist")
		}
		return err
	}

	return respond(w, http.StatusCreated, map[string]any{
		"id": sessID,
	})
}

func (h *sessionHandler) Delete(w http.ResponseWriter, r *http.Request) error {
	agentID := r.PathValue("agent")
	sessID := r.PathValue("session_id")

	if err := h.sessSvc.Delete(agent.ID(agentID), session.ID(sessID)); err != nil {
		return err
	}

	w.WriteHeader(http.StatusOK)
	return nil
}
