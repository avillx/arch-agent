package api

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/memory"
	"arch-agent/internal/types"
	"context"
	"errors"
	"net/http"
	"time"
)

var _ types.Validator = RequestDTO{}

type RequestDTO struct {
	Agent agent.ID  `json:"agent"`
	From  time.Time `json:"from"`
	To    time.Time `json:"to"`
}

func (r RequestDTO) Validate(_ context.Context) error {
	problems := map[string]string{}
	if r.Agent == "" {
		problems["agent"] = "agent must specified"
	}

	if r.From.IsZero() {
		problems["from"] = "from is required"
	}

	if !r.To.IsZero() && r.To.Before(r.From) {
		problems["to"] = "'to' must be after 'from'"
	}

	if len(problems) > 0 {
		return types.NewValidationError(problems)
	}

	return nil
}

type activityStore interface {
	GetRange(agent.ID, time.Time, time.Time) ([]agent.ActivityLog, error)
}

type activityHandler struct {
	store       activityStore
	activitySvc *memory.ActivityService
}

func (h *activityHandler) Activity(w http.ResponseWriter, r *http.Request) Response {

	type ActivityDTO struct {
		Date    string `json:"date"`
		Content string `json:"content"`
	}

	request, err := decode[RequestDTO](r)
	if err != nil {
		if problems := types.ResovleValidationProblems(err); len(problems) > 0 {
			return NewInvalidRequest(err)
		}
		return NewBadRequest(err.Error())
	}

	logs, err := h.store.GetRange(request.Agent, request.From, request.To)
	if err != nil {
		if errors.Is(err, types.ErrIsNotExist) {
			return NewNotFound("agent is not found")
		}
		return NewInternalError(err)
	}

	responseDTO := []ActivityDTO{}
	for _, l := range logs {
		responseDTO = append(responseDTO, ActivityDTO{
			Date:    l.Date.Format("2006-01-02"),
			Content: l.Content,
		})
	}

	return NewJSONResponse(http.StatusOK, responseDTO)
}

// GET /acivity/config
func (h *activityHandler) Config(w http.ResponseWriter, r *http.Request) Response {
	cfg := h.activitySvc.Config()
	return NewJSONResponse(http.StatusOK, cfg)
}

// POST /activity/config
func (h *activityHandler) SetConfig(w http.ResponseWriter, r *http.Request) Response {
	newCfg, err := decode[memory.ActivityConfig](r)
	if err != nil {
		return NewBadRequest(err.Error())
	}
	if err := h.activitySvc.SaveConfig(newCfg); err != nil {
		if errors.Is(err, types.ErrIsNotExist) {
			return NewBadRequest(err.Error())
		}
		return NewInternalError(err)
	}
	return NewResponse(http.StatusOK)
}
