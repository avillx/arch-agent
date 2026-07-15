package api

import (
	"arch-agent/internal/chat"
	"arch-agent/internal/session"
	"arch-agent/internal/task"
	"net/http"
)

func NewServer(taskSvc *task.Service, chatSvc *chat.Service, sessSvc *session.Service) http.Handler {
	h := http.NewServeMux()

	// tasks
	taskHandler := &TaskHandler{taskSvc: taskSvc}
	h.HandleFunc("GET /task/all", wrap(taskHandler.List))
	h.HandleFunc("POST /task/{name}", wrap(taskHandler.Create))
	h.HandleFunc("DELETE /task/{name}", wrap(taskHandler.Delete))

	chatHandler := &chatHandler{chatSvc: chatSvc}
	h.HandleFunc("POST /chat", wrap(chatHandler.Chat))

	sessHandler := &sessionHandler{sessSvc: sessSvc}
	h.HandleFunc("POST /session/{agent}", wrap(sessHandler.Create))
	h.HandleFunc("GET /session/{agent}", wrap(sessHandler.List))
	h.HandleFunc("GET /session/{agent}/{session_id}", wrap(sessHandler.Get))
	h.HandleFunc("DELETE /session/{agent}/{session_id}", wrap(sessHandler.Delete))

	// api v1 route
	v1 := http.NewServeMux()
	v1.Handle("/api/v1/", http.StripPrefix("/api/v1", h))

	return v1
}
