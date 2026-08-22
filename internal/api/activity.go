package api

import (
	"arch-agent/internal/agent"
	"net/http"
	"time"
)

type activityStore interface {
	GetRange(agent.ID, time.Time, time.Time) ([]agent.ActivityLog, error)
}

type activityHandler struct {
	store activityStore
}

func (h *activityHandler) Activity(w http.ResponseWriter, r *http.Request) Response {

	type RequestDTO struct {
		Agent agent.ID  `json:"agent"`
		From  time.Time `json:"from"`
		To    time.Time `json:"to,omitempty"`
	}

	type ActivityDTO struct {
		Date    string `json:"date"`
		Content string `json:"content"`
	}

	request, err := decode[RequestDTO](r)
	if err != nil {
		return NewInternalError(err)
	}

	logs, err := h.store.GetRange(request.Agent, request.From, request.To)
	if err != nil {
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
