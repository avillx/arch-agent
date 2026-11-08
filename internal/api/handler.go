package api

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/chat"
	"arch-agent/internal/mcp"
	"arch-agent/internal/memory"
	"arch-agent/internal/model"
	"arch-agent/internal/server"
	"arch-agent/internal/session"
	"arch-agent/internal/task"
	"arch-agent/internal/uuid"
	"encoding/json"
	"log/slog"
	"net/http"
)

type HTTPHandler struct {
	*http.ServeMux
	logger           *slog.Logger
	taskSvc          *task.Service
	chatSvc          *chat.Dispatcher
	sessSvc          *session.Service
	mcpSvc           *mcp.Service
	consolidationSvc *memory.ConsolidationService
	providerSvc      *model.ProviderService
	activitySvc      *memory.ActivityService
	memoryRepo       agent.MemoryRepo
	memoryIndexer    agent.MemoryIndexer
	activityStore    agent.ActivityRepo
	agentRepo        agent.Repo
	idGen            IDGenerator
}

func NewHTTPHandler(
	logger *slog.Logger,
	srv *server.Server,
) *HTTPHandler {

	h := &HTTPHandler{
		logger:           logger.WithGroup("api"),
		ServeMux:         http.NewServeMux(),
		taskSvc:          srv.TaskSvc,
		chatSvc:          srv.ChatDispatcherSvc,
		sessSvc:          srv.SessSvc,
		mcpSvc:           srv.MCPSvc,
		consolidationSvc: srv.MemoryConsolidationSvc,
		memoryRepo:       srv.MemoryFiles,
		memoryIndexer:    srv.MemoryFiles,
		activityStore:    srv.ActivityFiles,
		agentRepo:        srv.AgentFiles,
		providerSvc:      srv.ProviderSvc,
		activitySvc:      srv.ActivitySvc,
		idGen:            uuid.NewUUIDGenerator(),
	}

	h.registerRoutes()

	return h
}

func (s *HTTPHandler) registerRoutes() {
	taskHandler := &taskHandler{taskSvc: s.taskSvc}
	s.HandleFunc("GET /task", taskHandler.List)
	s.HandleFunc("POST /task/{name}", taskHandler.Create)
	s.HandleFunc("DELETE /task/{name}", taskHandler.Delete)
	s.HandleFunc("PATCH /task/{name}", taskHandler.Patch)

	sessHandler := &sessionHandler{sessSvc: s.sessSvc}
	s.HandleFunc("GET /session/{agent}", sessHandler.Sessions)
	s.HandleFunc("GET /session/{agent}/{session}", sessHandler.Get)
	s.HandleFunc("POST /session/{agent}", sessHandler.Create)
	s.HandleFunc("DELETE /session/{agent}/{session}", sessHandler.Delete)

	mcpHandler := &mcpHandler{mcpSvc: s.mcpSvc}
	s.HandleFunc("GET /mcp", mcpHandler.List)
	s.HandleFunc("POST /mcp/{mcp}", mcpHandler.Connect)
	s.HandleFunc("DELETE /mcp/{mcp}", mcpHandler.Disconnect)

	memoryHandler := NewMemoryHandler(s.consolidationSvc, s.memoryIndexer, s.memoryRepo, s.logger)
	s.HandleFunc("POST /memory/{agent}/consolidate", memoryHandler.Consolidate)
	s.HandleFunc("GET /memory/{agent}/{memory_name}", memoryHandler.Get)
	s.HandleFunc("GET /memory/{agent}", memoryHandler.List)
	s.HandleFunc("GET /memory/config", memoryHandler.GetConfig)
	s.HandleFunc("POST /memory/config", memoryHandler.SetConfig)

	healthHandler := &healthHandler{}
	s.HandleFunc("GET /health", healthHandler.Health)

	provToolHandler := NewProvidedToolsRouter(s.idGen)
	s.HandleFunc("POST /toolresult/{id}", provToolHandler.ResolveCall)

	chatHandler := NewChatHandler(s.chatSvc, provToolHandler, s.logger)
	s.HandleFunc("POST /chat/{agent}/{session}", chatHandler.Chat)
	s.HandleFunc("POST /chat/{agent}/{session}/interrupt", chatHandler.Interrupt)

	activityHandler := &activityHandler{store: s.activityStore, activitySvc: s.activitySvc}
	s.HandleFunc("GET /activity", activityHandler.Activity)
	s.HandleFunc("GET /activity/config", activityHandler.Config)
	s.HandleFunc("POST /activity/config", activityHandler.SetConfig)

	agentHandler := &agentHandler{repo: s.agentRepo}
	s.HandleFunc("GET /agent", agentHandler.List)
	s.HandleFunc("GET /agent/{id}", agentHandler.Read)
	s.HandleFunc("POST /agent/{id}", agentHandler.Create)
	s.HandleFunc("PUT /agent/{id}", agentHandler.Update)
	s.HandleFunc("DELETE /agent/{id}", agentHandler.Delete)

	providerHandler := &providerHandler{providerSvc: s.providerSvc}
	s.HandleFunc("GET /providers", providerHandler.GetAll)
	s.HandleFunc("GET /providers/{name}", providerHandler.GetProvider)
	s.HandleFunc("POST /providers", providerHandler.AddProvider)
	s.HandleFunc("PATCH /providers/{name}", providerHandler.UpdateProvider)
	s.HandleFunc("DELETE /providers/{name}", providerHandler.DeleteProvider)
	s.HandleFunc("POST /providers/{name}/models/{model}", providerHandler.SetModel)
	s.HandleFunc("DELETE /providers/{name}/models/{model}", providerHandler.DeleteModel)

}

func (s *HTTPHandler) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request) Response) {
	s.ServeMux.HandleFunc(pattern, s.wrapResponse(handler))
}

func (s *HTTPHandler) wrapResponse(h func(w http.ResponseWriter, r *http.Request) Response) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := s.logger.With("method", r.Method, "path", r.URL.Path)
		logger.Info("incoming request")

		response := h(w, r)

		// log error
		if cause, ok := response.(interface{ Err() error }); ok {
			logger.Error("internal", "error", cause.Err())
		}

		// silent case. no response provided
		if response == nil {
			return
		}

		logger.Info("respond", "status", response.StatusCode())

		// respond json
		if jr, ok := response.(JSONResponse); ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(jr.StatusCode())
			enc := json.NewEncoder(w)
			if err := enc.Encode(jr.Content()); err != nil {
				logger.Error("response encoding", "error", err)
			}
			return
		}

		// respond status code
		w.WriteHeader(response.StatusCode())
	}
}
