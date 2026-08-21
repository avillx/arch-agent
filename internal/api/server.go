package api

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/chat"
	"arch-agent/internal/mcp"
	"arch-agent/internal/memory"
	"arch-agent/internal/model"
	"arch-agent/internal/session"
	"arch-agent/internal/task"
	"arch-agent/internal/tools"
	"log/slog"
	"net/http"
)

func NewServer(
	taskSvc *task.Service,
	chatSvc *chat.Dispatcher,
	sessSvc *session.Service,
	toolsSvc *tools.Service,
	mcpSvc *mcp.Service,
	memoryRepo agent.MemoryRepo,
	memoryIndexer agent.MemoryIndexer,
	memorySvc *memory.Memory,
	activityStore activityStore,
	agentRepo agent.Repo,
	providerSvc *model.ProviderService,
) *http.ServeMux {

	h := NewAPIResponder(slog.Default())

	taskHandler := &taskHandler{taskSvc: taskSvc}
	h.HandleFunc("GET /task/all", taskHandler.List)
	h.HandleFunc("POST /task/{name}", taskHandler.Create)
	h.HandleFunc("DELETE /task/{name}", taskHandler.Delete)
	h.HandleFunc("PATCH /task/{name}", taskHandler.Patch)

	sessHandler := &sessionHandler{sessSvc: sessSvc}
	h.HandleFunc("POST /session/{agent}", sessHandler.Create)
	h.HandleFunc("GET /session/{agent}", sessHandler.List)
	h.HandleFunc("GET /session/{agent}/{session_id}", sessHandler.Get)
	h.HandleFunc("DELETE /session/{agent}/{session_id}", sessHandler.Delete)

	toolsHandler := &toolsHandler{toolSvc: toolsSvc}
	h.HandleFunc("GET /tools", toolsHandler.List)

	mcpHandler := &mcpHandler{mcpSvc: mcpSvc}
	h.HandleFunc("PATCH /mcp/reload", mcpHandler.Reload)
	h.HandleFunc("POST /mcp/disconnect/{id}", mcpHandler.Disconnect)
	h.HandleFunc("POST /mcp/connect", mcpHandler.Connect)
	h.HandleFunc("GET /mcp/list", mcpHandler.List)

	memoryHandler := NewMemoryHandler(memorySvc, memoryIndexer, memoryRepo)
	h.HandleFunc("POST /memory/{agent}/consolidate", memoryHandler.Consolidate)
	h.HandleFunc("GET /memory/{agent}/{memory_name}", memoryHandler.Get)
	h.HandleFunc("GET /memory/{agent}", memoryHandler.List)

	provToolHandler := NewProvidedToolsRouter()
	h.HandleFunc("POST /toolresult/{id}", provToolHandler.ResolveCall)

	chatHandler := &chatHandler{provToolRegister: provToolHandler, chatDispatcher: chatSvc}
	h.HandleFunc("POST /chat", chatHandler.Chat)
	h.HandleFunc("POST /chat/{agent}/{session}/interrupt", chatHandler.Interrupt)

	activityHandler := &activityHandler{store: activityStore}
	h.HandleFunc("GET /activity", activityHandler.Activity)

	agentHandler := &agentHandler{repo: agentRepo}
	h.HandleFunc("GET /agent/list", agentHandler.List)
	h.HandleFunc("GET /agent/{id}", agentHandler.Read)
	h.HandleFunc("POST /agent/{id}", agentHandler.Create)
	h.HandleFunc("PUT /agent/{id}", agentHandler.Update)
	h.HandleFunc("DELETE /agent/{id}", agentHandler.Delete)

	providerHandler := &providerHandler{providerSvc: providerSvc}
	h.HandleFunc("GET /providers", providerHandler.GetAll)
	h.HandleFunc("GET /providers/{name}", providerHandler.GetProvider)
	h.HandleFunc("POST /providers", providerHandler.AddProvider)
	h.HandleFunc("PATCH /providers/{name}", providerHandler.UpdateProvider)
	h.HandleFunc("DELETE /providers/{name}", providerHandler.DeleteProvider)
	h.HandleFunc("DELETE /providers/{name}/models/{model}", providerHandler.DeleteModel)
	h.HandleFunc("POST /providers/{name}/models/{model}", providerHandler.SetModel)

	return v1
}
