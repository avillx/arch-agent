package api

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/session"
	"arch-agent/internal/types"
	"errors"
	"net/http"
	"time"
)

type SessionDTO struct {
	ID           string          `json:"session_id"`
	Messages     []agent.Message `json:"messages"`
	InputTokens  int64           `json:"input_tokens"`
	OutputTokens int64           `json:"output_tokens"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
	Extras       map[string]any  `json:"extras,omitempty"`
}

func sessionToDTO() {

}

type sessionHandler struct {
	sessSvc *session.Service
}

func (h *sessionHandler) Get(w http.ResponseWriter, r *http.Request) error {
	agentID := r.PathValue("agent")
	sessID := r.PathValue("session_id")

	sess, err := h.sessSvc.Get(agent.ID(agentID), session.ID(sessID))
	if err != nil {
		return err
	}

	return respond(w, http.StatusOK, sess)
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
