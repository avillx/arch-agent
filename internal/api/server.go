package api

import (
	"arch-agent/internal/task"
	"net/http"
)

// New
func NewServer(taskSvc *task.Service) http.Handler {
	h := http.NewServeMux()

	// tasks
	taskHandler := &TaskHandler{taskSvc: taskSvc}
	h.HandleFunc("GET /task/all", wrap(taskHandler.list))
	h.HandleFunc("POST /task/{name}", wrap(taskHandler.new))
	h.HandleFunc("PATCH /task/{name}/start", wrap(taskHandler.start))
	h.HandleFunc("PATCH /task/{name}/stop", wrap(taskHandler.stop))
	h.HandleFunc("DELETE /task/{name}", wrap(taskHandler.delete))

	// api v1 route
	v1 := http.NewServeMux()
	v1.Handle("/api/v1/", http.StripPrefix("/api/v1", h))

	return v1
}
