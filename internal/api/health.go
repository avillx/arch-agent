package api

import "net/http"

type healthHandler struct{}

func (h *healthHandler) Health(w http.ResponseWriter, r *http.Request) Response {
	return NewResponse(http.StatusOK)
}
