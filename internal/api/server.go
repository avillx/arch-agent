package api

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/chat"
	"arch-agent/internal/mcp"
	"arch-agent/internal/memory"
	"arch-agent/internal/session"
	"arch-agent/internal/task"
	"arch-agent/internal/tools"
	"net/http"
)

func NewServer(
	addr string,
	taskSvc *task.Service,
	chatSvc *chat.Service,
	sessSvc *session.Service,
	toolsSvc *tools.Service,
	mcpSvc *mcp.Service,
	memoryRepo agent.MemoryRepo,
	memoryIndexer agent.MemoryIndexer,
	memorySvc *memory.Memory,
) *http.Server {
	h := http.NewServeMux()

	taskHandler := &taskHandler{taskSvc: taskSvc}
	h.HandleFunc("GET /task/all", wrap(taskHandler.List))
	h.HandleFunc("POST /task/{name}", wrap(taskHandler.Create))
	h.HandleFunc("DELETE /task/{name}", wrap(taskHandler.Delete))
	h.HandleFunc("PATCH /task/{name}", wrap(taskHandler.Patch))

	sessHandler := &sessionHandler{sessSvc: sessSvc}
	h.HandleFunc("POST /session/{agent}", wrap(sessHandler.Create))
	h.HandleFunc("GET /session/{agent}", wrap(sessHandler.List))
	h.HandleFunc("GET /session/{agent}/{session_id}", wrap(sessHandler.Get))
	h.HandleFunc("DELETE /session/{agent}/{session_id}", wrap(sessHandler.Delete))

	toolsHandler := &toolsHandler{toolSvc: toolsSvc}
	h.HandleFunc("GET /tools", wrap(toolsHandler.List))

	mcpHandler := &mcpHandler{mcpSvc: mcpSvc}
	h.HandleFunc("PATCH /mcp/reload", wrap(mcpHandler.Reload))
	h.HandleFunc("POST /mcp/disconnect/{id}", wrap(mcpHandler.Disconnect))
	h.HandleFunc("POST /mcp/connect", wrap(mcpHandler.Connect))
	h.HandleFunc("GET /mcp/list", wrap(mcpHandler.List))

	memoryHandler := &memoryHandler{memorySvc: memorySvc, memoryIndexer: memoryIndexer, memoryRepo: memoryRepo}
	h.HandleFunc("POST /memory/{agent}/consolidate", wrap(memoryHandler.Consolidate))
	h.HandleFunc("GET /memory/{agent}/{memory_name}", wrap(memoryHandler.Get))
	h.HandleFunc("GET /memory/{agent}", wrap(memoryHandler.List))

	chatHandler := &chatHandler{addr: addr, chatSvc: chatSvc}
	h.HandleFunc("POST /chat", wrap(chatHandler.Chat))

	// api v1 route
	v1 := http.NewServeMux()
	v1.Handle("/api/v1/", http.StripPrefix("/api/v1", h))

	return &http.Server{
		Addr:    addr,
		Handler: v1,
	}
}
