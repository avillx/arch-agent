package api

import (
	"arch-agent/internal/task"
	"net/http"
)

func NewServer(taskSvc *task.Service) http.Handler {
	h := http.NewServeMux()

	// tasks
	taskHandler := &TaskHandler{taskSvc: taskSvc}
	h.HandleFunc("GET /task/all", wrap(taskHandler.List))
	h.HandleFunc("POST /task/{name}", wrap(taskHandler.Create))
	h.HandleFunc("DELETE /task/{name}", wrap(taskHandler.Delete))

	// api v1 route
	v1 := http.NewServeMux()
	v1.Handle("/api/v1/", http.StripPrefix("/api/v1", h))

	return v1
}
